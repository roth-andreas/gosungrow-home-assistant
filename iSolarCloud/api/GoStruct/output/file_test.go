package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestIsJSONSyntaxError(t *testing.T) {
	if !IsJSONSyntaxError(&json.SyntaxError{}) {
		t.Fatal("expected JSON syntax errors to be detected")
	}
	if !IsJSONSyntaxError(json.Unmarshal([]byte(""), &map[string]string{})) {
		t.Fatal("expected empty JSON input to be detected")
	}
}

func TestPlainFileWriteReplacesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")
	if err := os.WriteFile(path, []byte(`{"old":true}`), 0600); err != nil {
		t.Fatalf("failed to seed file: %v", err)
	}

	if err := PlainFileWrite(path, []byte(`{"new":true}`), 0600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(got) != `{"new":true}` {
		t.Fatalf("unexpected file content: %q", got)
	}
}
