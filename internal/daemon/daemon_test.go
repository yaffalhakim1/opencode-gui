package daemon

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// startTestDaemon runs a daemon on a test port and returns a cancel.
func startTestDaemon(t *testing.T, d *Daemon) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go d.ListenAndServe(ctx) //nolint:errcheck
	// Wait for the listener.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://" + d.Addr() + "/health")
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("daemon did not start")
}

func TestRejectsBadToken(t *testing.T) {
	d := New("127.0.0.1:19681", "good-token")
	startTestDaemon(t, d)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _, err := websocket.Dial(ctx, "ws://127.0.0.1:19681/ws?token=wrong", nil)
	if err == nil {
		t.Fatal("a bad token was accepted")
	}
}

func TestCallsMethodAndGreetsOnConnect(t *testing.T) {
	d := New("127.0.0.1:19682", "tok")
	d.Handle("system.health", func(ctx context.Context, _ json.RawMessage) (any, error) {
		return map[string]any{"ok": true}, nil
	})
	startTestDaemon(t, d)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, "ws://127.0.0.1:19682/ws?token=tok", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer ws.Close(websocket.StatusNormalClosure, "")

	// The daemon greets with daemon.hello.
	_, first, err := ws.Read(ctx)
	if err != nil {
		t.Fatalf("read hello: %v", err)
	}
	var hello struct {
		Event string `json:"event"`
	}
	json.Unmarshal(first, &hello) //nolint:errcheck
	if hello.Event != "daemon.hello" {
		t.Errorf("first message = %q, want daemon.hello", string(first))
	}

	// A method call answers with the same id.
	if err := ws.Write(ctx, websocket.MessageText, []byte(`{"id":7,"method":"system.health","params":{}}`)); err != nil {
		t.Fatalf("write: %v", err)
	}
	for {
		_, data, err := ws.Read(ctx)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var m map[string]any
		json.Unmarshal(data, &m) //nolint:errcheck
		if m["id"] == nil {
			continue // an event
		}
		if m["id"].(float64) != 7 {
			t.Fatalf("id = %v, want 7", m["id"])
		}
		return
	}
}

func TestUnknownMethodReturnsError(t *testing.T) {
	d := New("127.0.0.1:19683", "tok")
	startTestDaemon(t, d)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, "ws://127.0.0.1:19683/ws?token=tok", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer ws.Close(websocket.StatusNormalClosure, "")

	ws.Read(ctx) //nolint:errcheck
	ws.Write(ctx, websocket.MessageText, []byte(`{"id":1,"method":"nope","params":{}}`)) //nolint:errcheck
	for {
		_, data, err := ws.Read(ctx)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var m map[string]any
		json.Unmarshal(data, &m) //nolint:errcheck
		if m["id"] == nil {
			continue
		}
		if m["error"] == nil {
			t.Fatalf("expected an error for an unknown method, got %s", string(data))
		}
		return
	}
}

func TestTokenIsURLSafeAndLong(t *testing.T) {
	tok := NewToken()
	if len(tok) < 32 {
		t.Errorf("token too short: %d chars", len(tok))
	}
	for _, r := range tok {
		ok := r == '-' || r == '_' ||
			(r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if !ok {
			t.Fatalf("token has a URL-unsafe character: %q", r)
		}
	}
	if NewToken() == tok {
		t.Error("two tokens are identical")
	}
}
