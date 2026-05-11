package cmd

import (
	"errors"
	"testing"

	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud"
)

func TestNormalizeLoginAppKey(t *testing.T) {
	if got := normalizeLoginAppKey(""); got != iSolarCloud.DefaultApiAppKey {
		t.Fatalf("empty app key should fall back to default: got %q", got)
	}
	if got := normalizeLoginAppKey(iSolarCloud.LegacyLoginAppKey); got != iSolarCloud.DefaultApiAppKey {
		t.Fatalf("legacy app key should fall back to default: got %q", got)
	}
	if got := normalizeLoginAppKey(iSolarCloud.OldLoginAppKey); got != iSolarCloud.OldLoginAppKey {
		t.Fatalf("non-empty app key should be preserved: got %q", got)
	}
}

func TestBuildLoginAttemptsPrioritizesConfiguredHostAndAppKey(t *testing.T) {
	attempts := buildLoginAttempts("https://custom.isolarcloud.example", iSolarCloud.OldLoginAppKey)
	if len(attempts) == 0 {
		t.Fatal("expected login attempts")
	}

	first := attempts[0]
	if first.Host != "https://custom.isolarcloud.example" {
		t.Fatalf("unexpected first host: %q", first.Host)
	}
	if first.AppKey != iSolarCloud.OldLoginAppKey {
		t.Fatalf("unexpected first app key: %q", first.AppKey)
	}

	seen := make(map[loginAttempt]bool, len(attempts))
	for _, attempt := range attempts {
		if seen[attempt] {
			t.Fatalf("duplicate login attempt found: %+v", attempt)
		}
		seen[attempt] = true
	}

	want := loginAttempt{
		Host:   "https://gateway.isolarcloud.com.hk",
		AppKey: iSolarCloud.LegacyLoginAppKey,
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
		{name: "dns no such host", err: errors.New("lookup augateway.isolarcloud.com: no such host"), want: true},
		{name: "docker dns temporary failure", err: errors.New("dial tcp: lookup augateway.isolarcloud.com on 127.0.0.11:53: temporary failure in name resolution"), want: false},
		{name: "external dns temporary failure", err: errors.New("dial tcp: lookup augateway.isolarcloud.com on 192.168.1.1:53: temporary failure in name resolution"), want: true},
		{name: "network timeout", err: errors.New("dial tcp 1.2.3.4:443: i/o timeout"), want: true},
		{name: "network unreachable", err: errors.New("dial tcp: connect: network is unreachable"), want: true},
		{name: "other error", err: errors.New("unexpected payload format"), want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tc := range tests {
		if got := shouldTryNextLoginAttempt(tc.err); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}
