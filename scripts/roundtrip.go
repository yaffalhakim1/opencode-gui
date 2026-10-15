//go:build ignore

// Round-trip smoke test against the registered opencode service.
// Run with an opencode server already running (`opencode serve --port 4096`).
//
//	go run scripts\roundtrip.go
package main

import (
	"context"
	"fmt"
	"time"

	"opencode-gui/internal/opencode"
)

func main() {
	ep, err := opencode.Discover()
	if err != nil {
		fmt.Println("FAIL discover:", err)
		return
	}
	fmt.Println("service:", ep.URL, "v"+ep.Version)

	srv := opencode.Attach(ep)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	if err := srv.WaitHealthy(ctx); err != nil {
		fmt.Println("FAIL health:", err)
		return
	}
	fmt.Println("healthy")

	sess, err := srv.CreateSession(ctx, "smoke")
	if err != nil {
		fmt.Println("FAIL create:", err)
		return
	}
	fmt.Println("session:", sess.ID)

	if err := srv.Prompt(ctx, sess.ID, "Reply with exactly: ROUNDTRIP-OK"); err != nil {
		fmt.Println("FAIL prompt:", err)
		return
	}
	fmt.Println("prompt admitted; waiting for reply…")

	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		msgs, err := srv.Messages(ctx, sess.ID)
		if err != nil {
			fmt.Println("poll error:", err)
			continue
		}
		for _, m := range msgs {
			if m.Info.Type != "assistant" {
				continue
			}
			for _, p := range m.Parts {
				if p.Type == "text" && p.Text != "" {
					fmt.Println("reply:", p.Text)
					fmt.Println("ROUNDTRIP-OK")
					return
				}
			}
		}
	}
	fmt.Println("FAIL: no reply in time")
}
