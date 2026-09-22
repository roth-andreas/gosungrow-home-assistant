package iSolarCloud

import (
	"errors"
	"strings"
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
		Host:   "https://gateway.isolarcloud.com.hk",
		AppKey: LegacyLoginAppKey,
	}
	if !seen[want] {
		t.Fatalf("expected fallback attempt %+v", want)
	}
}

func TestBuildLoginAttemptsIncludesIndianGatewayInStableHostOrder(t *testing.T) {
	attempts := BuildLoginAttempts("https://custom.isolarcloud.example", DefaultApiAppKey)
	wantHosts := []string{
		"https://custom.isolarcloud.example",
		"https://augateway.isolarcloud.com",
		"https://gateway.isolarcloud.com",
		"https://gateway.isolarcloud.eu",
		"https://gateway.isolarcloud.com.hk",
		"https://gateway.isolarcloud.com.cn",
		"https://gateway.isolarcloud.in",
	}

	var gotHosts []string
	for _, attempt := range attempts {
		if len(gotHosts) == 0 || gotHosts[len(gotHosts)-1] != attempt.Host {
			gotHosts = append(gotHosts, attempt.Host)
		}
	}
	if strings.Join(gotHosts, "\n") != strings.Join(wantHosts, "\n") {
		t.Fatalf("host order = %q, want %q", gotHosts, wantHosts)
	}
}

func TestBuildLoginAttemptsPrioritizesIndianGatewayWithoutDuplicatingHostGroup(t *testing.T) {
	attempts := BuildLoginAttempts("https://gateway.isolarcloud.in", DefaultApiAppKey)
	if len(attempts) == 0 || attempts[0].Host != "https://gateway.isolarcloud.in" {
		t.Fatalf("first attempt = %+v, want Indian gateway", attempts)
	}

	hostGroups := 0
	previousHost := ""
	for _, attempt := range attempts {
		if attempt.Host == previousHost {
			continue
		}
		if attempt.Host == "https://gateway.isolarcloud.in" {
			hostGroups++
		}
		previousHost = attempt.Host
	}
	if hostGroups != 1 {
		t.Fatalf("Indian gateway host groups = %d, want 1", hostGroups)
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
		{name: "http 500", err: errors.New("API httpResponse is 500 Internal Server Error"), want: true},
		{name: "http 502", err: errors.New("API httpResponse is 502 Bad Gateway"), want: true},
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

func TestIsDockerDNSError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "docker dns server misbehaving", err: errors.New("dial tcp: lookup gateway.isolarcloud.eu on 127.0.0.11:53: server misbehaving"), want: true},
		{name: "docker dns no such host", err: errors.New("dial tcp: lookup gateway.isolarcloud.eu on 127.0.0.11:53: no such host"), want: true},
		{name: "docker dns temporary failure", err: errors.New("dial tcp: lookup gateway.isolarcloud.eu on 127.0.0.11:53: temporary failure in name resolution"), want: true},
		{name: "external resolver", err: errors.New("dial tcp: lookup gateway.isolarcloud.eu on 192.168.1.1:53: server misbehaving"), want: false},
		{name: "other", err: errors.New("API httpResponse is 500 Internal Server Error"), want: false},
	}

	for _, tc := range tests {
		if got := IsDockerDNSError(tc.err); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestShouldTryNextLoginAttemptSkipsDockerDNSFailures(t *testing.T) {
	if ShouldTryNextLoginAttempt(errors.New("dial tcp: lookup gateway.isolarcloud.eu on 127.0.0.11:53: server misbehaving")) {
		t.Fatal("Docker DNS failures should not fan out through all login attempts")
	}
	if !ShouldTryNextLoginAttempt(errors.New("API httpResponse is 500 Internal Server Error")) {
		t.Fatal("regular recoverable gateway failures should still try fallback login attempts")
	}
}

func TestSummarizeLoginAttemptFailuresGroupsByHost(t *testing.T) {
	err := SummarizeLoginAttemptFailures([]LoginAttemptFailure{
		{
			Attempt: LoginAttempt{Host: "https://gateway.isolarcloud.eu", AppKey: DefaultApiAppKey},
			Err:     errors.New("dial tcp: lookup gateway.isolarcloud.eu on 127.0.0.11:53: server misbehaving"),
		},
		{
			Attempt: LoginAttempt{Host: "https://gateway.isolarcloud.eu", AppKey: OldLoginAppKey},
			Err:     errors.New("dial tcp: lookup gateway.isolarcloud.eu on 127.0.0.11:53: server misbehaving"),
		},
		{
			Attempt: LoginAttempt{Host: "https://augateway.isolarcloud.com", AppKey: DefaultApiAppKey},
			Err:     errors.New("dial tcp: lookup augateway.isolarcloud.com on 127.0.0.11:53: no such host"),
		},
	})
	if err == nil {
		t.Fatal("expected summarized error")
	}

	msg := err.Error()
	for _, expected := range []string{
		"login candidate sequence failed:",
		"first failure: https://gateway.isolarcloud.eu: dial tcp: lookup gateway.isolarcloud.eu on 127.0.0.11:53: server misbehaving",
		"terminal stop reason: https://augateway.isolarcloud.com: dial tcp: lookup augateway.isolarcloud.com on 127.0.0.11:53: no such host",
		"https://gateway.isolarcloud.eu (2 attempts): dial tcp: lookup gateway.isolarcloud.eu on 127.0.0.11:53: server misbehaving",
		"https://augateway.isolarcloud.com (1 attempts): dial tcp: lookup augateway.isolarcloud.com on 127.0.0.11:53: no such host",
	} {
		if !strings.Contains(msg, expected) {
			t.Fatalf("expected summary to contain %q, got %q", expected, msg)
		}
	}
	if got := ClassifyFailure(err); got != FailureClassRecoverableRemote {
		t.Fatalf("ClassifyFailure(summary) = %q, want %q", got, FailureClassRecoverableRemote)
	}
	if IsDockerDNSError(err) {
		t.Fatal("later Docker DNS text must not classify the login sequence as a Docker DNS outage")
	}
}

func TestFinalizeLoginAttemptFailuresPreservesSingleDockerDNSFailure(t *testing.T) {
	dnsErr := errors.New("dial tcp: lookup augateway.isolarcloud.com on 127.0.0.11:53: no such host")
	err := FinalizeLoginAttemptFailures([]LoginAttemptFailure{
		{
			Attempt: LoginAttempt{Host: "https://augateway.isolarcloud.com", AppKey: DefaultApiAppKey},
			Err:     dnsErr,
		},
	}, errors.New("unused fallback"))

	if !errors.Is(err, dnsErr) {
		t.Fatalf("FinalizeLoginAttemptFailures() = %v, want original DNS error", err)
	}
	if got := ClassifyFailure(err); got != FailureClassDockerDNS {
		t.Fatalf("ClassifyFailure(single failure) = %q, want %q", got, FailureClassDockerDNS)
	}
}

func TestSummarizeLoginAttemptFailuresBoundsDistinctMessagesPerHost(t *testing.T) {
	username := "person@example.test"
	password := "correct-horse-battery-staple"
	token := "12345_secret-token"
	err := SummarizeLoginAttemptFailures([]LoginAttemptFailure{
		{Attempt: LoginAttempt{Host: "https://gateway.example"}, Err: errors.New("first for " + username)},
		{Attempt: LoginAttempt{Host: "https://gateway.example"}, Err: errors.New("middle")},
		{Attempt: LoginAttempt{Host: "https://gateway.example"}, Err: errors.New("terminal with " + password + " and " + token)},
	}, username, password, token)
	if err == nil {
		t.Fatal("expected summarized error")
	}

	msg := err.Error()
	if !strings.Contains(msg, "https://gateway.example (3 attempts): first for <redacted> | terminal with <redacted> and <redacted>") {
		t.Fatalf("expected bounded ordered messages, got %q", msg)
	}
	if strings.Contains(msg, "middle") {
		t.Fatalf("middle distinct per-host message must be omitted in favor of the terminal reason, got %q", msg)
	}
	for _, secret := range []string{username, password, token} {
		if strings.Contains(msg, secret) {
			t.Fatalf("summary contains secret %q: %q", secret, msg)
		}
	}
}

func TestEndpointFailurePreservesStableClassification(t *testing.T) {
	sequenceErr := SummarizeLoginAttemptFailures([]LoginAttemptFailure{
		{Attempt: LoginAttempt{Host: "https://gateway.example"}, Err: errors.New("login rejected by gateway")},
		{Attempt: LoginAttempt{Host: "https://fallback.example"}, Err: errors.New("lookup fallback.example on 127.0.0.11:53: no such host")},
	})
	endpointErr := endpointFailure{err: sequenceErr}

	if !endpointErr.IsError() {
		t.Fatal("endpoint failure must report an error")
	}
	if got := ClassifyFailure(endpointErr.GetError()); got != FailureClassRecoverableRemote {
		t.Fatalf("ClassifyFailure(endpoint error) = %q, want %q", got, FailureClassRecoverableRemote)
	}
}
