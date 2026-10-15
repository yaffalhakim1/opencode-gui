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
	Time  struct {
		Created float64 `json:"created"`
		Updated float64 `json:"updated"`
		Idle    float64 `json:"idle"`
	} `json:"time"`
}

// LastActivity is when the session was last touched, in epoch milliseconds.
func (s Session) LastActivity() float64 {
	if s.Time.Updated > 0 {
		return s.Time.Updated
	}
	return s.Time.Created
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

// FileAttachment is one file sent with a prompt, per
// PromptInput.FileAttachment.
type FileAttachment struct {
	URI         string `json:"uri"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// Delivery says how a prompt reaches a running session: steer delivers it at
// the next step boundary, queue holds it until the turn ends.
type Delivery string

const (
	DeliverySteer Delivery = "steer"
	DeliveryQueue Delivery = "queue"
)

// PromptInput is one message admitted to a session.
type PromptInput struct {
	Text     string
	Agent    string
	Files    []FileAttachment
	Delivery Delivery
	Resume   *bool
}

// Prompt admits a message to a session. v2 prompts are async: the reply
// arrives through the session's messages, not as this call's result.
func (s *Server) Prompt(ctx context.Context, sessionID string, in PromptInput) error {
	body := map[string]any{"text": in.Text}
	if in.Agent != "" {
		body["agents"] = []string{in.Agent}
	}
	if len(in.Files) > 0 {
		body["files"] = in.Files
	}
	if in.Delivery != "" {
		body["delivery"] = string(in.Delivery)
	}
	if in.Resume != nil {
		body["resume"] = *in.Resume
	}
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
