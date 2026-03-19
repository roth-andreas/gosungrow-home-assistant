package cmd

import (
	"errors"
	"testing"

	"github.com/MickMake/GoSungrow/iSolarCloud"
)

func TestNormalizeLoginAppKey(t *testing.T) {
	if got := normalizeLoginAppKey(""); got != iSolarCloud.DefaultApiAppKey {
		t.Fatalf("empty app key should fall back to default: got %q", got)
	}
	if got := normalizeLoginAppKey(legacyLoginAppKey); got != iSolarCloud.DefaultApiAppKey {
		t.Fatalf("legacy app key should fall back to default: got %q", got)
	}
	if got := normalizeLoginAppKey(oldLoginAppKey); got != oldLoginAppKey {
		t.Fatalf("non-empty app key should be preserved: got %q", got)
	}
}

func TestBuildLoginAttemptsPrioritizesConfiguredHostAndAppKey(t *testing.T) {
	attempts := buildLoginAttempts("https://custom.isolarcloud.example", oldLoginAppKey)
	if len(attempts) == 0 {
		t.Fatal("expected login attempts")
	}

	first := attempts[0]
	if first.host != "https://custom.isolarcloud.example" {
		t.Fatalf("unexpected first host: %q", first.host)
	}
	if first.appKey != oldLoginAppKey {
		t.Fatalf("unexpected first app key: %q", first.appKey)
	}

	seen := make(map[loginAttempt]bool, len(attempts))
	for _, attempt := range attempts {
		if seen[attempt] {
			t.Fatalf("duplicate login attempt found: %+v", attempt)
		}
		seen[attempt] = true
	}

	want := loginAttempt{
		host:   "https://gateway.isolarcloud.eu",
		appKey: legacyLoginAppKey,
	}
	if !seen[want] {
		t.Fatalf("expected fallback attempt %+v", want)
	}
}

func TestShouldTryNextLoginAttempt(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "token invalid", err: errors.New("need to login again 'er_token_login_invalid'"), want: true},
		{name: "gateway rejected", err: errors.New("login rejected by gateway"), want: true},
		{name: "wrong app key", err: errors.New("appkey is incorrect"), want: true},
		{name: "other error", err: errors.New("network timeout"), want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tc := range tests {
		if got := shouldTryNextLoginAttempt(tc.err); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}
