package opencode

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// Event is one server-sent event from GET /api/event.
type Event struct {
	Name string          // the SSE "event:" field
	Data json.RawMessage // the SSE "data:" field (JSON)
}

// Subscribe streams events until ctx is canceled. The channel is closed
// when the stream ends; callers should reconnect (subscriptions are
// live-only, per the client docs).
func (s *Server) Subscribe(ctx context.Context) (<-chan Event, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.base+"/api/event", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.SetBasicAuth("opencode", s.pass)

	client := &http.Client{} // no timeout: this stream is long-lived
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, errStatus("GET /api/event", resp.StatusCode)
	}

	out := make(chan Event, 64)
	go func() {
		defer close(out)
		defer resp.Body.Close()
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		var name string
		for sc.Scan() {
			line := sc.Text()
			switch {
			case strings.HasPrefix(line, "event:"):
				name = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			case strings.HasPrefix(line, "data:"):
				data := json.RawMessage(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
				select {
				case out <- Event{Name: name, Data: data}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

// RefreshLoop subscribes, and whenever an event mentions sessionID, calls
// onChange (debounced) so the caller can re-read the session. Reconnects
// with backoff when the stream drops. Runs until ctx is canceled.
func (s *Server) RefreshLoop(ctx context.Context, sessionID string, onChange func()) {
	delay := time.Second
	for ctx.Err() == nil {
		events, err := s.Subscribe(ctx)
		if err != nil {
			time.Sleep(delay)
			if delay < 15*time.Second {
				delay *= 2
			}
			continue
		}
		delay = time.Second
		for e := range events {
			if sessionID != "" && !strings.Contains(string(e.Data), sessionID) {
				continue
			}
			onChange()
		}
	}
}
