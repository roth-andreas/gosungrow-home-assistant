package iSolarCloud

import (
	"encoding/json"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/getDeviceList"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/WebAppService/getDevicePointAttrs"
	"testing"
)

func TestDecodedDiscoveryDefinesProducersWithoutTelemetry(t *testing.T) {
	var inventory getDeviceList.ResultData
	if err := json.Unmarshal([]byte(`{"pageList":[
 {"ps_id":"100","ps_key":"100_55_1_1","device_type":55},
 {"ps_id":"100","ps_key":"100_55_1_2","device_type":55},
 {"ps_id":"100","ps_key":"opaque-ac","device_type":64,"device_name":"AC INVERTER"},
 {"ps_id":"100","ps_key":"opaque-dc","device_type":64},
 {"ps_id":"100","ps_key":"meter","device_type":7,"device_name":"Inverter"},
 {"ps_id":"100","ps_key":"plant","device_type":11,"device_name":"Inverter Plant"}
 ]}`), &inventory); err != nil {
		t.Fatal(err)
	}
	var ac, dc getDevicePointAttrs.Points
	if err := json.Unmarshal([]byte(`[{"id":"p24","name":"Active Power","unit":"kW"}]`), &ac); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`[{"id":"total_dc_power","name":"DC Power","unit":"kW"}]`), &dc); err != nil {
		t.Fatal(err)
	}
	inv := BuildPlantInventories(inventory.PageList, map[string]getDevicePointAttrs.Points{"opaque-ac": ac, "opaque-dc": dc, "meter": dc, "plant": dc})["100"]
	for key, want := range map[string]bool{"100_55_1_1": true, "100_55_1_2": true, "opaque-ac": true, "opaque-dc": true, "meter": false, "plant": false} {
		if got := inventoryProducer(inv.Devices[key]); got != want {
			t.Fatalf("producer %s=%t, want %t", key, got, want)
		}
	}
	topology := PlantTopology{PsID: "100", Complete: true, Inventory: inv, Devices: make(map[string]PlantTopologyDevice)}
	for key, d := range inv.Devices {
		topology.Devices[key] = PlantTopologyDevice{PsKey: key, DeviceType: d.DeviceType}
	}
	data := decodedPVSnapshot(t, []string{pvRow("100_55_1_1", 55, "p24", "0.2", "kW", "", "")}, "1")
	r := AddCanonicalPlantPVPower(&data, topology)
	if r.Added || r.Expected != 4 || r.Received != 1 || r.Reason != "missing_contributors" {
		t.Fatalf("independent expectation=%#v", r)
	}
}
func TestDecodedInventoryRejectsUnusableOrConflictingIdentity(t *testing.T) {
	for _, payload := range []string{
		`{"pageList":[{"ps_id":"100","ps_key":"x","device_type":55},{"ps_id":"100","ps_key":"x","device_type":1}]}`,
		`{"pageList":[{"ps_id":"100","ps_key":"","device_type":55}]}`,
		`{"pageList":[{"ps_id":"100","ps_key":"x","device_type":55},{"ps_id":"200","ps_key":"x","device_type":55}]}`,
	} {
		var decoded getDeviceList.ResultData
		if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
			t.Fatal(err)
		}
		inventories := BuildPlantInventories(decoded.PageList, nil)
		if !inventories["100"].Conflict {
			t.Fatalf("unsafe inventory=%#v", inventories)
		}
	}
}
