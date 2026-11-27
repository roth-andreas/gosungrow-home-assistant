package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/roth-andreas/gosungrow-home-assistant/cmdHassio"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/getDeviceList"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/queryDeviceList"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/WebAppService/getDevicePointAttrs"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api/GoStruct"
	"log"
	"strings"
	"testing"
	"time"
)

func TestDecodedPVDebugMetadataIsBoundedSortedAndSafe(t *testing.T) {
	transport := &pvTestTransport{}
	c := pvCommand(t, transport)
	c.log.SetLogLevel("debug")
	data := decodedNinePVCommandSnapshot(t, 104, false)
	logs := capturePVCommandLogs(t, func() { _ = c.publishPlantPVPower(&data, false) })
	records := []string{}
	for _, line := range strings.Split(logs, "\n") {
		if strings.Contains(line, "Plant PV aggregation detail:") {
			records = append(records, line[strings.Index(line, "device_key="):])
		}
		if strings.Contains(line, "Plant PV aggregation") {
			for _, secret := range []string{"SYNTHETIC_SECRET", "SYNTHETIC_PRIVATE_SERIAL", "value=", "0.50", "-0.27", "-0.23"} {
				if strings.Contains(line, secret) {
					t.Fatalf("unsafe aggregation metadata: %s", line)
				}
			}
		}
	}
	if len(records) != 100 || !strings.Contains(logs, "omitted=4") {
		t.Fatalf("unbounded diagnostics: records=%d logs=%s", len(records), logs)
	}
	for i := 1; i < len(records); i++ {
		if records[i] < records[i-1] {
			t.Fatalf("unsorted records: %q before %q", records[i-1], records[i])
		}
	}
}
func TestMicroinverterDiscoveredContextAndAutomaticGridRejection(t *testing.T) {
	target := haDashboardTarget{PsID: "100", PsKey: "100_55_1_1", PlantDevices: []dashboardPlantDevice{
		{PsKey: "100", DeviceType: 11}, {PsKey: "100_55_1_1", DeviceType: 55},
	}}
	for _, id := range []string{"sensor.gosungrow_virtual_100_55_1_1_p24", "sensor.gosungrow_querydevicelist_100_55_1_1_p24"} {
		state := dashboardTestState(id, "0", "kW")
		if role := dashboardCandidateDeviceRoleState(target, state); role != "inverter" {
			t.Fatalf("role=%s", role)
		}
		if _, _, _, compatible := dashboardScoreMetricCandidate(target, "pv_power", dashboardMetricProfileFor("pv_power"), state, true); !compatible {
			t.Fatalf("type-55 p24 not eligible: %s", id)
		}
	}
	for _, unit := range []string{"W", "kW", "MW"} {
		for _, value := range []string{"0.50", "-0.27", "-0.23", "0"} {
			state := dashboardTestState("sensor.gosungrow_virtual_100_7_0_0_grid_phase_a_active_power", value, unit)
			if resolved := resolveDashboardMetricEntity(target, "pv_power", []haState{state}, nil, true); resolved != "" {
				t.Fatalf("grid automatically selected: %s", resolved)
			}
		}
	}
}

type pvPublishToken struct {
	mqtt.Token
	failure  error
	complete bool
}

// Successful point rediscovery changes membership atomically; failures retain it.
func TestPVInventoryRediscoveryKeepsPriorMembershipOnFailure(t *testing.T) {
	originalLoader := mqttLoadDevicePoints
	defer func() { mqttLoadDevicePoints = originalLoader }()
	transport := &pvTestTransport{}
	c := pvCommand(t, transport)
	c.plantInventories["100"] = c.plantTopologies["100"].Inventory
	var discovered getDeviceList.ResultData
	if err := json.Unmarshal([]byte(`{"pageList":[{"ps_id":"100","ps_key":"100_55_1_1","device_type":55}]}`), &discovered); err != nil {
		t.Fatal(err)
	}
	c.Client.SungrowDevices = discovered.PageList
	mqttLoadDevicePoints = func(*iSolarCloud.PlantTreeDiscovery) (map[string]getDevicePointAttrs.Points, error) {
		return nil, errors.New("synthetic discovery unavailable")
	}
	if c.refreshPlantInventories() == nil {
		t.Fatal("discovery failure hidden")
	}
	if len(c.plantInventories["100"].Devices) != 9 {
		t.Fatal("failed rediscovery changed expected membership")
	}
	mqttLoadDevicePoints = func(*iSolarCloud.PlantTreeDiscovery) (map[string]getDevicePointAttrs.Points, error) {
		return map[string]getDevicePointAttrs.Points{}, nil
	}
	if err := c.refreshPlantInventories(); err != nil {
		t.Fatal(err)
	}
	if len(c.plantInventories["100"].Devices) != 1 {
		t.Fatal("successful rediscovery did not update membership")
	}
	data := decodedNinePVCommandSnapshot(t, 1, false)
	logs := capturePVCommandLogs(t, func() {
		if err := c.publishPlantPVPower(&data, false); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(logs, "expected=1 received=1") || !strings.Contains(logs, "outcome=published") {
		t.Fatalf("rediscovered membership: %s", logs)
	}
}

func (t pvPublishToken) WaitTimeout(time.Duration) bool { return t.complete }
func (t pvPublishToken) Error() error                   { return t.failure }

type pvPublication struct {
	topic    string
	retained bool
	payload  map[string]any
}
type pvTestTransport struct {
	mqtt.Client
	publications []pvPublication
	failAt       int
	timeout      bool
}

func (m *pvTestTransport) Subscribe(string, byte, mqtt.MessageHandler) mqtt.Token {
	return pvPublishToken{complete: true}
}

func (m *pvTestTransport) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	var decoded map[string]any
	_ = json.Unmarshal([]byte(fmt.Sprint(payload)), &decoded)
	m.publications = append(m.publications, pvPublication{topic, retained, decoded})
	var failure error
	if len(m.publications) == m.failAt {
		failure = errors.New("synthetic broker rejection")
	}
	return pvPublishToken{failure: failure, complete: !m.timeout}
}
func decodedNinePVCommandSnapshot(t *testing.T, count int, phases bool) api.DataMap {
	t.Helper()
	ep := queryDeviceList.Init(api.Web{})
	ep.Request.PsId.SetString("100")
	rows := make([]string, 0, 12)
	for i := 1; i <= count; i++ {
		rows = append(rows, fmt.Sprintf(`{"ps_id":"100","ps_key":"100_55_1_%d","device_type":55,"sn":"SYNTHETIC_PRIVATE_SERIAL","point_data":[{"point_id":"p24","point_name":"Active Power","value":0.2,"unit":"kW","time_stamp":"2026-10-04 10:00:00"}]}`, i))
	}
	rows = append(rows, `{"ps_id":"100","ps_key":"100","device_type":11}`, `{"ps_id":"100","ps_key":"100_64_0_0","device_type":64}`)
	if phases {
		rows = append(rows, `{"ps_id":"100","ps_key":"100_7_0_0","device_type":7,"point_data":[{"point_id":"phase_a","point_name":"Grid Phase A Power","value":0.50,"unit":"kW"},{"point_id":"phase_b","point_name":"Grid Phase B Power","value":-0.27,"unit":"kW"},{"point_id":"phase_c","point_name":"Grid Phase C Power","value":-0.23,"unit":"kW"}]}`)
	}
	if err := json.Unmarshal([]byte(`{"result_data":{"rowCount":9,"pageList":[`+strings.Join(rows, ",")+`]}}`), &ep.Response); err != nil {
		t.Fatal(err)
	}
	data := ep.GetData()
	data.ProcessMap()
	return data
}
func pvCommand(t *testing.T, transport *pvTestTransport) *CmdMqtt {
	t.Helper()
	c := NewCmdMqtt("info")
	c.syncCycle = 17
	c.Client = cmdHassio.New(cmdHassio.Mqtt{ClientId: "test", Host: "127.0.0.1", Port: "1883", Username: "synthetic", Password: "SYNTHETIC_SECRET", EntityPrefix: "gosungrow"}).WithClient(transport)
	c.Client.DeviceName = "GoSungrow"
	_, err := c.Client.SetDeviceConfig("GoSungrow", "100", "100", "Synthetic Plant", "Plant", "Sungrow", "Roof")
	if err != nil {
		t.Fatal(err)
	}
	topology := iSolarCloud.PlantTopology{PsID: "100", Complete: true, Devices: make(map[string]iSolarCloud.PlantTopologyDevice), Inventory: iSolarCloud.PlantInventory{Available: true, Devices: make(map[string]iSolarCloud.PlantInventoryDevice)}}
	for i := 1; i <= 9; i++ {
		key := fmt.Sprintf("100_55_1_%d", i)
		topology.Devices[key] = iSolarCloud.PlantTopologyDevice{PsID: "100", PsKey: key, UUID: int64(i), DeviceType: 55}
		topology.Inventory.Devices[key] = iSolarCloud.PlantInventoryDevice{PsKey: key, DeviceType: 55}
	}
	c.plantTopologies["100"] = topology
	c.endpoints = MqttEndPoints{"queryDeviceList": {Include: []string{"virtual.*"}}}
	return c
}
func capturePVCommandLogs(t *testing.T, run func()) string {
	t.Helper()
	var buffer bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&buffer)
	defer log.SetOutput(previous)
	run()
	return buffer.String()
}
func TestDecodedPVPublicationOutcomesAndCounts(t *testing.T) {
	for _, tc := range []struct {
		name, outcome, reason                   string
		count, failAt                           int
		filtered, unknown, unavailable, timeout bool
	}{
		{name: "published", outcome: "published", reason: "none", count: 9},
		{name: "missing", outcome: "suppressed", reason: "missing_contributors", count: 8},
		{name: "endpoint_filter", outcome: "filtered", reason: "endpoint_filter", count: 9, filtered: true},
		{name: "unknown_parent", outcome: "filtered", reason: "unknown_parent", count: 9, unknown: true},
		{name: "discovery_failure", outcome: "publication_failed", reason: "publication_failed", count: 9, failAt: 1},
		{name: "state_failure", outcome: "publication_failed", reason: "publication_failed", count: 9, failAt: 2},
		{name: "timeout", outcome: "publication_failed", reason: "publication_failed", count: 9, timeout: true},
		{name: "no_inventory", outcome: "suppressed", reason: "inventory_unavailable", count: 9, unavailable: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			transport := &pvTestTransport{failAt: tc.failAt, timeout: tc.timeout}
			c := pvCommand(t, transport)
			if tc.filtered {
				c.endpoints["queryDeviceList"] = MqttEndPoint{Include: []string{"virtual.*"}, Exclude: []string{"virtual.100.pv_power"}}
			}
			if tc.unknown {
				delete(c.Client.MqttDevices, "100")
			}
			if tc.unavailable {
				topology := c.plantTopologies["100"]
				topology.Inventory = iSolarCloud.PlantInventory{}
				c.plantTopologies["100"] = topology
			}
			data := decodedNinePVCommandSnapshot(t, tc.count, true)
			var err error
			logs := capturePVCommandLogs(t, func() { err = c.publishPlantPVPower(&data, false) })
			for _, evidence := range []string{"app_version=", "matcher_version=5", "observed_at=", "unique_id=GoSungrow-virtual-100-pv_power", "discovery_topic=", "state_topic=", "config_publication=", "state_publication=", "ha_observation=not_checked", "dashboard_binding=not_checked"} {
				if !strings.Contains(logs, evidence) {
					t.Fatalf("missing correlation evidence %q: %s", evidence, logs)
				}
			}
			if tc.name == "discovery_failure" && !strings.Contains(logs, "config_publication=failed state_publication=not_attempted") {
				t.Fatal(logs)
			}
			if tc.name == "state_failure" && !strings.Contains(logs, "config_publication=completed state_publication=failed") {
				t.Fatal(logs)
			}
			if strings.Count(logs, "INFO: Plant PV aggregation:") != 1 || !strings.Contains(logs, "outcome="+tc.outcome) || !strings.Contains(logs, "reason="+tc.reason) {
				t.Fatalf("final logs = %s", logs)
			}
			if tc.outcome == "publication_failed" && err == nil {
				t.Fatal("broker failure hidden")
			}
			if tc.outcome != "publication_failed" && err != nil {
				t.Fatal(err)
			}
			expected := "expected=9"
			if tc.unavailable {
				expected = "expected=unknown"
			}
			received := tc.count
			if tc.unavailable {
				received = 0
			}
			if !strings.Contains(logs, expected) || !strings.Contains(logs, fmt.Sprintf("received=%d", received)) {
				t.Fatalf("counts = %s", logs)
			}
			if tc.outcome == "published" {
				if len(transport.publications) != 2 {
					t.Fatalf("publications=%#v", transport.publications)
				}
				config, state := transport.publications[0], transport.publications[1]
				if !config.retained || !state.retained || !strings.HasSuffix(config.topic, "/config") || !strings.HasSuffix(state.topic, "/state") {
					t.Fatal("retained discovery/state pipeline not used")
				}
				if config.payload["unit_of_measurement"] != "kW" || config.payload["device_class"] != "power" || config.payload["state_class"] != "measurement" {
					t.Fatalf("discovery=%#v", config.payload)
				}
				device := config.payload["device"].(map[string]any)
				identifiers := device["identifiers"].([]any)
				if len(identifiers) != 1 || identifiers[0] != "gosungrow-100" {
					t.Fatalf("not plant device: %#v", device)
				}
				value := fmt.Sprint(state.payload["value"])
				if value != "1.8" && value != "1.800" {
					t.Fatalf("state=%#v", state.payload)
				}
			} else if tc.outcome != "publication_failed" && len(transport.publications) != 0 {
				t.Fatalf("suppression published: %#v", transport.publications)
			}
			if _, ok := data.Map["virtual.100.pv_power"]; ok {
				t.Fatal("canonical remains eligible for double publication")
			}
		})
	}
}

func TestCanonicalPVDiscoveryReuseIsNotReportedAsNewPublication(t *testing.T) {
	transport := &pvTestTransport{}
	c := pvCommand(t, transport)
	for _, status := range []string{"completed", "reused_previous_success"} {
		data := decodedNinePVCommandSnapshot(t, 9, false)
		logs := capturePVCommandLogs(t, func() {
			if err := c.publishPlantPVPower(&data, false); err != nil {
				t.Fatal(err)
			}
		})
		if !strings.Contains(logs, "config_publication="+status+" state_publication=completed") || !strings.Contains(logs, "outcome=published") {
			t.Fatal(logs)
		}
	}
	if len(transport.publications) != 3 {
		t.Fatalf("unchanged discovery was republished: %#v", transport.publications)
	}
}

func TestNormalizedSourceCaptureIsCombinedBoundedAndContainsNoMeasurements(t *testing.T) {
	c := pvCommand(t, &pvTestTransport{})
	c.log.SetLogLevel("debug")
	capture := &normalizedSourceCapture{}
	capture.add(c, decodedNinePVCommandSnapshot(t, 104, true))
	capture.add(c, decodedNinePVCommandSnapshot(t, 9, false))
	rows := capture.byPlant["100"]
	if len(rows) != 100 || capture.totals["100"] <= 100 {
		t.Fatalf("capture is not bounded across results: %d/%d", len(rows), capture.totals["100"])
	}
	for i := 1; i < len(rows); i++ {
		if normalizedSourceRecordKey(rows[i-1]) > normalizedSourceRecordKey(rows[i]) {
			t.Fatal("normalized capture is not sorted")
		}
	}
	logs := capturePVCommandLogs(t, func() { capture.write(c, 2) })
	for _, required := range []string{"Dashboard source point:", "app_version=", "matcher_version=5", "cycle=17 attempt=2", "endpoint=", "point_id=", "role=", "period=", "source_timestamp=", "provenance=", "filter_reason=", "publication=not_observed", "omitted="} {
		if !strings.Contains(logs, required) {
			t.Fatalf("missing %q: %s", required, logs)
		}
	}
	for _, forbidden := range []string{"SYNTHETIC_SECRET", "SYNTHETIC_PRIVATE_SERIAL", "value=", "\"sn\"", "access_token", "password"} {
		if strings.Contains(logs, forbidden) {
			t.Fatalf("unsafe normalized capture: %s", forbidden)
		}
	}
}

func TestNormalizedSourceCaptureKeepsOriginGapsExplicit(t *testing.T) {
	c := pvCommand(t, &pvTestTransport{})
	data := decodedNinePVCommandSnapshot(t, 9, false)
	for i := range data.Measurements {
		if data.Measurements[i].Current != nil {
			data.Measurements[i].Current.Source = GoStruct.MeasurementSource{}
		}
	}
	for _, entries := range data.Map {
		if entry := entries.GetEntry(api.LastEntry); entry != nil && entry.Current != nil {
			entry.Current.Source = GoStruct.MeasurementSource{}
		}
	}
	capture := &normalizedSourceCapture{}
	capture.add(c, data)
	found := false
	for _, rows := range capture.byPlant {
		for _, row := range rows {
			found = true
			if row.Point != "not_observed" || row.OriginalObserved || row.Provenance != "origin_not_observed" || row.NormalizedPoint == "" {
				t.Fatalf("normalization fabricated raw provenance: %#v", row)
			}
		}
	}
	if !found {
		t.Fatal("observed normalized points disappeared when raw provenance was missing")
	}
}

func TestSourceCapturePlantLookupPreservesOpaqueIDsAndRejectsAmbiguity(t *testing.T) {
	c := &CmdMqtt{dashboardTargets: []haDashboardTarget{{PsID: "opaque_plant:100", PsKey: "opaque_device:leaf_1", DeviceType: 55}}}
	parent := api.ParentDevice{Key: "opaque_device:leaf_1", PsId: "legacy_truncated_prefix"}
	plant, role := normalizedSourcePlant(c, parent)
	if plant != "opaque_plant:100" || role != 55 {
		t.Fatalf("opaque source identity changed: %s/%d", plant, role)
	}
	c.dashboardTargets = append(c.dashboardTargets, haDashboardTarget{PsID: "other_plant", PsKey: parent.Key, DeviceType: 55})
	if plant, _ := normalizedSourcePlant(c, parent); plant != "not_observed" {
		t.Fatalf("ambiguous source assigned to %q", plant)
	}
}
func TestDecodedPVDiscoveryRetriesAndRetainedStatePreservation(t *testing.T) {
	transport := &pvTestTransport{failAt: 1}
	c := pvCommand(t, transport)
	first := decodedNinePVCommandSnapshot(t, 9, false)
	_ = capturePVCommandLogs(t, func() {
		if c.publishPlantPVPower(&first, false) == nil {
			t.Fatal("failure expected")
		}
	})
	transport.failAt = 0
	restored := decodedNinePVCommandSnapshot(t, 9, false)
	_ = capturePVCommandLogs(t, func() {
		if err := c.publishPlantPVPower(&restored, false); err != nil {
			t.Fatal(err)
		}
	})
	if len(transport.publications) != 3 || !strings.HasSuffix(transport.publications[1].topic, "/config") {
		t.Fatal("failed first discovery was not retried")
	}
	partial := decodedNinePVCommandSnapshot(t, 8, false)
	_ = capturePVCommandLogs(t, func() {
		if err := c.publishPlantPVPower(&partial, false); err != nil {
			t.Fatal(err)
		}
	})
	if len(transport.publications) != 3 {
		t.Fatal("partial snapshot changed retained canonical state")
	}
	topology := c.plantTopologies["100"]
	topology.Complete = false
	topology.Reason = "cycle"
	c.plantTopologies["100"] = topology
	data := decodedNinePVCommandSnapshot(t, 9, false)
	_ = capturePVCommandLogs(t, func() {
		if err := c.publishPlantPVPower(&data, false); err != nil {
			t.Fatal(err)
		}
	})
	if len(transport.publications) != 3 {
		t.Fatal("invalid topology changed retained state")
	}
	// An ordinary entity remains publishable after canonical suppression.
	ordinary := api.NewDataMap()
	ordinary.Map["virtual.100_55_1_1.p24"] = data.Map["virtual.100_55_1_1.p24"]
	_, _ = c.Client.SetDeviceConfig("GoSungrow", "100", "100_55_1_1", "Microinverter", "Microinverter", "Sungrow", "Roof")
	_ = capturePVCommandLogs(t, func() {
		if err := c.Update("queryDeviceList", ordinary, false); err != nil {
			t.Fatal(err)
		}
	})
	if len(transport.publications) != 5 {
		t.Fatal("ordinary discovery/state stopped")
	}
}
func TestDecodedPVPlantDashboardComposition(t *testing.T) {
	data := decodedNinePVCommandSnapshot(t, 9, true)
	transport := &pvTestTransport{}
	c := pvCommand(t, transport)
	_ = capturePVCommandLogs(t, func() {
		if err := c.publishPlantPVPower(&data, false); err != nil {
			t.Fatal(err)
		}
	})
	target := haDashboardTarget{PsID: "100", PsKey: "100_55_1_1"}
	for i := 1; i <= 9; i++ {
		target.PlantDevices = append(target.PlantDevices, dashboardPlantDevice{PsKey: fmt.Sprintf("100_55_1_%d", i), DeviceType: 55})
	}
	target.PlantDevices = append(target.PlantDevices, dashboardPlantDevice{PsKey: "100", DeviceType: 11, DeviceName: "Inverter Plant"}, dashboardPlantDevice{PsKey: "100_64_0_0", DeviceType: 64, DeviceName: "iHomeManager"}, dashboardPlantDevice{PsKey: "100_7_0_0", DeviceType: 7, DeviceName: "Inverter meter"})
	if dashboardTargetInverterCount(target) != 9 {
		t.Fatalf("inverter-like count=%d", dashboardTargetInverterCount(target))
	}
	states := []haState{{EntityID: "sensor.gosungrow_virtual_100_pv_power", RegistryUniqueID: stringValue(transport.publications[0].payload["unique_id"]), State: fmt.Sprint(transport.publications[1].payload["value"]), Attributes: map[string]any{"unit_of_measurement": "kW"}}}
	for _, phase := range []string{"phase_a", "phase_b", "phase_c"} {
		var reading *api.DataEntry
		for i := range data.Measurements {
			if data.Measurements[i].Current.Source.PointID == phase {
				reading = data.Map[data.Measurements[i].EndPoint].GetEntry(api.LastEntry)
			}
		}
		if reading == nil {
			t.Fatalf("original phase identity missing: %s", phase)
		}
		entity := "sensor.gosungrow_" + strings.ReplaceAll(reading.EndPoint, ".", "_")
		states = append(states, haState{EntityID: entity, State: reading.Value.String(), Attributes: map[string]any{"unit_of_measurement": reading.Value.Unit(), "friendly_name": reading.Current.Source.PointName}})
	}
	resolved, _ := resolveDashboardMetricEntityWithTrace(target, "pv_power", "", states, nil, true)
	if resolved != states[0].EntityID {
		t.Fatalf("dashboard PV=%s", resolved)
	}
	config, err := renderDashboardConfig("../addon/gosungrow/assets/home-assistant-sungrow-flow.yaml", "Synthetic Plant", []haDashboardTarget{target}, defaultDashboardLocaleBundle)
	if err != nil {
		t.Fatal(err)
	}
	remapped, report := remapDashboardEntitiesWithReport(config, []haDashboardTarget{target}, states)
	found := false
	for _, trace := range report.Traces {
		if trace.Metric == "pv_power" && trace.Resolved == states[0].EntityID {
			found = true
		}
	}
	if !found {
		t.Fatalf("composed dashboard did not use published aggregate: %#v", report)
	}
	composed, _, sourceErr := applyDashboardSourceMappings(remapped, nil, nil, []haDashboardTarget{target}, states, report.Traces, "gosungrow", defaultDashboardLocaleBundle)
	if sourceErr != nil {
		t.Fatal(sourceErr)
	}
	card := findDashboardSourceMappingCard(composed, dashboardSourceMappingID(target))
	if got := anyMapToStringMap(card["defaults"])["pv_power"]; got != states[0].EntityID {
		t.Fatalf("fresh mapping=%q", got)
	}
	if recommendation := dashboardSemanticRecommendation(target, "pv_power", states, true); recommendation.Entity != states[0].EntityID || !recommendation.Confident {
		t.Fatalf("recommendation=%#v", recommendation)
	}
	for _, phase := range states[1:] {
		if _, _, _, compatible := dashboardScoreMetricCandidate(target, "pv_power", dashboardMetricProfileFor("pv_power"), phase, true); compatible {
			t.Fatalf("grid/phase auto-selected: %#v", phase)
		}
	}
	oldEntity := "sensor.gosungrow_virtual_100_55_1_1_p24"
	oldReading := data.Map["virtual.100_55_1_1.p24"].GetEntry(api.LastEntry)
	states = append(states, dashboardTestState(oldEntity, oldReading.Value.String(), "kW"))
	mappingID := dashboardSourceMappingID(target)
	for _, manual := range []bool{false, true} {
		currentCard := map[string]any{"type": dashboardSourceMappingCardType, "schema_version": 1, "mapping_id": mappingID, "defaults": map[string]any{"pv_power": oldEntity}, "pinned_defaults": map[string]any{"pv_power": oldEntity}}
		if manual {
			currentCard["overrides"] = map[string]any{"pv_power": oldEntity}
		}
		current := map[string]any{"views": []any{map[string]any{"cards": []any{currentCard}}}}
		base, err := renderDashboardConfig("../addon/gosungrow/assets/home-assistant-sungrow-flow.yaml", "Synthetic Plant", []haDashboardTarget{target}, defaultDashboardLocaleBundle)
		if err != nil {
			t.Fatal(err)
		}
		next, nextReport := remapDashboardEntitiesWithReport(base, []haDashboardTarget{target}, states)
		kept, overrides, sourceErr := applyDashboardSourceMappings(next, current, nil, []haDashboardTarget{target}, states, nextReport.Traces, "gosungrow", defaultDashboardLocaleBundle)
		if sourceErr != nil {
			t.Fatal(sourceErr)
		}
		saved := findDashboardSourceMappingCard(kept, mappingID)
		if manual {
			if overrides[mappingID]["pv_power"] != oldEntity {
				t.Fatal("manual PV source was replaced")
			}
		} else if anyMapToStringMap(saved["defaults"])["pv_power"] != oldEntity {
			t.Fatalf("pinned source changed: defaults=%v pins=%v", saved["defaults"], saved["pinned_defaults"])
		}
		if anyMapToStringMap(saved["recommendations"])["pv_power"] != states[0].EntityID {
			t.Fatal("canonical upgrade was not offered")
		}
	}
}

func TestDecodedNativePVPublishesWithoutInventoryOrTopology(t *testing.T) {
	transport := &pvTestTransport{}
	c := pvCommand(t, transport)
	c.plantTopologies = map[string]iSolarCloud.PlantTopology{}
	ep := queryDeviceList.Init(api.Web{})
	ep.Request.PsId.SetString("100")
	if err := json.Unmarshal([]byte(`{"result_data":{"rowCount":null,"pageList":[{"ps_id":"100","device_type":11,"point_data":[{"point_id":"p83076","value":4.5,"unit":"kW"}]}]}}`), &ep.Response); err != nil {
		t.Fatal(err)
	}
	data := ep.GetData()
	data.ProcessMap()
	logs := capturePVCommandLogs(t, func() {
		if err := c.publishPlantPVPower(&data, false); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(logs, "outcome=published source=native_plant expected=unknown") || len(transport.publications) != 2 || fmt.Sprint(transport.publications[1].payload["value"]) != "4.5" {
		t.Fatalf("native publication: %s %#v", logs, transport.publications)
	}
}
