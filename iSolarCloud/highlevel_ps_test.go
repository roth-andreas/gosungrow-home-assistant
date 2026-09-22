package iSolarCloud

import (
	"testing"

	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/WebIscmAppService/getPsTreeMenu"
)

func TestBuildPlantTopologiesValidatesParentRelationships(t *testing.T) {
	tests := []struct {
		name   string
		trees  PsTrees
		plant  string
		valid  bool
		reason string
	}{
		{name: "complete", trees: PsTrees{"100": {Devices: []getPsTreeMenu.Ps{
			testTreeDevice("100", "100_parent", 10, 0), testTreeDevice("100", "100_child", 11, 10),
		}}}, plant: "100", valid: true},
		{name: "dangling parent", trees: PsTrees{"100": {Devices: []getPsTreeMenu.Ps{
			testTreeDevice("100", "100_child", 11, 99),
		}}}, plant: "100", reason: "topology contains a dangling parent"},
		{name: "cross plant parent", trees: PsTrees{
			"100": {Devices: []getPsTreeMenu.Ps{testTreeDevice("100", "100_child", 11, 20)}},
			"200": {Devices: []getPsTreeMenu.Ps{testTreeDevice("200", "200_parent", 20, 0)}},
		}, plant: "100", reason: "topology contains a cross-plant parent"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			topology := BuildPlantTopologies(tc.trees)[tc.plant]
			if topology.Complete != tc.valid || topology.Reason != tc.reason {
				t.Fatalf("BuildPlantTopologies() = complete %v reason %q", topology.Complete, topology.Reason)
			}
		})
	}
}

func testTreeDevice(psID, psKey string, uuid, upUUID int64) getPsTreeMenu.Ps {
	var device getPsTreeMenu.Ps
	device.PsId.SetString(psID)
	device.PsKey.SetValue(psKey)
	device.UUID.SetValue(uuid)
	device.UpUUID.SetValue(upUUID)
	return device
}
