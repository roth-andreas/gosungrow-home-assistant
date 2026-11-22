package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/roth-andreas/gosungrow-home-assistant/cmdHassio"
)

const (
	selectDiscoveryPrefix = "homeassistant/select/GoSungrow/"
	optionUniqueIDPrefix  = "GoSungrow-option-"
)

var smokeOptionIDs = []string{"mqtt_loglevel", "loglevel", "fetchschedule", "sleepdelay", "servicestate"}

func TestRuntimeSelectHomeAssistant(t *testing.T) {
	baseURL := os.Getenv("HA_E2E_MQTT_CORE_URL")
	if baseURL == "" {
		t.Skip("run tools/ha-e2e/mqtt_options_smoke.sh to provision Home Assistant and Mosquitto")
	}
	scenario := os.Getenv("HA_E2E_MQTT_CASE")
	t.Logf("Home Assistant MQTT runtime select scenario: %s", scenario)
	address := os.Getenv("HA_E2E_MQTT_BROKER_ADDR")
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		t.Fatal(err)
	}
	broker := mqtt.NewClient(mqtt.NewClientOptions().
		AddBroker("tcp://" + address).
		SetClientID("gosungrow-options-smoke").
		SetConnectRetry(true).
		SetConnectRetryInterval(time.Second).
		SetConnectTimeout(5 * time.Second))
	waitSelectMQTT(t, broker.Connect())
	t.Cleanup(func() { broker.Disconnect(250) })

	switch scenario {
	case "fresh":
	case "retained", "registry":
		seedLegacySelectDiscovery(t, broker, scenario == "registry")
	default:
		t.Fatalf("unknown runtime select fixture %q", scenario)
	}

	ha := &selectSmokeAPI{baseURL: baseURL, client: &http.Client{Timeout: 10 * time.Second}}
	ha.onboard(t)
	entryID := ha.configureMQTT(t)
	ws, err := newHAWSClient(context.Background(), strings.Replace(baseURL, "http", "ws", 1)+"/api/websocket", ha.token)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ws.Close() })

	var prior selectSmokeRegistry
	if scenario != "fresh" {
		prior = waitSelectRegistry(t, ws, 20*time.Second, func(r selectSmokeRegistry) bool {
			return len(r.services) == 1 && (scenario == "retained" || r.allOn(r.services[0].ID))
		})
		if len(prior.services) != 1 {
			t.Fatalf("legacy service was not discovered: %+v", prior)
		}
	}
	if scenario == "registry" {
		if !prior.allOn(prior.services[0].ID) || len(prior.services[0].Connections) != 2 {
			t.Fatalf("legacy selects and connection aliases were not attached to the service: %+v", prior)
		}
		const renamedID = "select.gosungrow_custom_fetch_schedule"
		if err := ws.call(map[string]any{
			"type": "config/entity_registry/update", "entity_id": prior.entities[optionUniqueIDPrefix+"fetchschedule"].EntityID,
			"new_entity_id": renamedID,
		}, nil); err != nil {
			t.Fatal(err)
		}
		prior = readSelectRegistry(t, ws)
		if prior.entities[optionUniqueIDPrefix+"fetchschedule"].EntityID != renamedID {
			t.Fatal("fixture did not preserve the user rename before upgrade")
		}
	}

	var appClient *cmdHassio.Mqtt
	t.Cleanup(func() {
		if appClient != nil {
			_ = appClient.Disconnect()
		}
	})
	startAppMQTT := func() {
		command := NewCmdMqtt("info")
		command.Client = cmdHassio.New(cmdHassio.Mqtt{
			Host: host, Port: port, ClientId: DefaultServiceName, EntityPrefix: DefaultServiceName,
		})
		command.Client.DeviceName = DefaultServiceName
		appClient = command.Client
		if err := command.Client.Connect(); err != nil {
			t.Fatal(err)
		}
		if err := command.Options(); err != nil {
			t.Fatal(err)
		}
	}

	startAppMQTT()
	assertRetainedSelectDiscovery(t, broker)
	upgraded := waitSelectRegistry(t, ws, 8*time.Second, selectSmokeRegistry.isCorrect)
	if scenario == "registry" && !upgraded.isCorrect() {
		ha.postJSON(t, "/api/config/config_entries/entry/"+entryID+"/reload", nil, nil)
		upgraded = waitSelectRegistry(t, ws, 20*time.Second, selectSmokeRegistry.isCorrect)
		if upgraded.isCorrect() {
			t.Log("Existing select registry entries required one MQTT integration reload after app startup")
		}
	}
	if !upgraded.isCorrect() {
		t.Fatalf("options device and five select entries did not converge: %+v", upgraded)
	}
	if scenario != "fresh" && upgraded.services[0].ID != prior.services[0].ID {
		t.Fatal("upgrade replaced the service device")
	}

	if scenario == "registry" {
		if !reflect.DeepEqual(prior.entityIDs(), upgraded.entityIDs()) {
			t.Fatalf("upgrade changed entity IDs: before %v, after %v", prior.entityIDs(), upgraded.entityIDs())
		}
		if err := appClient.Disconnect(); err != nil {
			t.Fatal(err)
		}
		startAppMQTT()
		assertRetainedSelectDiscovery(t, broker)
		restarted := readSelectRegistry(t, ws)
		if !restarted.isCorrect() || restarted.services[0].ID != upgraded.services[0].ID ||
			restarted.options[0].ID != upgraded.options[0].ID || !reflect.DeepEqual(restarted.entityIDs(), upgraded.entityIDs()) {
			t.Fatalf("repeated startup changed the upgraded registry: %+v", restarted)
		}
	}

	fetchSchedule := upgraded.entities[optionUniqueIDPrefix+"fetchschedule"].EntityID
	waitSelectState(t, ws, fetchSchedule, "5m")
	if err := ws.call(map[string]any{
		"type": "call_service", "domain": "select", "service": "select_option",
		"target": map[string]any{"entity_id": fetchSchedule}, "service_data": map[string]any{"option": "6m"},
	}, nil); err != nil {
		t.Fatal(err)
	}
	waitSelectState(t, ws, fetchSchedule, "6m")
	stateTopic := selectDiscoveryPrefix + optionUniqueIDPrefix + "fetchschedule/state"
	states := make(chan mqtt.Message, 1)
	waitSelectMQTT(t, broker.Subscribe(stateTopic, 0, func(_ mqtt.Client, message mqtt.Message) { states <- message }))
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	select {
	case retained := <-states:
		if !retained.Retained() || string(retained.Payload()) != "6m" {
			t.Fatalf("expected retained fetch schedule 6m, received %q", retained.Payload())
		}
	case <-deadline.C:
		t.Fatal("fetch schedule was never retained")
	}
}

func waitSelectState(t *testing.T, ws *haWSClient, entityID, expected string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		states, err := ws.ListStates(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		for _, state := range states {
			if state.EntityID == entityID && state.State == expected {
				return
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("Home Assistant state for %s did not reach %q", entityID, expected)
}

func seedLegacySelectDiscovery(t *testing.T, broker mqtt.Client, registered bool) {
	t.Helper()
	service := cmdHassio.Config{
		Name: "GoSungrow Service", UniqueId: "GoSungrow", StateTopic: "homeassistant/sensor/GoSungrow/state",
		DeviceConfig: cmdHassio.DeviceConfig{Identifiers: []string{"GoSungrow"}, Name: "GoSungrow Service"},
	}
	waitSelectMQTT(t, broker.Publish("homeassistant/sensor/GoSungrow/config", 1, true, service.Json()))
	device := cmdHassio.Device{
		Identifiers: []string{"GoSungrow"},
		Connections: [][]string{{"GoSungrow", "GoSungrow-options"}, {"GoSungrow-options", "GoSungrow"}},
		Name:        "GoSungrow options - Options",
		ViaDevice:   "GoSungrow",
	}
	if registered {
		// Seed the registry left by an installation that accepted the old device identity.
		device.ViaDevice = ""
	}
	for _, id := range smokeOptionIDs {
		uniqueID := optionUniqueIDPrefix + id
		base := selectDiscoveryPrefix + uniqueID
		payload := cmdHassio.Select{
			Device: device, Name: cmdHassio.String(id), ObjectId: cmdHassio.String(uniqueID), UniqueId: cmdHassio.String(uniqueID),
			CommandTopic: cmdHassio.String(base + "/cmd"), StateTopic: cmdHassio.String(base + "/state"),
			Options: []string{"5m", "6m"}, Retain: true,
		}
		waitSelectMQTT(t, broker.Publish(base+"/config", 1, true, payload.Json()))
	}
}

func assertRetainedSelectDiscovery(t *testing.T, broker mqtt.Client) {
	t.Helper()
	packets := make(chan mqtt.Message, len(smokeOptionIDs))
	topic := selectDiscoveryPrefix + "+/config"
	waitSelectMQTT(t, broker.Subscribe(topic, 0, func(_ mqtt.Client, message mqtt.Message) { packets <- message }))
	defer func() { waitSelectMQTT(t, broker.Unsubscribe(topic)) }()
	expected := make(map[string]bool, len(smokeOptionIDs))
	for _, id := range smokeOptionIDs {
		expected[optionUniqueIDPrefix+id] = true
	}
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	for len(expected) > 0 {
		select {
		case packet := <-packets:
			var payload cmdHassio.Select
			if err := json.Unmarshal(packet.Payload(), &payload); err != nil {
				t.Fatal(err)
			}
			uniqueID := string(payload.UniqueId)
			base := selectDiscoveryPrefix + uniqueID
			if !expected[uniqueID] || packet.Topic() != base+"/config" || !packet.Retained() ||
				string(payload.ObjectId) != uniqueID || string(payload.CommandTopic) != base+"/cmd" ||
				string(payload.StateTopic) != base+"/state" || !reflect.DeepEqual(payload.Device.Identifiers, []string{"GoSungrow-options"}) ||
				payload.Device.ViaDevice != "GoSungrow" || payload.Device.Connections != nil {
				t.Fatalf("unexpected retained discovery on %s: %s", packet.Topic(), packet.Payload())
			}
			delete(expected, uniqueID)
		case <-deadline.C:
			t.Fatalf("missing retained discovery for %v", expected)
		}
	}
}

func waitSelectMQTT(t *testing.T, token mqtt.Token) {
	t.Helper()
	if !token.WaitTimeout(10 * time.Second) {
		t.Fatal("MQTT fixture operation timed out")
	}
	if err := token.Error(); err != nil {
		t.Fatal(err)
	}
}

type selectSmokeDevice struct {
	ID          string     `json:"id"`
	Identifiers [][]string `json:"identifiers"`
	Connections [][]string `json:"connections"`
	ViaDeviceID string     `json:"via_device_id"`
}

type selectSmokeRegistry struct {
	services []selectSmokeDevice
	options  []selectSmokeDevice
	entities map[string]haEntityRegistryEntry
}

func (r selectSmokeRegistry) allOn(deviceID string) bool {
	if len(r.entities) != len(smokeOptionIDs) {
		return false
	}
	for _, id := range smokeOptionIDs {
		if r.entities[optionUniqueIDPrefix+id].DeviceID != deviceID {
			return false
		}
	}
	return true
}

func (r selectSmokeRegistry) isCorrect() bool {
	return len(r.services) == 1 && len(r.options) == 1 &&
		r.options[0].ViaDeviceID == r.services[0].ID && r.allOn(r.options[0].ID)
}

func (r selectSmokeRegistry) entityIDs() map[string]string {
	ids := make(map[string]string, len(r.entities))
	for uniqueID, entity := range r.entities {
		ids[uniqueID] = entity.EntityID
	}
	return ids
}

func readSelectRegistry(t *testing.T, ws *haWSClient) selectSmokeRegistry {
	t.Helper()
	var devices []selectSmokeDevice
	if err := ws.call(map[string]any{"type": "config/device_registry/list"}, &devices); err != nil {
		t.Fatal(err)
	}
	entities, err := ws.ListEntityRegistry(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	result := selectSmokeRegistry{entities: make(map[string]haEntityRegistryEntry)}
	for _, device := range devices {
		for _, identifier := range device.Identifiers {
			if reflect.DeepEqual(identifier, []string{"mqtt", "GoSungrow"}) {
				result.services = append(result.services, device)
			}
			if reflect.DeepEqual(identifier, []string{"mqtt", "GoSungrow-options"}) {
				result.options = append(result.options, device)
			}
		}
	}
	for _, entry := range entities {
		if entry.Platform == "mqtt" && strings.HasPrefix(entry.EntityID, "select.") && strings.HasPrefix(entry.UniqueID, optionUniqueIDPrefix) {
			result.entities[entry.UniqueID] = entry
		}
	}
	return result
}

func waitSelectRegistry(t *testing.T, ws *haWSClient, timeout time.Duration, ready func(selectSmokeRegistry) bool) selectSmokeRegistry {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		registry := readSelectRegistry(t, ws)
		if ready(registry) || time.Now().After(deadline) {
			return registry
		}
		time.Sleep(250 * time.Millisecond)
	}
}

type selectSmokeAPI struct {
	baseURL string
	client  *http.Client
	token   string
}

func (ha *selectSmokeAPI) onboard(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(150 * time.Second)
	for {
		response, err := ha.client.Get(ha.baseURL + "/api/onboarding")
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("Home Assistant onboarding did not become ready")
		}
		time.Sleep(time.Second)
	}
	clientID := ha.baseURL + "/"
	var user struct {
		AuthCode string `json:"auth_code"`
	}
	ha.postJSON(t, "/api/onboarding/users", map[string]string{
		"name": "GoSungrow CI", "username": "gosungrow-ci", "password": "gosungrow-ci-password", "client_id": clientID, "language": "en",
	}, &user)
	form := url.Values{"grant_type": {"authorization_code"}, "code": {user.AuthCode}, "client_id": {clientID}}
	var tokens struct {
		AccessToken string `json:"access_token"`
	}
	ha.post(t, "/auth/token", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()), &tokens)
	ha.token = tokens.AccessToken
	ha.postJSON(t, "/api/onboarding/core_config", map[string]string{}, nil)
	ha.postJSON(t, "/api/onboarding/analytics", map[string]string{}, nil)
	ha.postJSON(t, "/api/onboarding/integration", map[string]string{
		"client_id": clientID, "redirect_uri": clientID + "?auth_callback=1",
	}, nil)
}

func (ha *selectSmokeAPI) configureMQTT(t *testing.T) string {
	t.Helper()
	const flowPath = "/api/config/config_entries/flow"
	var start struct {
		FlowID string `json:"flow_id"`
	}
	ha.postJSON(t, flowPath, map[string]string{"handler": "mqtt"}, &start)
	if start.FlowID == "" {
		t.Fatal("Home Assistant did not start the MQTT config flow")
	}
	var result struct {
		Type   string            `json:"type"`
		Errors map[string]string `json:"errors"`
		Result struct {
			EntryID string `json:"entry_id"`
		} `json:"result"`
	}
	ha.postJSON(t, flowPath+"/"+start.FlowID, map[string]any{
		"broker": "mqtt", "port": 1883,
		"other_settings": map[string]any{"set_client_cert": false, "set_ca_cert": "off"},
	}, &result)
	if result.Type != "create_entry" || result.Result.EntryID == "" {
		t.Fatalf("MQTT config flow did not create an entry: type %q, errors %v", result.Type, result.Errors)
	}
	return result.Result.EntryID
}

func (ha *selectSmokeAPI) postJSON(t *testing.T, path string, body, result any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(payload)
	}
	ha.post(t, path, "application/json", reader, result)
}

func (ha *selectSmokeAPI) post(t *testing.T, path, contentType string, body io.Reader, result any) {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, ha.baseURL+path, body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", contentType)
	if ha.token != "" {
		request.Header.Set("Authorization", "Bearer "+ha.token)
	}
	response, err := ha.client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		t.Fatalf("Home Assistant POST %s returned %s", path, response.Status)
	}
	if result != nil {
		if err := json.NewDecoder(response.Body).Decode(result); err != nil {
			t.Fatalf("decode Home Assistant POST %s: %v", path, err)
		}
	}
}
