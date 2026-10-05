package cmd

import (
	"errors"
	"testing"

	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud"
)

func TestApiLoginPersistsBeforePromotionAndRemembersWinningGateway(t *testing.T) {
	f := newMQTTRecoveryFixture(t)
	c := cmds.Api
	f.loginHost = "augateway.isolarcloud.com"
	oldClient, oldToken := c.SunGrow, c.SunGrow.GetToken()
	persisted := false
	apiWriteConfig = func() error {
		if c.SunGrow != oldClient || c.SunGrow.GetToken() != oldToken || c.Url != "https://augateway.isolarcloud.com" {
			t.Fatal("candidate promoted before configuration persistence")
		}
		persisted = true
		return nil
	}
	if err := c.ApiLogin(true); err != nil {
		t.Fatal(err)
	}
	if !persisted || c.SunGrow != oldClient || c.SunGrow.ApiRoot.ServerUrl.String() != c.Url {
		t.Fatal("candidate was not promoted into the stable client")
	}
	f.loginHost = ""
	apiWriteConfig = func() error { return nil }
	start := len(f.requests)
	if err := c.ApiLogin(true); err != nil {
		t.Fatal(err)
	}
	if f.requests[start] != "augateway.isolarcloud.com/v1/userService/login" {
		t.Fatalf("winning gateway not prioritized: %s", f.requests[start])
	}
}

func TestApiLoginFailedPersistenceRestoresConfigurationAndSession(t *testing.T) {
	f := newMQTTRecoveryFixture(t)
	c := cmds.Api
	oldURL, oldKey, oldLogin, oldToken := c.Url, c.AppKey, c.LastLogin, c.ApiToken
	f.loginHost = "augateway.isolarcloud.com"
	failure := errors.New("synthetic configuration persistence failure")
	apiWriteConfig = func() error { return failure }
	if err := c.ApiLogin(true); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if c.Url != oldURL || c.AppKey != oldKey || c.LastLogin != oldLogin || c.ApiToken != oldToken || c.SunGrow.GetToken() != oldToken || c.SunGrow.ApiRoot.ServerUrl.String() != oldURL {
		t.Fatal("failed persistence changed active session/configuration")
	}
}

func TestApiLoginSameTokenSkipsConfigurationWrite(t *testing.T) {
	f := newMQTTRecoveryFixture(t)
	f.fixedToken = cmds.Api.SunGrow.GetToken()
	writes, logins := f.configWrites, f.loginCalls
	if err := cmds.Api.ApiLogin(true); err != nil {
		t.Fatal(err)
	}
	if f.configWrites != writes || f.loginCalls != logins+1 {
		t.Fatal("forced login reused cache or rewrote unchanged token configuration")
	}
}

func TestApiBootstrapLoginRetainsValidTokenAndEndpointCaches(t *testing.T) {
	f := newMQTTRecoveryFixture(t)
	calls, writes := len(f.requests), f.configWrites
	if err := cmds.Api.ApiLogin(false); err != nil {
		t.Fatal(err)
	}
	if len(f.requests) != calls || f.configWrites != writes {
		t.Fatal("bootstrap login bypassed valid token or endpoint cache")
	}
}

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
		Host:   "https://gateway.isolarcloud.in",
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
