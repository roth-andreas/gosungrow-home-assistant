package iSolarCloud

import (
	"fmt"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/getDeviceList"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/WebAppService/getDevicePointAttrs"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api/GoStruct"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api/GoStruct/valueTypes"
	"math"
	"sort"
	"strings"
)

type PlantTopologyDevice struct {
	PsID, PsKey              string
	UUID, UpUUID, DeviceType int64
}
type PlantTopology struct {
	PsID      string
	Devices   map[string]PlantTopologyDevice
	Complete  bool
	Reason    string
	Inventory PlantInventory
}

// PlantInventoryDevice contains independently discovered role and point metadata.
type PlantInventoryDevice struct {
	PsKey, Name string
	DeviceType  int64
	Points      getDevicePointAttrs.Points
}

// PlantInventory remains stable until successful device and point rediscovery.
type PlantInventory struct {
	Devices             map[string]PlantInventoryDevice
	Available, Conflict bool
}

// PlantPVPowerRecord contains safe diagnostic metadata, never a numeric reading.
type PlantPVPowerRecord struct {
	DeviceKey             string
	DeviceType            int64
	PointID, Unit, Reason string
	Valid                 bool
}

// PlantPVPowerResult describes aggregation eligibility, not MQTT publication success.
type PlantPVPowerResult struct {
	Added                                bool
	Source                               string
	Contributors                         int
	Reason                               string
	ExpectedKnown                        bool
	Expected, Received, ValidAC, ValidDC int
	Records                              []PlantPVPowerRecord
	OmittedRecords                       int
}
type plantPowerCandidate struct {
	entry  *api.DataEntry
	value  float64
	valid  bool
	rank   int
	reason string
}
type producerCandidates struct {
	device   PlantTopologyDevice
	ac, dc   *plantPowerCandidate
	received bool
}

var plantNativePVPointOrder = map[string]int{"p83076": 0, "p83076_map": 1, "p83033": 2, "p83002": 3, "plant_power": 4, "pv_power": 5, "solar_power": 6}
var plantACPointOrder = map[string]int{"p24": 0, "inverter_ac_power": 1, "total_active_power": 2, "active_power": 3}
var plantDCPointOrder = map[string]int{"total_dc_power": 0, "dc_power": 1}

// BuildPlantInventories preserves independent discovery, never current telemetry.
func BuildPlantInventories(devices getDeviceList.Devices, points map[string]getDevicePointAttrs.Points) map[string]PlantInventory {
	out := make(map[string]PlantInventory)
	owners := make(map[string]string)
	missingProducerPlant := false
	for _, d := range devices {
		psID, key := strings.TrimSpace(d.PsId.String()), strings.TrimSpace(d.PsKey.String())
		if psID == "" {
			if d.DeviceType.Value() == 1 || d.DeviceType.Value() == 55 {
				missingProducerPlant = true
			}
			continue
		}
		inv := out[psID]
		if inv.Devices == nil {
			inv.Devices = make(map[string]PlantInventoryDevice)
			inv.Available = true
		}
		device := PlantInventoryDevice{PsKey: key, Name: d.DeviceName.String(), DeviceType: d.DeviceType.Value(), Points: points[key]}
		if key == "" {
			if inventoryProducer(device) {
				inv.Conflict = true
			}
		} else if prior, ok := inv.Devices[key]; ok && (prior.DeviceType != device.DeviceType || prior.Name != device.Name) {
			inv.Conflict = true
		} else {
			if owner, exists := owners[key]; exists && owner != psID {
				inv.Conflict = true
				other := out[owner]
				other.Conflict = true
				out[owner] = other
			}
			owners[key] = psID
			inv.Devices[key] = device
		}
		out[psID] = inv
	}
	if missingProducerPlant {
		for psID, inventory := range out {
			inventory.Available = false
			out[psID] = inventory
		}
	}
	return out
}

func BuildPlantTopologies(trees PsTrees) map[string]PlantTopology {
	out := make(map[string]PlantTopology)
	for psID, tree := range trees {
		t := PlantTopology{PsID: psID, Devices: make(map[string]PlantTopologyDevice), Complete: true}
		uuids := make(map[int64]PlantTopologyDevice)
		if len(tree.Devices) == 0 {
			t.Complete = false
			t.Reason = "topology contains no devices"
		}
		for _, raw := range tree.Devices {
			d := PlantTopologyDevice{PsID: raw.PsId.String(), PsKey: raw.PsKey.String(), UUID: raw.UUID.Value(), UpUUID: raw.UpUUID.Value(), DeviceType: raw.DeviceType.Value()}
			if d.PsID == "" {
				d.PsID = psID
			}
			if d.PsID != psID || strings.TrimSpace(d.PsKey) == "" {
				t.Complete = false
				t.Reason = "topology contains a missing key or cross-plant device"
				continue
			}
			if _, ok := t.Devices[d.PsKey]; ok {
				t.Complete = false
				t.Reason = "topology contains duplicate keys"
			}
			if _, ok := uuids[d.UUID]; d.UUID != 0 && ok {
				t.Complete = false
				t.Reason = "topology contains duplicate UUIDs"
			}
			t.Devices[d.PsKey] = d
			if d.UUID != 0 {
				uuids[d.UUID] = d
			}
		}
		for _, d := range t.Devices {
			seen := make(map[int64]bool)
			if d.UUID != 0 {
				seen[d.UUID] = true
			}
			for parent := d.UpUUID; parent != 0; {
				if seen[parent] {
					t.Complete = false
					t.Reason = "topology contains a cycle"
					break
				}
				seen[parent] = true
				p, ok := uuids[parent]
				if !ok {
					t.Complete = false
					t.Reason = "topology contains a dangling parent"
					for otherID, other := range trees {
						if otherID == psID {
							continue
						}
						for _, node := range other.Devices {
							if node.UUID.Value() == parent {
								t.Reason = "topology contains a cross-plant parent"
							}
						}
					}
					break
				}
				parent = p.UpUUID
			}
		}
		out[psID] = t
	}
	return out
}

func powerAlias(s string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(strings.ToLower(strings.TrimSpace(s)), "-", " ")), "_")
}
func powerClass(id string) (string, int, bool) {
	if rank, ok := plantNativePVPointOrder[id]; ok {
		return "native", rank, true
	}
	if rank, ok := plantACPointOrder[id]; ok {
		return "ac", rank, true
	}
	if rank, ok := plantDCPointOrder[id]; ok {
		return "dc", rank, true
	}
	return "", 0, false
}
func forbiddenPowerMetadata(parts ...string) bool {
	text := powerAlias(strings.Join(parts, " "))
	for _, token := range strings.Split(text, "_") {
		switch token {
		case "grid", "meter", "phase", "channel", "l1", "l2", "l3":
			return true
		}
	}
	return false
}
func measurementClass(s GoStruct.MeasurementSource) (string, int, string) {
	if (s.Endpoint != "AppService.queryDeviceList" && s.Endpoint != "discovery") || s.Derived {
		return "", 0, "derived_or_unknown_origin"
	}
	class, rank, known := powerClass(powerAlias(s.PointID))
	named, nrank, nknown := powerClass(powerAlias(s.PointName))
	if !known && nknown {
		class, rank = named, nrank
	}
	if forbiddenPowerMetadata(s.PointID, s.PointName, s.GroupName) {
		return class, rank, "grid_or_phase"
	}
	if known && nknown && class != named {
		return class, rank, "identity_conflict"
	}
	if known {
		return class, rank, ""
	}
	if nknown {
		return named, nrank, ""
	}
	return "", 0, "unrecognized_point"
}
func inventoryProducer(d PlantInventoryDevice) bool {
	switch d.DeviceType {
	case 1, 55:
		return true
	case 7, 11, 22:
		return false
	}
	for _, p := range d.Points {
		s := GoStruct.MeasurementSource{Endpoint: "discovery", PointID: p.Id.String(), PointName: p.Name.String(), GroupName: p.PointGroupName}
		class, _, reason := measurementClass(s)
		if reason == "" && (class == "dc" || class == "ac" && strings.Contains(strings.ToLower(d.Name), "inverter")) {
			return true
		}
	}
	return false
}
func sortedLatestEntries(data *api.DataMap) []*api.DataEntry {
	entries := make([]*api.DataEntry, 0, len(data.Map)+len(data.Measurements))
	for i := range data.Measurements {
		entries = append(entries, &data.Measurements[i])
	}
	for _, key := range data.Sort() {
		if e := data.Map[key].GetEntry(api.LastEntry); e != nil && e.Point != nil {
			entries = append(entries, e)
		}
	}
	return entries
}
func candidateFor(e *api.DataEntry, rank int) *plantPowerCandidate {
	value, valid := plantPowerKilowatts(e)
	reason := ""
	if !valid {
		reason = "invalid_values"
		if e.Value.Unit() != "W" && e.Value.Unit() != "kW" && e.Value.Unit() != "MW" {
			reason = "incompatible_units"
		}
	}
	if !e.Current.Source.NumericValid {
		valid = false
		reason = "invalid_values"
	}
	return &plantPowerCandidate{entry: e, value: value, valid: valid, rank: rank, reason: reason}
}
func preferredPlantCandidate(a, b *plantPowerCandidate) *plantPowerCandidate {
	if a == nil || b.valid && !a.valid || a.valid == b.valid && (b.rank < a.rank || b.rank == a.rank && candidateKey(b) < candidateKey(a)) {
		return b
	}
	return a
}
func candidateKey(c *plantPowerCandidate) string {
	s := c.entry.Current.Source
	return s.PointID + "\x00" + s.Endpoint + "\x00" + c.entry.EndPoint
}

// Conflicting representations of one measurement cannot be used to repair a tier.
func usableMeasurements(entries []*api.DataEntry) ([]*api.DataEntry, map[string]bool) {
	groups := make(map[string][]*api.DataEntry)
	conflicts := make(map[string]bool)
	for _, e := range entries {
		if e.Current == nil {
			continue
		}
		s := e.Current.Source
		if s.Endpoint != "AppService.queryDeviceList" || s.Derived {
			continue
		}
		key := s.PsID + "\x00" + s.PsKey + "\x00" + s.PointID
		groups[key] = append(groups[key], e)
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]*api.DataEntry, 0, len(groups))
	for _, key := range keys {
		copies := groups[key]
		first := copies[0]
		fv, fok := plantPowerKilowatts(first)
		for _, e := range copies[1:] {
			ev, eok := plantPowerKilowatts(e)
			a, b := first.Current.Source, e.Current.Source
			if fok != eok || (fok && fv != ev) || a.NumericValid != b.NumericValid || a.PointName != b.PointName || a.GroupName != b.GroupName || a.DeviceType != b.DeviceType || !a.Timestamp.Equal(b.Timestamp) || a.Unit != b.Unit {
				conflicts[key] = true
			}
		}
		if conflicts[key] {
			for _, e := range copies {
				conflicts[e.EndPoint] = true
			}
		}
		out = append(out, first)
	}
	return out, conflicts
}

// AddCanonicalPlantPVPower evaluates one plant-scoped collection snapshot.
func AddCanonicalPlantPVPower(data *api.DataMap, topology PlantTopology) PlantPVPowerResult {
	result := PlantPVPowerResult{Source: "none", Reason: "inventory_unavailable"}
	if data == nil || topology.PsID == "" {
		return result
	}
	entries, conflicts := usableMeasurements(sortedLatestEntries(data))
	producers := make(map[string]*producerCandidates)
	inv := topology.Inventory
	inventoryConflict := inv.Conflict
	for _, device := range data.MeasurementDevices {
		if device.DeviceType != 1 && device.DeviceType != 55 {
			continue
		}
		known, ok := inv.Devices[device.PsKey]
		if !ok || known.DeviceType != device.DeviceType || device.PsID != "" && device.PsID != topology.PsID {
			inventoryConflict = true
		}
	}
	for key, d := range inv.Devices {
		if !inventoryProducer(d) {
			continue
		}
		td, ok := topology.Devices[key]
		if !ok && topology.Complete {
			topology.Complete = false
			topology.Reason = "topology missing an expected producer"
		}
		if ok && td.DeviceType != d.DeviceType {
			inventoryConflict = true
		}
		td.PsKey = key
		td.DeviceType = d.DeviceType
		producers[key] = &producerCandidates{device: td}
	}
	leaves := producerLeaves(producers, topology)
	result.ExpectedKnown = inv.Available && !inventoryConflict && topology.Complete
	if result.ExpectedKnown {
		result.Expected = len(leaves)
	}
	native, blocked, records, pointConflict := collectPlantMeasurements(entries, conflicts, topology, inv, producers)
	inventoryConflict = inventoryConflict || pointConflict
	result.Records = records
	if inventoryConflict {
		result.ExpectedKnown = false
	}
	for _, p := range leaves {
		if p.received {
			result.Received++
		}
		if p.ac != nil && p.ac.valid {
			result.ValidAC++
		}
		if p.dc != nil && p.dc.valid {
			result.ValidDC++
		}
	}
	sort.Slice(result.Records, func(i, j int) bool {
		a, b := result.Records[i], result.Records[j]
		return fmt.Sprintf("%s\x00%d\x00%s\x00%s\x00%t\x00%s", a.DeviceKey, a.DeviceType, a.PointID, a.Unit, a.Valid, a.Reason) < fmt.Sprintf("%s\x00%d\x00%s\x00%s\x00%t\x00%s", b.DeviceKey, b.DeviceType, b.PointID, b.Unit, b.Valid, b.Reason)
	})
	if len(result.Records) > 100 {
		result.OmittedRecords = len(result.Records) - 100
		result.Records = result.Records[:100]
	}
	if native != nil {
		addPlantPVPowerEntry(data, topology.PsID, native.entry, native.value)
		result.Added = true
		result.Source = "native_plant"
		result.Contributors = 1
		result.Reason = "none"
		return result
	}
	switch {
	case !inv.Available:
		result.Reason = "inventory_unavailable"
	case inventoryConflict:
		result.Reason = "inventory_conflict"
	case !topology.Complete && len(topology.Devices) == 0:
		result.Reason = "topology_unavailable"
	case !topology.Complete:
		result.Reason = "topology_invalid"
	case len(leaves) == 0:
		result.Reason = "no_producers"
	default:
		selected, source, ok := completeProducerTier(leaves, true)
		if !ok {
			selected, source, ok = completeProducerTier(leaves, false)
		}
		if ok {
			total := 0.0
			oldest := selected[0].entry
			for _, c := range selected {
				total += c.value
				if c.entry.Date.Time.Before(oldest.Date.Time) {
					oldest = c.entry
				}
			}
			if !math.IsInf(total, 0) && !math.IsNaN(total) {
				addPlantPVPowerEntry(data, topology.PsID, oldest, total)
				result.Added = true
				result.Source = source
				result.Contributors = len(selected)
				result.Reason = "none"
				return result
			}
			blocked["invalid_values"] = true
		}
		switch {
		case blocked["conflicting_points"]:
			result.Reason = "conflicting_points"
		case result.Received < result.Expected:
			result.Reason = "missing_contributors"
		case blocked["incompatible_units"]:
			result.Reason = "incompatible_units"
		case blocked["invalid_values"]:
			result.Reason = "invalid_values"
		default:
			result.Reason = "incomplete_basis"
		}
	}
	return result
}

// collectPlantMeasurements applies source semantics before selecting numeric candidates.
func collectPlantMeasurements(entries []*api.DataEntry, conflicts map[string]bool, topology PlantTopology, inv PlantInventory, producers map[string]*producerCandidates) (*plantPowerCandidate, map[string]bool, []PlantPVPowerRecord, bool) {
	var native *plantPowerCandidate
	inventoryConflict := false
	var records []PlantPVPowerRecord
	blocked := make(map[string]bool)
	for _, e := range entries {
		s := e.Current.Source
		if s.PsID != topology.PsID {
			continue
		}
		class, rank, reject := measurementClass(s)
		if conflicts[e.EndPoint] {
			reject = "conflicting_points"
		}
		if reject != "" {
			if reject == "identity_conflict" {
				blocked["conflicting_points"] = true
			}
		}
		if class == "" && s.Unit != "W" && s.Unit != "kW" && s.Unit != "MW" {
			continue
		}
		record := PlantPVPowerRecord{DeviceKey: s.PsKey, DeviceType: s.DeviceType, PointID: s.PointID, Unit: s.Unit, Valid: s.NumericValid, Reason: reject}
		if class == "native" && (s.DeviceType == 11 || s.PsKey == topology.PsID || s.PsKey == "") {
			c := candidateFor(e, rank)
			if reject == "" && c.valid {
				native = preferredPlantCandidate(native, c)
			}
			if record.Reason == "" {
				record.Reason = c.reason
			}
		} else if class == "ac" || class == "dc" {
			d, known := inv.Devices[s.PsKey]
			if !known && s.DeviceType != 7 && s.DeviceType != 11 && s.DeviceType != 22 {
				inventoryConflict = true
			}
			if known && d.DeviceType != s.DeviceType {
				inventoryConflict = true
			}
			p := producers[s.PsKey]
			if p != nil {
				p.received = true
				if class == "ac" && d.DeviceType != 1 && d.DeviceType != 55 && !strings.Contains(strings.ToLower(d.Name), "inverter") {
					record.Reason = "not_inverter"
				}
				c := candidateFor(e, rank)
				if record.Reason == "" {
					record.Reason = c.reason
				} else {
					c.valid = false
					c.reason = record.Reason
				}
				if reject == "conflicting_points" {
					blocked["conflicting_points"] = true
				}
				if c.reason != "" {
					blocked[c.reason] = true
				}
				if class == "ac" {
					p.ac = preferredPlantCandidate(p.ac, c)
				} else {
					p.dc = preferredPlantCandidate(p.dc, c)
				}
			}
		}
		record.Valid = s.NumericValid && e.Value.Valid && e.Value.IsNumber() && !math.IsNaN(e.Value.ValueFloat()) && !math.IsInf(e.Value.ValueFloat(), 0)
		if record.Reason == "" {
			record.Reason = "none"
		}
		records = append(records, record)
	}
	return native, blocked, records, inventoryConflict
}

func producerLeaves(producers map[string]*producerCandidates, topology PlantTopology) []*producerCandidates {
	byUUID := make(map[int64]PlantTopologyDevice)
	for _, d := range topology.Devices {
		if d.UUID != 0 {
			byUUID[d.UUID] = d
		}
	}
	parents := make(map[string]bool)
	for _, p := range producers {
		seen := make(map[int64]bool)
		for parent := p.device.UpUUID; parent != 0 && !seen[parent]; {
			seen[parent] = true
			d, ok := byUUID[parent]
			if !ok {
				break
			}
			if _, ok := producers[d.PsKey]; ok {
				parents[d.PsKey] = true
			}
			parent = d.UpUUID
		}
	}
	keys := make([]string, 0, len(producers))
	for key := range producers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	leaves := make([]*producerCandidates, 0, len(keys))
	for _, key := range keys {
		if !parents[key] {
			leaves = append(leaves, producers[key])
		}
	}
	return leaves
}
func completeProducerTier(leaves []*producerCandidates, ac bool) ([]*plantPowerCandidate, string, bool) {
	selected := make([]*plantPowerCandidate, 0, len(leaves))
	for _, p := range leaves {
		c := p.dc
		if ac {
			c = p.ac
		}
		if c == nil || !c.valid {
			return nil, "", false
		}
		selected = append(selected, c)
	}
	if ac {
		return selected, "summed_device_ac", true
	}
	return selected, "summed_device_dc", true
}

func plantPowerKilowatts(entry *api.DataEntry) (float64, bool) {
	if entry != nil && entry.Current != nil && entry.Current.Source.Endpoint != "" && !entry.Current.Source.NumericValid {
		return 0, false
	}
	if entry == nil || entry.Point == nil || !entry.Valid || !entry.Point.Valid || !entry.Value.Valid || !entry.Value.IsNumber() {
		return 0, false
	}
	value := entry.Value.ValueFloat()
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false
	}
	switch strings.TrimSpace(entry.Value.Unit()) {
	case "W":
		value /= 1000
	case "kW":
	case "MW":
		value *= 1000
	default:
		return 0, false
	}
	return value, !math.IsNaN(value) && !math.IsInf(value, 0)
}

func addPlantPVPowerEntry(data *api.DataMap, psID string, source *api.DataEntry, value float64) {
	if source == nil || source.Current == nil {
		return
	}
	current := source.Current.Copy()
	current.Source.Derived = true
	current.DataStructure.Endpoint = GoStruct.NewEndPointPath("virtual", psID, "pv_power")
	current.DataStructure.PointId = "pv_power"
	current.DataStructure.PointName = "Plant PV Power"
	current.DataStructure.PointGroupName = "Plant"
	current.DataStructure.PointDevice = psID
	current.DataStructure.PointUnit = "kW"
	current.DataStructure.PointUpdateFreq = GoStruct.UpdateFreq5Mins
	current.DataStructure.PointValueType = "Power"
	current.DataStructure.ValueType = "Power"
	current.IsOk = true

	if math.Abs(value) <= math.MaxFloat64/1000 {
		value = math.Round(value*1000) / 1000
	}
	unitValue := valueTypes.SetUnitValueFloat("kW", "Power", value)
	unitValue.SetDeviceId(psID)
	current.SetUnitValue(unitValue)
	point := api.CreatePoint(&current, psID)
	point.Id = "pv_power"
	point.Description = "Plant PV Power"
	point.GroupName = "Plant"
	point.Unit = "kW"
	point.UpdateFreq = GoStruct.UpdateFreq5Mins
	point.ValueType = "Power"
	point.Valid = true

	endpoint := fmt.Sprintf("virtual.%s.pv_power", psID)
	data.Add(api.DataEntry{
		Current: &current, EndPoint: endpoint, Point: &point, Parent: api.NewParentDevice(psID),
		Date: source.Date, Value: unitValue, Valid: true,
	})
}
