package login

import (
	"os"
	"path/filepath"
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

func TestReadTokenFileCorruptIsNotFatal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	tokenPath := filepath.Join(home, ".GoSungrow", "AppService_login.json")
	if err := os.MkdirAll(filepath.Dir(tokenPath), 0700); err != nil {
		t.Fatalf("failed to create token dir: %v", err)
	}
	if err := os.WriteFile(tokenPath, []byte(""), 0600); err != nil {
		t.Fatalf("failed to write corrupt token file: %v", err)
	}

	endpoint := Init(api.Web{})
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
