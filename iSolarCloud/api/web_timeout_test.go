package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestWebUsesBoundedPrivateHTTPClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(80 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	web := Web{client: http.Client{Timeout: 20 * time.Millisecond}}
	request, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = web.do(request)
	if err == nil {
		t.Fatal("expected request timeout")
	}
	var netError interface{ Timeout() bool }
	if !errors.As(err, &netError) || !netError.Timeout() {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func TestWebSucceedsAfterTimedOutRequest(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			time.Sleep(60 * time.Millisecond)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	web := Web{client: http.Client{Timeout: 15 * time.Millisecond}}
	request, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	if _, err := web.do(request); err == nil {
		t.Fatal("expected first request to time out")
	}

	request, _ = http.NewRequest(http.MethodGet, server.URL, nil)
	response, err := web.do(request)
	if err != nil {
		t.Fatalf("subsequent request failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("status=%d", response.StatusCode)
	}
}

func TestWebAppliesDefaultTimeoutWithoutChangingDefaultClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	defer server.Close()
	web := Web{}
	request, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	if web.client.Timeout != 0 {
		t.Fatal("unexpected initial timeout")
	}
	response, err := web.do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if web.client.Timeout != 60*time.Second {
		t.Fatalf("default timeout=%s", web.client.Timeout)
	}
}
