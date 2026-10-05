package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/login"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/WebAppService/getDevicePointAttrs"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/roth-andreas/gosungrow-home-assistant/cmdHassio"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/getDeviceList"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api/GoStruct/valueTypes"
)

type mqttRecoveryTransport func(*http.Request) (*http.Response, error)

func (f mqttRecoveryTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type mqttRecoveryFixture struct {
	command                                      *CmdMqtt
	broker                                       *pvTestTransport
	requests                                     []string
	loginCalls, collectionCalls, configWrites    int
	remoteDown, dnsDown, loginFailed, fatalQuery bool
	tokenFailures                                int
	remoteQueryFailures                          int
	fixedToken, loginHost                        string
	clock                                        time.Time
}

func newMQTTRecoveryFixture(t *testing.T) *mqttRecoveryFixture {
	t.Helper()
	oldAPI, oldWriter := cmds.Api, apiWriteConfig
	t.Cleanup(func() { cmds.Api = oldAPI; apiWriteConfig = oldWriter })
	f := &mqttRecoveryFixture{clock: time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC), broker: &pvTestTransport{}}
	f.command = pvCommand(t, f.broker)
	f.command.log.SetLogLevel("error")
	f.command.endpoints["queryDeviceList"] = MqttEndPoint{Include: []string{"virtual.100.pv_power"}}
	f.command.optionSleepDelay, f.command.optionFetchSchedule = 0, -time.Second
	f.command.now = func() time.Time { return f.clock }
	f.command.Client.SungrowDevices = nil
	rows, tree := []string{}, []string{}
	for i := 1; i <= 9; i++ {
		key := fmt.Sprintf("100_55_1_%d", i)
		f.command.Client.SungrowDevices = append(f.command.Client.SungrowDevices, testDeviceListDevice("100", key, 55))
		rows = append(rows, fmt.Sprintf(`{"ps_id":"100","ps_key":%q,"device_type":55,"uuid":%d,"point_data":[{"point_id":"p24","point_name":"Active Power","value":0.2,"unit":"kW","time_stamp":"2026-10-05 12:00:00"}]}`, key, i))
		tree = append(tree, fmt.Sprintf(`{"ps_id":"100","ps_key":%q,"device_type":55,"uuid":%d,"up_uuid":0}`, key, i))
	}
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token.json")
	sg := iSolarCloud.NewSunGro("https://gateway.isolarcloud.eu", dir)
	sg.ApiRoot = sg.ApiRoot.WithTransport(mqttRecoveryTransport(func(r *http.Request) (*http.Response, error) {
		f.requests = append(f.requests, r.URL.Host+r.URL.Path)
		isLogin := strings.HasSuffix(r.URL.Path, "/login")
		if isLogin {
			f.loginCalls++
		}
		if strings.HasSuffix(r.URL.Path, "/queryDeviceList") {
			f.collectionCalls++
		}
		if f.dnsDown {
			return nil, errors.New("lookup EU on 127.0.0.11:53: no such host")
		}
		if f.remoteDown || (isLogin && (f.loginFailed || (f.loginHost != "" && r.URL.Host != f.loginHost))) {
			if r.URL.Host == "gateway.isolarcloud.com.cn" {
				return nil, errors.New("lookup CN on 127.0.0.11:53: no such host")
			}
			return nil, errors.New("synthetic gateway timeout")
		}
		result := `{}`
		switch {
		case isLogin:
			token := fmt.Sprintf("fresh_%d", f.loginCalls)
			if f.fixedToken != "" {
				token = f.fixedToken
			}
			result = fmt.Sprintf(`{"token":%q,"user_id":"1","login_state":"1","loginLastDate":%q}`, token, time.Now().Format("2006-01-02 15:04:05"))
		case strings.HasSuffix(r.URL.Path, "/getUserList"):
			result = `[]`
		case strings.HasSuffix(r.URL.Path, "/getPsList"):
			result = `{"pageList":[{"ps_id":"100"}]}`
		case strings.HasSuffix(r.URL.Path, "/getDeviceList"):
			result = `{"pageList":[` + strings.Join(rows, ",") + `]}`
		case strings.HasSuffix(r.URL.Path, "/getPsTreeMenu"):
			result = `{"list":[` + strings.Join(tree, ",") + `]}`
		case strings.HasSuffix(r.URL.Path, "/getDevicePointAttrs"):
			result = `[]`
		case strings.HasSuffix(r.URL.Path, "/queryDeviceList"):
			if f.remoteQueryFailures > 0 {
				f.remoteQueryFailures--
				return nil, errors.New("synthetic gateway timeout")
			}
			if f.fatalQuery {
				return nil, errors.New("synthetic malformed request")
			}
			if f.tokenFailures != 0 {
				if f.tokenFailures > 0 {
					f.tokenFailures--
				}
				return mqttFixtureResponse(`{"req_serial_num":"synthetic","result_code":"E00003","result_msg":"er_token_login_invalid","result_data":{}}`), nil
			}
			result = `{"pageList":[` + strings.Join(rows, ",") + `]}`
		default:
			t.Errorf("unexpected API path %s", r.URL.Path)
		}
		return mqttFixtureResponse(`{"req_serial_num":"synthetic","result_code":"1","result_msg":"success","result_data":` + result + `}`), nil
	}))
	if err := sg.Init(); err != nil {
		t.Fatal(err)
	}
	sg.AuthDetails = &login.SunGrowAuth{TokenPath: func() string { return tokenPath }}
	cmds.Api = &CmdApi{SunGrow: sg, Url: sg.ApiRoot.ServerUrl.String(), AppKey: iSolarCloud.DefaultApiAppKey, Username: "synthetic", Password: "synthetic-secret"}
	apiWriteConfig = func() error { f.configWrites++; return nil }
	if err := cmds.Api.ApiLogin(false); err != nil {
		t.Fatal(err)
	}
	return f
}
func mqttFixtureResponse(body string) *http.Response {
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func TestMQTTCyclesRecoverFailedLoginSequenceWithoutRestart(t *testing.T) {
	f := newMQTTRecoveryFixture(t)
	c := f.command
	if err := c.Cron(); err != nil {
		t.Fatal(err)
	}
	beforeRefresh, published := c.Client.LastRefresh, len(f.broker.publications)
	if published == 0 {
		t.Fatal("fixture did not publish retained MQTT state")
	}
	f.remoteDown = true
	for cycle := 0; cycle < 3; cycle++ {
		before := len(f.requests)
		f.clock = f.clock.Add(time.Minute)
		if err := c.Cron(); err != nil {
			t.Fatal(err)
		}
		if len(f.requests) <= before || !cmds.Api.SunGrow.NeedLogin {
			t.Fatal("recoverable failure became a sticky no-network cycle")
		}
		if c.Client.LastRefresh != beforeRefresh || len(f.broker.publications) != published {
			t.Fatal("failed cycle changed refresh time or retained publications")
		}
	}
	f.remoteDown = false
	start := len(f.requests)
	f.clock = f.clock.Add(time.Minute)
	if err := c.Cron(); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(f.requests[start], "gateway.isolarcloud.eu/v1/userService/login") {
		t.Fatalf("recovery started at wrong gateway: %s", f.requests[start])
	}
	if cmds.Api.SunGrow.NeedLogin || c.rediscoveryPending || c.Client.LastRefresh == beforeRefresh || len(f.broker.publications) <= published {
		t.Fatal("healthy EU did not resume discovery and publication")
	}
	for _, publication := range f.broker.publications {
		if !publication.retained {
			t.Fatal("publication lost retained semantics")
		}
	}
}

func TestMQTTDirectDockerDNSRetriesNetworkWithoutLogin(t *testing.T) {
	f := newMQTTRecoveryFixture(t)
	c := f.command
	if err := c.Cron(); err != nil {
		t.Fatal(err)
	}
	beforeLogin, published, refreshed := f.loginCalls, len(f.broker.publications), c.Client.LastRefresh
	f.dnsDown = true
	for cycle := 0; cycle < 6; cycle++ {
		before := len(f.requests)
		if err := c.Cron(); err != nil {
			t.Fatal(err)
		}
		if len(f.requests) != before+1 || f.loginCalls != beforeLogin || cmds.Api.SunGrow.NeedLogin {
			t.Fatal("DNS retry reused an error or initiated authentication")
		}
		idx := cycle
		if idx >= len(dockerDNSRetryDelays) {
			idx = len(dockerDNSRetryDelays) - 1
		}
		if c.nextSyncDelay() != dockerDNSRetryDelays[idx] {
			t.Fatal("incorrect DNS retry delay")
		}
		if c.Client.LastRefresh != refreshed || len(f.broker.publications) != published {
			t.Fatal("DNS failure changed retained state")
		}
	}
	f.dnsDown = false
	f.clock = f.clock.Add(time.Hour)
	if err := c.Cron(); err != nil {
		t.Fatal(err)
	}
	if c.dockerDNSErrorCount != 0 || c.nextSyncDelay() != c.optionFetchSchedule || c.Client.LastRefresh == refreshed {
		t.Fatal("DNS recovery did not reset scheduling")
	}
}

func TestMQTTTokenInvalidLoginFailureDefersAndRecovers(t *testing.T) {
	f := newMQTTRecoveryFixture(t)
	f.tokenFailures, f.loginFailed = 1, true
	if err := f.command.Cron(); err != nil {
		t.Fatalf("recoverable forced login terminated MQTT: %v", err)
	}
	if !cmds.Api.SunGrow.NeedLogin || !f.command.Client.LastRefresh.IsZero() {
		t.Fatal("failed forced login lost authentication obligation")
	}
	before := len(f.requests)
	f.loginFailed = false
	if err := f.command.Cron(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.requests[before], "/login") || f.command.Client.LastRefresh.IsZero() {
		t.Fatal("next cycle failed to authenticate before collection")
	}
}

func TestMQTTRediscoveryFailureRetriesWithoutAnotherLogin(t *testing.T) {
	f := newMQTTRecoveryFixture(t)
	oldLoader := mqttLoadDevices
	t.Cleanup(func() { mqttLoadDevices = oldLoader })
	f.tokenFailures = 1
	discoveryCalls := 0
	mqttLoadDevices = func() (getDeviceList.Devices, error) {
		discoveryCalls++
		if discoveryCalls == 1 {
			cmds.Api.SunGrow.Error = errors.New("gateway timeout during synthetic discovery")
			return nil, cmds.Api.SunGrow.Error
		}
		return oldLoader()
	}
	priorDevices, _ := json.Marshal(f.command.Client.SungrowDevices)
	if err := f.command.Cron(); err != nil {
		t.Fatal(err)
	}
	if !f.command.rediscoveryPending || cmds.Api.SunGrow.NeedLogin {
		t.Fatal("discovery failure lost its separate obligation")
	}
	afterDevices, _ := json.Marshal(f.command.Client.SungrowDevices)
	if string(priorDevices) != string(afterDevices) {
		t.Fatal("failed rediscovery replaced devices")
	}
	before := f.loginCalls
	if err := f.command.Cron(); err != nil {
		t.Fatal(err)
	}
	if f.loginCalls != before || f.command.rediscoveryPending || f.command.Client.LastRefresh.IsZero() {
		t.Fatal("rediscovery retry forced another login or failed to collect")
	}
}

func TestMQTTCycleBoundsForcedLoginAndCollectionReplay(t *testing.T) {
	f := newMQTTRecoveryFixture(t)
	f.tokenFailures = -1
	before := f.loginCalls
	if err := f.command.Cron(); err != nil {
		t.Fatal(err)
	}
	if f.loginCalls != before+1 || f.collectionCalls != 2 || !cmds.Api.SunGrow.NeedLogin {
		t.Fatalf("cycle exceeded retry limits: logins=%d collections=%d", f.loginCalls-before, f.collectionCalls)
	}
	before, collections := f.loginCalls, f.collectionCalls
	if err := f.command.Cron(); err != nil {
		t.Fatal(err)
	}
	if f.loginCalls != before+1 || f.collectionCalls != collections+1 {
		t.Fatal("pending-auth cycle replayed collection or forced login twice")
	}
}

func TestMQTTPendingAuthenticationDNSBackoffPreservesAnchor(t *testing.T) {
	f := newMQTTRecoveryFixture(t)
	cmds.Api.SunGrow.RequireAuthentication()
	f.dnsDown = true
	before := f.loginCalls
	if err := f.command.Cron(); err != nil {
		t.Fatal(err)
	}
	if f.loginCalls != before+1 || !cmds.Api.SunGrow.NeedLogin || f.command.nextSyncDelay() != 15*time.Second {
		t.Fatal("pending authentication did not retain DNS obligation")
	}
	f.dnsDown = false
	if err := f.command.Cron(); err != nil {
		t.Fatal(err)
	}
	if cmds.Api.SunGrow.NeedLogin || f.command.Client.LastRefresh.IsZero() || f.command.dockerDNSErrorCount != 0 {
		t.Fatal("pending DNS authentication failed to recover")
	}
}

func TestMQTTFatalPhaseErrorsPropagate(t *testing.T) {
	for _, phase := range []string{"authentication", "persistence", "rediscovery", "collection"} {
		t.Run(phase, func(t *testing.T) {
			f := newMQTTRecoveryFixture(t)
			fatal := errors.New("synthetic fatal local error")
			oldDevices := mqttLoadDevices
			t.Cleanup(func() { mqttLoadDevices = oldDevices })
			switch phase {
			case "authentication":
				cmds.Api.Password = ""
				cmds.Api.SunGrow.RequireAuthentication()
			case "persistence":
				apiWriteConfig = func() error { return errors.New("synthetic configuration write i/o timeout") }
				cmds.Api.SunGrow.RequireAuthentication()
			case "rediscovery":
				f.command.rediscoveryPending = true
				mqttLoadDevices = func() (getDeviceList.Devices, error) { return nil, fatal }
			case "collection":
				f.fatalQuery = true
			}
			if err := f.command.Cron(); err == nil || iSolarCloud.ShouldRecoverGatewayError(err) {
				t.Fatalf("fatal %s error swallowed: %v", phase, err)
			}
			if !f.command.Client.LastRefresh.IsZero() {
				t.Fatal("fatal cycle changed LastRefresh")
			}
		})
	}
}

func TestMQTTSyncRunnerUsesDNSDeadlinesAndPropagatesFatalError(t *testing.T) {
	c := NewCmdMqtt("error")
	c.dockerDNSErrorCount = 1
	ticks, deadlines := make(chan struct{}, 4), make(chan time.Time, 1)
	delays := make(chan time.Duration, 1)
	fatal := errors.New("synthetic fatal sync failure")
	result := make(chan error, 1)
	go func() {
		result <- c.runScheduledSync(ticks, func() error { return fatal }, func(delay time.Duration) <-chan time.Time { delays <- delay; return deadlines })
	}()
	if delay := <-delays; delay != 15*time.Second {
		t.Fatalf("DNS deadline=%s", delay)
	}
	ticks <- struct{}{}
	select {
	case err := <-result:
		t.Fatalf("cron shortened DNS deadline: %v", err)
	default:
	}
	deadlines <- time.Now()
	select {
	case err := <-result:
		if !errors.Is(err, fatal) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("fatal cycle failed to terminate runner")
	}
}

func TestMQTTEndpointRecoveryRetainsSeparateReplayBound(t *testing.T) {
	f := newMQTTRecoveryFixture(t)
	f.remoteQueryFailures = 2
	before := f.loginCalls
	if err := f.command.Cron(); err != nil {
		t.Fatal(err)
	}
	if f.collectionCalls != 2 || f.loginCalls != before+1 || cmds.Api.SunGrow.NeedLogin || !f.command.Client.LastRefresh.IsZero() {
		t.Fatal("endpoint recovery exceeded one session recovery and replay")
	}
	if err := f.command.Cron(); err != nil {
		t.Fatal(err)
	}
	if f.collectionCalls != 3 || f.command.Client.LastRefresh.IsZero() {
		t.Fatal("new cycle reused endpoint replay failure")
	}
}

func TestMQTTMetadataFailurePreservesInventoryAndRetriesDiscovery(t *testing.T) {
	f := newMQTTRecoveryFixture(t)
	oldLoader := mqttLoadDevicePoints
	t.Cleanup(func() { mqttLoadDevicePoints = oldLoader })
	f.command.rediscoveryPending = true
	f.command.plantInventories["100"] = f.command.plantTopologies["100"].Inventory
	before, _ := json.Marshal(f.command.Client.SungrowDevices)
	mqttLoadDevicePoints = func() (map[string]getDevicePointAttrs.Points, error) {
		return nil, errors.New("synthetic gateway timeout")
	}
	logins := f.loginCalls
	if err := f.command.Cron(); err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(f.command.Client.SungrowDevices)
	if string(before) != string(after) || len(f.command.plantInventories["100"].Devices) != 9 || !f.command.rediscoveryPending {
		t.Fatal("failed metadata discovery changed committed membership")
	}
	mqttLoadDevicePoints = oldLoader
	if err := f.command.Cron(); err != nil {
		t.Fatal(err)
	}
	if f.loginCalls != logins || f.command.rediscoveryPending {
		t.Fatal("metadata retry unnecessarily reauthenticated")
	}
}

func TestMQTTTopologyFailureDoesNotPoisonCollection(t *testing.T) {
	f := newMQTTRecoveryFixture(t)
	oldLoader := mqttLoadPlantTrees
	t.Cleanup(func() { mqttLoadPlantTrees = oldLoader })
	mqttLoadPlantTrees = func() (iSolarCloud.PsTrees, error) {
		cmds.Api.SunGrow.Error = errors.New("synthetic invalid topology")
		return nil, cmds.Api.SunGrow.Error
	}
	f.command.rediscoveryPending = true
	if err := f.command.Cron(); err != nil {
		t.Fatal(err)
	}
	if f.command.Client.LastRefresh.IsZero() || f.collectionCalls != 1 || cmds.Api.SunGrow.Error != nil || f.command.plantTopologies["100"].Complete {
		t.Fatal("nonfatal topology failure poisoned the cycle")
	}
}

func TestMQTTFatalSequenceClassificationOverridesEarlierTokenText(t *testing.T) {
	f := newMQTTRecoveryFixture(t)
	original := mqttApiLogin
	t.Cleanup(func() { mqttApiLogin = original })
	fatal := iSolarCloud.SummarizeLoginAttemptFailures([]iSolarCloud.LoginAttemptFailure{
		{Attempt: iSolarCloud.LoginAttempt{Host: "EU"}, Err: errors.New("need to login again 'er_token_login_invalid'")},
		{Attempt: iSolarCloud.LoginAttempt{Host: "AU"}, Err: errors.New("synthetic malformed response")},
	})
	mqttApiLogin = func(bool) error { return fatal }
	cmds.Api.SunGrow.RequireAuthentication()
	if err := f.command.Cron(); !errors.Is(err, fatal) {
		t.Fatalf("fatal sequence was swallowed based on historical token text: %v", err)
	}
}

func TestRefreshPlantTopologiesKeepsFailureNonfatal(t *testing.T) {
	originalLoader := mqttLoadPlantTrees
	defer func() { mqttLoadPlantTrees = originalLoader }()
	mqttLoadPlantTrees = func() (iSolarCloud.PsTrees, error) {
		return nil, errors.New("tree unavailable")
	}

	c := NewCmdMqtt("")
	c.Client = &cmdHassio.Mqtt{SungrowDevices: getDeviceList.Devices{
		testDeviceListDevice("100", "100_55_1_1", 55),
	}}
	c.refreshPlantTopologies()

	topology, ok := c.plantTopologies["100"]
	if !ok || topology.Complete || topology.Reason != "plant topology unavailable" {
		t.Fatalf("topology after nonfatal failure = %#v", topology)
	}
}

func TestNormalizeEntityMeasurementKeepsReactivePowerMetadataAligned(t *testing.T) {
	tests := []struct {
		name  string
		unit  string
		value float64
	}{
		{name: "base reading", unit: "var", value: 750},
		{name: "equivalent kilo reading", unit: "kVar", value: 0.75},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			entry := &api.DataEntry{
				Point: &api.Point{Unit: tc.unit, ValueType: "Reactive Power"},
				Value: valueTypes.SetUnitValueFloat(tc.unit, "Reactive Power", tc.value),
			}
			unit := normalizeEntityMeasurement(entry)
			if entry.Value.Value() != 750 {
				t.Fatalf("state value = %v, want 750", entry.Value.Value())
			}
			if unit != "var" || entry.Value.Unit() != unit || entry.Point.Unit != unit {
				t.Fatalf("units disagree: returned=%q value=%q point=%q", unit, entry.Value.Unit(), entry.Point.Unit)
			}
			if entry.Point.ValueType != "Reactive Power" {
				t.Fatalf("point type = %q", entry.Point.ValueType)
			}
		})
	}
}

func TestNormalizeEntityMeasurementLeavesUnrelatedMetadataUntouched(t *testing.T) {
	entry := &api.DataEntry{
		Point: &api.Point{Unit: "%", ValueType: "Percent"},
		Value: valueTypes.SetUnitValueFloat("%", "Percent", 75),
	}
	unit := normalizeEntityMeasurement(entry)
	if unit != "%" || entry.Point.Unit != "%" || entry.Value.Value() != 75 {
		t.Fatalf("unrelated measurement changed: %#v", entry)
	}
}

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
		{name: "request timeout", err: errors.New("context deadline exceeded (Client.Timeout exceeded while awaiting headers)"), want: true},
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

func TestCmdMqttDockerDNSBackoffSequenceAndCap(t *testing.T) {
	c := NewCmdMqtt("")
	err := errors.New("dial tcp: lookup gateway.isolarcloud.eu on 127.0.0.11:53: server misbehaving")
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }

	want := []time.Duration{
		15 * time.Second,
		30 * time.Second,
		60 * time.Second,
		120 * time.Second,
		300 * time.Second,
		300 * time.Second,
	}
	for i, delay := range want {
		c.recordDockerDNSError(err)
		if got := c.nextSyncDelay(); got != delay {
			t.Fatalf("attempt %d: got delay %s want %s", i+1, got, delay)
		}
	}
	if c.dockerDNSOutageAt != now {
		t.Fatalf("outage start changed: got %s want %s", c.dockerDNSOutageAt, now)
	}
}

func TestCmdMqttDockerDNSRecoveryResetsNormalSchedule(t *testing.T) {
	c := NewCmdMqtt("")
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }
	err := errors.New("dial tcp: lookup gateway.isolarcloud.eu on 127.0.0.11:53: server misbehaving")
	c.recordDockerDNSError(err)
	now = now.Add(45 * time.Second)
	c.clearDockerDNSOutage()

	if c.dockerDNSErrorCount != 0 {
		t.Fatalf("expected Docker DNS failure count reset, got %d", c.dockerDNSErrorCount)
	}
	if !c.dockerDNSOutageAt.IsZero() {
		t.Fatalf("expected outage timestamp reset, got %s", c.dockerDNSOutageAt)
	}
	if got := c.nextSyncDelay(); got != c.optionFetchSchedule {
		t.Fatalf("got normal delay %s want %s", got, c.optionFetchSchedule)
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

func TestCmdMqttRetryStartupDockerDNSErrorDoesNotRelogin(t *testing.T) {
	c := NewCmdMqtt("")

	originalLogin := mqttApiLogin
	defer func() { mqttApiLogin = originalLogin }()

	loginCalls := 0
	mqttApiLogin = func(force bool) error {
		loginCalls++
		return nil
	}

	expected := errors.New("dial tcp: lookup gateway.isolarcloud.eu on 127.0.0.11:53: server misbehaving")
	runCalls := 0
	err := c.retryStartupRecoverable("device discovery", func() error {
		runCalls++
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected original DNS error, got %v", err)
	}
	if loginCalls != 0 {
		t.Fatalf("expected no login refresh for DNS failure, got %d", loginCalls)
	}
	if runCalls != 1 {
		t.Fatalf("expected one startup attempt before wrapper retry, got %d", runCalls)
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
