package cmd

import (
	"errors"
	"testing"
)

func TestCmdMqttIsTokenInvalidError(t *testing.T) {
	c := NewCmdMqtt("")

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "token code", err: errors.New("er_token_login_invalid"), want: true},
		{name: "need login", err: errors.New("Need to login again"), want: true},
		{name: "other", err: errors.New("mqtt publish failed"), want: false},
	}

	for _, tc := range tests {
		if got := c.isTokenInvalidError(tc.err); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestCmdMqttRetryStartupTokenInvalidRelogsAndRetries(t *testing.T) {
	c := NewCmdMqtt("")

	originalLogin := mqttApiLogin
	defer func() { mqttApiLogin = originalLogin }()

	loginCalls := 0
	mqttApiLogin = func(force bool) error {
		if !force {
			t.Fatal("expected forced login refresh")
		}
		loginCalls++
		return nil
	}

	runCalls := 0
	err := c.retryStartupTokenInvalid("metadata discovery", func() error {
		runCalls++
		if runCalls == 1 {
			return errors.New("need to login again 'er_token_login_invalid'")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loginCalls != 1 {
		t.Fatalf("expected 1 login refresh, got %d", loginCalls)
	}
	if runCalls != 2 {
		t.Fatalf("expected 2 execution attempts, got %d", runCalls)
	}
}

func TestCmdMqttRetryStartupTokenInvalidLeavesNonTokenErrorsAlone(t *testing.T) {
	c := NewCmdMqtt("")

	originalLogin := mqttApiLogin
	defer func() { mqttApiLogin = originalLogin }()

	loginCalls := 0
	mqttApiLogin = func(force bool) error {
		loginCalls++
		return nil
	}

	expected := errors.New("broker unavailable")
	err := c.retryStartupTokenInvalid("device discovery", func() error {
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected original error, got %v", err)
	}
	if loginCalls != 0 {
		t.Fatalf("expected no login refresh, got %d", loginCalls)
	}
}
