package iSolarCloud

import (
	"net/http"
	"path"
	"testing"

	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/WebIscmAppService/getPsTreeMenu"
)

func TestPlantTreeDiscoveryKeepsDenialDistinctFromAnEmptyTree(t *testing.T) {
	const deniedID, healthyID = "100_101", "200"
	treeCalls, attrsCalls := 0, 0
	sg, auth, _ := recoveryClient(t, recoveryTransport(func(request *http.Request) (*http.Response, error) {
		switch path.Base(request.URL.Path) {
		case "login":
			return recoveryResponse(recoveryLoginResponse("synthetic_token")), nil
		case "getUserList":
			return recoveryResponse(`{"req_serial_num":"synthetic","result_code":"1","result_msg":"success","result_data":[]}`), nil
		case "getPsList":
			return recoveryResponse(`{"req_serial_num":"synthetic","result_code":"1","result_msg":"success","result_data":{"pageList":[{"ps_id":"100_101"},{"ps_id":"200"}]}}`), nil
		case "getPsTreeMenu":
			treeCalls++
			if treeCalls == 1 {
				response := recoveryResponse("synthetic forbidden body")
				response.StatusCode = http.StatusForbidden
				return response, nil
			}
			return recoveryResponse(`{"req_serial_num":"synthetic","result_code":"1","result_msg":"success","result_data":{"list":[]}}`), nil
		case "getDevicePointAttrs":
			attrsCalls++
			return recoveryResponse(`{"req_serial_num":"synthetic","result_code":"1","result_msg":"success","result_data":[]}`), nil
		default:
			t.Fatalf("unexpected endpoint %s", request.URL.Path)
			return nil, nil
		}
	}))
	if err := sg.AuthenticateSession(auth, false, nil); err != nil {
		t.Fatal(err)
	}
	var attempt PlantTreeDiscovery
	trees, err := attempt.Trees(sg)
	if err != nil || !attempt.Denied(deniedID) || attempt.Denied(healthyID) || len(trees) != 1 {
		t.Fatalf("discovery of a denied and empty tree: trees=%v err=%v", trees, err)
	}
	if tree, ok := trees[healthyID]; !ok || len(tree.Devices) != 0 {
		t.Fatalf("empty successful tree was lost: %+v", trees)
	}
	if _, err := attempt.DevicePointAttrs(sg); err != nil || treeCalls != 2 || attrsCalls != 0 {
		t.Fatalf("metadata retried a denied tree: err=%v trees=%d attrs=%d", err, treeCalls, attrsCalls)
	}
	var next PlantTreeDiscovery
	trees, err = next.Trees(sg)
	if err != nil || next.HasDenials() || len(trees) != 2 || treeCalls != 3 {
		t.Fatalf("later discovery did not retry the denied plant: err=%v trees=%v requests=%d", err, trees, treeCalls)
	}
}

func TestPlantTreeDiscoveryRequiresAnActualHTTPForbiddenResponse(t *testing.T) {
	sg, auth, _ := recoveryClient(t, recoveryTransport(func(request *http.Request) (*http.Response, error) {
		switch path.Base(request.URL.Path) {
		case "login":
			return recoveryResponse(recoveryLoginResponse("synthetic_token")), nil
		case "getUserList":
			return recoveryResponse(`{"req_serial_num":"synthetic","result_code":"1","result_msg":"success","result_data":[]}`), nil
		case "getPsList":
			return recoveryResponse(`{"req_serial_num":"synthetic","result_code":"1","result_msg":"success","result_data":{"pageList":[{"ps_id":"100"}]}}`), nil
		case "getPsTreeMenu":
			return recoveryResponse(`{"req_serial_num":"synthetic","result_code":"-1","result_msg":"API httpResponse is 403 Forbidden","result_data":{}}`), nil
		default:
			t.Fatalf("unexpected endpoint %s", request.URL.Path)
			return nil, nil
		}
	}))
	if err := sg.AuthenticateSession(auth, false, nil); err != nil {
		t.Fatal(err)
	}
	var attempt PlantTreeDiscovery
	if _, err := attempt.Trees(sg); ClassifyFailure(err) != FailureClassNonRecoverable || attempt.HasDenials() {
		t.Fatalf("application error treated as an HTTP denial: err=%v denied=%v", err, attempt.HasDenials())
	}
}

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
		{name: "cycle", trees: PsTrees{"100": {Devices: []getPsTreeMenu.Ps{
			testTreeDevice("100", "100_parent", 10, 11), testTreeDevice("100", "100_child", 11, 10),
		}}}, plant: "100", reason: "topology contains a cycle"},
		{name: "self cycle", trees: PsTrees{"100": {Devices: []getPsTreeMenu.Ps{
			testTreeDevice("100", "100_child", 11, 11),
		}}}, plant: "100", reason: "topology contains a cycle"},
		{name: "UUID reuse across plants is valid", trees: PsTrees{
			"100": {Devices: []getPsTreeMenu.Ps{testTreeDevice("100", "100_parent", 10, 0), testTreeDevice("100", "100_child", 11, 10)}},
			"200": {Devices: []getPsTreeMenu.Ps{testTreeDevice("200", "200_parent", 10, 0), testTreeDevice("200", "200_child", 11, 10)}},
		}, plant: "100", valid: true},
		{name: "duplicate UUID", trees: PsTrees{"100": {Devices: []getPsTreeMenu.Ps{
			testTreeDevice("100", "100_parent", 10, 0), testTreeDevice("100", "100_child", 10, 0),
		}}}, plant: "100", reason: "topology contains duplicate UUIDs"},
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
