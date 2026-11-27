package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/WebIscmAppService/getPsTreeMenu"
)

func TestDashboardDiscoveryUsesOnlyDeniedPlantInventory(t *testing.T) {
	fixture := newMQTTRecoveryFixture(t)
	var requested []string
	treeRequests := 0
	fixture.denyRequest = func(request *http.Request) bool {
		endpoint := path.Base(request.URL.Path)
		switch endpoint {
		case "getPsTreeMenu":
			requested = append(requested, endpoint)
			treeRequests++
			return treeRequests != 2
		case "getDeviceList":
			requested = append(requested, endpoint)
		}
		return false
	}
	fixture.resultData = func(endpoint, fallback string) string {
		switch endpoint {
		case "getPsList":
			return `{"pageList":[{"ps_id":"200_201"},{"ps_id":"100_101","ps_name":"Roof"}]}`
		case "getPsTreeMenu":
			return `{"list":[{"ps_id":"200_201","ps_key":"other_14_1_1","device_type":14}]}`
		case "getDeviceList":
			return `{"pageList":[{"ps_id":"100_101","ps_key":"opaque_B","device_type":14,"device_name":"Z"},{"ps_id":"100_101","ps_key":"opaque_A","device_type":" 014 "},{"ps_id":"100_101","ps_key":"opaque_B","device_type":"14","device_name":"A"},{"ps_id":"100_101","device_type":null}]}`
		}
		return fallback
	}
	discovery, err := dashboardDiscoveryFromCloud(cmds.Api.SunGrow, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := discovery.selectTargets(nil)
	if got.reason != "" || !reflect.DeepEqual(targetPSKeys(got.targets), []string{"opaque_A", "opaque_B", "other_14_1_1"}) {
		t.Fatalf("unexpected selection: %+v", got)
	}
	if got.targets[1].DeviceName != "A" || got.targets[1].PlantName != "Roof" {
		t.Fatalf("inventory names not resolved deterministically: %+v", got.targets[1])
	}
	wantRequests := []string{"getPsTreeMenu", "getPsTreeMenu", "getDeviceList"}
	if !reflect.DeepEqual(requested, wantRequests) || fixture.loginCalls != 1 {
		t.Fatalf("requests=%v logins=%d, want %v and one login", requested, fixture.loginCalls, wantRequests)
	}
	if denial := fixture.denialOutput.String(); strings.Count(denial, "status=403") != 1 || strings.Contains(denial, "PRIVATE") {
		t.Fatalf("unexpected denial diagnostic: %s", denial)
	}
	requested = nil
	explicit, err := dashboardDiscoveryFromCloud(cmds.Api.SunGrow, []string{" 100_101 ", "100_101"})
	if err != nil {
		t.Fatal(err)
	}
	got = explicit.selectTargets(nil)
	if got.reason != "" || len(got.targets) != 2 || got.targets[0].PlantName != "Plant 100_101" ||
		!reflect.DeepEqual(requested, []string{"getPsTreeMenu"}) {
		t.Fatalf("explicit scope or cached inventory changed: selection=%+v requests=%v", got, requested)
	}
}

func TestDashboardInventoryInvalidTypeSurvivesCacheAndMQTTReuse(t *testing.T) {
	fixture := newMQTTRecoveryFixture(t)
	fixture.command.DashboardEnabled = true
	fixture.denyRequest = func(request *http.Request) bool { return path.Base(request.URL.Path) == "getPsTreeMenu" }
	fixture.resultData = func(endpoint, fallback string) string {
		if endpoint == "getDeviceList" {
			return `{"pageList":[{"ps_id":"100","ps_key":"eligible","device_type":22},{"ps_id":"100","ps_key":"unverifiable","device_type":null}]}`
		}
		return fallback
	}
	for _, origin := range []string{"live", "cache", "MQTT"} {
		t.Run(origin, func(t *testing.T) {
			before := countDashboardFixtureRequests(fixture.requests, "getDeviceList")
			var discovery *dashboardDiscovery
			if origin == "MQTT" {
				devices, err := cmds.Api.SunGrow.GetDeviceList()
				if err != nil {
					t.Fatal(err)
				}
				fixture.command.refreshPlantTopologies(devices)
				discovery = fixture.command.dashboardDiscovery
			} else {
				var err error
				discovery, err = dashboardDiscoveryFromCloud(cmds.Api.SunGrow, nil)
				if err != nil {
					t.Fatal(err)
				}
			}
			selection := discovery.selectTargets(nil)
			if selection.reason != dashboardCoverageIncomplete || len(selection.targets) != 0 {
				t.Fatalf("invalid plant was accepted: %+v", selection)
			}
			after := countDashboardFixtureRequests(fixture.requests, "getDeviceList")
			if origin == "live" && after != before+1 || origin != "live" && after != before {
				t.Fatalf("%s fetched inventory %d times", origin, after-before)
			}
		})
	}
	before := fixture.collectionCalls
	if err := fixture.command.Cron(); err != nil || fixture.collectionCalls <= before {
		t.Fatalf("invalid dashboard inventory interrupted MQTT collection: %v", err)
	}
}

func TestDashboardDiscoveryDoesNotUseInventoryForSuccessfulEmptyTree(t *testing.T) {
	fixture := newMQTTRecoveryFixture(t)
	fixture.resultData = func(endpoint, fallback string) string {
		if endpoint == "getPsTreeMenu" {
			return `{"list":[]}`
		}
		return fallback
	}
	discovery, err := dashboardDiscoveryFromCloud(cmds.Api.SunGrow, nil)
	if err != nil {
		t.Fatal(err)
	}
	selection := discovery.selectTargets(nil)
	if len(selection.targets) != 0 || selection.protected || countDashboardFixtureRequests(fixture.requests, "getDeviceList") != 0 {
		t.Fatalf("successful empty tree unexpectedly activated fallback: %+v", selection)
	}
}

func countDashboardFixtureRequests(requests []string, endpoint string) int {
	count := 0
	for _, request := range requests {
		if path.Base(request) == endpoint {
			count++
		}
	}
	return count
}

func dashboardTestTree(plant string, keys []string, deviceType int64) iSolarCloud.PsTree {
	tree := iSolarCloud.PsTree{Devices: make([]getPsTreeMenu.Ps, len(keys))}
	for i, key := range keys {
		device := &tree.Devices[i]
		device.PsId.SetString(plant)
		device.PsKey.SetValue(key)
		device.DeviceType.SetValue(deviceType)
	}
	return tree
}

func TestDashboardFallbackPreservesBestRankAndVerifiedViewOrder(t *testing.T) {
	const plant = "100"
	inventory := dashboardTestTree(plant, []string{"100_A", "100_B"}, 22)
	discovery := &dashboardDiscovery{plants: []string{plant}, denied: map[string]bool{plant: true},
		inventories: iSolarCloud.PsTrees{plant: inventory}}
	if got := discovery.selectTargets(nil); len(got.targets) != 1 || got.targets[0].PsKey != "100_A" {
		t.Fatalf("fresh discovery did not select the lexical eligible other type: %+v", got)
	}
	state := &haDashboardState{TargetPsKeys: []string{"100_B"}, InventoryTargetPlants: []string{plant}}
	recovery := &dashboardDiscovery{plants: []string{plant}, trees: iSolarCloud.PsTrees{plant: inventory}}
	for _, attempt := range []*dashboardDiscovery{discovery, recovery} {
		got := attempt.selectTargets(state)
		if got.reason != "" || len(got.targets) != 1 || got.targets[0].PsKey != "100_B" || !got.keepViews {
			t.Fatalf("saved tied target lost: %+v", got)
		}
		old := map[string]any{"views": []any{map[string]any{"title": "Saved title", "path": "saved-path"}}}
		state.DashboardHash, _ = hashCanonicalJSON(old)
		desired := map[string]any{"views": []any{map[string]any{"title": "Changed label", "path": "new-path"}}}
		if !preserveDashboardViews(desired, old, state, got.targets) || !reflect.DeepEqual(desired, old) {
			t.Fatalf("verified saved views lost: %v", desired)
		}
	}
	changed := &dashboardDiscovery{plants: []string{plant}, trees: iSolarCloud.PsTrees{
		plant: dashboardTestTree(plant, []string{"replacement"}, 14)}}
	if got := changed.selectTargets(state); got.reason != "" || got.keepViews || len(got.targets) != 1 || got.targets[0].PsKey != "replacement" {
		t.Fatalf("complete recovery failed to accept genuinely different target: %+v", got)
	}
}

func TestDashboardDiscoveryDefersBeforeMutation(t *testing.T) {
	const plant = "100"
	inventory := func(keys ...string) *dashboardDiscovery {
		return &dashboardDiscovery{plants: []string{plant}, denied: map[string]bool{plant: true},
			inventories: iSolarCloud.PsTrees{plant: dashboardTestTree(plant, keys, 14)}}
	}
	for _, test := range []struct {
		name      string
		discovery *dashboardDiscovery
		reason    string
		saved     bool
	}{
		{"incomplete coverage", inventory(), dashboardCoverageIncomplete, false},
		{"fresh path conflict", inventory("100_14_1_A", "100_14_1-A"), dashboardIdentityConflict, false},
		{"mixed case-fold conflict", &dashboardDiscovery{plants: []string{"100", "200"}, denied: map[string]bool{"100": true},
			inventories: iSolarCloud.PsTrees{"100": dashboardTestTree("100", []string{"100_14_1_A"}, 14)},
			trees:       iSolarCloud.PsTrees{"200": dashboardTestTree("200", []string{"100_14_1_a"}, 14)}}, dashboardIdentityConflict, false},
		{"retained path on recovery", &dashboardDiscovery{plants: []string{plant},
			trees: iSolarCloud.PsTrees{plant: dashboardTestTree(plant, []string{"100_A", "100_B"}, 14)}}, dashboardIdentityConflict, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			assets, root := t.TempDir(), t.TempDir()
			t.Setenv("GOSUNGROW_CONFIG", filepath.Join(root, "config.json"))
			if err := os.WriteFile(filepath.Join(assets, dashboardTemplateFile), []byte("views:\n  - cards: []\n"), 0600); err != nil {
				t.Fatal(err)
			}
			var current map[string]any
			var originalState []byte
			if test.saved {
				current = map[string]any{"views": []any{
					map[string]any{"title": "First", "path": "retained"},
					map[string]any{"title": "Second", "path": "retained"},
				}}
				hash, err := hashCanonicalJSON(current)
				if err != nil {
					t.Fatal(err)
				}
				if err := saveDashboardState(dashboardStatePath(), &haDashboardState{DashboardURLPath: defaultDashboardURLPath,
					DashboardHash: hash, TargetPsKeys: []string{"100_A", "100_B"}, InventoryTargetPlants: []string{plant}}); err != nil {
					t.Fatal(err)
				}
				originalState, err = os.ReadFile(dashboardStatePath())
				if err != nil {
					t.Fatal(err)
				}
			}
			endpoint := dashboardReadOnlyFixture(t, current)
			opts := haDashboardInstallOptions{AssetDir: assets, SupervisorToken: "fixture-token", HomeAssistantWSURL: endpoint,
				Language: "en", DashboardURLPath: defaultDashboardURLPath, discovery: test.discovery,
				onTargetsVerified: func([]haDashboardTarget) { t.Error("deferral replaced verified runtime targets") }}
			if err := reconcileManagedDashboardWithAssetRoots(context.Background(), nil, opts, []string{root}); err == nil || err.Error() != test.reason {
				t.Fatalf("reconciliation returned %v, want %s", err, test.reason)
			}
			if _, err := os.Stat(filepath.Join(root, "www")); !os.IsNotExist(err) {
				t.Fatalf("dashboard assets were staged: %v", err)
			}
			state, err := os.ReadFile(dashboardStatePath())
			if test.saved && (err != nil || string(state) != string(originalState)) || !test.saved && !os.IsNotExist(err) {
				t.Fatalf("dashboard state was altered: %v", err)
			}
		})
	}
	if !dashboardTargetIdentityConflict([]haDashboardTarget{{PsID: "100", PsKey: "first"}, {PsID: "100", PsKey: "second"}},
		func(haDashboardTarget) string { return "identical-mapping" }) {
		t.Fatal("same mapping ID on distinct targets was not rejected")
	}
}

func dashboardReadOnlyFixture(t *testing.T, saved map[string]any) string {
	t.Helper()
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connection, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade dashboard websocket: %v", err)
			return
		}
		defer connection.Close()
		var auth map[string]any
		if connection.WriteJSON(map[string]any{"type": "auth_required"}) != nil ||
			connection.ReadJSON(&auth) != nil ||
			connection.WriteJSON(map[string]any{"type": "auth_ok"}) != nil {
			t.Error("could not authenticate the dashboard websocket")
			return
		}
		for {
			var request map[string]any
			if err := connection.ReadJSON(&request); err != nil {
				return
			}
			if request["type"] != "lovelace/config" {
				t.Errorf("unexpected Home Assistant operation: %v", request["type"])
				return
			}
			response := map[string]any{"id": request["id"], "type": "result", "success": saved != nil}
			if saved == nil {
				response["error"] = map[string]any{"code": "config_not_found", "message": "missing dashboard"}
			} else {
				response["result"] = saved
			}
			if err := connection.WriteJSON(response); err != nil {
				return
			}
		}
	}))
	t.Cleanup(server.Close)
	return "ws" + strings.TrimPrefix(server.URL, "http")
}

// Run by the pinned Home Assistant browser fixture against disposable Core.
func TestDashboardFallbackHomeAssistantFixture(t *testing.T) {
	root := os.Getenv("HA_E2E_CONFIG_DIR")
	if root == "" {
		t.Skip("requires the pinned Home Assistant browser fixture")
	}
	t.Setenv("GOSUNGROW_CONFIG", filepath.Join(root, "fixture-gosungrow.json"))
	fixture := newMQTTRecoveryFixture(t)
	fixture.denyRequest = func(request *http.Request) bool { return path.Base(request.URL.Path) == "getPsTreeMenu" }
	fixture.resultData = func(endpoint, fallback string) string {
		switch endpoint {
		case "getPsList":
			return `{"pageList":[{"ps_id":"100","ps_name":"Fixture plant"}]}`
		case "getDeviceList":
			return `{"pageList":[{"ps_id":"100","ps_key":"100_14_1_1","device_type":14,"device_name":"Fixture ESS"}]}`
		}
		return fallback
	}
	opts := haDashboardInstallOptions{
		AssetDir: filepath.Join("..", "addon", "gosungrow", "assets"), HomeAssistantDir: root,
		HomeAssistantWSURL: os.Getenv("HA_E2E_WS_URL"), SupervisorToken: os.Getenv("HA_E2E_TOKEN"),
		Language: "en", DashboardURLPath: defaultDashboardURLPath, DashboardTitle: defaultDashboardTitle,
		DashboardIcon: defaultDashboardIcon, ShowInSidebar: true,
	}
	cmds.Api.setCloudOperation(cloudDashboardDiscovery, 0)
	var err error
	opts.discovery, err = dashboardDiscoveryFromCloud(cmds.Api.SunGrow, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := reconcileManagedDashboardWithAssetRoots(context.Background(), nil, opts, []string{root}); err != nil {
		t.Fatal(err)
	}
	if denial := fixture.denialOutput.String(); strings.Count(denial, "status=403") != 1 || !strings.Contains(denial, "operation=dashboard_discovery") {
		t.Fatalf("expected one sanitized dashboard tree denial: %s", denial)
	}
	state, err := loadDashboardState(dashboardStatePath())
	if err != nil || state == nil || state.AssetMode != dashboardAssetModeEnhanced ||
		!reflect.DeepEqual(state.InventoryTargetPlants, []string{"100"}) {
		t.Fatalf("fallback was not saved with verified assets and inventory metadata: state=%+v err=%v", state, err)
	}
}
