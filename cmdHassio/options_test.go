package cmdHassio

import "testing"

func TestOptionsSetCanonicalizesKnownOptionCasing(t *testing.T) {
	var options Options
	options.New()

	if err := options.Create("loglevel", "Log Level", nil, "Enabled", "Disabled"); err != nil {
		t.Fatalf("create option: %v", err)
	}
	if err := options.Set("loglevel", "disabled"); err != nil {
		t.Fatalf("set option: %v", err)
	}

	if got := options.Get("loglevel"); got != "Disabled" {
		t.Fatalf("expected canonical option value Disabled, got %q", got)
	}
}

func TestOptionsSetPreservesUnknownOptionValues(t *testing.T) {
	var options Options
	options.New()

	if err := options.Create("loglevel", "Log Level", nil, "Enabled", "Disabled"); err != nil {
		t.Fatalf("create option: %v", err)
	}
	if err := options.Set("loglevel", "debug"); err != nil {
		t.Fatalf("set option: %v", err)
	}

	if got := options.Get("loglevel"); got != "debug" {
		t.Fatalf("expected unknown option value to pass through, got %q", got)
	}
}
