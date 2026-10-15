package opencode

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverReadsServiceFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	sub := filepath.Join(dir, "opencode")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"id":"x","version":"2.0.28","url":"http://127.0.0.1:49374","pid":1,"password":"secret"}`
	if err := os.WriteFile(filepath.Join(sub, "service.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	ep, err := Discover()
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if ep.URL != "http://127.0.0.1:49374" || ep.Password != "secret" {
		t.Errorf("got %+v", ep)
	}
	if got := Attach(ep).Port(); got != 49374 {
		t.Errorf("Port() = %d, want 49374", got)
	}
}

func TestDiscoverFailsWithoutService(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	if _, err := Discover(); err == nil {
		t.Error("expected an error when no service is registered")
	}
}

func TestDiscoverRejectsIncompleteFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)
	sub := filepath.Join(dir, "opencode")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "service.json"), []byte(`{"url":"http://x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Discover(); err == nil {
		t.Error("expected an error for a service file with no password")
	}
}
