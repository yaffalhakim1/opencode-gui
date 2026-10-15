// Package opencode manages the opencode server child process and speaks
// its HTTP API.
package opencode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

const (
	startPort  = 4096
	healthPoll = 250 * time.Millisecond
	healthWait = 60 * time.Second
)

// Server is one running opencode server.
type Server struct {
	base string // e.g. http://127.0.0.1:4096
	pass string
	http *http.Client
}

// Attach connects to an already registered server.
func Attach(ep Endpoint) *Server {
	return &Server{
		base: ep.URL,
		pass: ep.Password,
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

// Port returns the server's TCP port.
func (s *Server) Port() int {
	return portOf(s.base)
}

// Get calls a GET endpoint and decodes JSON into out (nil to ignore body).
func (s *Server) Get(ctx context.Context, path string, out any) error {
	return s.do(ctx, http.MethodGet, path, nil, out)
}

// Post calls a POST endpoint with a JSON body and decodes the response.
func (s *Server) Post(ctx context.Context, path string, body, out any) error {
	return s.do(ctx, http.MethodPost, path, body, out)
}

func (s *Server) do(ctx context.Context, method, path string, body, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.base+path, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("opencode", s.pass)
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s %s: %s %s", method, path, resp.Status, bytes.TrimSpace(msg))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// Healthy reports whether the server answers GET /api/info.
func (s *Server) Healthy(ctx context.Context) bool {
	return s.Get(ctx, "/api/info", nil) == nil
}

// WaitHealthy polls GET /api/info until it answers or the deadline passes.
func (s *Server) WaitHealthy(ctx context.Context) error {
	deadline := time.Now().Add(healthWait)
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if s.Healthy(ctx) {
			return nil
		}
		time.Sleep(healthPoll)
	}
	return fmt.Errorf("opencode at %s did not become healthy", s.base)
}

func portOf(base string) int {
	u := base
	for i := len(u) - 1; i >= 0; i-- {
		if u[i] == ':' {
			var p int
			fmt.Sscanf(u[i+1:], "%d", &p)
			return p
		}
	}
	return 0
}

// freePort reports whether a local port can be bound.
func freePort(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}
