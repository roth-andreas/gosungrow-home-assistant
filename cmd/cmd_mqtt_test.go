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

func TestCmdMqttIsRecoverableGatewayError(t *testing.T) {
	c := NewCmdMqtt("")

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "token invalid", err: errors.New("er_token_login_invalid"), want: true},
		{name: "http 500", err: errors.New("API httpResponse is 500 Internal Server Error"), want: true},
		{name: "other", err: errors.New("mqtt publish failed"), want: false},
	}

	for _, tc := range tests {
		if got := c.isRecoverableGatewayError(tc.err); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestCmdMqttRetryStartupRecoverableRelogsAndRetries(t *testing.T) {
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
	err := c.retryStartupRecoverable("metadata discovery", func() error {
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

func TestCmdMqttRetryStartupRecoverableRetriesHttp500(t *testing.T) {
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
	err := c.retryStartupRecoverable("metadata discovery", func() error {
		runCalls++
		if runCalls < 3 {
			return errors.New("API httpResponse is 500 Internal Server Error")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loginCalls != 2 {
		t.Fatalf("expected 2 login refreshes, got %d", loginCalls)
	}
	if runCalls != 3 {
		t.Fatalf("expected 3 execution attempts, got %d", runCalls)
	}
}

func TestCmdMqttRetryStartupRecoverableLeavesNonRecoverableErrorsAlone(t *testing.T) {
	c := NewCmdMqtt("")

	originalLogin := mqttApiLogin
	defer func() { mqttApiLogin = originalLogin }()

	loginCalls := 0
	mqttApiLogin = func(force bool) error {
		loginCalls++
		return nil
	}

	expected := errors.New("broker unavailable")
	err := c.retryStartupRecoverable("device discovery", func() error {
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected original error, got %v", err)
	}
	if loginCalls != 0 {
		t.Fatalf("expected no login refresh, got %d", loginCalls)
	}
}

func TestMergeDefaultMqttEndpointsAddsRequiredVirtualIncludes(t *testing.T) {
	endpoints := MqttEndPoints{
		"queryDeviceList": {
			Include: []string{"legacy.*"},
			Exclude: []string{"custom.exclude"},
		},
	}

	changed, err := mergeDefaultMqttEndpoints(&endpoints)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected endpoint config to change")
	}
	if !stringSliceContains(endpoints["queryDeviceList"].Include, "legacy.*") {
		t.Fatalf("expected custom include to be preserved: %#v", endpoints["queryDeviceList"].Include)
	}
	if !stringSliceContains(endpoints["queryDeviceList"].Include, "virtual.*") {
		t.Fatalf("expected required virtual include to be added: %#v", endpoints["queryDeviceList"].Include)
	}
	if !stringSliceContains(endpoints["queryDeviceList"].Exclude, "custom.exclude") {
		t.Fatalf("expected custom exclude to be preserved: %#v", endpoints["queryDeviceList"].Exclude)
	}
	if _, ok := endpoints["queryDeviceRealTimeDataByPsKeys"]; !ok {
		t.Fatal("expected missing default endpoint to be added")
	}
}

func TestMergeDefaultMqttEndpointsLeavesCurrentDefaultsUnchanged(t *testing.T) {
	endpoints, err := defaultMqttEndpoints()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	changed, err := mergeDefaultMqttEndpoints(&endpoints)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Fatal("expected current default endpoint config to remain unchanged")
	}
}
