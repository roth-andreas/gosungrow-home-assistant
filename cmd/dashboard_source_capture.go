package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/roth-andreas/gosungrow-home-assistant/defaults"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api"
)

type normalizedSourceRecord struct {
	Plant, Device, Endpoint, Point, Metric, Unit, Period, Provenance, Rejection string
	NormalizedPoint                                                             string
	OriginalObserved                                                            bool
	Role                                                                        int64
	Timestamp                                                                   time.Time
	Valid                                                                       bool
}

type normalizedSourceCapture struct {
	byPlant map[string][]normalizedSourceRecord
	totals  map[string]int
}

func normalizedSourceRecordKey(r normalizedSourceRecord) string {
	return r.Endpoint + "\x00" + r.Device + "\x00" + r.Point + "\x00" + r.NormalizedPoint + "\x00" + r.Provenance + "\x00" + r.Timestamp.UTC().Format(time.RFC3339Nano) + "\x00" + r.Rejection + "\x00" + r.Unit + "\x00" + r.Period
}

func normalizedSourcePlant(command *CmdMqtt, parent api.ParentDevice) (string, int64) {
	matches := make(map[string]int64)
	add := func(plant string, role int64) {
		if previous, found := matches[plant]; found && previous != role {
			role = 0
		}
		matches[plant] = role
	}
	if command.Client != nil {
		for _, device := range command.Client.SungrowDevices {
			if device.PsKey.Value() == parent.Key || device.PsId.String() == parent.Key {
				add(device.PsId.String(), device.DeviceType.Value())
			}
		}
	}
	for plant, inventory := range command.plantInventories {
		if device, found := inventory.Devices[parent.Key]; found {
			add(plant, device.DeviceType)
		}
	}
	for _, target := range command.dashboardTargets {
		if parent.Key == target.PsID || parent.Key == target.PsKey {
			add(target.PsID, target.DeviceType)
		}
		for _, device := range target.PlantDevices {
			if device.PsKey == parent.Key {
				add(target.PsID, device.DeviceType)
			}
		}
	}
	if len(matches) == 1 {
		for plant, role := range matches {
			return plant, role
		}
	}
	return "not_observed", 0
}

func (c *normalizedSourceCapture) add(command *CmdMqtt, data api.DataMap) {
	entries := make([]*api.DataEntry, 0, len(data.Measurements)+len(data.Map))
	for i := range data.Measurements {
		entries = append(entries, &data.Measurements[i])
	}
	for _, key := range data.Sort() {
		entries = append(entries, data.Map[key].GetEntry(api.LastEntry))
	}
	for _, entry := range entries {
		if entry == nil || entry.Current == nil {
			continue
		}
		source := entry.Current.Source
		original := source.Endpoint != ""
		point := ""
		if entry.Point != nil {
			point = entry.Point.Id
		}
		if !original {
			// General reflection does not retain every raw point identifier.
			// Keep the observed normalized point and make that gap explicit.
			if data.EndPoint != nil {
				source.Endpoint = string(data.EndPoint.GetArea()) + "." + string(data.EndPoint.GetName())
			}
			source.PointID = "not_observed"
			source.PsID, source.DeviceType = normalizedSourcePlant(command, entry.Parent)
			source.PsKey, source.Unit, source.NumericValid = entry.Parent.Key, entry.Value.Unit(), entry.Value.Valid
		}
		if source.PsID == "" {
			source.PsID = "not_observed"
		}
		metric := ""
		for _, definition := range dashboardSourceMetricDefinitions {
			contract, ok := dashboardSemanticContractFor(definition.Metric)
			if !ok || !contract.daily {
				continue
			}
			identity := source.PointID
			if !original {
				identity = point
			}
			if _, matches := dashboardSemanticSource(identity, contract.sources); matches {
				metric = definition.Metric
				break
			}
		}
		if metric == "" {
			switch source.Unit {
			case "W", "kW", "MW":
				metric = "pv_power_review"
			case "Wh", "kWh", "MWh":
				metric = "daily_energy_review"
			default:
				continue
			}
		}
		record := normalizedSourceRecord{Plant: source.PsID, Device: source.PsKey, Endpoint: source.Endpoint, Point: source.PointID, NormalizedPoint: point, OriginalObserved: original, Metric: metric, Unit: source.Unit, Role: source.DeviceType, Timestamp: source.Timestamp, Valid: source.NumericValid, Provenance: "native", Rejection: "not_observed"}
		if !original {
			record.Provenance, record.Rejection = "origin_not_observed", "origin_not_observed"
		}
		if source.Derived {
			record.Provenance = "derived"
		}
		if entry.Point != nil {
			record.Period = entry.Point.UpdateFreq
		}
		if inventory, ok := command.plantInventories[source.PsID]; ok {
			if device, ok := inventory.Devices[source.PsKey]; ok {
				record.Role = device.DeviceType
			}
		}
		switch {
		case original && !source.NumericValid:
			record.Rejection = "invalid_numeric"
		case entry.Point == nil || !entry.Point.Valid:
			record.Rejection = "point_filtered"
		case !command.endpoints.IsOK(entry):
			record.Rejection = "endpoint_filtered"
		case !entry.Value.Valid:
			record.Rejection = "invalid_normalized_value"
		case record.Period == "":
			record.Rejection = "period_not_observed"
		}
		if c.byPlant == nil {
			c.byPlant, c.totals = make(map[string][]normalizedSourceRecord), make(map[string]int)
		}
		c.totals[record.Plant]++
		rows := append(c.byPlant[record.Plant], record)
		sort.SliceStable(rows, func(i, j int) bool { return normalizedSourceRecordKey(rows[i]) < normalizedSourceRecordKey(rows[j]) })
		if len(rows) > 100 {
			rows = rows[:100]
		}
		c.byPlant[record.Plant] = rows
	}
}

func (c *normalizedSourceCapture) write(command *CmdMqtt, attempt uint64) {
	plants := make([]string, 0, len(c.byPlant))
	for plant := range c.byPlant {
		plants = append(plants, plant)
	}
	sort.Strings(plants)
	for _, plant := range plants {
		for _, row := range c.byPlant[plant] {
			// No values, friendly names, device names, serials, or raw payloads.
			command.log.Debug("Dashboard source point: app_version=%s matcher_version=%d observed_at=%s cycle=%d attempt=%d stage=normalized ps_id=%q ps_key=%q endpoint=%q point_id=%q normalized_point=%q original_observed=%t metric=%s role=%d unit=%q period=%q source_timestamp=%s valid=%t provenance=%s filter_reason=%s publication=not_observed ha_observation=not_checked dashboard_binding=not_checked\n",
				defaults.BinaryVersion, dashboardSemanticMatcherVersion, time.Now().UTC().Format(time.RFC3339Nano), command.syncCycle, attempt,
				boundedDashboardSourceField(row.Plant), boundedDashboardSourceField(row.Device), boundedDashboardSourceField(row.Endpoint), boundedDashboardSourceField(row.Point), boundedDashboardSourceField(row.NormalizedPoint), row.OriginalObserved, row.Metric, row.Role, boundedDashboardSourceField(row.Unit), boundedDashboardSourceField(row.Period), dashboardPublicationTime(row.Timestamp), row.Valid, row.Provenance, row.Rejection)
		}
		command.log.Debug("Dashboard source points: cycle=%d attempt=%d ps_id=%q observed=%d omitted=%d\n", command.syncCycle, attempt, boundedDashboardSourceField(plant), c.totals[plant], c.totals[plant]-len(c.byPlant[plant]))
	}
	if len(plants) == 0 {
		command.log.Debug("Dashboard source points: cycle=%d attempt=%d stage=normalized observation=no_relevant_point_observed api_absence=not_proven\n", command.syncCycle, attempt)
	}
}

func boundedDashboardSourceField(value string) string {
	value = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, value)
	if len(value) > 160 {
		return fmt.Sprintf("%s...", value[:160])
	}
	return value
}
