package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/getDeviceList"
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

func TestCmdMqttIsDockerDNSError(t *testing.T) {
	c := NewCmdMqtt("")

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "docker dns server misbehaving", err: errors.New("dial tcp: lookup gateway.isolarcloud.eu on 127.0.0.11:53: server misbehaving"), want: true},
		{name: "docker dns no such host", err: errors.New("dial tcp: lookup gateway.isolarcloud.eu on 127.0.0.11:53: no such host"), want: true},
		{name: "docker dns temporary failure", err: errors.New("dial tcp: lookup gateway.isolarcloud.eu on 127.0.0.11:53: temporary failure in name resolution"), want: true},
		{name: "non docker dns", err: errors.New("dial tcp: lookup gateway.isolarcloud.eu on 192.168.1.1:53: server misbehaving"), want: false},
		{name: "other", err: errors.New("API httpResponse is 500 Internal Server Error"), want: false},
	}

	for _, tc := range tests {
		if got := c.isDockerDNSError(tc.err); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestCmdMqttShouldRestartAfterRepeatedDockerDNSErrors(t *testing.T) {
	c := NewCmdMqtt("")
	err := errors.New("dial tcp: lookup gateway.isolarcloud.eu on 127.0.0.11:53: server misbehaving")

	for i := 1; i < dockerDNSRuntimeRestartThreshold; i++ {
		if c.shouldRestartAfterDockerDNSError(err) {
			t.Fatalf("attempt %d should not restart before threshold", i)
		}
	}
	if !c.shouldRestartAfterDockerDNSError(err) {
		t.Fatal("expected restart once Docker DNS failures reach threshold")
	}
}

func TestCmdMqttShouldRestartAfterDockerDNSErrorResetsOnOtherError(t *testing.T) {
	c := NewCmdMqtt("")
	c.dockerDNSErrorCount = 1

	if c.shouldRestartAfterDockerDNSError(errors.New("API httpResponse is 500 Internal Server Error")) {
		t.Fatal("non-Docker-DNS error should not restart")
	}
	if c.dockerDNSErrorCount != 0 {
		t.Fatalf("expected Docker DNS failure count reset, got %d", c.dockerDNSErrorCount)
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

func TestDescribeRealtimePsKeySelectionPrefersType14(t *testing.T) {
	devices := getDeviceList.Devices{
		testDeviceListDevice("100", "100_11_1_1", 11),
		testDeviceListDevice("100", "100_14_1_1", 14),
	}

	got := describeRealtimePsKeySelection(devices)
	want := "1 plant: ps_id=100 ps_key=100_14_1_1 device_type=14 source=device-type-14"
	if got != want {
		t.Fatalf("unexpected realtime selection: %q", got)
	}
}

func TestDescribeRealtimePsKeySelectionPrefersType11OverCommunicationModuleFallback(t *testing.T) {
	devices := getDeviceList.Devices{
		testDeviceListDevice("100", "100_22_247_1", 22),
		testDeviceListDevice("100", "100_11_0_0", 11),
	}

	got := describeRealtimePsKeySelection(devices)
	want := "1 plant: ps_id=100 ps_key=100_11_0_0 device_type=11 source=device-type-11"
	if got != want {
		t.Fatalf("unexpected realtime selection: %q", got)
	}
}

func TestSelectRealtimePsKeyTargetsReturnsOneTargetPerPlant(t *testing.T) {
	devices := getDeviceList.Devices{
		testDeviceListDevice("200", "200_22_247_1", 22),
		testDeviceListDevice("100", "100_11_0_0", 11),
		testDeviceListDevice("200", "200_14_1_1", 14),
		testDeviceListDevice("100", "100_14_1_1", 14),
	}

	targets := selectRealtimePsKeyTargets(devices)
	if len(targets) != 2 {
		t.Fatalf("expected two realtime targets, got %#v", targets)
	}
	if targets[0].PsID != "100" || targets[0].PsKey != "100_14_1_1" {
		t.Fatalf("unexpected first target: %#v", targets[0])
	}
	if targets[1].PsID != "200" || targets[1].PsKey != "200_14_1_1" {
		t.Fatalf("unexpected second target: %#v", targets[1])
	}
}

func TestSelectRealtimePsKeyTargetsIgnoresDevicesWithoutPsKey(t *testing.T) {
	devices := getDeviceList.Devices{
		testDeviceListDevice("100", "", 14),
		testDeviceListDevice("100", "100_11_0_0", 11),
	}

	targets := selectRealtimePsKeyTargets(devices)
	if len(targets) != 1 {
		t.Fatalf("expected one realtime target, got %#v", targets)
	}
	if targets[0].PsKey != "100_11_0_0" {
		t.Fatalf("unexpected selected target: %#v", targets[0])
	}
}

func TestSelectRealtimePsKeyTargetsDerivesPsIDFromPsKey(t *testing.T) {
	devices := getDeviceList.Devices{
		testDeviceListDevice("", "300_14_1_1", 14),
	}

	targets := selectRealtimePsKeyTargets(devices)
	if len(targets) != 1 {
		t.Fatalf("expected one realtime target, got %#v", targets)
	}
	if targets[0].PsID != "300" || targets[0].PsKey != "300_14_1_1" {
		t.Fatalf("unexpected selected target: %#v", targets[0])
	}
}

func TestDescribeRealtimePsKeySelectionIncludesMultiplePlants(t *testing.T) {
	devices := getDeviceList.Devices{
		testDeviceListDevice("200", "200_14_1_1", 14),
		testDeviceListDevice("100", "100_11_0_0", 11),
	}

	got := describeRealtimePsKeySelection(devices)
	if !strings.Contains(got, "2 plants") {
		t.Fatalf("expected multi-plant summary, got %q", got)
	}
	if !strings.Contains(got, "ps_id=100 ps_key=100_11_0_0") || !strings.Contains(got, "ps_id=200 ps_key=200_14_1_1") {
		t.Fatalf("expected both selected plants in summary, got %q", got)
	}
}

func TestBuildMqttEndpointBatchesSplitsRealtimePerTarget(t *testing.T) {
	batches := buildMqttEndpointBatches(
		[]string{"queryDeviceList", realtimeEndpointName, "getPsList"},
		[]realtimePsKeyTarget{
			{PsID: "100", PsKey: "100_14_1_1"},
			{PsID: "200", PsKey: "200_11_0_0"},
		},
	)

	if len(batches) != 3 {
		t.Fatalf("expected non-realtime plus two realtime batches, got %#v", batches)
	}
	if strings.Join(batches[0].Endpoints, ",") != "queryDeviceList,getPsList" || len(batches[0].Args) != 0 {
		t.Fatalf("unexpected non-realtime batch: %#v", batches[0])
	}
	if strings.Join(batches[1].Endpoints, ",") != realtimeEndpointName || strings.Join(batches[1].Args, ",") != "PsKeyList:100_14_1_1" {
		t.Fatalf("unexpected first realtime batch: %#v", batches[1])
	}
	if strings.Join(batches[2].Endpoints, ",") != realtimeEndpointName || strings.Join(batches[2].Args, ",") != "PsKeyList:200_11_0_0" {
		t.Fatalf("unexpected second realtime batch: %#v", batches[2])
	}
}

func TestBuildMqttEndpointBatchesSkipsRealtimeWithoutTargets(t *testing.T) {
	batches := buildMqttEndpointBatches([]string{"queryDeviceList", realtimeEndpointName}, nil)
	if len(batches) != 1 {
		t.Fatalf("expected only non-realtime batch, got %#v", batches)
	}
	if strings.Join(batches[0].Endpoints, ",") != "queryDeviceList" {
		t.Fatalf("unexpected batch: %#v", batches[0])
	}
}

func TestFormatSungrowDeviceTypeSummary(t *testing.T) {
	devices := getDeviceList.Devices{
		testDeviceListDevice("", "", 22),
		testDeviceListDevice("", "", 14),
		testDeviceListDevice("", "", 22),
	}

	got := formatSungrowDeviceTypeSummary(devices)
	if got != "14=1, 22=2" {
		t.Fatalf("unexpected device type summary: %q", got)
	}
}

func testDeviceListDevice(psID string, psKey string, deviceType int64) getDeviceList.Device {
	var device getDeviceList.Device
	device.PsId.SetString(psID)
	device.PsKey.SetValue(psKey)
	device.DeviceType.SetValue(deviceType)
	return device
}
