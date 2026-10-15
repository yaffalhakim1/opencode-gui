//go:build ignore

// Phone-flow smoke test against a RUNNING app: connects to its daemon over
// the Tailscale IP and calls the documented methods.
//
//	go run scripts\phoneflow.go <host> <token>
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/coder/websocket"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: phoneflow <host> <token>")
		return
	}
	host, token := os.Args[1], os.Args[2]
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	url := fmt.Sprintf("ws://%s:9680/ws?token=%s", host, token)
	ws, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		fmt.Println("FAIL dial:", err)
		return
	}
	defer ws.Close(websocket.StatusNormalClosure, "")
	fmt.Println("connected to", host)

	call := func(id int, method string, params any) map[string]any {
		body, _ := json.Marshal(map[string]any{"id": id, "method": method, "params": params})
		if err := ws.Write(ctx, websocket.MessageText, body); err != nil {
			fmt.Println("FAIL write:", err)
			return nil
		}
		for {
			_, data, err := ws.Read(ctx)
			if err != nil {
				fmt.Println("FAIL read:", err)
				return nil
			}
			var m map[string]any
			json.Unmarshal(data, &m)
			if m["id"] == nil {
				continue // an event
			}
			if m["error"] != nil {
				fmt.Printf("%s -> error %v\n", method, m["error"])
			} else {
				b, _ := json.Marshal(m["result"])
				s := string(b)
				if len(s) > 160 {
					s = s[:160] + "…"
				}
				fmt.Printf("%s -> %s\n", method, s)
			}
			return m
		}
	}

	call(1, "system.health", map[string]any{})
	if r := call(2, "session.create", map[string]any{"title": "from phone"}); r != nil {
		if data, ok := r["result"].(map[string]any); ok {
			if id, ok := data["id"].(string); ok {
				call(3, "session.prompt", map[string]any{"id": id, "text": "Reply with exactly: PHONE-OK"})
				time.Sleep(12 * time.Second)
				call(4, "session.messages", map[string]any{"id": id})
				fmt.Println("PHONE-FLOW-OK")
			}
		}
	}
}
