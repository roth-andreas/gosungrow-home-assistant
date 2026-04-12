package iSolarCloud

import (
	"errors"
	"testing"
)

func TestBuildLoginAttemptsPrioritizesConfiguredHostAndAppKey(t *testing.T) {
	attempts := BuildLoginAttempts("https://custom.isolarcloud.example", OldLoginAppKey)
	if len(attempts) == 0 {
		t.Fatal("expected login attempts")
	}

	first := attempts[0]
	if first.Host != "https://custom.isolarcloud.example" {
		t.Fatalf("unexpected first host: %q", first.Host)
	}
	if first.AppKey != OldLoginAppKey {
		t.Fatalf("unexpected first app key: %q", first.AppKey)
	}

	seen := make(map[LoginAttempt]bool, len(attempts))
	for _, attempt := range attempts {
		if seen[attempt] {
			t.Fatalf("duplicate login attempt found: %+v", attempt)
		}
		seen[attempt] = true
	}

	want := LoginAttempt{
		Host:   "https://gateway.isolarcloud.eu",
		AppKey: LegacyLoginAppKey,
	}
	if !seen[want] {
		t.Fatalf("expected fallback attempt %+v", want)
	}
}

func TestShouldRecoverGatewayError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "token invalid", err: errors.New("need to login again 'er_token_login_invalid'"), want: true},
		{name: "gateway rejected", err: errors.New("login rejected by gateway"), want: true},
		{name: "wrong app key", err: errors.New("appkey is incorrect"), want: true},
		{name: "dns no such host", err: errors.New("lookup augateway.isolarcloud.com: no such host"), want: true},
		{name: "dns server misbehaving", err: errors.New("dial tcp: lookup gateway.isolarcloud.eu on 127.0.0.11:53: server misbehaving"), want: true},
		{name: "network timeout", err: errors.New("dial tcp 1.2.3.4:443: i/o timeout"), want: true},
		{name: "network unreachable", err: errors.New("dial tcp: connect: network is unreachable"), want: true},
		{name: "other error", err: errors.New("unexpected payload format"), want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tc := range tests {
		if got := ShouldRecoverGatewayError(tc.err); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}
