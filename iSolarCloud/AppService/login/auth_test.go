package login

import (
	"testing"

	"github.com/roth-andreas/gosungrow-home-assistant/iSolarCloud/api"
)

func TestReadTokenFileMissingIsNotFatal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	endpoint := Init(api.Web{})
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
