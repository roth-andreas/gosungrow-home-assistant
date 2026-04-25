package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/WebIscmAppService/getPsTreeMenu"
)

func testPsTreeDevice(psID string, psKey string, deviceType int64, plantName string, deviceName string) getPsTreeMenu.Ps {
	var ps getPsTreeMenu.Ps
	ps.PsId.SetString(psID)
	ps.PsKey.SetValue(psKey)
	ps.DeviceType.SetValue(deviceType)
	ps.PsName.SetString(plantName)
	ps.DeviceName.SetString(deviceName)
	return ps
}

func TestDiscoverDashboardTargetsPrefersDeviceType14(t *testing.T) {
	trees := map[string]iSolarCloud.PsTree{
		"100": {
			Devices: []getPsTreeMenu.Ps{
				testPsTreeDevice("100", "100_11_1_1", 11, "Roof", "Other Device"),
				testPsTreeDevice("100", "100_14_1_1", 14, "Roof", "Inverter"),
			},
		},
	}

	targets, err := discoverDashboardTargetsFromTrees(trees)
	if err != nil {
		t.Fatalf("discoverDashboardTargetsFromTrees: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %#v", targets)
	}
	if targets[0].PsKey != "100_14_1_1" {
		t.Fatalf("expected type 14 ps key, got %#v", targets[0])
	}
	if targets[0].DeviceType != 14 {
		t.Fatalf("expected device type 14, got %#v", targets[0])
	}
	if targets[0].SelectionSource != "preferred-device-type-14" {
		t.Fatalf("expected preferred type-14 selection, got %#v", targets[0])
	}
}

func TestDiscoverDashboardTargetsPrefersType11WhenNoEssTargetExists(t *testing.T) {
	trees := map[string]iSolarCloud.PsTree{
		"100": {
			Devices: []getPsTreeMenu.Ps{
				testPsTreeDevice("100", "100_22_247_1", 22, "Roof", "Communication Module"),
				testPsTreeDevice("100", "100_11_0_0", 11, "Roof", "Plant"),
				testPsTreeDevice("100", "100_12_1_1", 12, "Roof", "Battery"),
			},
		},
	}

	targets, err := discoverDashboardTargetsFromTrees(trees)
	if err != nil {
		t.Fatalf("discoverDashboardTargetsFromTrees: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 preferred non-ESS target, got %#v", targets)
	}
	if targets[0].PsKey != "100_11_0_0" {
		t.Fatalf("expected type 11 target, got %#v", targets[0])
	}
	if targets[0].DeviceType != 11 {
		t.Fatalf("expected preferred device type 11, got %#v", targets[0])
	}
	if targets[0].SelectionSource != "preferred-device-type-11" {
		t.Fatalf("expected preferred type-11 selection source, got %#v", targets[0])
	}
}

func TestDiscoverDashboardTargetsReturnsClearErrorWhenNoValidPsKeysExist(t *testing.T) {
	trees := map[string]iSolarCloud.PsTree{
		"100": {
			Devices: []getPsTreeMenu.Ps{
				testPsTreeDevice("100", "", 11, "Roof", "Invalid Device"),
			},
		},
	}

	_, err := discoverDashboardTargetsFromTrees(trees)
	if err == nil {
		t.Fatal("expected error when no valid ps_key exists")
	}
	if got := err.Error(); got != "no Sungrow devices with a valid ps_key were discovered" {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestRenderDashboardConfigTargetsAndReplacesPsKeys(t *testing.T) {
	templateDir := t.TempDir()
	templatePath := filepath.Join(templateDir, dashboardTemplateFile)

	template := `title: Template
views:
  - title: Prototype
    path: prototype
    cards:
      - type: tile
        entity: sensor.gosungrow_virtual_YOUR_ESS_PS_KEY_p13112
`
	if err := os.WriteFile(templatePath, []byte(template), 0600); err != nil {
		t.Fatalf("write template: %v", err)
	}

	config, err := renderDashboardConfig(templatePath, "GoSungrow Flow", []haDashboardTarget{
		{PsID: "100", PsKey: "5072099_14_1_1", ViewTitle: "Roof", ViewPath: "roof"},
		{PsID: "101", PsKey: "5080000_14_1_1", ViewTitle: "Garage", ViewPath: "garage"},
	}, defaultDashboardLocaleBundle)
	if err != nil {
		t.Fatalf("renderDashboardConfig: %v", err)
	}

	if got := config["title"]; got != "GoSungrow Flow" {
		t.Fatalf("unexpected dashboard title: %v", got)
	}

	views, ok := config["views"].([]any)
	if !ok || len(views) != 2 {
		t.Fatalf("expected 2 generated views, got %#v", config["views"])
	}

	firstView := views[0].(map[string]any)
	if firstView["title"] != "Roof" || firstView["path"] != "roof" {
		t.Fatalf("unexpected first view metadata: %#v", firstView)
	}

	rendered, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("marshal rendered config: %v", err)
	}
	text := string(rendered)
	if strings.Contains(text, "YOUR_ESS_PS_KEY") {
		t.Fatal("dashboard placeholder was not replaced")
	}
	if !strings.Contains(text, "sensor.gosungrow_virtual_5072099_14_1_1_p13112") {
		t.Fatal("expected first ps key replacement in rendered config")
	}
}

func TestRenderDashboardConfigPreservesMultiplePrototypeViews(t *testing.T) {
	templateDir := t.TempDir()
	templatePath := filepath.Join(templateDir, dashboardTemplateFile)

	template := `title: Template
views:
  - title: Overview
    path: overview
    cards:
      - type: tile
        entity: sensor.gosungrow_virtual_YOUR_ESS_PS_KEY_p13112
  - title: Trends
    path: trends
    cards:
      - type: tile
        entity: sensor.gosungrow_virtual_YOUR_ESS_PS_KEY_p13141
`
	if err := os.WriteFile(templatePath, []byte(template), 0600); err != nil {
		t.Fatalf("write template: %v", err)
	}

	config, err := renderDashboardConfig(templatePath, "GoSungrow Flow", []haDashboardTarget{
		{PsID: "100", PsKey: "5072099_14_1_1", ViewTitle: "Roof", ViewPath: "roof"},
	}, defaultDashboardLocaleBundle)
	if err != nil {
		t.Fatalf("renderDashboardConfig: %v", err)
	}

	views, ok := config["views"].([]any)
	if !ok || len(views) != 2 {
		t.Fatalf("expected 2 generated views, got %#v", config["views"])
	}

	firstView := views[0].(map[string]any)
	secondView := views[1].(map[string]any)
	if firstView["title"] != "Overview" || firstView["path"] != "overview" {
		t.Fatalf("unexpected first view metadata: %#v", firstView)
	}
	if secondView["title"] != "Trends" || secondView["path"] != "trends" {
		t.Fatalf("unexpected second view metadata: %#v", secondView)
	}
}

func TestDashboardStateRoundTripAndCanonicalHash(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("GOSUNGROW_CONFIG", configPath)

	statePath := dashboardStatePath()
	if want := filepath.Join(filepath.Dir(configPath), dashboardStateFileName); statePath != want {
		t.Fatalf("unexpected dashboard state path: got %q want %q", statePath, want)
	}

	state := &haDashboardState{
		DashboardURLPath: "gosungrow-flow",
		DashboardHash:    "abc123",
		TargetPsKeys:     []string{"5072099_14_1_1"},
		UpdatedAt:        "2026-03-19T12:00:00Z",
	}
	if err := saveDashboardState(statePath, state); err != nil {
		t.Fatalf("saveDashboardState: %v", err)
	}

	loaded, err := loadDashboardState(statePath)
	if err != nil {
		t.Fatalf("loadDashboardState: %v", err)
	}
	if loaded == nil || loaded.DashboardURLPath != state.DashboardURLPath || loaded.DashboardHash != state.DashboardHash {
		t.Fatalf("unexpected loaded state: %#v", loaded)
	}

	hashA, err := hashCanonicalJSON(map[string]any{"b": float64(2), "a": float64(1)})
	if err != nil {
		t.Fatalf("hashCanonicalJSON A: %v", err)
	}
	hashB, err := hashCanonicalJSON(map[string]any{"a": float64(1), "b": float64(2)})
	if err != nil {
		t.Fatalf("hashCanonicalJSON B: %v", err)
	}
	if hashA != hashB {
		t.Fatalf("expected stable canonical hash, got %q and %q", hashA, hashB)
	}
}

func TestInstallDashboardCardAssetWritesVersionedResource(t *testing.T) {
	assetDir := t.TempDir()
	configDir := t.TempDir()
	cardSource := filepath.Join(assetDir, dashboardCardFileName)
	cardBody := []byte("console.log('gosungrow card');")

	if err := os.WriteFile(cardSource, cardBody, 0600); err != nil {
		t.Fatalf("write card source: %v", err)
	}

	resourceURL, version, err := installDashboardCardAsset(assetDir, configDir)
	if err != nil {
		t.Fatalf("installDashboardCardAsset: %v", err)
	}

	if !strings.HasPrefix(resourceURL, "/local/"+dashboardCardResourceDir+"/"+dashboardCardFileName+"?v=") {
		t.Fatalf("unexpected resource URL: %q", resourceURL)
	}
	if strings.TrimSpace(version) == "" {
		t.Fatal("expected non-empty asset version")
	}

	targetPath := filepath.Join(configDir, "www", dashboardCardResourceDir, dashboardCardFileName)
	targetBody, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read installed card: %v", err)
	}
	if string(targetBody) != string(cardBody) {
		t.Fatalf("unexpected installed card content: %q", string(targetBody))
	}
}

func TestUniqueNonEmptyStrings(t *testing.T) {
	values := uniqueNonEmptyStrings([]string{"", "/config", " /config ", "/homeassistant", "/config"})
	if len(values) != 2 {
		t.Fatalf("unexpected unique values: %#v", values)
	}
	if values[0] != "/config" || values[1] != "/homeassistant" {
		t.Fatalf("unexpected order/content: %#v", values)
	}
}

func TestDashboardCardDataURI(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), dashboardCardFileName)
	if err := os.WriteFile(sourcePath, []byte("console.log('gosungrow');"), 0600); err != nil {
		t.Fatalf("write source: %v", err)
	}

	resourceURL, err := dashboardCardDataURI(sourcePath, "abc123")
	if err != nil {
		t.Fatalf("dashboardCardDataURI: %v", err)
	}

	if !strings.HasPrefix(resourceURL, "data:text/javascript;base64,") {
		t.Fatalf("unexpected data uri prefix: %q", resourceURL)
	}
	if !strings.HasSuffix(resourceURL, "#v=abc123") {
		t.Fatalf("unexpected data uri version suffix: %q", resourceURL)
	}
	if got := resourceURLBase(resourceURL); got != "data:text/javascript;base64," {
		t.Fatalf("unexpected normalized resource base: %q", got)
	}
}

func TestHAWSClientDashboardCalls(t *testing.T) {
	upgrader := websocket.Upgrader{}
	sawResourceUpdate := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer supervisor-token" {
			t.Fatalf("unexpected authorization header: %q", got)
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("upgrade websocket: %v", err)
		}
		defer conn.Close()

		if err := conn.WriteJSON(map[string]any{"type": "auth_required"}); err != nil {
			t.Fatalf("write auth_required: %v", err)
		}

		var auth map[string]any
		if err := conn.ReadJSON(&auth); err != nil {
			t.Fatalf("read auth: %v", err)
		}
		if auth["type"] != "auth" || auth["access_token"] != "supervisor-token" {
			t.Fatalf("unexpected auth payload: %#v", auth)
		}
		if err := conn.WriteJSON(map[string]any{"type": "auth_ok"}); err != nil {
			t.Fatalf("write auth_ok: %v", err)
		}

		for {
			var request map[string]any
			if err := conn.ReadJSON(&request); err != nil {
				return
			}

			response := map[string]any{
				"id":      request["id"],
				"type":    "result",
				"success": true,
			}

			switch request["type"] {
			case "lovelace/dashboards/list":
				response["result"] = []map[string]any{{
					"id":       "dashboard-id",
					"url_path": "gosungrow-flow",
					"title":    "GoSungrow Flow",
				}}
			case "lovelace/resources":
				response["result"] = []map[string]any{{
					"id":   "resource-id",
					"url":  "/local/gosungrow/gosungrow-energy-flow-card-v2.js?v=old",
					"type": "module",
				}}
			case "lovelace/resources/create", "lovelace/resources/update":
				if _, ok := request["res_type"]; !ok {
					t.Fatalf("expected res_type in resource request: %#v", request)
				}
				if _, ok := request["resource_type"]; ok {
					t.Fatalf("did not expect resource_type in resource request: %#v", request)
				}
				if request["type"] == "lovelace/resources/update" {
					sawResourceUpdate = true
				}
				response["result"] = map[string]any{}
			case "lovelace/config":
				response["result"] = map[string]any{
					"title": "GoSungrow Flow",
					"views": []any{},
				}
			default:
				response["result"] = map[string]any{}
			}

			if err := conn.WriteJSON(response); err != nil {
				t.Fatalf("write websocket response: %v", err)
			}
		}
	}))
	defer server.Close()

	ctx := context.Background()
	client, err := newHAWSClient(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), "supervisor-token")
	if err != nil {
		t.Fatalf("newHAWSClient: %v", err)
	}
	defer client.Close()

	dashboards, err := client.ListDashboards(ctx)
	if err != nil {
		t.Fatalf("ListDashboards: %v", err)
	}
	if len(dashboards) != 1 || dashboards[0].ID != "dashboard-id" {
		t.Fatalf("unexpected dashboards: %#v", dashboards)
	}

	config, err := client.GetConfig(ctx, "gosungrow-flow")
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if config["title"] != "GoSungrow Flow" {
		t.Fatalf("unexpected config: %#v", config)
	}

	opts := haDashboardInstallOptions{
		DashboardURLPath: "gosungrow-flow",
		DashboardTitle:   "GoSungrow Flow",
		DashboardIcon:    "mdi:solar-power",
		ShowInSidebar:    true,
	}
	if err := client.UpdateDashboard(ctx, "dashboard-id", opts); err != nil {
		t.Fatalf("UpdateDashboard: %v", err)
	}
	if err := client.CreateDashboard(ctx, opts); err != nil {
		t.Fatalf("CreateDashboard: %v", err)
	}
	if err := client.SaveConfig(ctx, "gosungrow-flow", map[string]any{"title": "GoSungrow Flow", "views": []any{}}); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	if err := client.EnsureResource(ctx, "data:text/javascript;base64,Zm9v#v=new", dashboardCardResourceType); err != nil {
		t.Fatalf("EnsureResource: %v", err)
	}
	if !sawResourceUpdate {
		t.Fatal("expected EnsureResource to update the existing managed dashboard card resource")
	}
}

func TestBundledDashboardTemplateRenders(t *testing.T) {
	assetDir := filepath.Join("..", "addon", "gosungrow", "assets")
	templatePath := filepath.Join(assetDir, dashboardTemplateFile)

	config, err := renderDashboardConfig(templatePath, "GoSungrow Flow", []haDashboardTarget{
		{PsID: "100", PsKey: "5072099_14_1_1", ViewTitle: "Roof", ViewPath: "roof"},
	}, defaultDashboardLocaleBundle)
	if err != nil {
		t.Fatalf("render bundled dashboard: %v", err)
	}

	rendered, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("marshal bundled dashboard: %v", err)
	}

	views, ok := config["views"].([]any)
	if !ok || len(views) != 2 {
		t.Fatalf("expected bundled dashboard to render 2 views, got %#v", config["views"])
	}

	text := string(rendered)
	if !strings.Contains(text, "\"type\":\"custom:gosungrow-energy-flow-card-v2\"") {
		t.Fatal("expected custom GoSungrow flow card in bundled dashboard")
	}
	if !strings.Contains(text, "sensor.gosungrow_virtual_5072099_14_1_1_pv_to_grid_power") {
		t.Fatal("expected pv_to_grid flow sensor in bundled dashboard")
	}
	if !strings.Contains(text, "sensor.gosungrow_virtual_5072099_14_1_1_grid_to_load_power") {
		t.Fatal("expected grid_to_load flow sensor in bundled dashboard")
	}
	if !strings.Contains(text, "sensor.gosungrow_virtual_5072099_14_1_1_p13141") {
		t.Fatal("expected battery soc sensor in bundled dashboard")
	}
	if !strings.Contains(text, "sensor.gosungrow_virtual_5072099_14_1_1_p13112") {
		t.Fatal("expected daily PV yield sensor in bundled dashboard")
	}
}

func TestBundledDashboardTemplateRendersSwedishAndInjectsCardLabels(t *testing.T) {
	assetDir := filepath.Join("..", "addon", "gosungrow", "assets")
	templatePath := filepath.Join(assetDir, dashboardTemplateFile)
	localeBundle, _, err := localizedDashboardBundle(assetDir, "sv-SE")
	if err != nil {
		t.Fatalf("localizedDashboardBundle: %v", err)
	}

	config, err := renderDashboardConfig(templatePath, "GoSungrow Flow", []haDashboardTarget{
		{PsID: "100", PsKey: "5072099_14_1_1", ViewTitle: "Roof", ViewPath: "roof"},
	}, localeBundle)
	if err != nil {
		t.Fatalf("render bundled dashboard: %v", err)
	}

	rendered, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("marshal bundled dashboard: %v", err)
	}
	text := string(rendered)
	if !strings.Contains(text, "\"heading\":\"Liveflöde\"") {
		t.Fatal("expected swedish localized heading")
	}
	if !strings.Contains(text, "\"name\":\"PV till last\"") {
		t.Fatal("expected swedish localized tile label")
	}
	if !strings.Contains(text, "\"labels\":{\"node_battery\":\"Batteri\"") && !strings.Contains(text, "\"node_battery\":\"Batteri\"") {
		t.Fatal("expected localized flow-card labels to be injected")
	}
}

func TestWriteDashboardInstallDiagnosticsIncludesSummaryAndUnresolvedRefs(t *testing.T) {
	var buf bytes.Buffer
	writeDashboardInstallDiagnostics(&buf, dashboardInstallDiagnostics{
		DiagnosticContext:    "Reconciling after MQTT startup (1)",
		HAStatesLoaded:       1284,
		GoSungrowStatesFound: 42,
		DashboardRefsFound:   23,
		RemappedRefs:         18,
		RemappedPreview: []dashboardEntityRemap{
			{
				From:   "sensor.gosungrow_virtual_123_pv_power",
				To:     "sensor.gosungrow_123_pv_information_pv_power",
				Metric: "pv_power",
			},
		},
		BatteryDetectionKnown: true,
		BatteryTargetsFound:   0,
		BatteryTargetsTotal:   1,
		TargetDiagnostics: []dashboardTargetDiagnostics{
			{
				PlantName:       "Roof",
				DeviceName:      "String inverter",
				PsID:            "1203332",
				PsKey:           "1203332_22_247_1",
				ViewPath:        "1203332-22-247-1",
				DeviceType:      22,
				SelectionSource: "fallback-first-valid-ps-key",
				GoSungrowStates: 97,
				VirtualStates:   0,
				ExampleGoSungrowStates: []string{
					"sensor.gosungrow_1203332_pv_information_pv_power",
					"sensor.gosungrow_1203332_grid_information_grid_power",
				},
			},
		},
		DashboardSaved:      true,
		DashboardSaveReason: "configuration changed",
		UnresolvedRefs: []dashboardUnresolvedEntityRef{
			{
				Entity: "sensor.gosungrow_virtual_123_pv_power",
				Reason: "no usable candidate entity matched metric \"pv_power\"",
			},
		},
	})

	text := buf.String()
	for _, expected := range []string{
		"Dashboard diagnostics:",
		"- context: Reconciling after MQTT startup (1)",
		"- HA states loaded: 1284",
		"- GoSungrow states found: 42",
		"- dashboard entity refs found: 23",
		"- remapped refs: 18",
		"- unresolved refs: 1",
		"- battery detected: false",
		"- dashboard saved: yes (configuration changed)",
		"Dashboard targets:",
		"- target[1]: plant=\"Roof\" device=\"String inverter\" ps_id=1203332 ps_key=1203332_22_247_1 device_type=22 selection=fallback-first-valid-ps-key view=1203332-22-247-1 gosungrow_states=97 virtual_states=0",
		"warning: selected fallback non-ESS device_type=22; full ESS virtual metrics may be unavailable",
		"warning: no target-specific gosungrow_virtual states were found",
		"example gosungrow states: sensor.gosungrow_1203332_pv_information_pv_power, sensor.gosungrow_1203332_grid_information_grid_power",
		"Remapped dashboard refs:",
		"- pv_power: sensor.gosungrow_virtual_123_pv_power -> sensor.gosungrow_123_pv_information_pv_power",
		"Unresolved dashboard refs:",
		"- sensor.gosungrow_virtual_123_pv_power: no usable candidate entity matched metric \"pv_power\"",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected diagnostics output to contain %q, got:\n%s", expected, text)
		}
	}
}
