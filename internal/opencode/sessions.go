package opencode

import "context"

// envelope is the v2 Success wrapper: { location?, data }.
type envelope[T any] struct {
	Data T `json:"data"`
}

// Session is an opencode session.
type Session struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// Message is one session message. v2 messages carry content blocks, not
// parts: {type: "assistant"|"user", content: [{type: "text", text}]}.
// Finish is set on assistant messages once the model stopped.
type Message struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Content []Part `json:"content"`
	Finish  string `json:"finish"`
}

// Part is one content block of a message. Assistant messages may also
// carry reasoning and tool blocks, which the chat view filters out.
type Part struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// MessagePair is one message with a flattened text view.
type MessagePair struct {
	Info  Message
	Parts []Part
}

// CreateSession makes a new session, optionally titled.
func (s *Server) CreateSession(ctx context.Context, title string) (Session, error) {
	body := map[string]any{}
	if title != "" {
		body["title"] = title
	}
	var out envelope[Session]
	err := s.Post(ctx, "/api/session", body, &out)
	return out.Data, err
}

// ListSessions returns all sessions (newest first).
func (s *Server) ListSessions(ctx context.Context) ([]Session, error) {
	var out envelope[[]Session]
	err := s.Get(ctx, "/api/session?order=desc", &out)
	return out.Data, err
}

// Prompt admits text to a session. v2 prompts are async: the reply arrives
// through the session's messages, not as this call's result.
func (s *Server) Prompt(ctx context.Context, sessionID, text string) error {
	body := map[string]any{"text": text}
	var out envelope[any]
	return s.Post(ctx, "/api/session/"+sessionID+"/prompt", body, &out)
}

// Interrupt stops a running session.
func (s *Server) Interrupt(ctx context.Context, sessionID string) error {
	var out any
	return s.Post(ctx, "/api/session/"+sessionID+"/interrupt", map[string]any{}, &out)
}

// Messages lists a session's messages.
func (s *Server) Messages(ctx context.Context, sessionID string) ([]MessagePair, error) {
	var raw envelope[[]Message]
	if err := s.Get(ctx, "/api/session/"+sessionID+"/message", &raw); err != nil {
		return nil, err
	}
	out := make([]MessagePair, len(raw.Data))
	for i, m := range raw.Data {
		out[i] = MessagePair{Info: m, Parts: m.Content}
	}
	return out, nil
}

// Agent is one registered agent, per Agent.Info.
type Agent struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Model *struct {
		ID         string `json:"id"`
		ProviderID string `json:"providerID"`
	} `json:"model"`
}

// Agents lists the registered agents.
func (s *Server) Agents(ctx context.Context) ([]Agent, error) {
	var out envelope[[]Agent]
	err := s.Get(ctx, "/api/agent", &out)
	return out.Data, err
}
