package valueTypes

import "testing"

func TestReactivePowerUnitNormalization(t *testing.T) {
	tests := []struct {
		name  string
		unit  string
		value float64
		want  float64
	}{
		{name: "canonical", unit: "var", value: 750, want: 750},
		{name: "mixed base", unit: "Var", value: -750, want: -750},
		{name: "base acronym", unit: "VAr", value: 0, want: 0},
		{name: "uppercase base", unit: "VAR", value: 750, want: 750},
		{name: "lower kilo", unit: "kvar", value: 0.75, want: 750},
		{name: "sungrow kilo", unit: "kVar", value: -0.75, want: -750},
		{name: "ha alias kilo", unit: "kVAr", value: 0.75, want: 750},
		{name: "uppercase kilo", unit: "KVAR", value: 0.75, want: 750},
		{name: "mixed mega", unit: "Mvar", value: 0.001, want: 1000},
		{name: "camel mega", unit: "MVar", value: 0.001, want: 1000},
		{name: "acronym mega", unit: "MVAr", value: 0.001, want: 1000},
		{name: "uppercase mega", unit: "MVAR", value: -0.001, want: -1000},
		{name: "lower milli", unit: "mvar", value: 750, want: 0.75},
		{name: "camel milli", unit: "mVar", value: 750, want: 0.75},
		{name: "mixed milli", unit: "mVAr", value: -750, want: -0.75},
		{name: "uppercase suffix milli", unit: "mVAR", value: 750, want: 0.75},
		{name: "trimmed", unit: "  kVar\t", value: 0.75, want: 750},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SetUnitValueFloat(tc.unit, "", tc.value)
			if got.Unit() != "var" {
				t.Fatalf("unit = %q, want var", got.Unit())
			}
			if got.Value() != tc.want {
				t.Fatalf("value = %v, want %v", got.Value(), tc.want)
			}
			if got.Type() != "Reactive Power" {
				t.Fatalf("type = %q, want Reactive Power", got.Type())
			}
		})
	}
}

func TestReactivePowerNormalizationInputTypes(t *testing.T) {
	tests := []struct {
		name string
		got  UnitValue
		want float64
	}{
		{name: "integer", got: SetUnitValueInteger("kVar", "", 2), want: 2000},
		{name: "numeric integer string", got: SetUnitValueString("kVar", "", "2"), want: 2000},
		{name: "numeric float string", got: SetUnitValueString("kVar", "", "0.75"), want: 750},
		{name: "numeric scientific string", got: SetUnitValueString("kVar", "", "7.5e-1"), want: 750},
		{name: "numeric integer scientific string", got: SetUnitValueString("kVar", "", "2e0"), want: 2000},
		{name: "integer milli exact", got: SetUnitValueInteger("mvar", "", 2000), want: 2},
		{name: "integer milli fractional", got: SetUnitValueInteger("mvar", "", 750), want: 0.75},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got.Unit() != "var" || tc.got.Value() != tc.want {
				t.Fatalf("got %v %s, want %v var", tc.got.Value(), tc.got.Unit(), tc.want)
			}
		})
	}
}

func TestReactivePowerNormalizationUnavailableUnknownAndIdempotent(t *testing.T) {
	unavailable := SetUnitValueString(" kVar ", "", "--")
	if unavailable.Unit() != "var" || unavailable.String() != "--" || unavailable.Valid {
		t.Fatalf("unavailable value changed unexpectedly: %#v", unavailable)
	}

	unknown := SetUnitValueFloat("KVAr", "", 0.75)
	if unknown.Unit() != "KVAr" || unknown.Value() != 0.75 {
		t.Fatalf("unknown unit was modified: %v %s", unknown.Value(), unknown.Unit())
	}

	value := SetUnitValueFloat("kVar", "", 0.75)
	value.UnitValueFix()
	if value.Unit() != "var" || value.Value() != 750 {
		t.Fatalf("normalization was not idempotent: %v %s", value.Value(), value.Unit())
	}
}

func TestUnitValueFixNonReactiveRegression(t *testing.T) {
	tests := []struct {
		unit     string
		value    float64
		wantUnit string
		want     float64
	}{
		{unit: "W", value: 1500, wantUnit: "kW", want: 1.5},
		{unit: "Wh", value: 1500, wantUnit: "kWh", want: 1.5},
		{unit: "%", value: 75, wantUnit: "%", want: 75},
	}
	for _, tc := range tests {
		got := SetUnitValueFloat(tc.unit, "", tc.value)
		if got.Unit() != tc.wantUnit || got.Value() != tc.want {
			t.Fatalf("%s: got %v %s, want %v %s", tc.unit, got.Value(), got.Unit(), tc.want, tc.wantUnit)
		}
	}
}
