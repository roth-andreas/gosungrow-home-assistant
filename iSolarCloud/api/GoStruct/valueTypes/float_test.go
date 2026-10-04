package valueTypes

import (
	"encoding/json"
	"math"
	"testing"
)

func TestFloatMissingDoesNotBecomeValidZero(t *testing.T) {
	for _, input := range []string{"null", `"--"`, `"invalid"`, `""`} {
		value := SetFloatValue(9)
		if err := json.Unmarshal([]byte(input), &value); err != nil {
			t.Fatal(err)
		}
		if value.Valid || value.ToUnitValue().Valid {
			t.Fatalf("%s became valid: %#v", input, value)
		}
	}
	var zero Float
	if err := json.Unmarshal([]byte("0"), &zero); err != nil || !zero.Valid || !zero.ToUnitValue().Valid {
		t.Fatalf("genuine zero invalid: %#v error=%v", zero, err)
	}
}
func FuzzFloatJSON(f *testing.F) {
	for _, seed := range []string{"null", "0", `"--"`, `"NaN"`, "1e308", "{}", "[]"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		var value Float
		err := json.Unmarshal([]byte(input), &value)
		if err == nil && input == "null" && value.Valid {
			t.Fatal("null became zero")
		}
		converted := value.ToUnitValue()
		if value.Valid && !math.IsNaN(value.Value()) && converted.ValueFloat() != value.Value() {
			t.Fatal("conversion changed numeric value")
		}
	})
}
