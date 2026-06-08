package valueTypes

import "testing"

func TestIntegerSetStringRejectsNullPlaceholder(t *testing.T) {
	integer := SetIntegerString("null")

	if integer.Error != nil {
		t.Fatalf("unexpected error: %v", integer.Error)
	}
	if integer.Valid {
		t.Fatal("expected null placeholder integer to be invalid")
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
