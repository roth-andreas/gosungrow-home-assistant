package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestDashboardObserverUsesSeparateSnapshotConnectionAndCancelsIdleRead(t *testing.T) {
	target := haDashboardTarget{PsID: "100", PsKey: "100_55_1_1"}
	state := haState{EntityID: "sensor.renamed_plant_pv_2", State: "0", LastUpdated: time.Now().UTC().Format(time.RFC3339Nano), Attributes: map[string]any{"unit_of_measurement": "kW"}}
	upgrader := websocket.Upgrader{}
	events := make(chan map[string]any, 4)
	subscribed := make(chan struct{}, 1)
	var connections, snapshotConnection, subscriptionConnection int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		conn, err := upgrader.Upgrade(w, request, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		connectionID := atomic.AddInt32(&connections, 1)
		if err := conn.WriteJSON(map[string]any{"type": "auth_required"}); err != nil {
			return
		}
		var auth map[string]any
		if conn.ReadJSON(&auth) != nil {
			return
		}
		if err := conn.WriteJSON(map[string]any{"type": "auth_ok"}); err != nil {
			return
		}
		for {
			var command map[string]any
			if conn.ReadJSON(&command) != nil {
				return
			}
			response := map[string]any{"id": command["id"], "type": "result", "success": true}
			switch command["type"] {
			case "subscribe_events":
				atomic.StoreInt32(&subscriptionConnection, connectionID)
				if conn.WriteJSON(response) != nil {
					return
				}
				subscribed <- struct{}{}
				closed := make(chan struct{})
				go func() { defer close(closed); _, _, _ = conn.ReadMessage() }()
				for {
					select {
					case <-closed:
						return
					case event := <-events:
						if conn.WriteJSON(map[string]any{"id": command["id"], "type": "event", "event": event}) != nil {
							return
						}
					}
				}
			case "get_states":
				atomic.StoreInt32(&snapshotConnection, connectionID)
				response["result"] = []haState{state}
			case "config/entity_registry/list":
				response["result"] = []haEntityRegistryEntry{{EntityID: state.EntityID, UniqueID: dashboardPlantPVUniqueID(target), Platform: "mqtt"}}
			case "lovelace/config":
				response["result"] = map[string]any{}
			default:
				t.Errorf("observer sent an unexpected operation: %v", command["type"])
				return
			}
			if conn.WriteJSON(response) != nil {
				return
			}
		}
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &dashboardCoordinator{targets: []haDashboardTarget{target}, changes: make(chan struct{}, 1), updates: make(chan struct{}, 1)}
	opts := haDashboardInstallOptions{HomeAssistantWSURL: "ws" + strings.TrimPrefix(server.URL, "http"), SupervisorToken: "synthetic-test-token"}
	ready, done := make(chan struct{}, 1), make(chan error, 1)
	var previous *dashboardEligibilityView
	go func() { done <- c.observeConnection(ctx, opts, &previous, func() { ready <- struct{}{} }) }()
	dashboardTestReceive(t, subscribed)
	dashboardTestReceive(t, ready)
	if atomic.LoadInt32(&subscriptionConnection) == atomic.LoadInt32(&snapshotConnection) {
		t.Fatal("snapshot and subscription shared a websocket reader")
	}
	// The all-events subscription must ignore unrelated data shapes safely.
	events <- map[string]any{"event_type": "unrelated", "data": map[string]any{"new_state": "not a state object"}}
	unavailable := state
	unavailable.State = "unavailable"
	events <- map[string]any{"event_type": "state_changed", "data": map[string]any{"entity_id": state.EntityID, "new_state": unavailable}}
	dashboardTestReceive(t, c.changes)
	cancel()
	if err := dashboardTestReceive(t, done); err != context.Canceled {
		t.Fatalf("observer cancellation: %v", err)
	}
}
