package valueTypes

import (
	"encoding/json"
	"testing"
)

func TestUuidSetStringRejectsCompositePlaceholderWithoutError(t *testing.T) {
	uuid := SetUuidString("3109704_3109705")

	if uuid.Error != nil {
		t.Fatalf("unexpected error: %v", uuid.Error)
	}
	if uuid.Valid {
		t.Fatal("expected composite placeholder uuid to be invalid")
	}
	if got := uuid.String(); got != "3109704_3109705" {
		t.Fatalf("expected raw string to be retained, got %q", got)
	}
}

func TestUuidUnmarshalRejectsCompositePlaceholderWithoutError(t *testing.T) {
	var uuid Uuid

	if err := json.Unmarshal([]byte(`"3109704_3109705"`), &uuid); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if uuid.Valid {
		t.Fatal("expected composite placeholder uuid to be invalid")
	}
}

func TestUuidSetStringRejectsNullPlaceholderWithoutError(t *testing.T) {
	uuid := SetUuidString("null")

	if uuid.Error != nil {
		t.Fatalf("unexpected error: %v", uuid.Error)
	}
	if uuid.Valid {
		t.Fatal("expected null placeholder uuid to be invalid")
	}
}
