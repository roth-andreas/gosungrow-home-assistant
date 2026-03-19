package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestRenderDashboardConfigInlinesAssetsAndTargets(t *testing.T) {
	assetDir := t.TempDir()
	templatePath := filepath.Join(assetDir, dashboardTemplateFile)

	template := `title: Template
views:
  - title: Prototype
    path: prototype
    cards:
      - type: picture-elements
        image: /local/gosungrow/SungrowEnergy2.png
        elements:
          - type: state-label
            entity: sensor.gosungrow_virtual_YOUR_ESS_PS_KEY_p13112
`
	if err := os.WriteFile(templatePath, []byte(template), 0600); err != nil {
		t.Fatalf("write template: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "SungrowEnergy2.png"), []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, 0600); err != nil {
		t.Fatalf("write asset: %v", err)
	}

	config, err := renderDashboardConfig(templatePath, assetDir, "GoSungrow Flow", []haDashboardTarget{
		{PsID: "100", PsKey: "5072099_14_1_1", ViewTitle: "Roof", ViewPath: "roof"},
		{PsID: "101", PsKey: "5080000_14_1_1", ViewTitle: "Garage", ViewPath: "garage"},
	})
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
	if !strings.Contains(text, "data:image/png;base64,") {
		t.Fatal("expected image asset to be embedded as a data URI")
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

func TestHAWSClientDashboardCalls(t *testing.T) {
	upgrader := websocket.Upgrader{}

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
}

func TestBundledDashboardTemplateRenders(t *testing.T) {
	assetDir := filepath.Join("..", "addon", "gosungrow", "assets")
	templatePath := filepath.Join(assetDir, dashboardTemplateFile)

	config, err := renderDashboardConfig(templatePath, assetDir, "GoSungrow Flow", []haDashboardTarget{
		{PsID: "100", PsKey: "5072099_14_1_1", ViewTitle: "Roof", ViewPath: "roof"},
	})
	if err != nil {
		t.Fatalf("render bundled dashboard: %v", err)
	}

	rendered, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("marshal bundled dashboard: %v", err)
	}

	text := string(rendered)
	if !strings.Contains(text, "sensor.gosungrow_virtual_5072099_14_1_1_pv_to_grid_power") {
		t.Fatal("expected pv_to_grid flow sensor in bundled dashboard")
	}
	if !strings.Contains(text, "sensor.gosungrow_virtual_5072099_14_1_1_grid_to_load_power") {
		t.Fatal("expected grid_to_load flow sensor in bundled dashboard")
	}
	if !strings.Contains(text, "binary_sensor.gosungrow_virtual_5072099_14_1_1_pv_to_load_active") {
		t.Fatal("expected pv_to_load active binary sensor in bundled dashboard")
	}
	if !strings.Contains(text, "binary_sensor.gosungrow_virtual_5072099_14_1_1_grid_to_load_active") {
		t.Fatal("expected grid_to_load active binary sensor in bundled dashboard")
	}
	if !strings.Contains(text, "data:image/svg+xml;base64,") {
		t.Fatal("expected bundled dashboard base SVG to be embedded as a data URI")
	}
}
