// Package app holds services the UI calls directly: session management
// against the registered opencode server.
package app

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"opencode-gui/internal/opencode"
)

// Sessions talks to the opencode server and tracks the active session.
type Sessions struct {
	mu       sync.Mutex
	srv      *opencode.Server
	activeID string
}

// Start attaches to the registered opencode server. It fails when no
// server is registered — start one with `opencode serve --port 4096`.
func (s *Sessions) Start(ctx context.Context) error {
	ep, err := opencode.Discover()
	if err != nil {
		return fmt.Errorf("start opencode serve first, or run opencode with a server: %w", err)
	}
	srv := opencode.Attach(ep)
	if err := srv.WaitHealthy(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	s.srv = srv
	s.mu.Unlock()
	return nil
}

// Port reports the opencode server port, 0 if not attached.
func (s *Sessions) Port() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.srv == nil {
		return 0
	}
	return s.srv.Port()
}

// List returns all sessions, creating one if none exist.
func (s *Sessions) List(ctx context.Context) ([]opencode.Session, error) {
	srv, err := s.server()
	if err != nil {
		return nil, err
	}
	sessions, err := srv.ListSessions(ctx)
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		sess, err := srv.CreateSession(ctx, "New session")
		if err != nil {
			return nil, err
		}
		sessions = []opencode.Session{sess}
		s.setActive(sess.ID)
	}
	return sessions, nil
}

// New creates a session and makes it active.
func (s *Sessions) New(ctx context.Context, title string) (opencode.Session, error) {
	srv, err := s.server()
	if err != nil {
		return opencode.Session{}, err
	}
	sess, err := srv.CreateSession(ctx, title)
	if err != nil {
		return opencode.Session{}, err
	}
	s.setActive(sess.ID)
	return sess, nil
}

// Prompt admits text to a session. The reply arrives asynchronously; poll
// Messages for it (the SSE bridge replaces polling on day 2).
func (s *Sessions) Prompt(ctx context.Context, sessionID, text string) error {
	srv, err := s.server()
	if err != nil {
		return err
	}
	return srv.Prompt(ctx, sessionID, text)
}

// Interrupt stops a running session.
func (s *Sessions) Interrupt(ctx context.Context, sessionID string) error {
	srv, err := s.server()
	if err != nil {
		return err
	}
	return srv.Interrupt(ctx, sessionID)
}

// Messages returns a session's messages with their parts.
func (s *Sessions) Messages(ctx context.Context, sessionID string) ([]opencode.MessagePair, error) {
	srv, err := s.server()
	if err != nil {
		return nil, err
	}
	return srv.Messages(ctx, sessionID)
}

// Active returns the active session ID.
func (s *Sessions) Active() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeID
}

// SetActive selects a session.
func (s *Sessions) SetActive(id string) { s.setActive(id) }

func (s *Sessions) setActive(id string) {
	s.mu.Lock()
	s.activeID = id
	s.mu.Unlock()
}

func (s *Sessions) server() (*opencode.Server, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.srv == nil {
		return nil, errors.New("not attached to an opencode server")
	}
	return s.srv, nil
}
