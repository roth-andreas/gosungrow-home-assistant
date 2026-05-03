package api

import (
	"strings"
	"testing"
)

func TestRequestCommonStringRedactsToken(t *testing.T) {
	const token = "secret-token-that-must-not-appear"

	text := (RequestCommon{
		Appkey:    "app-key",
		Lang:      "_en_US",
		SysCode:   "200",
		Token:     token,
		UserId:    "123",
		ValidFlag: "1,3",
	}).String()

	if strings.Contains(text, token) {
		t.Fatalf("expected token to be redacted, got:\n%s", text)
	}
	if !strings.Contains(text, "Token:\t<redacted>") {
		t.Fatalf("expected redacted token marker, got:\n%s", text)
	}
}

func TestRedactSecretLeavesEmptyValuesEmpty(t *testing.T) {
	if got := RedactSecret(" "); got != "" {
		t.Fatalf("expected empty secret to stay empty, got %q", got)
	}
}
