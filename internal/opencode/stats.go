package opencode

import (
	"context"
	"fmt"
)

// TokenUsage is how many tokens a range used.
type TokenUsage struct {
	Input     float64 `json:"input"`
	Output    float64 `json:"output"`
	Reasoning float64 `json:"reasoning"`
	Cache     struct {
		Read  float64 `json:"read"`
		Write float64 `json:"write"`
	} `json:"cache"`
}

// ModelUsage is one model's share of a range's activity.
type ModelUsage struct {
	Model  ModelRef   `json:"model"`
	Steps  int        `json:"steps"`
	Tokens TokenUsage `json:"tokens"`
	Cost   float64    `json:"cost"`
}

// Stats is aggregate session activity for a time range, per
// SessionStats.Info. Only the fields the UI shows are decoded.
// Range.From and Range.To are epoch milliseconds, as the server sends them.
type Stats struct {
	Range struct {
		From float64 `json:"from"`
		To   float64 `json:"to"`
	} `json:"range"`
	Sessions   int          `json:"sessions"`
	Subagents  int          `json:"subagents"`
	Prompts    int          `json:"prompts"`
	Steps      int          `json:"steps"`
	Tokens     TokenUsage   `json:"tokens"`
	Cost       float64      `json:"cost"`
	ActiveDays int          `json:"activeDays"`
	Streak     int          `json:"streak"`
	Models     []ModelUsage `json:"models"`
	Activity   []Activity   `json:"activity"`
}

// Activity is one day's step count, per SessionStats.Activity.
type Activity struct {
	Date  string `json:"date"`
	Steps int    `json:"steps"`
}

// Usage returns aggregate activity between two epoch-millisecond bounds.
// A zero bound is left out, so the server uses its own range.
func (s *Server) Usage(ctx context.Context, from, to int64) (Stats, error) {
	var out envelope[Stats]
	q := "/api/experimental/session/stats?tools=none"
	if from > 0 {
		q += fmt.Sprintf("&from=%d", from)
	}
	if to > 0 {
		q += fmt.Sprintf("&to=%d", to)
	}
	err := s.Get(ctx, q, &out)
	return out.Data, err
}
