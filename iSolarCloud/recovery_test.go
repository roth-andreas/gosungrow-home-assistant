package iSolarCloud

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/getPsDetail"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/getPsList"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/login"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api/GoStruct/output"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
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

type recoveryTransport func(*http.Request) (*http.Response, error)

func (f recoveryTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func recoveryResponse(body string) *http.Response {
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}
func recoveryLoginResponse(token string) string {
	return fmt.Sprintf(`{"req_serial_num":"synthetic","result_code":"1","result_msg":"success","result_data":{"token":%q,"user_id":"1","login_state":"1","loginLastDate":%q}}`, token, time.Now().Format("2006-01-02 15:04:05"))
}
func recoveryClient(t *testing.T, transport http.RoundTripper) (*SunGrow, login.SunGrowAuth, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	sg := NewSunGro("https://gateway.isolarcloud.eu", dir)
	sg.ApiRoot = sg.ApiRoot.WithTransport(transport)
	if err := sg.Init(); err != nil {
		t.Fatal(err)
	}
	auth := login.SunGrowAuth{AppKey: DefaultApiAppKey, UserAccount: "synthetic", UserPassword: "synthetic-secret", Force: true, TokenPath: func() string { return path }}
	return sg, auth, path
}

func TestRecoveryCandidatesPreserveSessionAndPersistedResponse(t *testing.T) {
	mode := "healthy"
	hosts := []string{}
	sg, auth, tokenPath := recoveryClient(t, recoveryTransport(func(r *http.Request) (*http.Response, error) {
		hosts = append(hosts, r.URL.Host)
		if mode == "failed" {
			if r.URL.Host == "gateway.isolarcloud.com.cn" {
				return nil, errors.New("lookup gateway.isolarcloud.com.cn on 127.0.0.11:53: no such host")
			}
			return nil, errors.New("synthetic gateway timeout")
		}
		if strings.HasSuffix(r.URL.Path, "/login") {
			return recoveryResponse(recoveryLoginResponse(mode)), nil
		}
		return recoveryResponse(`{"req_serial_num":"synthetic","result_code":"1","result_msg":"success","result_data":[]}`), nil
	}))
	if err := sg.AuthenticateSession(auth, false, nil); err != nil {
		t.Fatal(err)
	}
	oldAuth, oldDetails := sg.Auth, sg.AuthDetails
	before, _ := os.ReadFile(tokenPath)
	mode = "failed"
	hosts = nil
	err := sg.AuthenticateSession(auth, true, nil)
	if err == nil || ClassifyFailure(err) != FailureClassRecoverableRemote || !sg.NeedLogin {
		t.Fatalf("pending remote recovery: %v", err)
	}
	after, _ := os.ReadFile(tokenPath)
	if string(after) != string(before) || sg.AuthDetails != oldDetails || sg.Auth.Auth != oldAuth.Auth || !reflect.DeepEqual(sg.Auth.Response, oldAuth.Response) || sg.ApiRoot.ServerUrl.String() != "https://gateway.isolarcloud.eu" {
		t.Fatal("failed candidate replaced the retained session or token")
	}
	if len(hosts) < 2 || hosts[0] != "gateway.isolarcloud.eu" {
		t.Fatalf("no real candidate attempts: %v", hosts)
	}
	sg.BeginRetry()
	_ = sg.GetByStruct(getPsList.EndPointName, nil, time.Second)
	if !sg.NeedLogin || len(hosts) == 0 || sg.Error == nil {
		t.Fatal("retained token authorized collection while pending")
	}
	mode = "recovered"
	hosts = nil
	sg.BeginRetry()
	if err = sg.AuthenticateSession(auth, true, nil); err != nil {
		t.Fatal(err)
	}
	if sg.NeedLogin || sg.GetToken() != "recovered" || hosts[0] != "gateway.isolarcloud.eu" {
		t.Fatalf("recovery did not start from EU: %v", hosts)
	}
	after, _ = os.ReadFile(tokenPath)
	if string(after) == string(before) {
		t.Fatal("successful response was not persisted")
	}
}

func TestForcedAuthenticationIgnoresBothTokenAndResponseCaches(t *testing.T) {
	requests := 0
	sg, auth, tokenPath := recoveryClient(t, recoveryTransport(func(r *http.Request) (*http.Response, error) {
		requests++
		if strings.HasSuffix(r.URL.Path, "/login") {
			return recoveryResponse(recoveryLoginResponse("fresh")), nil
		}
		return recoveryResponse(`{"req_serial_num":"synthetic","result_code":"1","result_msg":"success","result_data":[]}`), nil
	}))
	var response login.Response
	if err := json.Unmarshal([]byte(recoveryLoginResponse("cached")), &response); err != nil {
		t.Fatal(err)
	}
	if err := output.FileWrite(tokenPath, response, 0600); err != nil {
		t.Fatal(err)
	}
	endpoint := login.Init(sg.ApiRoot)
	endpoint = login.Assert(endpoint.SetRequestByJson(output.Json(`{"user_account":"synthetic","user_password":"synthetic-secret","login_type":"1","strong_weak_password":"1","rememberMe":false,"supportTotp":"1","isNamePassword":true}`)))
	if err := sg.ApiRoot.WebCacheWrite(endpoint, []byte(recoveryLoginResponse("cached"))); err != nil {
		t.Fatal(err)
	}
	if err := sg.AuthenticateSession(auth, true, nil); err != nil {
		t.Fatal(err)
	}
	if requests != 2 || sg.GetToken() != "fresh" {
		t.Fatalf("forced login reused cache: requests=%d token=%s", requests, sg.GetToken())
	}
}

func TestPendingAuthenticationStopsOnFirstDockerDNSAndRetriesAnchor(t *testing.T) {
	down, calls := false, 0
	sg, auth, _ := recoveryClient(t, recoveryTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if down {
			return nil, errors.New("lookup gateway.isolarcloud.eu on 127.0.0.11:53: no such host")
		}
		if strings.HasSuffix(r.URL.Path, "/login") {
			return recoveryResponse(recoveryLoginResponse("healthy")), nil
		}
		return recoveryResponse(`{"req_serial_num":"synthetic","result_code":"1","result_msg":"success","result_data":[]}`), nil
	}))
	if err := sg.AuthenticateSession(auth, false, nil); err != nil {
		t.Fatal(err)
	}
	sg.RequireAuthentication()
	down = true
	for i := 0; i < 2; i++ {
		calls = 0
		sg.BeginRetry()
		err := sg.AuthenticateSession(auth, true, nil)
		if !IsDockerDNSError(err) || calls != 1 || !sg.NeedLogin || sg.lastSuccessfulHost != "https://gateway.isolarcloud.eu" {
			t.Fatalf("DNS obligation lost or rotated: err=%v calls=%d", err, calls)
		}
	}
	down = false
	sg.BeginRetry()
	if err := sg.AuthenticateSession(auth, true, nil); err != nil || sg.NeedLogin {
		t.Fatalf("pending authentication did not recover: %v", err)
	}
}

func TestPersistenceFailurePreventsPromotion(t *testing.T) {
	token := "original"
	sg, auth, path := recoveryClient(t, recoveryTransport(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "/login") {
			return recoveryResponse(recoveryLoginResponse(token)), nil
		}
		return recoveryResponse(`{"req_serial_num":"synthetic","result_code":"1","result_msg":"success","result_data":[]}`), nil
	}))
	if err := sg.AuthenticateSession(auth, false, nil); err != nil {
		t.Fatal(err)
	}
	oldDetails := sg.AuthDetails
	token = "replacement"
	failure := errors.New("synthetic configuration write failure")
	if err := sg.AuthenticateSession(auth, true, func(*SunGrow) error { return failure }); !errors.Is(err, failure) {
		t.Fatalf("persistence error=%v", err)
	}
	if sg.GetToken() != "original" || sg.AuthDetails != oldDetails {
		t.Fatal("failed configuration persistence promoted candidate")
	}
	// Token-file failure also cannot promote, using an isolated directory path.
	sg.Error = nil
	auth.TokenPath = func() string { return filepath.Dir(path) }
	if err := sg.AuthenticateSession(auth, true, nil); err == nil || ShouldRecoverGatewayError(err) {
		t.Fatalf("token I/O failure was not fatal: %v", err)
	}
	if sg.GetToken() != "original" {
		t.Fatal("token-file failure promoted candidate")
	}
}

func TestRuntimeCandidateListsPrependSuccessfulHostAndKeyIndependently(t *testing.T) {
	attempts := buildSessionAttempts("https://configured.example", DefaultApiAppKey, "https://gateway.isolarcloud.eu", OldLoginAppKey)
	if attempts[0] != (LoginAttempt{"https://gateway.isolarcloud.eu", OldLoginAppKey}) || attempts[1].AppKey != DefaultApiAppKey {
		t.Fatalf("runtime ordering: %v", attempts[:2])
	}
	seen := map[LoginAttempt]bool{}
	for _, attempt := range attempts {
		if seen[attempt] {
			t.Fatal("duplicate candidate")
		}
		seen[attempt] = true
	}
	if !seen[LoginAttempt{"https://configured.example", OldLoginAppKey}] {
		t.Fatal("successful key was not prepended for configured gateway")
	}
}

func TestCollectorReportsPlantDiscoveryFailureWithoutAnotherLookup(t *testing.T) {
	calls := 0
	sg, _, _ := recoveryClient(t, recoveryTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("lookup EU on 127.0.0.11:53: no such host")
	}))
	data := sg.NewSunGrowData()
	data.SetPsIds()
	if data.Error == nil || calls != 1 {
		t.Fatalf("discovery error not propagated immediately: calls=%d err=%v", calls, data.Error)
	}
}

func TestFailedCandidateValidationDoesNotPersistAndRedactsCandidateToken(t *testing.T) {
	validating := false
	sg, auth, path := recoveryClient(t, recoveryTransport(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "/login") {
			token := "retained"
			if validating {
				token = "PRIVATE_CANDIDATE_TOKEN"
			}
			return recoveryResponse(recoveryLoginResponse(token)), nil
		}
		if validating {
			return nil, errors.New("gateway timeout validating PRIVATE_CANDIDATE_TOKEN")
		}
		return recoveryResponse(`{"req_serial_num":"synthetic","result_code":"1","result_msg":"success","result_data":[]}`), nil
	}))
	if err := sg.AuthenticateSession(auth, false, nil); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	cached, _ := sg.ApiRoot.WebCacheRead(sg.Auth)
	validating = true
	err := sg.AuthenticateSession(auth, true, nil)
	after, _ := os.ReadFile(path)
	afterCache, _ := sg.ApiRoot.WebCacheRead(sg.Auth)
	if err == nil || strings.Contains(err.Error(), "PRIVATE_CANDIDATE_TOKEN") || sg.GetToken() != "retained" || string(before) != string(after) || string(cached) != string(afterCache) {
		t.Fatalf("failed candidate validation leaked or persisted its token: %v", err)
	}
}

func TestEndpointReplayPreservesOpaqueRequestAndTimeoutWithPromotedSession(t *testing.T) {
	moving := false
	sg, auth, _ := recoveryClient(t, recoveryTransport(func(r *http.Request) (*http.Response, error) {
		if moving && r.URL.Host == "gateway.isolarcloud.eu" {
			return nil, errors.New("synthetic gateway timeout")
		}
		if strings.HasSuffix(r.URL.Path, "/login") {
			token := "old"
			if moving {
				token = "new"
			}
			return recoveryResponse(recoveryLoginResponse(token)), nil
		}
		return recoveryResponse(`{"req_serial_num":"synthetic","result_code":"1","result_msg":"success","result_data":[]}`), nil
	}))
	if err := sg.AuthenticateSession(auth, false, nil); err != nil {
		t.Fatal(err)
	}
	endpoint := sg.GetEndpoint(getPsDetail.EndPointName).SetRequestByJson(output.Json(`{"ps_id":"opaque_plant_id"}`)).SetCacheTimeout(37 * time.Second)
	if err := endpoint.GetError(); err != nil {
		t.Fatal(err)
	}
	moving = true
	if err := sg.AuthenticateSession(auth, true, nil); err != nil {
		t.Fatal(err)
	}
	replay := sg.rebuildEndpointForCurrentGateway(endpoint)
	rebuilt := getPsDetail.Assert(replay)
	if replay.GetError() != nil || replay.GetRequestJson() != endpoint.GetRequestJson() || replay.GetCacheTimeout() != 37*time.Second || rebuilt.Request.Token != "new" || rebuilt.ApiRoot.ServerUrl.String() != DefaultHost {
		t.Fatal("replay lost request, timeout, or promoted session")
	}
}
