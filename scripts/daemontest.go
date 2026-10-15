//go:build ignore

// Daemon smoke test: starts the daemon on a test port with a known token,
// connects, and calls system.health. Run: go run scripts\daemontest.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coder/websocket"

	"opencode-gui/internal/daemon"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	d := daemon.New("127.0.0.1:19680", "testtoken")
	d.Handle("system.health", func(ctx context.Context, _ json.RawMessage) (any, error) {
		return map[string]any{"ok": true, "port": 4096}, nil
	})
	go d.ListenAndServe(ctx) //nolint:errcheck
	time.Sleep(500 * time.Millisecond)

	// Wrong token must be refused.
	if _, _, err := websocket.Dial(ctx, "ws://127.0.0.1:19680/ws?token=wrong", nil); err == nil {
		fmt.Println("FAIL: bad token accepted")
		return
	}
	fmt.Println("bad token refused: ok")

	ws, _, err := websocket.Dial(ctx, "ws://127.0.0.1:19680/ws?token=testtoken", nil)
	if err != nil {
		fmt.Println("FAIL dial:", err)
		return
	}
	defer ws.Close(websocket.StatusNormalClosure, "")

	// First message is daemon.hello.
	_, hello, err := ws.Read(ctx)
	if err != nil {
		fmt.Println("FAIL read hello:", err)
		return
	}
	fmt.Println("hello:", string(hello))

	req := `{"id":1,"method":"system.health","params":{}}`
	if err := ws.Write(ctx, websocket.MessageText, []byte(req)); err != nil {
		fmt.Println("FAIL write:", err)
		return
	}
	for {
		_, data, err := ws.Read(ctx)
		if err != nil {
			fmt.Println("FAIL read:", err)
			return
		}
		var m map[string]any
		json.Unmarshal(data, &m)
		if m["id"] != nil {
			fmt.Println("result:", string(data))
			fmt.Println("DAEMON-OK")
			return
		}
	}
}
