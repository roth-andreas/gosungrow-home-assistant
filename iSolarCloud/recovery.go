package iSolarCloud

import (
	"errors"
	"strconv"
	"strings"

	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/AppService/login"
	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api"
)

const (
	OldLoginAppKey    = "A5C22A880B97303FCB902069C6B042AB"
	LegacyLoginAppKey = "93D72E60331ABDCDC7B39ADC2D1F32B3"
)

type LoginAttempt struct {
	Host   string
	AppKey string
}

type LoginAttemptFailure struct {
	Attempt LoginAttempt
	Err     error
}

// FailureClass is the stable recovery category carried across process boundaries.
type FailureClass string

const (
	// FailureClassRecoverableRemote selects login refresh and normal remote retry.
	FailureClassRecoverableRemote FailureClass = "recoverable_remote"
	// FailureClassDockerDNS selects Docker resolver backoff without login refresh.
	FailureClassDockerDNS FailureClass = "docker_dns"
	// FailureClassNonRecoverable stops the app wrapper retry loop.
	FailureClassNonRecoverable FailureClass = "non_recoverable"
	// FailureClassOperatorActionRequired stops automated recovery until a user repairs a safe local condition.
	FailureClassOperatorActionRequired FailureClass = "operator_action_required"
)

type classifiedFailure interface {
	FailureClass() FailureClass
}

type loginAttemptSequenceError struct {
	class   FailureClass
	message string
	cause   error
}

type endpointFailure struct {
	api.EndPoint
	err error
}

// Local persistence failure cannot be repaired by gateway rotation, even when
// its text contains a timeout or a previous authentication diagnostic.
type sessionPersistenceError struct{ cause error }

func (e *sessionPersistenceError) Error() string              { return e.cause.Error() }
func (e *sessionPersistenceError) Unwrap() error              { return e.cause }
func (e *sessionPersistenceError) FailureClass() FailureClass { return FailureClassNonRecoverable }

var _ api.EndPoint = endpointFailure{}

func (e endpointFailure) GetError() error {
	return e.err
}

func (e endpointFailure) IsError() bool {
	return e.err != nil
}

func (e *loginAttemptSequenceError) Error() string {
	return e.message
}

func (e *loginAttemptSequenceError) Unwrap() error {
	return e.cause
}

func (e *loginAttemptSequenceError) FailureClass() FailureClass {
	return e.class
}

func NormalizeLoginAppKey(appKey string) string {
	appKey = strings.TrimSpace(appKey)
	if appKey == "" || appKey == LegacyLoginAppKey {
		return DefaultApiAppKey
	}
	return appKey
}

func appendUniqueLoginAttempt(list []LoginAttempt, item LoginAttempt) []LoginAttempt {
	host := strings.TrimSpace(item.Host)
	key := strings.TrimSpace(item.AppKey)
	if host == "" || key == "" {
		return list
	}
	for _, existing := range list {
		if existing.Host == host && existing.AppKey == key {
			return list
		}
	}
	return append(list, LoginAttempt{Host: host, AppKey: key})
}

func BuildLoginAttempts(host string, appKey string) []LoginAttempt {
	return buildSessionAttempts(host, appKey, "", "")
}

func buildSessionAttempts(host, appKey, successfulHost, successfulKey string) []LoginAttempt {
	candidates := make([]LoginAttempt, 0)
	hosts := []string{
		successfulHost,
		host,
		DefaultHost,
		"https://gateway.isolarcloud.com",
		"https://gateway.isolarcloud.eu",
		"https://gateway.isolarcloud.com.hk",
		"https://gateway.isolarcloud.com.cn",
		"https://gateway.isolarcloud.in",
	}
	appKeys := []string{
		successfulKey,
		NormalizeLoginAppKey(appKey),
		DefaultApiAppKey,
		OldLoginAppKey,
		LegacyLoginAppKey,
	}
	for _, host := range hosts {
		for _, appKey := range appKeys {
			candidates = appendUniqueLoginAttempt(candidates, LoginAttempt{Host: host, AppKey: appKey})
		}
	}
	return candidates
}

func hasRecoverableGatewayMessage(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "login_state=-1") ||
		strings.Contains(msg, "login rejected by gateway") ||
		strings.Contains(msg, "appkey is incorrect") ||
		strings.Contains(msg, "need to login again") ||
		strings.Contains(msg, "er_token_login_invalid") ||
		strings.Contains(msg, "cannot login") ||
		strings.Contains(msg, "api httpresponse is 5") ||
		strings.Contains(msg, "internal server error") ||
		strings.Contains(msg, "bad gateway") ||
		strings.Contains(msg, "service unavailable") ||
		strings.Contains(msg, "gateway timeout") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "temporary failure in name resolution") ||
		strings.Contains(msg, "server misbehaving") ||
		strings.Contains(msg, "network is unreachable") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "context deadline exceeded") ||
		strings.Contains(msg, "i/o timeout")
}

func hasDockerDNSMessage(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "127.0.0.11:53") &&
		(strings.Contains(msg, "no such host") ||
			strings.Contains(msg, "temporary failure in name resolution") ||
			strings.Contains(msg, "server misbehaving"))
}

// ClassifyFailure returns the stable recovery category for err.
func ClassifyFailure(err error) FailureClass {
	if err == nil {
		return ""
	}

	var classified classifiedFailure
	if errors.As(err, &classified) {
		return classified.FailureClass()
	}
	if hasDockerDNSMessage(err) {
		return FailureClassDockerDNS
	}
	if hasRecoverableGatewayMessage(err) {
		return FailureClassRecoverableRemote
	}
	return FailureClassNonRecoverable
}

func ShouldRecoverGatewayError(err error) bool {
	class := ClassifyFailure(err)
	return class == FailureClassRecoverableRemote || class == FailureClassDockerDNS
}

func IsDockerDNSError(err error) bool {
	return ClassifyFailure(err) == FailureClassDockerDNS
}

func ShouldTryNextLoginAttempt(err error) bool {
	return ShouldRecoverGatewayError(err) && !IsDockerDNSError(err)
}

func SummarizeLoginAttemptFailures(failures []LoginAttemptFailure, secrets ...string) error {
	if len(failures) == 0 {
		return nil
	}
	if len(failures) == 1 {
		return failures[0].Err
	}

	type hostSummary struct {
		Attempts int
		Messages []string
	}

	order := make([]string, 0, len(failures))
	summaries := make(map[string]*hostSummary, len(failures))
	for _, failure := range failures {
		host := strings.TrimSpace(failure.Attempt.Host)
		if host == "" {
			host = "<unknown-host>"
		}
		summary, ok := summaries[host]
		if !ok {
			summary = &hostSummary{}
			summaries[host] = summary
			order = append(order, host)
		}
		summary.Attempts++

		msg := ""
		if failure.Err != nil {
			msg = redactFailureMessage(strings.TrimSpace(failure.Err.Error()), secrets)
		}
		if msg == "" {
			msg = "unknown error"
		}
		duplicate := false
		for _, existing := range summary.Messages {
			if existing == msg {
				duplicate = true
				break
			}
		}
		if !duplicate && len(summary.Messages) < 2 {
			summary.Messages = append(summary.Messages, msg)
		}
	}

	terminalFailure := failures[len(failures)-1]
	terminalHost := strings.TrimSpace(terminalFailure.Attempt.Host)
	if terminalHost == "" {
		terminalHost = "<unknown-host>"
	}
	terminalMessage := "unknown error"
	if terminalFailure.Err != nil && strings.TrimSpace(terminalFailure.Err.Error()) != "" {
		terminalMessage = redactFailureMessage(strings.TrimSpace(terminalFailure.Err.Error()), secrets)
	}
	terminalSummary := summaries[terminalHost]
	terminalRecorded := false
	for _, message := range terminalSummary.Messages {
		if message == terminalMessage {
			terminalRecorded = true
			break
		}
	}
	if !terminalRecorded {
		if len(terminalSummary.Messages) < 2 {
			terminalSummary.Messages = append(terminalSummary.Messages, terminalMessage)
		} else {
			terminalSummary.Messages[len(terminalSummary.Messages)-1] = terminalMessage
		}
	}

	parts := make([]string, 0, len(order))
	for _, host := range order {
		summary := summaries[host]
		message := strings.Join(summary.Messages, " | ")
		if len(summary.Messages) == 0 {
			message = "unknown error"
		}
		parts = append(parts, host+" ("+strconv.Itoa(summary.Attempts)+" attempts): "+message)
	}

	first := failures[0]
	terminal := terminalFailure
	class := ClassifyFailure(terminal.Err)
	if class == FailureClassDockerDNS {
		class = FailureClassRecoverableRemote
	}
	message := "login candidate sequence failed: first failure: " + formatLoginAttemptFailure(first, secrets) +
		"; terminal stop reason: " + formatLoginAttemptFailure(terminal, secrets) +
		"; attempts by host: " + strings.Join(parts, "; ")

	return &loginAttemptSequenceError{
		class:   class,
		message: message,
		cause:   terminal.Err,
	}
}

func formatLoginAttemptFailure(failure LoginAttemptFailure, secrets []string) string {
	host := strings.TrimSpace(failure.Attempt.Host)
	if host == "" {
		host = "<unknown-host>"
	}
	message := "unknown error"
	if failure.Err != nil && strings.TrimSpace(failure.Err.Error()) != "" {
		message = redactFailureMessage(strings.TrimSpace(failure.Err.Error()), secrets)
	}
	return host + ": " + message
}

func redactFailureMessage(message string, secrets []string) string {
	for _, secret := range secrets {
		if secret != "" {
			message = strings.ReplaceAll(message, secret, "<redacted>")
		}
	}
	return message
}

// FinalizeLoginAttemptFailures preserves a single failure and summarizes a sequence.
func FinalizeLoginAttemptFailures(failures []LoginAttemptFailure, fallback error, secrets ...string) error {
	if len(failures) == 0 {
		return fallback
	}
	if len(failures) == 1 {
		return failures[0].Err
	}
	return SummarizeLoginAttemptFailures(failures, secrets...)
}

// AuthenticateSession evaluates isolated candidates and promotes only a persisted
// session. Runtime recovery and CLI login share this boundary.
func (sg *SunGrow) AuthenticateSession(auth login.SunGrowAuth, runtime bool, persist func(*SunGrow) error) error {
	if sg == nil {
		return errors.New("sungrow instance not configured")
	}
	if sg.recovering {
		return sg.Error
	}
	if sg.configuredHost == "" {
		sg.configuredHost = sg.ApiRoot.ServerUrl.String()
		sg.configuredKey = NormalizeLoginAppKey(auth.AppKey)
	}
	attempts := BuildLoginAttempts(sg.configuredHost, sg.configuredKey)
	if runtime {
		auth.Force = true
		attempts = buildSessionAttempts(sg.configuredHost, sg.configuredKey, sg.lastSuccessfulHost, sg.lastSuccessfulKey)
	}
	sg.recovering = true
	defer func() { sg.recovering = false }()
	failures := make([]LoginAttemptFailure, 0, len(attempts))
	secrets := []string{auth.UserAccount, auth.UserPassword, sg.GetToken()}
	for _, attempt := range attempts {
		root := sg.ApiRoot
		if auth.Force {
			root = root.WithoutCache()
		}
		candidate := &SunGrow{ApiRoot: root, Directory: sg.Directory, OutputType: sg.OutputType, SaveAsFile: sg.SaveAsFile, recovering: true}
		candidate.ApiRoot.Error = nil
		candidate.ApiRoot.Body = nil
		err := candidate.ApiRoot.SetUrl(attempt.Host)
		if err == nil {
			err = candidate.Init()
		}
		candidateAuth := auth
		candidateAuth.AppKey = attempt.AppKey
		if err == nil {
			err = candidate.authenticate(candidateAuth)
		}
		if err == nil {
			// Persistence errors are fatal and never trigger another candidate.
			if err = candidate.Auth.Persist(); err == nil && auth.Force && candidate.validationResponse != nil {
				err = candidate.ApiRoot.WebCacheWrite(candidate.validationResponse, []byte(candidate.validationResponse.GetResponseJson()))
			}
			if err == nil && persist != nil {
				err = persist(candidate)
			}
			if err != nil {
				sg.Error = &sessionPersistenceError{cause: err}
				return sg.Error
			}
			candidate.ApiRoot = candidate.ApiRoot.WithCache()
			candidate.Auth.ApiRoot = candidate.ApiRoot
			candidate.Error = nil
			// Rebuild endpoint templates against the promoted, cache-enabled root.
			if err = candidate.Init(); err != nil {
				sg.Error = err
				return err
			}
			candidate.lastSuccessfulHost, candidate.lastSuccessfulKey = attempt.Host, attempt.AppKey
			candidate.configuredHost, candidate.configuredKey = sg.configuredHost, sg.configuredKey
			candidate.persistSession = persist
			candidate.validationResponse = nil
			*sg = *candidate
			return nil
		}
		secrets = append(secrets, candidate.GetToken())
		failures = append(failures, LoginAttemptFailure{Attempt: attempt, Err: err})
		if !ShouldTryNextLoginAttempt(err) {
			break
		}
	}
	sg.Error = FinalizeLoginAttemptFailures(failures, errors.New("no login candidates"), secrets...)
	if runtime && ShouldRecoverGatewayError(sg.Error) {
		sg.NeedLogin = true
	}
	return sg.Error
}

func (sg *SunGrow) recoverGatewaySession(force bool) error {
	if sg == nil || sg.AuthDetails == nil {
		return errors.New("no auth details available for recovery")
	}
	auth := *sg.AuthDetails
	auth.Force = force
	return sg.AuthenticateSession(auth, true, sg.persistSession)
}

func (sg *SunGrow) rebuildEndpointForCurrentGateway(endpoint api.EndPoint) api.EndPoint {
	areaAndName := endpoint.GetArea().String() + "." + endpoint.GetName().String()
	retry := sg.GetEndpoint(areaAndName)
	if sg.Error != nil {
		return retry
	}

	retry = retry.SetCacheTimeout(endpoint.GetCacheTimeout())
	reqJSON := endpoint.GetRequestJson()
	if string(reqJSON) != "" {
		retry = retry.SetRequestByJson(reqJSON)
		if retry.IsError() {
			sg.Error = retry.GetError()
			return retry
		}
	}

	return retry
}

func (sg *SunGrow) callEndpointWithRecovery(endpoint api.EndPoint) api.EndPoint {
	endpoint = endpoint.Call()
	if !endpoint.IsError() {
		sg.Error = nil
		return endpoint
	}

	sg.Error = endpoint.GetError()
	if sg.IsLoggedOut() || sg.recovering || !ShouldRecoverGatewayError(sg.Error) || IsDockerDNSError(sg.Error) || sg.AuthDetails == nil {
		return endpoint
	}

	if err := sg.recoverGatewaySession(true); err != nil {
		return endpointFailure{EndPoint: endpoint, err: err}
	}

	retry := sg.rebuildEndpointForCurrentGateway(endpoint)
	if retry.IsError() {
		return retry
	}

	retry = retry.Call()
	sg.Error = retry.GetError()
	return retry
}
