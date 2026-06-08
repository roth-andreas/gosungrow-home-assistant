package queryDeviceRealTimeDataByPsKeys

import "testing"

func TestPsIDFromPsKeyUsesPlantComponent(t *testing.T) {
	psID := psIDFromPsKey("5238227_7_7_1")

	if !psID.HasValue() {
		t.Fatal("expected ps_id derived from ps_key to be usable")
	}
	if got := psID.String(); got != "5238227" {
		t.Fatalf("unexpected ps_id: %q", got)
	}
}

func TestPsIDFromPsKeyRejectsNullPlaceholder(t *testing.T) {
	psID := psIDFromPsKey("null_7_7_1")

	if psID.HasValue() {
		t.Fatalf("expected null placeholder ps_id to be rejected, got %q", psID.String())
	}
}
