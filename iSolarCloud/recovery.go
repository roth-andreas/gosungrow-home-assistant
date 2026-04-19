package iSolarCloud

import (
	"errors"
	"strings"

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
	candidates := make([]LoginAttempt, 0)
	hosts := []string{
		host,
		DefaultHost,
		"https://gateway.isolarcloud.com",
		"https://gateway.isolarcloud.eu",
		"https://gateway.isolarcloud.com.cn",
	}
	appKeys := []string{
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

func ShouldRecoverGatewayError(err error) bool {
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

func (sg *SunGrow) recoverGatewaySession(force bool) error {
	if sg == nil {
		return errors.New("sungrow instance not configured")
	}
	if sg.AuthDetails == nil {
		return errors.New("no auth details available for recovery")
	}
	if sg.recovering {
		return sg.Error
	}

	cacheDir := sg.ApiRoot.GetCacheDir()
	auth := *sg.AuthDetails
	auth.AppKey = NormalizeLoginAppKey(auth.AppKey)
	auth.Force = force
	attempts := BuildLoginAttempts(sg.ApiRoot.ServerUrl.String(), auth.AppKey)

	var firstRetriableErr error
	var lastErr error
	exhaustedRetriable := true

	sg.recovering = true
	defer func() {
		sg.recovering = false
	}()

	for idx, attempt := range attempts {
		if idx > 0 {
			sg.Logout()
		}

		replacement := NewSunGro(attempt.Host, cacheDir)
		if replacement.Error != nil {
			sg.Error = replacement.Error
			return replacement.Error
		}
		replacement.Directory = sg.Directory
		replacement.OutputType = sg.OutputType
		replacement.SaveAsFile = sg.SaveAsFile
		if err := replacement.Init(); err != nil {
			sg.Error = err
			return err
		}

		sg.ApiRoot = replacement.ApiRoot
		sg.Areas = replacement.Areas
		sg.Auth = replacement.Auth
		sg.Error = nil
		sg.NeedLogin = false

		auth.AppKey = attempt.AppKey
		if err := sg.Login(auth); err == nil {
			sg.Error = nil
			return nil
		} else {
			lastErr = err
			if !ShouldRecoverGatewayError(err) {
				exhaustedRetriable = false
				break
			}
			if firstRetriableErr == nil {
				firstRetriableErr = err
			}
		}
	}

	if exhaustedRetriable && firstRetriableErr != nil {
		sg.Error = firstRetriableErr
		return firstRetriableErr
	}
	if lastErr != nil {
		sg.Error = lastErr
		return lastErr
	}
	return sg.Error
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
	if sg.IsLoggedOut() || sg.recovering || !ShouldRecoverGatewayError(sg.Error) || sg.AuthDetails == nil {
		return endpoint
	}

	if err := sg.recoverGatewaySession(true); err != nil {
		return endpoint.SetError("%s", err)
	}

	retry := sg.rebuildEndpointForCurrentGateway(endpoint)
	if retry.IsError() {
		return retry
	}

	retry = retry.Call()
	sg.Error = retry.GetError()
	return retry
}
