package valueTypes

import (
	"encoding/json"
	"testing"
)

func TestIntegerSetStringRejectsNullPlaceholder(t *testing.T) {
	integer := SetIntegerString("null")

	if integer.Error != nil {
		t.Fatalf("unexpected error: %v", integer.Error)
	}
	if integer.Valid {
		t.Fatal("expected null placeholder integer to be invalid")
	}
}

func TestIntegerSetStringRejectsCompositePlaceholderWithoutError(t *testing.T) {
	integer := SetIntegerString("3109704_3109705")

	if integer.Error != nil {
		t.Fatalf("unexpected error: %v", integer.Error)
	}
	if integer.Valid {
		t.Fatal("expected composite placeholder integer to be invalid")
	}
	if got := integer.String(); got != "3109704_3109705" {
		t.Fatalf("expected raw string to be retained, got %q", got)
	}
}

func TestIntegerUnmarshalRejectsCompositePlaceholderWithoutError(t *testing.T) {
	var integer Integer

	if err := json.Unmarshal([]byte(`"3109704_3109705"`), &integer); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if integer.Valid {
		t.Fatal("expected composite placeholder integer to be invalid")
	}
}

func TestCountSetStringRejectsNullPlaceholder(t *testing.T) {
	count := SetCountString("null")

	if count.Error != nil {
		t.Fatalf("unexpected error: %v", count.Error)
	}
	if count.Valid {
		t.Fatal("expected null placeholder count to be invalid")
	}
}

func TestCountUnmarshalDistinguishesNullFromZero(t *testing.T) {
	count := SetCountString("7")
	if err := json.Unmarshal([]byte("null"), &count); err != nil || count.Valid {
		t.Fatalf("null count = %+v, err = %v", count, err)
	}
	if err := json.Unmarshal([]byte("0"), &count); err != nil || !count.Valid || count.Value() != 0 {
		t.Fatalf("zero count = %+v, err = %v", count, err)
	}
}

func TestCountSetStringRejectsCompositePlaceholderWithoutError(t *testing.T) {
	count := SetCountString("3109704_3109705")

	if count.Error != nil {
		t.Fatalf("unexpected error: %v", count.Error)
	}
	if count.Valid {
		t.Fatal("expected composite placeholder count to be invalid")
	}
}
