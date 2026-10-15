package main

import (
	"context"
	"encoding/json"
	"fmt"

	"opencode-gui/internal/daemon"
)

// startDaemon hosts the phone-facing WebSocket server and registers the
// methods from docs/PROTOCOL.md.
func (st *state) startDaemon(ctx context.Context) {
	token := daemon.NewToken()
	d := daemon.New("0.0.0.0:9680", token)
	st.daemon = d

	d.Handle("system.health", func(ctx context.Context, _ json.RawMessage) (any, error) {
		return map[string]any{"ok": true, "port": st.sessions.Port()}, nil
	})
	d.Handle("session.list", func(ctx context.Context, _ json.RawMessage) (any, error) {
		return st.sessions.List(ctx)
	})
	d.Handle("session.create", func(ctx context.Context, p json.RawMessage) (any, error) {
		var body struct {
			Title string `json:"title"`
		}
		_ = json.Unmarshal(p, &body)
		return st.sessions.New(ctx, body.Title)
	})
	d.Handle("session.messages", func(ctx context.Context, p json.RawMessage) (any, error) {
		var body struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(p, &body); err != nil {
			return nil, err
		}
		return st.sessions.Messages(ctx, body.ID)
	})
	d.Handle("session.prompt", func(ctx context.Context, p json.RawMessage) (any, error) {
		var body struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(p, &body); err != nil {
			return nil, err
		}
		return true, st.sessions.Prompt(ctx, body.ID, body.Text)
	})
	d.Handle("session.interrupt", func(ctx context.Context, p json.RawMessage) (any, error) {
		var body struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(p, &body); err != nil {
			return nil, err
		}
		return true, st.sessions.Interrupt(ctx, body.ID)
	})
	d.Handle("session.diff", func(ctx context.Context, p json.RawMessage) (any, error) {
		var body struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(p, &body); err != nil {
			return nil, err
		}
		return st.sessions.Diff(ctx, body.ID)
	})

	go func() {
		if err := d.ListenAndServe(ctx); err != nil {
			daemon.Logf("daemon: %v", err)
		}
	}()
	st.daemonURL = st.daemonURLFor(token)
}

// daemonURLFor builds the URL a phone should use, preferring Tailscale.
func (st *state) daemonURLFor(token string) string {
	host := daemon.TailscaleIPv4()
	if host == "" {
		host = daemon.TailscaleName()
	}
	if host == "" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("ws://%s:9680/ws?token=%s", host, token)
}
