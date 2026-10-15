// Package opencode talks to a running opencode server, following the v2
// service model: the server registers itself in service.json with its URL
// and password, and clients authenticate with Basic opencode:<password>.
package opencode

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Endpoint is a registered opencode server: its base URL and password.
// See https://opencode.ai/v2/docs/build/client/ (Service.discover).
type Endpoint struct {
	URL      string `json:"url"`
	Password string `json:"password"`
	PID      int    `json:"pid"`
	Version  string `json:"version"`
}

// serviceFile is where opencode registers a running server:
// $XDG_STATE_HOME/opencode/service.json, or ~/.local/state/opencode/.
func serviceFile() (string, error) {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "opencode", "service.json"), nil
}

// Discover returns the registered server, or an error when none is
// registered (no server has been started yet).
func Discover() (Endpoint, error) {
	path, err := serviceFile()
	if err != nil {
		return Endpoint{}, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return Endpoint{}, fmt.Errorf("no registered opencode service: %w", err)
	}
	var ep Endpoint
	if err := json.Unmarshal(b, &ep); err != nil {
		return Endpoint{}, fmt.Errorf("service.json malformed: %w", err)
	}
	if ep.URL == "" || ep.Password == "" {
		return Endpoint{}, errors.New("service.json missing url or password")
	}
	return ep, nil
}
