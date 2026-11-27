package api

import (
	"net/http"
	"strings"
	"time"
)

// SessionOrigin describes how the session sent with a request was obtained.
type SessionOrigin string

const (
	SessionCached    SessionOrigin = "cached"
	SessionFresh     SessionOrigin = "fresh"
	SessionCandidate SessionOrigin = "candidate"
	SessionNone      SessionOrigin = "none"
	SessionUnknown   SessionOrigin = "unknown"
)

// WithSessionEvidence attaches diagnostic facts without changing the wire request.
func (req RequestCommon) WithSessionEvidence(origin SessionOrigin, validatingSession bool) RequestCommon {
	req.sessionOrigin = origin
	req.validatingSession = validatingSession
	return req
}

func (req RequestCommon) sessionEvidence() (SessionOrigin, bool) {
	switch req.sessionOrigin {
	case SessionCached, SessionFresh, SessionCandidate:
		return req.sessionOrigin, req.validatingSession
	default:
		return SessionUnknown, req.validatingSession
	}
}

// ForbiddenResponse carries only catalog and transport facts safe for diagnostics.
type ForbiddenResponse struct {
	At             time.Time
	Endpoint       string
	Path           string
	Gateway        string
	Status         int
	SessionOrigin  SessionOrigin
	Authentication bool
}

// WithForbiddenResponseHandler attaches the synchronous observer for HTTP 403 responses.
func (w Web) WithForbiddenResponseHandler(handler func(ForbiddenResponse)) Web {
	w.forbiddenResponse = handler
	return w
}

func (w *Web) reportForbidden(endpoint EndPoint, request *http.Request, hasSession bool) {
	if w.forbiddenResponse == nil {
		return
	}
	if w.httpResponse.Request != nil {
		request = w.httpResponse.Request
	}
	origin := SessionNone
	validating := false
	if hasSession {
		origin = SessionUnknown
		if evidence, ok := endpoint.RequestRef().(interface{ sessionEvidence() (SessionOrigin, bool) }); ok {
			origin, validating = evidence.sessionEvidence()
		}
		if request.Method != http.MethodPost {
			origin = SessionNone
		}
	}
	name := endpoint.GetArea().String() + "." + endpoint.GetName().String()
	w.forbiddenResponse(ForbiddenResponse{
		At:             time.Now().UTC(),
		Endpoint:       name,
		Path:           endpoint.GetUrl().EndPoint.Path,
		Gateway:        diagnosticGateway(request.URL.Hostname()),
		Status:         http.StatusForbidden,
		SessionOrigin:  origin,
		Authentication: validating || name == "AppService.login",
	})
}

func diagnosticGateway(host string) string {
	switch host = strings.ToLower(host); host {
	case "augateway.isolarcloud.com", "gateway.isolarcloud.com", "gateway.isolarcloud.eu",
		"gateway.isolarcloud.com.hk", "gateway.isolarcloud.com.cn", "gateway.isolarcloud.in":
		return host
	default:
		return "custom"
	}
}
