package valueTypes

import (
	"encoding/json"
	"testing"
)

func TestPsIdNumericRoundTrip(t *testing.T) {
	psID := SetPsIdString("5520557")
	if psID.Error != nil {
		t.Fatalf("unexpected error: %v", psID.Error)
	}
	if !psID.Valid {
		t.Fatal("expected numeric ps_id to be valid")
	}
	if !psID.Numeric {
		t.Fatal("expected numeric ps_id to be marked numeric")
	}
	if psID.Value() != 5520557 {
		t.Fatalf("unexpected numeric value: %d", psID.Value())
	}

	data, err := json.Marshal(psID)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if string(data) != "5520557" {
		t.Fatalf("expected numeric JSON, got %s", data)
	}
}

func TestPsIdAcceptsCompositeString(t *testing.T) {
	psID := SetPsIdString("5520557_5520558")
	if psID.Error != nil {
		t.Fatalf("unexpected error: %v", psID.Error)
	}
	if !psID.Valid {
		t.Fatal("expected composite ps_id to be valid")
	}
	if psID.Numeric {
		t.Fatal("expected composite ps_id to stay string-based")
	}
	if psID.String() != "5520557_5520558" {
		t.Fatalf("unexpected string value: %q", psID.String())
	}
	if psID.Value() != 0 {
		t.Fatalf("expected non-numeric ps_id to keep numeric value 0, got %d", psID.Value())
	}

	data, err := json.Marshal(psID)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if string(data) != `"5520557_5520558"` {
		t.Fatalf("expected string JSON, got %s", data)
	}
}
