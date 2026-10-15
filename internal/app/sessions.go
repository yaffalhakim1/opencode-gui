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

// Prompt admits a message to a session. The reply arrives asynchronously;
// the SSE bridge refreshes it into the UI.
func (s *Sessions) Prompt(ctx context.Context, sessionID string, in opencode.PromptInput) error {
	srv, err := s.server()
	if err != nil {
		return err
	}
	return srv.Prompt(ctx, sessionID, in)
}

// Models lists the models the server offers.
func (s *Sessions) Models(ctx context.Context) ([]opencode.Model, error) {
	srv, err := s.server()
	if err != nil {
		return nil, err
	}
	return srv.Models(ctx)
}

// SwitchModel changes a session's model.
func (s *Sessions) SwitchModel(ctx context.Context, sessionID string, m opencode.ModelRef) error {
	srv, err := s.server()
	if err != nil {
		return err
	}
	return srv.SwitchModel(ctx, sessionID, m)
}

// SwitchAgent changes a session's agent.
func (s *Sessions) SwitchAgent(ctx context.Context, sessionID, agentID string) error {
	srv, err := s.server()
	if err != nil {
		return err
	}
	return srv.SwitchAgent(ctx, sessionID, agentID)
}

// ListDir lists a workspace directory.
func (s *Sessions) ListDir(ctx context.Context, path string) ([]opencode.FSEntry, error) {
	srv, err := s.server()
	if err != nil {
		return nil, err
	}
	return srv.ListDir(ctx, path)
}

// Permissions lists the tool calls waiting for an answer.
func (s *Sessions) Permissions(ctx context.Context) ([]opencode.PermissionRequest, error) {
	srv, err := s.server()
	if err != nil {
		return nil, err
	}
	return srv.Permissions(ctx)
}

// ReplyPermission answers a pending permission request.
func (s *Sessions) ReplyPermission(ctx context.Context, sessionID, requestID string, d opencode.PermissionDecision) error {
	srv, err := s.server()
	if err != nil {
		return err
	}
	return srv.ReplyPermission(ctx, sessionID, requestID, d)
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

// Skills lists the agent skills.
func (s *Sessions) Skills(ctx context.Context) ([]opencode.Skill, error) {
	srv, err := s.server()
	if err != nil {
		return nil, err
	}
	return srv.Skills(ctx)
}

// Usage returns aggregate activity for a range.
func (s *Sessions) Usage(ctx context.Context, from, to int64) (opencode.Stats, error) {
	srv, err := s.server()
	if err != nil {
		return opencode.Stats{}, err
	}
	return srv.Usage(ctx, from, to)
}

// Agents lists the registered agents.
func (s *Sessions) Agents(ctx context.Context) ([]opencode.Agent, error) {
	srv, err := s.server()
	if err != nil {
		return nil, err
	}
	return srv.Agents(ctx)
}

// Diff returns a session's file changes.
func (s *Sessions) Diff(ctx context.Context, sessionID string) ([]opencode.FileDiff, error) {
	srv, err := s.server()
	if err != nil {
		return nil, err
	}
	return srv.Diff(ctx, sessionID)
}

// Watch streams events for a session, calling onChange (already debounced
// upstream) so the caller can re-read state. Runs until ctx is canceled.
func (s *Sessions) Watch(ctx context.Context, sessionID string, onChange func()) {
	srv, err := s.server()
	if err != nil {
		return
	}
	srv.RefreshLoop(ctx, sessionID, onChange)
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
