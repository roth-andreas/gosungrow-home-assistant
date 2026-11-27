package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRequestTimestampMillisDiagnosticOffset(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		offset time.Duration
	}{
		{name: "valid trimmed value", value: " 1500 ", offset: 1500 * time.Millisecond},
		{name: "empty value", value: "", offset: 0},
		{name: "invalid value", value: "not-a-number", offset: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("GOSUNGROW_TIMESTAMP_OFFSET_MS", test.value)
			web := Web{timeOffset: 2 * time.Second}
			before := time.Now().Add(web.timeOffset + test.offset).UnixMilli()
			got := web.requestTimestampMillis()
			after := time.Now().Add(web.timeOffset + test.offset).UnixMilli()
			if got < before || got > after {
				t.Fatalf("timestamp=%d, want range [%d,%d]", got, before, after)
			}
		})
	}
}

func TestCurrentGMTHeaderUsesCompatibilityFormat(t *testing.T) {
	_, offsetSeconds := time.Now().Zone()
	hours := offsetSeconds / 3600
	want := fmt.Sprintf("GMT%%2B%d", hours)
	if hours < 0 {
		want = fmt.Sprintf("GMT-%d", -hours)
	}
	if got := currentGMTHeader(); got != want {
		t.Fatalf("currentGMTHeader()=%q, want %q", got, want)
	}
}

type denialTestEndpoint struct {
	EndPoint
	request RequestCommon
}

func (e denialTestEndpoint) GetArea() AreaName     { return "AppService" }
func (e denialTestEndpoint) GetName() EndPointName { return "getPsList" }
func (e denialTestEndpoint) GetUrl() EndPointUrl {
	return SetUrl("/v1/powerStationService/getPsList?private_query=private_value")
}
func (e denialTestEndpoint) IsRequestValid() error   { return nil }
func (e denialTestEndpoint) RequestRef() interface{} { return e.request }

func TestHTTP403EvidenceContainsOnlySafeTransportMetadata(t *testing.T) {
	t.Setenv("GOSUNGROW_TRACE_HTTP", "")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Request-ID", "private_request_id")
		http.Error(w, "private_response_body", http.StatusForbidden)
	}))
	defer server.Close()

	tests := []struct {
		name    string
		request RequestCommon
		want    SessionOrigin
	}{
		{"no session", RequestCommon{}, SessionNone},
		{"unknown session", RequestCommon{Token: "private_token"}, SessionUnknown},
		{"cached session", RequestCommon{Token: "private_token"}.WithSessionEvidence(SessionCached, false), SessionCached},
		{"fresh session", RequestCommon{Token: "private_token"}.WithSessionEvidence(SessionFresh, false), SessionFresh},
		{"candidate session", RequestCommon{Token: "private_token"}.WithSessionEvidence(SessionCandidate, true), SessionCandidate},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var records []ForbiddenResponse
			web := Web{}.WithForbiddenResponseHandler(func(record ForbiddenResponse) { records = append(records, record) })
			if err := web.SetUrl(server.URL); err != nil {
				t.Fatal(err)
			}
			if _, err := web.getApi(denialTestEndpoint{request: test.request}); !errors.Is(err, ErrHTTPForbidden) {
				t.Fatalf("HTTP 403 returned %v, want ErrHTTPForbidden", err)
			}
			if len(records) != 1 {
				t.Fatalf("got %d denial records, want one", len(records))
			}
			record := records[0]
			if record.Gateway != "custom" || record.Path != "/v1/powerStationService/getPsList" ||
				record.Endpoint != "AppService.getPsList" || record.Status != 403 || record.At.Location() != time.UTC ||
				record.SessionOrigin != test.want || strings.Contains(fmt.Sprint(record), "private_") {
				t.Fatalf("unexpected denial record: %+v", record)
			}
		})
	}
}
