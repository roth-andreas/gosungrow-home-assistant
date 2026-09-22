package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
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

func TestEnrichDashboardStatesWithEntityRegistry(t *testing.T) {
	states := []haState{{EntityID: "sensor.renamed", State: "4"}, {EntityID: "sensor.untouched", State: "5"}}
	registry := []haEntityRegistryEntry{{EntityID: "SENSOR.RENAMED", UniqueID: "gosungrow_100_11_0_0_p13112", DeviceID: "device-1", Platform: "mqtt"}}
	got := enrichDashboardStatesWithRegistry(states, registry)
	if got[0].RegistryUniqueID != registry[0].UniqueID || got[0].RegistryDeviceID != "device-1" || got[0].RegistryPlatform != "mqtt" {
		t.Fatalf("registry metadata missing: %#v", got[0])
	}
	if got[1].RegistryUniqueID != "" {
		t.Fatalf("unmatched state was changed: %#v", got[1])
	}
	if states[0].RegistryUniqueID != "" {
		t.Fatal("input states were mutated")
	}
}

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
	if len(targets[0].PlantDevices) != 2 {
		t.Fatalf("expected all plant devices in diagnostics metadata, got %#v", targets[0].PlantDevices)
	}
	if !targets[0].PlantDevices[1].Selected {
		t.Fatalf("expected selected target to be marked in plant devices, got %#v", targets[0].PlantDevices)
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
		DashboardURLPath:       "gosungrow-flow",
		DashboardHash:          "abc123",
		DashboardStructureHash: "structure123",
		TargetPsKeys:           []string{"5072099_14_1_1"},
		AssetMode:              dashboardAssetModeEnhanced,
		AssetURL:               "/local/gosungrow/gosungrow-dashboard-cards.abc123abc123.js",
		AssetHash:              strings.Repeat("a", 64),
		PreviousAssetURL:       "/local/gosungrow/gosungrow-dashboard-cards.def456def456.js",
		PreviousAssetHash:      strings.Repeat("d", 64),
		UpdatedAt:              "2026-03-19T12:00:00Z",
	}
	if err := saveDashboardState(statePath, state); err != nil {
		t.Fatalf("saveDashboardState: %v", err)
	}

	loaded, err := loadDashboardState(statePath)
	if err != nil {
		t.Fatalf("loadDashboardState: %v", err)
	}
	if loaded == nil || loaded.DashboardURLPath != state.DashboardURLPath || loaded.DashboardHash != state.DashboardHash || loaded.DashboardStructureHash != state.DashboardStructureHash || loaded.AssetMode != state.AssetMode || loaded.AssetHash != state.AssetHash || loaded.PreviousAssetHash != state.PreviousAssetHash {
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

func TestLoadDashboardStateAcceptsLegacyStateWithoutAssetFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), dashboardStateFileName)
	legacy := []byte(`{"dashboard_url_path":"gosungrow-flow","dashboard_hash":"legacy","updated_at":"2026-01-01T00:00:00Z"}`)
	if err := os.WriteFile(path, legacy, 0600); err != nil {
		t.Fatal(err)
	}
	state, err := loadDashboardState(path)
	if err != nil {
		t.Fatalf("loadDashboardState: %v", err)
	}
	if state.DashboardURLPath != "gosungrow-flow" || state.DashboardHash != "legacy" || state.AssetMode != "" || state.AssetURL != "" || state.AssetHash != "" {
		t.Fatalf("unexpected migrated legacy state: %#v", state)
	}
}

func TestInstallDashboardCardAssetWritesVersionedResource(t *testing.T) {
	assetDir := t.TempDir()
	configDir := t.TempDir()
	cardSource := filepath.Join(assetDir, dashboardCardSourceFile)
	cardBody := []byte("console.log('gosungrow card');")

	if err := os.WriteFile(cardSource, cardBody, 0600); err != nil {
		t.Fatalf("write card source: %v", err)
	}

	resourceURL, version, err := installDashboardCardAsset(assetDir, configDir)
	if err != nil {
		t.Fatalf("installDashboardCardAsset: %v", err)
	}

	expectedHash := fmt.Sprintf("%x", sha256.Sum256(cardBody))
	if version != expectedHash {
		t.Fatalf("unexpected full asset hash: got %q want %q", version, expectedHash)
	}
	expectedURL := "/local/gosungrow/gosungrow-dashboard-cards." + expectedHash[:12] + ".js"
	if resourceURL != expectedURL {
		t.Fatalf("unexpected resource URL: got %q want %q", resourceURL, expectedURL)
	}

	targetPath := filepath.Join(configDir, "www", dashboardCardResourceDir, "gosungrow-dashboard-cards."+expectedHash[:12]+".js")
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

func TestVerifyDashboardCardAssetRequiresExactResponse(t *testing.T) {
	body := []byte("customElements.define('x-test', class extends HTMLElement {});")
	expectedHash := fmt.Sprintf("%x", sha256.Sum256(body))
	tests := []struct {
		name        string
		status      int
		contentType string
		body        []byte
		wantError   string
	}{
		{name: "verified", status: http.StatusOK, contentType: "text/javascript; charset=utf-8", body: body},
		{name: "status", status: http.StatusNotFound, contentType: "text/javascript", body: body, wantError: "HTTP 404"},
		{name: "redirect", status: http.StatusFound, contentType: "text/javascript", body: body, wantError: "HTTP 302"},
		{name: "mime", status: http.StatusOK, contentType: "text/plain", body: body, wantError: "non-JavaScript MIME"},
		{name: "hash", status: http.StatusOK, contentType: "application/javascript", body: []byte("different"), wantError: "hash mismatch"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/local/gosungrow/gosungrow-dashboard-cards.0123456789ab.js" {
					t.Fatalf("unexpected path %q", r.URL.Path)
				}
				if got := r.Header.Get("Authorization"); got != "" {
					t.Fatalf("static request leaked authorization header %q", got)
				}
				w.Header().Set("Content-Type", tt.contentType)
				w.WriteHeader(tt.status)
				_, _ = w.Write(tt.body)
			}))
			defer server.Close()

			wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/websocket"
			verification, err := verifyDashboardCardAsset(context.Background(), wsURL, "/local/gosungrow/gosungrow-dashboard-cards.0123456789ab.js", expectedHash)
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("verifyDashboardCardAsset: %v", err)
				}
				if verification.StatusCode != http.StatusOK || verification.MIMEType != "text/javascript" || verification.Route != dashboardHTTPRouteWebsocketOrigin {
					t.Fatalf("unexpected verification: %#v", verification)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("expected error containing %q, got %v", tt.wantError, err)
			}
		})
	}
}

type dashboardHTTPClientFunc func(*http.Request) (*http.Response, error)

func (f dashboardHTTPClientFunc) Do(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestVerifyDashboardCardAssetSeparatesSupervisorAndStaticOrigins(t *testing.T) {
	body := []byte("customElements.define('x-test', class extends HTMLElement {});")
	expectedHash := fmt.Sprintf("%x", sha256.Sum256(body))
	requests := 0
	client := dashboardHTTPClientFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		if got, want := request.URL.String(), "http://homeassistant:8123/local/gosungrow/gosungrow-dashboard-cards.0123456789ab.js"; got != want {
			t.Fatalf("unexpected static URL: got %q want %q", got, want)
		}
		if request.URL.Hostname() == "supervisor" {
			t.Fatal("static request reached Supervisor proxy")
		}
		if got := request.Header.Get("Authorization"); got != "" {
			t.Fatalf("static request leaked authorization header %q", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/javascript"}},
			Body:       io.NopCloser(bytes.NewReader(body)),
			Request:    request,
		}, nil
	})

	verification, err := verifyDashboardCardAssetWithClient(
		context.Background(),
		client,
		"ws://supervisor/core/websocket",
		"/local/gosungrow/gosungrow-dashboard-cards.0123456789ab.js",
		expectedHash,
	)
	if err != nil {
		t.Fatalf("verifyDashboardCardAssetWithClient: %v", err)
	}
	if requests != 1 {
		t.Fatalf("unexpected request count: %d", requests)
	}
	if verification.Route != dashboardHTTPRouteDirectCore {
		t.Fatalf("unexpected verification route: %q", verification.Route)
	}
}

func TestDashboardHTTPResourceURLPreservesStandaloneBasePath(t *testing.T) {
	got, route, err := dashboardHTTPResourceURL(
		"wss://user:secret@example.test/ha/api/websocket?token=secret#fragment",
		"/local/gosungrow/card.js",
	)
	if err != nil {
		t.Fatalf("dashboardHTTPResourceURL: %v", err)
	}
	if want := "https://example.test/ha/local/gosungrow/card.js"; got != want {
		t.Fatalf("unexpected resource URL: got %q want %q", got, want)
	}
	if route != dashboardHTTPRouteWebsocketOrigin {
		t.Fatalf("unexpected route: %q", route)
	}
}

func TestNativeDashboardFallbackRemovesCustomCardsAndKeepsCanonicalOrder(t *testing.T) {
	config := map[string]any{
		"views": []any{
			map[string]any{"path": "overview", "cards": []any{map[string]any{
				"type": dashboardEnergyFlowCardType,
				"entities": map[string]any{
					"battery_soc": "sensor.soc", "grid_power": "sensor.grid", "solar_power": "sensor.pv",
					"load_power": "sensor.load", "battery_power": "sensor.battery", "pv_to_load_power": "sensor.pv_load",
					"pv_to_battery_power": "sensor.pv_battery", "pv_to_grid_power": "sensor.pv_grid",
					"grid_to_load_power": "sensor.grid_load", "battery_to_load_power": "sensor.battery_load",
				},
			}}},
			map[string]any{"path": "aggregates", "cards": []any{map[string]any{
				"type": "custom:gosungrow-energy-summary-card-v1", "entities": map[string]any{"production": "sensor.production", "consumption": "sensor.consumption"},
			}}},
			map[string]any{"path": "data-sources", "cards": []any{map[string]any{"type": dashboardSourceMappingCardType}}},
		},
	}

	fallback, err := nativeDashboardFallback(config, defaultDashboardLocaleBundle)
	if err != nil {
		t.Fatalf("nativeDashboardFallback: %v", err)
	}
	if dashboardConfigContainsCustomGoSungrow(fallback) {
		t.Fatalf("fallback contains custom cards: %#v", fallback)
	}
	views := fallback["views"].([]any)
	flow := views[0].(map[string]any)["cards"].([]any)[0].(map[string]any)
	rows := flow["entities"].([]any)
	want := []string{"sensor.pv", "sensor.load", "sensor.grid", "sensor.battery", "sensor.pv_load", "sensor.pv_battery", "sensor.pv_grid", "sensor.grid_load", "sensor.battery_load", "sensor.soc"}
	if len(rows) != len(want) {
		t.Fatalf("unexpected fallback row count: %#v", rows)
	}
	for index, entity := range want {
		if got := rows[index].(map[string]any)["entity"]; got != entity {
			t.Fatalf("row %d: got %v want %s", index, got, entity)
		}
	}
	if got := views[2].(map[string]any)["cards"].([]any)[0].(map[string]any)["type"]; got != "markdown" {
		t.Fatalf("source mapping was not replaced with native informational card: %v", got)
	}
}

func TestCleanupDashboardAssetFilesRetainsActiveAndPrevious(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "www", dashboardCardResourceDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	active := strings.Repeat("a", 64)
	previous := strings.Repeat("b", 64)
	for _, name := range []string{
		"gosungrow-dashboard-cards." + active[:12] + ".js",
		"gosungrow-dashboard-cards." + previous[:12] + ".js",
		"gosungrow-dashboard-cards.cccccccccccc.js",
		dashboardCardSourceFile,
		"unrelated.js",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := cleanupDashboardAssetFiles(root, active, previous); err != nil {
		t.Fatalf("cleanupDashboardAssetFiles: %v", err)
	}
	for _, name := range []string{"gosungrow-dashboard-cards." + active[:12] + ".js", "gosungrow-dashboard-cards." + previous[:12] + ".js", "unrelated.js"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("expected %s to remain: %v", name, err)
		}
	}
	for _, name := range []string{"gosungrow-dashboard-cards.cccccccccccc.js", dashboardCardSourceFile} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be removed, got %v", name, err)
		}
	}
}

func TestHAWSClientDashboardCalls(t *testing.T) {
	upgrader := websocket.Upgrader{}
	sawResourceUpdate := false
	sawResourceReload := false

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
			case "lovelace/resources/list":
				response["result"] = []map[string]any{{
					"id":   "resource-id",
					"url":  "data:text/javascript;base64,Zm9v#v=old",
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
			case "get_services":
				response["result"] = map[string]any{"lovelace": map[string]any{"reload_resources": map[string]any{}}}
			case "call_service":
				if request["domain"] == "lovelace" && request["service"] == "reload_resources" {
					sawResourceReload = true
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
	if err := client.EnsureResource(ctx, "/local/gosungrow/gosungrow-dashboard-cards.0123456789ab.js", dashboardCardResourceType); err != nil {
		t.Fatalf("EnsureResource: %v", err)
	}
	if !sawResourceUpdate {
		t.Fatal("expected EnsureResource to update the existing managed dashboard card resource")
	}
	reloaded, err := client.ReloadResourcesIfSupported(ctx)
	if err != nil || !reloaded || !sawResourceReload {
		t.Fatalf("expected supported resource reload request, reloaded=%t saw=%t err=%v", reloaded, sawResourceReload, err)
	}
}

func TestDashboardReloadServiceSupportIsModeAware(t *testing.T) {
	if dashboardReloadServiceSupported(map[string]map[string]json.RawMessage{"lovelace": {}}) {
		t.Fatal("storage mode without a reload service must not request the YAML-only action")
	}
	if !dashboardReloadServiceSupported(map[string]map[string]json.RawMessage{"lovelace": {"reload_resources": json.RawMessage(`{}`)}}) {
		t.Fatal("exposed reload service was not detected")
	}
}

func TestHAWSClientEnsureResourceUpdatesStaleManagedCardURLs(t *testing.T) {
	tests := []struct {
		name        string
		existingURL string
	}{
		{
			name:        "unversioned local resource",
			existingURL: "/local/gosungrow/gosungrow-energy-flow-card-v2.js",
		},
		{
			name:        "old versioned local resource",
			existingURL: "/local/gosungrow/gosungrow-energy-flow-card-v2.js?v=old",
		},
		{
			name:        "legacy basename-only resource",
			existingURL: "/local/gosungrow-energy-flow-card-v2.js",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upgrader := websocket.Upgrader{}
			updatedURL := ""
			created := false

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
					case "lovelace/resources/list":
						response["result"] = []map[string]any{{
							"id":   "resource-id",
							"url":  tt.existingURL,
							"type": dashboardCardResourceType,
						}}
					case "lovelace/resources/update":
						updatedURL, _ = request["url"].(string)
						if got := request["resource_id"]; got != "resource-id" {
							t.Fatalf("unexpected resource id: %#v", request)
						}
						response["result"] = map[string]any{}
					case "lovelace/resources/create":
						created = true
						response["result"] = map[string]any{}
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

			newURL := "/local/gosungrow/gosungrow-dashboard-cards.0123456789ab.js"
			if err := client.EnsureResource(ctx, newURL, dashboardCardResourceType); err != nil {
				t.Fatalf("EnsureResource: %v", err)
			}
			if created {
				t.Fatal("expected stale managed resource to be updated, not duplicated")
			}
			if updatedURL != newURL {
				t.Fatalf("unexpected updated URL: got %q want %q", updatedURL, newURL)
			}
		})
	}
}

func TestManagedResourceActivationCleanupAndRollback(t *testing.T) {
	upgrader := websocket.Upgrader{}
	resources := []haResourceMetadata{
		{ID: "managed-primary", URL: "https://cdn.example/gosungrow-energy-flow-card-v2.js", ResourceType: dashboardCardResourceType},
		{ID: "managed-duplicate", URL: "data:text/javascript;base64,Zm9v#v=old", ResourceType: dashboardCardResourceType},
		{ID: "unrelated", URL: "/local/community/other-card.js", ResourceType: dashboardCardResourceType},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("upgrade websocket: %v", err)
		}
		defer conn.Close()
		_ = conn.WriteJSON(map[string]any{"type": "auth_required"})
		var auth map[string]any
		if err := conn.ReadJSON(&auth); err != nil {
			return
		}
		_ = conn.WriteJSON(map[string]any{"type": "auth_ok"})
		for {
			var request map[string]any
			if err := conn.ReadJSON(&request); err != nil {
				return
			}
			response := map[string]any{"id": request["id"], "type": "result", "success": true, "result": map[string]any{}}
			switch request["type"] {
			case "lovelace/resources/list":
				response["result"] = resources
			case "lovelace/resources/update":
				id := fmt.Sprint(request["resource_id"])
				for index := range resources {
					if fmt.Sprint(resources[index].ID) == id {
						resources[index].URL = fmt.Sprint(request["url"])
						resources[index].ResourceType = fmt.Sprint(request["res_type"])
					}
				}
			case "lovelace/resources/delete":
				id := fmt.Sprint(request["resource_id"])
				kept := resources[:0]
				for _, resource := range resources {
					if fmt.Sprint(resource.ID) != id {
						kept = append(kept, resource)
					}
				}
				resources = kept
			case "lovelace/resources/create":
				resources = append(resources, haResourceMetadata{ID: fmt.Sprintf("created-%d", len(resources)), URL: fmt.Sprint(request["url"]), ResourceType: fmt.Sprint(request["res_type"])})
			}
			if err := conn.WriteJSON(response); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	ctx := context.Background()
	client, err := newHAWSClient(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), "supervisor-token")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	canonicalURL := "/local/gosungrow/gosungrow-dashboard-cards.0123456789ab.js"
	change, err := client.ActivateManagedResource(ctx, canonicalURL)
	if err != nil {
		t.Fatalf("ActivateManagedResource: %v", err)
	}
	if change.Action != "updated" || len(change.Before) != 2 || change.Canonical.URL != canonicalURL {
		t.Fatalf("unexpected resource change: %#v", change)
	}
	if err := client.RemoveDuplicateManagedResources(ctx, change.Canonical.ID); err != nil {
		t.Fatalf("RemoveDuplicateManagedResources: %v", err)
	}
	if len(resources) != 2 || resources[0].URL != canonicalURL || resources[1].ID != "unrelated" {
		t.Fatalf("unexpected resources after cleanup: %#v", resources)
	}
	if err := client.RestoreManagedResources(ctx, change.Before); err != nil {
		t.Fatalf("RestoreManagedResources: %v", err)
	}
	managed := managedDashboardResources(resources)
	if len(managed) != 2 || managed[0].URL != change.Before[0].URL || managed[1].URL != change.Before[1].URL {
		t.Fatalf("managed resource snapshot was not restored: %#v", resources)
	}
	if len(resources) != 3 {
		t.Fatalf("unrelated resource changed during rollback: %#v", resources)
	}
}

func TestMatchesManagedDashboardCardResourceMigrationForms(t *testing.T) {
	for _, value := range []string{
		"data:text/javascript;base64,Zm9v#v=old",
		"/local/gosungrow/gosungrow-energy-flow-card-v2.js",
		"https://cdn.example/assets/gosungrow-energy-flow-card-v2.js?v=1",
		"/local/gosungrow/gosungrow-dashboard-cards.0123456789ab.js",
	} {
		if !matchesManagedDashboardCardResource(value) {
			t.Fatalf("expected managed resource match for %q", value)
		}
	}
	if matchesManagedDashboardCardResource("/local/community/unrelated-gosungrow-card.js") {
		t.Fatal("unrelated resource was classified as managed")
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
	if !ok || len(views) != 4 {
		t.Fatalf("expected bundled dashboard to render 4 views, got %#v", config["views"])
	}
	if got := views[1].(map[string]any)["path"]; got != "aggregates" {
		t.Fatalf("expected second bundled dashboard view to be aggregates, got %v", got)
	}
	if got := views[2].(map[string]any)["path"]; got != "trends" {
		t.Fatalf("expected third bundled dashboard view to be trends, got %v", got)
	}
	if got := views[3].(map[string]any)["path"]; got != "data-sources" {
		t.Fatalf("expected fourth bundled dashboard view to be data-sources, got %v", got)
	}
	if got := views[3].(map[string]any)["type"]; got != "panel" {
		t.Fatalf("expected data-sources to use a full-width panel view, got %v", got)
	}

	text := string(rendered)
	if !strings.Contains(text, "\"type\":\"custom:gosungrow-energy-flow-card-v2\"") {
		t.Fatal("expected custom GoSungrow flow card in bundled dashboard")
	}
	if !strings.Contains(text, "\"type\":\"custom:gosungrow-energy-summary-card-v1\"") {
		t.Fatal("expected custom GoSungrow energy summary card in bundled dashboard")
	}
	if !strings.Contains(text, "\"type\":\"custom:gosungrow-source-mapping-card-v1\"") {
		t.Fatal("expected custom GoSungrow source mapping card in bundled dashboard")
	}
	if !strings.Contains(text, "\"buckets\":{\"day\":14,\"month\":12,\"year\":5}") {
		t.Fatal("expected summary card bucket defaults in bundled dashboard")
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
	if !strings.Contains(text, "sensor.gosungrow_virtual_5072099_14_1_1_p13199") {
		t.Fatal("expected summary consumption sensor in bundled dashboard")
	}
	if !strings.Contains(text, "sensor.gosungrow_virtual_5072099_14_1_1_p13029") {
		t.Fatal("expected summary battery discharge sensor in bundled dashboard")
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
	if !strings.Contains(text, "\"title\":\"Energisammanfattning\"") {
		t.Fatal("expected swedish localized summary card title")
	}
	if !strings.Contains(text, "\"period_month\":\"Månad\"") {
		t.Fatal("expected swedish localized summary card labels")
	}
}

func TestWriteDashboardInstallDiagnosticsIncludesSummaryAndUnresolvedRefs(t *testing.T) {
	var buf bytes.Buffer
	writeDashboardInstallDiagnostics(&buf, dashboardInstallDiagnostics{
		DiagnosticContext:    "Reconciling after MQTT startup (1)",
		AssetPhase:           "committed",
		AssetHash:            strings.Repeat("a", 64),
		AssetURL:             "/local/gosungrow/gosungrow-dashboard-cards.aaaaaaaaaaaa.js",
		AssetHTTPRoute:       dashboardHTTPRouteDirectCore,
		AssetHTTPStatus:      http.StatusOK,
		AssetMIMEType:        "text/javascript",
		ResourceAction:       "updated",
		DashboardMode:        dashboardAssetModeEnhanced,
		RollbackResult:       "not required",
		CleanupResult:        "complete",
		HAStatesLoaded:       1284,
		GoSungrowStatesFound: 42,
		DashboardRefsFound:   23,
		RemappedRefs:         18,
		RemappedPreview: []dashboardEntityRemap{
			{
				From:   "sensor.gosungrow_virtual_123_pv_power",
				To:     "sensor.gosungrow_123_pv_information_pv_power",
				Metric: "pv_power",
				Source: "inverter-level",
			},
		},
		Debug:                 true,
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
				PlantDevices: []dashboardPlantDevice{
					{
						PlantName:       "Roof",
						DeviceName:      "Large inverter",
						PsID:            "1203332",
						PsKey:           "1203332_1_1_1",
						DeviceType:      1,
						SelectionSource: "preferred-device-type-1",
					},
					{
						PlantName:       "Roof",
						DeviceName:      "String inverter",
						PsID:            "1203332",
						PsKey:           "1203332_22_247_1",
						DeviceType:      22,
						Selected:        true,
						SelectionSource: "fallback-first-valid-ps-key",
					},
				},
				GoSungrowStates: 97,
				VirtualStates:   0,
				ExampleGoSungrowStates: []string{
					"sensor.gosungrow_1203332_pv_information_pv_power",
					"sensor.gosungrow_1203332_grid_information_grid_power",
				},
			},
		},
		MetricTraces: []dashboardMetricTrace{
			{
				Placeholder: "sensor.gosungrow_virtual_123_pv_power",
				Metric:      "pv_power",
				TargetPsKey: "1203332_22_247_1",
				Resolved:    "sensor.gosungrow_123_pv_information_pv_power",
				Source:      "inverter-level",
				Candidates: []dashboardMetricCandidate{
					{
						Entity: "sensor.gosungrow_123_pv_information_pv_power",
						Metric: "pv_power",
						Score:  240,
						State:  "2.60",
						Unit:   "kW",
						Source: "inverter-level",
						Reason: "usable candidate",
					},
				},
			},
		},
		AggregateHints: []dashboardAggregateHint{
			{
				Metric: "pv_power",
				Entity: "sensor.gosungrow_123_pv_information_pv_power",
				State:  "2.60",
				Unit:   "kW",
				Source: "inverter-level",
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
		"- asset: phase=committed hash=" + strings.Repeat("a", 64) + " url=/local/gosungrow/gosungrow-dashboard-cards.aaaaaaaaaaaa.js http_route=direct-core http_status=200 mime=text/javascript resource_action=updated dashboard_mode=enhanced rollback=not required cleanup=complete",
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
		"warning: selected fallback target while other inverter-like devices exist",
		"warning: no target-specific gosungrow_virtual states were found",
		"example gosungrow states: sensor.gosungrow_1203332_pv_information_pv_power, sensor.gosungrow_1203332_grid_information_grid_power",
		"plant devices:",
		"available ps_id=1203332 ps_key=1203332_1_1_1 device=\"Large inverter\" device_type=1 selection=preferred-device-type-1",
		"selected ps_id=1203332 ps_key=1203332_22_247_1 device=\"String inverter\" device_type=22 selection=fallback-first-valid-ps-key",
		"Remapped dashboard refs:",
		"- pv_power [inverter-level]: sensor.gosungrow_virtual_123_pv_power -> sensor.gosungrow_123_pv_information_pv_power",
		"Potential aggregate sources:",
		"- pv_power [inverter-level] state=2.60 kW entity=sensor.gosungrow_123_pv_information_pv_power",
		"Dashboard metric candidates:",
		"candidate score=240 source=inverter-level state=2.60 kW entity=sensor.gosungrow_123_pv_information_pv_power reason=usable candidate",
		"Unresolved dashboard refs:",
		"- sensor.gosungrow_virtual_123_pv_power: no usable candidate entity matched metric \"pv_power\"",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected diagnostics output to contain %q, got:\n%s", expected, text)
		}
	}
}
