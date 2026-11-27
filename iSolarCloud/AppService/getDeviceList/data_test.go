package getDeviceList

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
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

func TestDashboardDeviceTypeRequiresPositiveIntegerWireValue(t *testing.T) {
	for _, test := range []struct {
		value string
		want  int64
	}{
		{``, 0}, {`null`, 0}, {`""`, 0}, {`"--"`, 0}, {`"NuLl"`, 0},
		{`true`, 0}, {`[]`, 0}, {`{}`, 0}, {`1.0`, 0}, {`1e1`, 0}, {`"ten"`, 0},
		{`"+1"`, 0}, {`0`, 0}, {`-1`, 0}, {`"-1"`, 0}, {`9223372036854775808`, 0},
		{`14`, 14}, {`" 014 "`, 14}, {`9223372036854775807`, 9223372036854775807},
	} {
		t.Run(test.value, func(t *testing.T) {
			wire := `{}`
			if test.value != "" {
				wire = `{"device_type":` + test.value + `}`
			}
			var device Device
			if err := json.Unmarshal([]byte(wire), &device); err != nil {
				t.Fatal(err)
			}
			if got := device.DashboardDeviceType(); got != test.want {
				t.Fatalf("dashboard device type = %d, want %d", got, test.want)
			}
		})
	}
}

func FuzzDashboardDeviceType(f *testing.F) {
	for _, seed := range []string{"14", "\" 014 \"", "null", "true", "-1", "13.5"} {
		f.Add(seed)
	}
	digits := regexp.MustCompile("^[0-9]+$")
	f.Fuzz(func(t *testing.T, input string) {
		raw := json.RawMessage(input)
		wire, err := json.Marshal(map[string]json.RawMessage{"device_type": raw})
		if err != nil {
			return
		}
		var device Device
		if err := json.Unmarshal(wire, &device); err != nil {
			t.Fatalf("valid JSON contained an unhandled field: %v", err)
		}
		token := strings.TrimSpace(input)
		if len(token) > 0 && token[0] == '"' {
			if err := json.Unmarshal(raw, &token); err != nil {
				t.Fatal(err)
			}
			token = strings.TrimSpace(token)
		}
		var want int64
		if digits.MatchString(token) {
			if value, err := strconv.ParseInt(token, 10, 64); err == nil {
				want = value
			}
		}
		if got := device.DashboardDeviceType(); got != want {
			t.Fatalf("decoded type = %d, want %d", got, want)
		}
	})
}
