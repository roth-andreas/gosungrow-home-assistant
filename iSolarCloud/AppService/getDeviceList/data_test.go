package getDeviceList

import (
	"encoding/json"
	"testing"
)

func TestDeviceUnmarshalAcceptsCompositePsId(t *testing.T) {
	raw := []byte(`{
		"ps_key":"5520557_14_1_1",
		"ps_id":"5520557_5520558",
		"device_type":14,
		"device_id":5520558,
		"device_name":"Hybrid Inverter"
	}`)

	var device Device
	if err := json.Unmarshal(raw, &device); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if got := device.PsId.String(); got != "5520557_5520558" {
		t.Fatalf("unexpected ps_id: %q", got)
	}
	if !device.PsId.Valid {
		t.Fatal("expected composite ps_id to decode as valid")
	}
}
