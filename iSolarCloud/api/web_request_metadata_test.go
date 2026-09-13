package api

import (
	"fmt"
	"testing"
	"time"
)

func TestRequestTimestampMillisDiagnosticOffset(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		offset time.Duration
	}{
		{name: "valid trimmed value", value: " 1500 ", offset: 1500 * time.Millisecond},
		{name: "empty value", value: "", offset: 0},
		{name: "invalid value", value: "not-a-number", offset: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("GOSUNGROW_TIMESTAMP_OFFSET_MS", test.value)
			web := Web{timeOffset: 2 * time.Second}
			before := time.Now().Add(web.timeOffset + test.offset).UnixMilli()
			got := web.requestTimestampMillis()
			after := time.Now().Add(web.timeOffset + test.offset).UnixMilli()
			if got < before || got > after {
				t.Fatalf("timestamp=%d, want range [%d,%d]", got, before, after)
			}
		})
	}
}

func TestCurrentGMTHeaderUsesCompatibilityFormat(t *testing.T) {
	_, offsetSeconds := time.Now().Zone()
	hours := offsetSeconds / 3600
	want := fmt.Sprintf("GMT%%2B%d", hours)
	if hours < 0 {
		want = fmt.Sprintf("GMT-%d", -hours)
	}
	if got := currentGMTHeader(); got != want {
		t.Fatalf("currentGMTHeader()=%q, want %q", got, want)
	}
}
