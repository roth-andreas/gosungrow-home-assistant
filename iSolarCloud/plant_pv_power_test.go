package iSolarCloud

import (
	"testing"
	"time"

	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api/GoStruct"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api/GoStruct/valueTypes"
)

func TestAddCanonicalPlantPVPowerSumsCompleteDCLeaves(t *testing.T) {
	data := api.NewDataMap()
	addPlantPowerTestEntry(&data, "raw.one.total_dc_power", "100_55_1_1", "total_dc_power", 1.2, "kW", time.Date(2026, 9, 21, 10, 5, 0, 0, time.UTC))
	addPlantPowerTestEntry(&data, "raw.two.total_dc_power", "100_55_1_2", "total_dc_power", 800, "W", time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC))
	topology := PlantTopology{PsID: "100", Complete: true, Devices: map[string]PlantTopologyDevice{
		"100_55_1_1": {PsID: "100", PsKey: "100_55_1_1", UUID: 1},
		"100_55_1_2": {PsID: "100", PsKey: "100_55_1_2", UUID: 2},
	}}

	result := AddCanonicalPlantPVPower(&data, topology)
	if !result.Added || result.Source != "summed_device_dc" || result.Contributors != 2 {
		t.Fatalf("AddCanonicalPlantPVPower() = %#v", result)
	}
	entry := data.Map["virtual.100.pv_power"].GetEntry(api.LastEntry)
	if got := entry.Value.ValueFloat(); got != 2 {
		t.Fatalf("plant PV value = %v, want 2", got)
	}
	if got := entry.Value.Unit(); got != "kW" {
		t.Fatalf("plant PV unit = %q, want kW", got)
	}
	if got := entry.Date.Time; !got.Equal(time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("plant PV timestamp = %s", got)
	}
}

func TestAddCanonicalPlantPVPowerNativeTotalWins(t *testing.T) {
	data := api.NewDataMap()
	addPlantPowerTestEntry(&data, "query.100.p83076", "100_11_0_0", "p83076", 4.5, "kW", time.Now())
	addPlantPowerTestEntry(&data, "raw.one.total_dc_power", "100_55_1_1", "total_dc_power", 3, "kW", time.Now())
	topology := PlantTopology{PsID: "100", Complete: false, Reason: "unavailable", Devices: map[string]PlantTopologyDevice{
		"100_55_1_1": {PsID: "100", PsKey: "100_55_1_1", UUID: 1},
	}}

	result := AddCanonicalPlantPVPower(&data, topology)
	if !result.Added || result.Source != "native_plant" {
		t.Fatalf("AddCanonicalPlantPVPower() = %#v", result)
	}
	if got := data.Map["virtual.100.pv_power"].GetEntry(api.LastEntry).Value.ValueFloat(); got != 4.5 {
		t.Fatalf("plant PV value = %v, want 4.5", got)
	}
}

func TestAddCanonicalPlantPVPowerPrefersCompleteACAndExcludesParent(t *testing.T) {
	data := api.NewDataMap()
	now := time.Now()
	addPlantPowerTestEntry(&data, "raw.parent.p24", "100_1_1_0", "p24", 9, "kW", now)
	addPlantPowerTestEntry(&data, "raw.one.p24", "100_1_1_1", "p24", 1, "kW", now)
	addPlantPowerTestEntry(&data, "raw.two.p24", "100_1_1_2", "p24", 2, "kW", now)
	addPlantPowerTestEntry(&data, "raw.one.total_dc_power", "100_1_1_1", "total_dc_power", 4, "kW", now)
	addPlantPowerTestEntry(&data, "raw.two.total_dc_power", "100_1_1_2", "total_dc_power", 5, "kW", now)
	topology := PlantTopology{PsID: "100", Complete: true, Devices: map[string]PlantTopologyDevice{
		"100_1_1_0": {PsID: "100", PsKey: "100_1_1_0", UUID: 10, DeviceType: 1},
		"100_1_1_1": {PsID: "100", PsKey: "100_1_1_1", UUID: 11, UpUUID: 10, DeviceType: 1},
		"100_1_1_2": {PsID: "100", PsKey: "100_1_1_2", UUID: 12, UpUUID: 10, DeviceType: 1},
	}}

	result := AddCanonicalPlantPVPower(&data, topology)
	if !result.Added || result.Source != "summed_device_ac" || result.Contributors != 2 {
		t.Fatalf("AddCanonicalPlantPVPower() = %#v", result)
	}
	if got := data.Map["virtual.100.pv_power"].GetEntry(api.LastEntry).Value.ValueFloat(); got != 3 {
		t.Fatalf("plant PV value = %v, want 3", got)
	}
}

func TestAddCanonicalPlantPVPowerRejectsIncompleteTierOrTopology(t *testing.T) {
	tests := []struct {
		name     string
		topology PlantTopology
		entries  []plantPowerTestValue
	}{
		{
			name: "mixed incomplete basis",
			topology: PlantTopology{PsID: "100", Complete: true, Devices: map[string]PlantTopologyDevice{
				"100_1_1_1":  {PsID: "100", PsKey: "100_1_1_1", UUID: 1, DeviceType: 1},
				"100_55_1_2": {PsID: "100", PsKey: "100_55_1_2", UUID: 2},
			}},
			entries: []plantPowerTestValue{{"one", "100_1_1_1", "p24", 1, "kW"}, {"two", "100_55_1_2", "total_dc_power", 2, "kW"}},
		},
		{
			name: "invalid topology",
			topology: PlantTopology{PsID: "100", Complete: false, Reason: "dangling parent", Devices: map[string]PlantTopologyDevice{
				"100_55_1_1": {PsID: "100", PsKey: "100_55_1_1", UUID: 1},
			}},
			entries: []plantPowerTestValue{{"one", "100_55_1_1", "total_dc_power", 1, "kW"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := api.NewDataMap()
			for _, entry := range tc.entries {
				addPlantPowerTestEntry(&data, entry.endpoint, entry.device, entry.point, entry.value, entry.unit, time.Now())
			}
			if result := AddCanonicalPlantPVPower(&data, tc.topology); result.Added {
				t.Fatalf("unexpected aggregate: %#v", result)
			}
			if _, exists := data.Map["virtual.100.pv_power"]; exists {
				t.Fatal("plant aggregate was published")
			}
		})
	}
}

func TestBuildPlantTopologiesRejectsEmptyPlant(t *testing.T) {
	topologies := BuildPlantTopologies(PsTrees{"100": {}})
	topology := topologies["100"]
	if topology.Complete || topology.Reason != "topology contains no devices" {
		t.Fatalf("BuildPlantTopologies() = %#v", topology)
	}
}

type plantPowerTestValue struct {
	endpoint string
	device   string
	point    string
	value    float64
	unit     string
}

func addPlantPowerTestEntry(data *api.DataMap, endpoint, device, pointID string, value float64, unit string, when time.Time) {
	current := &GoStruct.Reflect{IsOk: true}
	current.DataStructure.Endpoint = GoStruct.NewEndPointPath(stringsToPath(endpoint)...)
	current.DataStructure.PointId = pointID
	current.DataStructure.PointName = pointID
	current.DataStructure.PointDevice = device
	current.DataStructure.PointUnit = unit
	current.DataStructure.PointUpdateFreq = GoStruct.UpdateFreq5Mins
	unitValue := valueTypes.SetUnitValueFloat(unit, "Power", value)
	unitValue.SetDeviceId(device)
	current.SetUnitValue(unitValue)
	data.Add(api.DataEntry{
		Current: current, EndPoint: endpoint,
		Point:  &api.Point{Id: pointID, Description: pointID, Unit: unit, UpdateFreq: GoStruct.UpdateFreq5Mins, ValueType: "Power", Valid: true},
		Parent: api.NewParentDevice(device), Date: valueTypes.SetDateTimeValue(when), Value: unitValue, Valid: true,
	})
}

func stringsToPath(endpoint string) []string {
	var parts []string
	start := 0
	for index, char := range endpoint {
		if char == '.' {
			parts = append(parts, endpoint[start:index])
			start = index + 1
		}
	}
	return append(parts, endpoint[start:])
}
