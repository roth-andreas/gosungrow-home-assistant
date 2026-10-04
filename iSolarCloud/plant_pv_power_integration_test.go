package iSolarCloud

import (
	"encoding/json"
	"fmt"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/queryDeviceList"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api/GoStruct/valueTypes"
	"strings"
	"testing"
)

func TestDecodedCandidateFallbackUnitsAndCopies(t *testing.T) {
	for _, tc := range []struct {
		name   string
		rows   []string
		source string
		value  float64
		reason string
	}{
		{"lower_valid_AC_alias", append(ninePVRows()[:8], pvRow("100_55_1_9", 55, "p24", "null", "kW", "", ""), pvRow("100_55_1_9", 55, "active_power", "200", "W", "", "")), "summed_device_ac", 1.8, ""},
		{"readable_alias", append(ninePVRows()[:8], pvRow("100_55_1_9", 55, "unknown", "0.2", "kW", " Total-Active Power ", "")), "summed_device_ac", 1.8, ""},
		{"incompatible_unit", append(ninePVRows()[:8], pvRow("100_55_1_9", 55, "p24", "0.2", "kWh", "", "")), "", 0, "incompatible_units"},
		{"conflicting_API_rows", append(ninePVRows(), pvRow("100_55_1_9", 55, "p24", "0.7", "kW", "", "")), "", 0, "conflicting_points"},
		{"recognized_conflicting_name", append(ninePVRows()[:8], pvRow("100_55_1_9", 55, "p24", "0.2", "kW", "DC Power", "")), "", 0, "conflicting_points"},
		{"recognized_grid_metadata", append(ninePVRows()[:8], pvRow("100_55_1_9", 55, "p24", "0.2", "kW", "", "Grid Phase A")), "", 0, "incomplete_basis"},
		{"unusable_key", append(ninePVRows()[:8], pvRow("", 55, "p24", "0.2", "kW", "", "")), "", 0, "inventory_conflict"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := decodedPVSnapshot(t, tc.rows, "9")
			if tc.source != "" {
				requirePV(t, &data, ninePVTopology(), tc.source, tc.value)
				return
			}
			r := AddCanonicalPlantPVPower(&data, ninePVTopology())
			if r.Added || r.Reason != tc.reason {
				t.Fatalf("candidate = %#v", r)
			}
			if tc.reason != "inventory_conflict" && r.Received != 9 {
				t.Fatalf("recognized measurements = %d, want 9", r.Received)
			}
		})
	}
}

func decodedPVSnapshot(t *testing.T, rows []string, rowCount string) api.DataMap {
	t.Helper()
	ep := queryDeviceList.Init(api.Web{})
	ep.Request.PsId.SetString("100")
	suffix := ""
	if rowCount != "" {
		suffix = ",\"rowCount\":" + rowCount
	}
	if err := json.Unmarshal([]byte("{\"result_data\":{\"pageList\":["+strings.Join(rows, ",")+"]"+suffix+"}}"), &ep.Response); err != nil {
		t.Fatal(err)
	}
	data := ep.GetData()
	data.ProcessMap()
	if data.Error != nil {
		t.Fatal(data.Error)
	}
	return data
}
func pvRow(key string, typ int64, point, value, unit, name, group string) string {
	return fmt.Sprintf(`{"ps_id":"100","ps_key":%q,"device_type":%d,"point_data":[{"point_id":%q,"point_name":%q,"value":%s,"unit":%q,"point_group_name":%q,"time_stamp":"2026-10-04 10:00:00"}]}`, key, typ, point, name, value, unit, group)
}
func ninePVTopology() PlantTopology {
	t := PlantTopology{PsID: "100", Complete: true, Devices: make(map[string]PlantTopologyDevice), Inventory: PlantInventory{Available: true, Devices: make(map[string]PlantInventoryDevice)}}
	for i := 1; i <= 9; i++ {
		key := fmt.Sprintf("100_55_1_%d", i)
		t.Devices[key] = PlantTopologyDevice{PsID: "100", PsKey: key, UUID: int64(i), DeviceType: 55}
		t.Inventory.Devices[key] = PlantInventoryDevice{PsKey: key, DeviceType: 55}
	}
	return t
}
func ninePVRows() []string {
	rows := make([]string, 0, 9)
	for i := 1; i <= 9; i++ {
		rows = append(rows, pvRow(fmt.Sprintf("100_55_1_%d", i), 55, "p24", "0.2", "kW", "Active Power", "Inverter"))
	}
	return rows
}
func requirePV(t *testing.T, data *api.DataMap, topology PlantTopology, source string, value float64) PlantPVPowerResult {
	t.Helper()
	r := AddCanonicalPlantPVPower(data, topology)
	if !r.Added || r.Source != source {
		t.Fatalf("aggregation = %#v", r)
	}
	e := data.Map["virtual.100.pv_power"].GetEntry(api.LastEntry)
	if e.Value.ValueFloat() != value || e.Value.Unit() != "kW" || e.Parent.Key != "100" {
		t.Fatalf("canonical measurement = %#v", e)
	}
	return r
}

func TestDecodedMicroinvertersKeepOwnValues(t *testing.T) {
	topology := ninePVTopology()
	for i := 3; i <= 9; i++ {
		key := fmt.Sprintf("100_55_1_%d", i)
		delete(topology.Devices, key)
		delete(topology.Inventory.Devices, key)
	}
	data := decodedPVSnapshot(t, []string{pvRow("100_55_1_1", 55, "p24", "0.8", "kW", "", ""), pvRow("100_55_1_2", 55, "p24", "0.9", "kW", "", "")}, "2")
	for key, want := range map[string]float64{"virtual.100_55_1_1.p24": .8, "virtual.100_55_1_2.p24": .9} {
		e := data.Map[key].GetEntry(api.LastEntry)
		if e.Value.ValueFloat() != want {
			t.Fatalf("%s = %v", key, e.Value.ValueFloat())
		}
	}
	r := requirePV(t, &data, topology, "summed_device_ac", 1.7)
	if r.Expected != 2 || r.Received != 2 || r.ValidAC != 2 {
		t.Fatalf("counts = %#v", r)
	}
}
func TestDecodedRowCountNeverEstablishesProducerCoverage(t *testing.T) {
	for _, count := range []string{"9", "10", "1", "", "null"} {
		t.Run("complete_"+count, func(t *testing.T) {
			rows := append(ninePVRows(), pvRow("100", 11, "unrelated", "0", "kW", "", ""))
			data := decodedPVSnapshot(t, rows, count)
			r := requirePV(t, &data, ninePVTopology(), "summed_device_ac", 1.8)
			if r.Expected != 9 || r.Received != 9 {
				t.Fatalf("counts = %#v", r)
			}
		})
	}
	rows := append(ninePVRows()[:8], pvRow("100", 11, "unrelated", "0", "kW", "", ""))
	data := decodedPVSnapshot(t, rows, "9")
	r := AddCanonicalPlantPVPower(&data, ninePVTopology())
	if r.Added || r.Expected != 9 || r.Received != 8 || r.Reason != "missing_contributors" {
		t.Fatalf("partial = %#v", r)
	}
}
func TestDecodedMissingInvalidZeroAndUnrecognizedContributors(t *testing.T) {
	for _, value := range []string{"null", `"--"`, `"invalid"`, `"NaN"`} {
		t.Run(value, func(t *testing.T) {
			rows := ninePVRows()
			rows[8] = pvRow("100_55_1_9", 55, "p24", value, "kW", "", "")
			data := decodedPVSnapshot(t, rows, "9")
			r := AddCanonicalPlantPVPower(&data, ninePVTopology())
			if r.Added || r.Expected != 9 || r.Received != 9 || r.ValidAC != 8 || r.Reason != "invalid_values" {
				t.Fatalf("invalid = %#v", r)
			}
		})
	}
	rows := ninePVRows()
	rows[8] = pvRow("100_55_1_9", 55, "unknown", "0.2", "kW", "unknown", "")
	data := decodedPVSnapshot(t, rows, "9")
	r := AddCanonicalPlantPVPower(&data, ninePVTopology())
	if r.Added || r.Expected != 9 || r.Received != 8 {
		t.Fatalf("unrecognized = %#v", r)
	}
	rows[8] = pvRow("100_55_1_9", 55, "p24", "0", "kW", "", "")
	data = decodedPVSnapshot(t, rows, "9")
	requirePV(t, &data, ninePVTopology(), "summed_device_ac", 1.6)
	data = decodedPVSnapshot(t, ninePVRows(), "9")
	requirePV(t, &data, ninePVTopology(), "summed_device_ac", 1.8)
}
func TestDecodedNativeProvenanceAndSemanticSafety(t *testing.T) {
	for _, pointID := range []string{"p83076_map", "plant_power", "pv_power", "solar_power"} {
		t.Run("original_ID_"+pointID, func(t *testing.T) {
			data := decodedPVSnapshot(t, []string{pvRow("100", 11, pointID, "4.5", "kW", "", "")}, "1")
			requirePV(t, &data, PlantTopology{PsID: "100"}, "native_plant", 4.5)
			if data.Measurements[0].Current.Source.PointID != pointID {
				t.Fatalf("original point ID lost: %#v", data.Measurements[0].Current.Source)
			}
			if pointID == "p83076_map" {
				if _, ok := data.Map["virtual.100.83076_map"]; !ok {
					t.Fatal("legacy MQTT identity was renamed")
				}
			}
		})
	}
	data := decodedPVSnapshot(t, append(ninePVRows()[:8], pvRow("100_11_0_0", 11, "p83076", "4.5", "kW", "", "")), "9")
	rawFound := false
	for _, entries := range data.Map {
		e := entries.GetEntry(api.LastEntry)
		if e.Point.Id == "value" && e.Current.Source.PointID == "p83076" {
			rawFound = true
		}
	}
	if !rawFound {
		for _, entries := range data.Map {
			e := entries.GetEntry(api.LastEntry)
			if e.Point.Id == "value" {
				t.Logf("raw endpoint=%s parent=%s device=%s path=%s", e.EndPoint, e.Parent.Key, e.Current.DataStructure.PointDevice, e.Current.FieldPath.String())
			}
		}
		t.Fatal("raw value lost original source identity")
	}
	requirePV(t, &data, PlantTopology{PsID: "100"}, "native_plant", 4.5)
	for _, tc := range []struct{ name, point, alias, group string }{
		{"id_name_conflict", "p83076", "DC Power", ""},
		{"grid", "p83076", "Grid Active Power", ""},
		{"phase", "p83076", "", "Phase A"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := decodedPVSnapshot(t, []string{pvRow("100", 11, tc.point, "4.5", "kW", tc.alias, tc.group)}, "1")
			if r := AddCanonicalPlantPVPower(&d, PlantTopology{PsID: "100"}); r.Added {
				t.Fatalf("unsafe native = %#v", r)
			}
		})
	}
	data = decodedPVSnapshot(t, []string{pvRow("100", 11, "unrecognized", "4.5", "kW", "Plant Power", "")}, "1")
	requirePV(t, &data, PlantTopology{PsID: "100"}, "native_plant", 4.5)
	data = decodedPVSnapshot(t, []string{pvRow("100", 11, "p83076", "4.5", "kW", "", "")}, "1")
	entry := data.Map["virtual.100.p83076"].GetEntry(api.LastEntry)
	entry.Value = valueTypes.SetUnitValueFloat("kW", "Power", 9)
	if r := AddCanonicalPlantPVPower(&data, PlantTopology{PsID: "100"}); r.Added {
		t.Fatalf("conflicting copy used: %#v", r)
	}
	data = decodedPVSnapshot(t, []string{pvRow("100", 11, "p83076", "4.5", "kW", "", "")}, "1")
	for i := range data.Measurements {
		data.Measurements[i].Current.Source.Derived = true
	}
	for _, entries := range data.Map {
		e := entries.GetEntry(api.LastEntry)
		if e.Current.Source.PointID == "p83076" {
			e.Current.Source.Derived = true
		}
	}
	if r := AddCanonicalPlantPVPower(&data, PlantTopology{PsID: "100"}); r.Added {
		t.Fatalf("derived native used: %#v", r)
	}
}
func TestDecodedCompleteDCFallbackAndNoMixedRepair(t *testing.T) {
	rows := ninePVRows()[:8]
	for i := 1; i <= 9; i++ {
		rows = append(rows, pvRow(fmt.Sprintf("100_55_1_%d", i), 55, "total_dc_power", "0.3", "kW", "", ""))
	}
	data := decodedPVSnapshot(t, rows, "17")
	requirePV(t, &data, ninePVTopology(), "summed_device_dc", 2.7)
	data = decodedPVSnapshot(t, append(ninePVRows()[:8], pvRow("100_55_1_9", 55, "total_dc_power", "0.3", "kW", "", "")), "9")
	if r := AddCanonicalPlantPVPower(&data, ninePVTopology()); r.Added || r.Reason != "incomplete_basis" {
		t.Fatalf("mixed = %#v", r)
	}
}
func TestDecodedIndependentInventoryAndIdentityConflicts(t *testing.T) {
	for _, tc := range []struct {
		name     string
		topology PlantTopology
		rows     []string
		reason   string
	}{
		{"unavailable", PlantTopology{PsID: "100", Complete: true}, ninePVRows(), "inventory_unavailable"},
		{"conflicting_inventory", func() PlantTopology { x := ninePVTopology(); x.Inventory.Conflict = true; return x }(), ninePVRows(), "inventory_conflict"},
		{"unknown_telemetry", ninePVTopology(), append(ninePVRows(), pvRow("100_55_1_10", 55, "p24", "0.2", "kW", "", "")), "inventory_conflict"},
		{"type_conflict", ninePVTopology(), append(ninePVRows()[:8], pvRow("100_55_1_9", 1, "p24", "0.2", "kW", "", "")), "inventory_conflict"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := decodedPVSnapshot(t, tc.rows, "9")
			r := AddCanonicalPlantPVPower(&d, tc.topology)
			if r.Added || r.Reason != tc.reason {
				t.Fatalf("inventory = %#v", r)
			}
		})
	}
}
func TestDecodedIntermediateTopologyAndCycle(t *testing.T) {
	topology := ninePVTopology()
	for i := 3; i <= 9; i++ {
		key := fmt.Sprintf("100_55_1_%d", i)
		delete(topology.Devices, key)
		delete(topology.Inventory.Devices, key)
	}
	parent := topology.Devices["100_55_1_1"]
	parent.UUID = 1
	topology.Devices[parent.PsKey] = parent
	child := topology.Devices["100_55_1_2"]
	child.UpUUID = 20
	topology.Devices[child.PsKey] = child
	topology.Devices["100_64_0_0"] = PlantTopologyDevice{PsKey: "100_64_0_0", DeviceType: 64, UUID: 20, UpUUID: 1}
	rows := []string{pvRow(parent.PsKey, 55, "p24", "9", "kW", "", ""), pvRow(child.PsKey, 55, "p24", "0.2", "kW", "", "")}
	data := decodedPVSnapshot(t, rows, "2")
	r := requirePV(t, &data, topology, "summed_device_ac", .2)
	if r.Expected != 1 || r.Contributors != 1 {
		t.Fatalf("leaves = %#v", r)
	}
	data = decodedPVSnapshot(t, rows[:1], "1")
	r = AddCanonicalPlantPVPower(&data, topology)
	if r.Added || r.Expected != 1 || r.Received != 0 {
		t.Fatalf("parent fallback = %#v", r)
	}
	topology.Complete = false
	topology.Reason = "topology contains a cycle"
	data = decodedPVSnapshot(t, rows, "2")
	if r := AddCanonicalPlantPVPower(&data, topology); r.Added || r.Reason != "topology_invalid" {
		t.Fatalf("cycle = %#v", r)
	}
	data = decodedPVSnapshot(t, append(rows, pvRow("100", 11, "p83076", "4.5", "kW", "", "")), "3")
	requirePV(t, &data, topology, "native_plant", 4.5)
}
