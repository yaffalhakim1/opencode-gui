package opencode

import "context"

// ModelRef identifies a model for a session, per Model.Ref.
type ModelRef struct {
	ID         string `json:"id"`
	ProviderID string `json:"providerID"`
	Variant    string `json:"variant,omitempty"`
}

// ModelVariant is one reasoning-effort variant of a model.
type ModelVariant struct {
	ID       string `json:"id"`
	Settings struct {
		ReasoningEffort string `json:"reasoningEffort"`
	} `json:"settings"`
}

// Model is one model the server offers, per Model.Info.
type Model struct {
	ID           string         `json:"id"`
	ModelID      string         `json:"modelID"`
	ProviderID   string         `json:"providerID"`
	Name         string         `json:"name"`
	Family       string         `json:"family"`
	Status       string         `json:"status"`
	Enabled      bool           `json:"enabled"`
	Variants     []ModelVariant `json:"variants"`
	Capabilities struct {
		Tools  bool     `json:"tools"`
		Input  []string `json:"input"`
		Output []string `json:"output"`
	} `json:"capabilities"`
}

// Models lists every model the server offers.
func (s *Server) Models(ctx context.Context) ([]Model, error) {
	var out envelope[[]Model]
	err := s.Get(ctx, "/api/model", &out)
	return out.Data, err
}

// SwitchModel changes the model a session's later turns use.
func (s *Server) SwitchModel(ctx context.Context, sessionID string, m ModelRef) error {
	var out any
	return s.Post(ctx, "/api/session/"+sessionID+"/model", map[string]any{"model": m}, &out)
}

// SwitchAgent changes the agent a session's later turns use.
func (s *Server) SwitchAgent(ctx context.Context, sessionID, agentID string) error {
	var out any
	return s.Post(ctx, "/api/session/"+sessionID+"/agent", map[string]any{"agent": agentID}, &out)
}
