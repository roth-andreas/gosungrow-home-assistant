package login

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api/GoStruct/output"
)

type authTransport func(*http.Request) (*http.Response, error)

func (f authTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestForcedAuthenticationStagesBothTokenAndLoginCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")
	body := func(token string) string {
		return fmt.Sprintf(`{"req_serial_num":"synthetic","result_code":"1","result_msg":"success","result_data":{"token":%q,"login_state":"1","loginLastDate":%q}}`, token, time.Now().Format("2006-01-02 15:04:05"))
	}
	calls := 0
	root := api.Web{}
	if err := root.SetUrl("https://gateway.isolarcloud.eu"); err != nil {
		t.Fatal(err)
	}
	if err := root.SetCacheDir(dir); err != nil {
		t.Fatal(err)
	}
	root = root.WithTransport(authTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body("fresh")))}, nil
	}))
	endpoint := Init(root)
	endpoint = Assert(endpoint.SetRequestByJson(output.Json(`{"user_account":"synthetic","user_password":"synthetic-secret","login_type":"1","strong_weak_password":"1","rememberMe":false,"supportTotp":"1","isNamePassword":true}`)))
	var cached Response
	if err := json.Unmarshal([]byte(body("cached")), &cached); err != nil {
		t.Fatal(err)
	}
	if err := output.FileWrite(path, cached, 0600); err != nil {
		t.Fatal(err)
	}
	if err := root.WebCacheWrite(endpoint, []byte(body("cached"))); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	beforeCache, _ := root.WebCacheRead(endpoint)
	auth := &SunGrowAuth{AppKey: "synthetic-key", UserAccount: "synthetic", UserPassword: "synthetic-secret", Force: true, TokenPath: func() string { return path }}
	if err := endpoint.Authenticate(auth); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	afterCache, _ := root.WebCacheRead(endpoint)
	if calls != 1 || endpoint.Token() != "fresh" || string(after) != string(before) || string(afterCache) != string(beforeCache) {
		t.Fatal("forced candidate reused cache or persisted before commit")
	}
	if err := endpoint.Persist(); err != nil {
		t.Fatal(err)
	}
	after, _ = os.ReadFile(path)
	afterCache, _ = root.WebCacheRead(endpoint)
	if !strings.Contains(string(after), "fresh") || !strings.Contains(string(afterCache), "fresh") {
		t.Fatal("commit did not persist the fresh response")
	}
}

func TestTokenStoreWriteFailureIsReported(t *testing.T) {
	dir := t.TempDir()
	root := api.Web{}
	_ = root.SetUrl("https://gateway.isolarcloud.eu")
	_ = root.SetCacheDir(dir)
	root = root.WithTransport(authTransport(func(*http.Request) (*http.Response, error) {
		body := fmt.Sprintf(`{"req_serial_num":"synthetic","result_code":"1","result_msg":"success","result_data":{"token":"fresh","loginLastDate":%q}}`, time.Now().Format("2006-01-02 15:04:05"))
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	}))
	endpoint := Init(root)
	reads := 0
	auth := &SunGrowAuth{AppKey: "synthetic-key", UserAccount: "synthetic", UserPassword: "synthetic-secret", Force: true, TokenPath: func() string {
		reads++
		if reads == 1 {
			return filepath.Join(dir, "missing.json")
		}
		return dir
	}}
	if err := endpoint.Authenticate(auth); err != nil {
		t.Fatal(err)
	}
	if err := endpoint.Persist(); err == nil {
		t.Fatal("token-file replacement failure was ignored")
	}
}

func TestReadTokenFileMissingIsNotFatal(t *testing.T) {
	endpoint := Init(api.Web{})
	endpoint.Auth.TokenPath = func() string { return filepath.Join(t.TempDir(), "token.json") }
	if endpoint.Auth == nil {
		t.Fatal("expected auth state")
	}

	if err := endpoint.readTokenFile(); err != nil {
		t.Fatalf("missing token file should not be fatal: %v", err)
	}
	if !endpoint.Auth.newToken {
		t.Fatal("missing token file should require a fresh login")
	}
}

func TestReadTokenFileCorruptIsNotFatal(t *testing.T) {
	home := t.TempDir()

	tokenPath := filepath.Join(home, ".GoSungrow", "AppService_login.json")
	if err := os.MkdirAll(filepath.Dir(tokenPath), 0700); err != nil {
		t.Fatalf("failed to create token dir: %v", err)
	}
	if err := os.WriteFile(tokenPath, []byte(""), 0600); err != nil {
		t.Fatalf("failed to write corrupt token file: %v", err)
	}

	endpoint := Init(api.Web{})
	endpoint.Auth.TokenPath = func() string { return tokenPath }
	if endpoint.Auth == nil {
		t.Fatal("expected auth state")
	}

	if err := endpoint.readTokenFile(); err != nil {
		t.Fatalf("corrupt token file should not be fatal: %v", err)
	}
	if !endpoint.Auth.newToken {
		t.Fatal("corrupt token file should require a fresh login")
	}
	if _, err := os.Stat(tokenPath); !os.IsNotExist(err) {
		t.Fatalf("expected corrupt token file to be removed, stat error: %v", err)
	}
}
