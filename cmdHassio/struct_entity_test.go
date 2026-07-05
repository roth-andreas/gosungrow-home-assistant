package cmdHassio

import (
	"testing"

	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api/GoStruct"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api/GoStruct/valueTypes"
)

func TestFixConfigDoesNotMarkTextSensorsAsMeasurements(t *testing.T) {
	value := valueTypes.SetUnitValueString("", "", "Energy Storage System1")
	config := EntityConfig{
		Value: &value,
		Point: &api.Point{UpdateFreq: GoStruct.UpdateFreqInstant},
	}

	config.FixConfig()

	if config.StateClass != "" {
		t.Fatalf("expected text sensor state class to be empty, got %q", config.StateClass)
	}
	if config.Units != "" {
		t.Fatalf("expected text sensor unit to be empty, got %q", config.Units)
	}
	if config.LastResetValueTemplate != "" {
		t.Fatalf("expected text sensor last reset template to be empty, got %q", config.LastResetValueTemplate)
	}
	if config.ValueTemplate != "{{ value_json.value }}" {
		t.Fatalf("unexpected text value template: %q", config.ValueTemplate)
	}
}

func TestFixConfigKeepsMeasurementMetadataForNumericSensors(t *testing.T) {
	value := valueTypes.SetUnitValueFloat("W", "Power", 123.45)
	config := EntityConfig{
		Units: value.Unit(),
		Value: &value,
		Point: &api.Point{UpdateFreq: GoStruct.UpdateFreqInstant},
	}

	config.FixConfig()

	if config.StateClass != "measurement" {
		t.Fatalf("expected numeric sensor state class measurement, got %q", config.StateClass)
	}
	if config.Units != "kW" {
		t.Fatalf("expected numeric sensor unit kW after unit normalization, got %q", config.Units)
	}
	if config.DeviceClass != "power" {
		t.Fatalf("expected power device class, got %q", config.DeviceClass)
	}
}

func TestFixConfigKeepsTotalMetadataForNumericTotals(t *testing.T) {
	value := valueTypes.SetUnitValueFloat("kWh", "Energy", 42)
	config := EntityConfig{
		Units: value.Unit(),
		Value: &value,
		Point: &api.Point{UpdateFreq: GoStruct.UpdateFreqTotal},
	}

	config.FixConfig()

	if config.StateClass != "total" {
		t.Fatalf("expected total sensor state class total, got %q", config.StateClass)
	}
	if config.LastResetValueTemplate == "" {
		t.Fatal("expected total sensor last reset template")
	}
}
