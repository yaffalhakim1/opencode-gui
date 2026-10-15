package opencode

import "context"

// Skill is one agent skill, per Skill.Info.
type Skill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Autoinvoke  bool   `json:"autoinvoke"`
	Path        string `json:"path"`
	Content     string `json:"content"`
}

// Skills lists the skills the server knows.
func (s *Server) Skills(ctx context.Context) ([]Skill, error) {
	var out envelope[[]Skill]
	err := s.Get(ctx, "/api/skill", &out)
	return out.Data, err
}
