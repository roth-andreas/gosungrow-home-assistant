package cmdHassio

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func TestOptionsSetCanonicalizesKnownOptionCasing(t *testing.T) {
	var options Options
	options.New()

	if err := options.Create("loglevel", "Log Level", nil, "Enabled", "Disabled"); err != nil {
		t.Fatalf("create option: %v", err)
	}
	if err := options.Set("loglevel", "disabled"); err != nil {
		t.Fatalf("set option: %v", err)
	}

	if got := options.Get("loglevel"); got != "Disabled" {
		t.Fatalf("expected canonical option value Disabled, got %q", got)
	}
}

func TestOptionsSetPreservesUnknownOptionValues(t *testing.T) {
	var options Options
	options.New()

	if err := options.Create("loglevel", "Log Level", nil, "Enabled", "Disabled"); err != nil {
		t.Fatalf("create option: %v", err)
	}
	if err := options.Set("loglevel", "debug"); err != nil {
		t.Fatalf("set option: %v", err)
	}

	if got := options.Get("loglevel"); got != "debug" {
		t.Fatalf("expected unknown option value to pass through, got %q", got)
	}
}

func TestRuntimeOptionsPublishRegisteredDevice(t *testing.T) {
	transport := &optionDiscoveryTransport{t: t}
	client := New(Mqtt{
		ClientId: "GoSungrow", EntityPrefix: "GoSungrow", Host: "localhost", Port: "1883",
	}).WithClient(transport)
	if err := client.GetError(); err != nil {
		t.Fatal(err)
	}
	client.DeviceName = "GoSungrow"
	client.registerOptionsDevice()

	for _, id := range []string{"mqtt_loglevel", "loglevel", "fetchschedule", "sleepdelay", "servicestate"} {
		t.Run(id, func(t *testing.T) {
			transport.t = t
			if err := client.CreateOption(id, id, func(mqtt.Client, mqtt.Message) {}); err != nil {
				t.Fatal(err)
			}
			entityID := "GoSungrow-option-" + id
			baseTopic := "homeassistant/select/GoSungrow/" + entityID
			if transport.topic != baseTopic+"/config" || transport.subscription != baseTopic+"/cmd" {
				t.Fatalf("published to %q and subscribed to %q, want topics under %q", transport.topic, transport.subscription, baseTopic)
			}
			var payload Select
			if err := json.Unmarshal(transport.payload, &payload); err != nil {
				t.Fatal(err)
			}
			if string(payload.ObjectId) != entityID || string(payload.UniqueId) != entityID ||
				string(payload.CommandTopic) != baseTopic+"/cmd" || string(payload.StateTopic) != baseTopic+"/state" {
				t.Fatalf("wrong select identity or topics in %s", transport.payload)
			}
			if !payload.Retain || payload.CommandTemplate != "{{ value }}" || payload.ValueTemplate != "{{ value }}" {
				t.Fatalf("wrong select retention or templates in %s", transport.payload)
			}
			if !reflect.DeepEqual(payload.Device.Identifiers, []string{"GoSungrow-options"}) || payload.Device.ViaDevice != "GoSungrow" {
				t.Fatalf("wrong options device in %s", transport.payload)
			}
			var fields struct {
				Device map[string]json.RawMessage `json:"device"`
			}
			if err := json.Unmarshal(transport.payload, &fields); err != nil {
				t.Fatal(err)
			}
			if _, present := fields.Device["connections"]; present {
				t.Fatalf("options discovery must omit device connections: %s", transport.payload)
			}
		})
	}
}

type optionDiscoveryTransport struct {
	mqtt.Client
	t            *testing.T
	topic        string
	subscription string
	payload      []byte
}

func (m *optionDiscoveryTransport) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	m.t.Helper()
	if qos != 0 || !retained {
		m.t.Fatalf("publish %q with QoS %d and retained %t, want QoS 0 and retained", topic, qos, retained)
	}
	m.topic = topic
	m.payload = []byte(payload.(string))
	return optionDiscoveryToken{}
}

func (m *optionDiscoveryTransport) Subscribe(topic string, qos byte, handler mqtt.MessageHandler) mqtt.Token {
	m.t.Helper()
	if qos != 0 || handler == nil {
		m.t.Fatalf("subscribe %q with QoS %d and handler set %t, want QoS 0 and a handler", topic, qos, handler != nil)
	}
	m.subscription = topic
	return optionDiscoveryToken{}
}

type optionDiscoveryToken struct{ mqtt.Token }

func (optionDiscoveryToken) WaitTimeout(time.Duration) bool { return true }
func (optionDiscoveryToken) Error() error                   { return nil }
