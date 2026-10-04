package queryDeviceList

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api/GoStruct"
)

func FuzzQueryPointJSON(f *testing.F) {
	for _, seed := range []string{`{"point_id":"p83076_map","value":4.5}`, `{"point_id":"plant_power","value":null}`, `{"point_id":24,"value":0}`, `{"point_id":"phase_a","value":"--"}`, `{}`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		var point PointStruct
		if json.Unmarshal([]byte(input), &point) != nil {
			return
		}
		var fields struct {
			PointID json.RawMessage `json:"point_id"`
		}
		if json.Unmarshal([]byte(input), &fields) != nil {
			return
		}
		var original string
		if json.Unmarshal(fields.PointID, &original) != nil {
			return
		}
		if _, err := strconv.ParseInt(strings.TrimPrefix(original, "p"), 10, 64); err == nil {
			return
		}
		if original != point.sourcePointID {
			t.Fatalf("original API identity lost: %q -> %q", original, point.sourcePointID)
		}
	})
}

func TestDecodedCopiesPreserveValidityIdentityAndNonPowerUnits(t *testing.T) {
	endpoint := Init(api.Web{})
	endpoint.Request.PsId.SetString("100")
	if err := json.Unmarshal([]byte(`{"result_data":{"pageList":[{"ps_id":"100","ps_key":"100_55_1_1","device_type":55,"point_data":[{"point_id":"p24","value":null,"unit":"kW"},{"point_id":"energy","point_name":"Energy","value":2.5,"unit":"kWh"}]},{"ps_id":"100","ps_key":"100_55_1_2","device_type":55,"point_data":[{"point_id":"p24","value":0.9,"unit":"kW"}]}]}}`), &endpoint.Response); err != nil {
		t.Fatal(err)
	}
	data := endpoint.GetData()
	data.ProcessMap()
	invalid := data.Map["virtual.100_55_1_1.p24"].GetEntry(api.LastEntry)
	if invalid.Value.Valid || invalid.Current.Source.NumericValid {
		t.Fatal("null copy became valid")
	}
	second := data.Map["virtual.100_55_1_2.p24"].GetEntry(api.LastEntry)
	if second.Value.ValueFloat() != .9 || second.Parent.Key != "100_55_1_2" {
		t.Fatal("device-local copy lost identity or value")
	}
	energy := data.Map["virtual.100_55_1_1.energy"].GetEntry(api.LastEntry)
	if energy.Point.ValueType == "Power" || energy.Value.Unit() != "kWh" {
		t.Fatalf("energy coerced to power: %#v", energy)
	}
	rawFound := false
	for _, entries := range data.Map {
		entry := entries.GetEntry(api.LastEntry)
		if entry.Point.Id == "value" && entry.Current.Source.PsKey == "100_55_1_1" && entry.Current.Source.PointID == "p24" {
			rawFound = true
			if entry.Value.Valid {
				t.Fatal("raw null became valid zero")
			}
		}
	}
	if !rawFound {
		t.Fatal("raw MQTT point lost API identity")
	}
}

func TestVirtualPointBuildersSkipMissingSourcePoints(t *testing.T) {
	endpoint := EndPoint{}
	entries := api.NewDataMap()
	epp := GoStruct.NewEndPointPath("virtual", "100_14_1_1")

	assertDoesNotPanic(t, "SetBatteryPoints", func() {
		endpoint.SetBatteryPoints(epp, entries)
	})
	assertDoesNotPanic(t, "SetPvPoints", func() {
		endpoint.SetPvPoints(epp, entries)
	})
	assertDoesNotPanic(t, "SetGridPoints", func() {
		endpoint.SetGridPoints(epp, entries)
	})
	assertDoesNotPanic(t, "SetLoadPoints", func() {
		endpoint.SetLoadPoints(epp, entries)
	})
}

func assertDoesNotPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("%s panicked with missing source points: %v", name, recovered)
		}
	}()
	fn()
}
