package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"

	"opencode-gui/internal/app"
)

// state is what the window shows. The view builds the interface from it;
// replies arrive on goroutines and repaint via win.Update.
type state struct {
	sessions *app.Sessions
	win      *mygo.Window
	mu       sync.Mutex

	ids     []string // session IDs, sidebar order
	title   map[string]string
	replies map[string]string // session id → accumulated reply text
	active  string            // session id
	prompt  string
	status  string
	busy    bool

	sidebarWidth float32
	md           markdown.Options
}

func main() {
	st := &state{
		sessions: &app.Sessions{},
		title:    map[string]string{},
		replies:  map[string]string{},
	}
	mygo.App.WhenReady(func() {
		if err := st.sessions.Start(context.Background()); err != nil {
			log.Printf("opencode server: %v", err)
		}
		list, err := st.sessions.List(context.Background())
		if err != nil {
			log.Printf("list sessions: %v", err)
		}
		for _, s := range list {
			st.ids = append(st.ids, s.ID)
			st.title[s.ID] = s.Title
			if st.active == "" {
				st.active = s.ID
			}
		}
		st.win = mygo.NewWindow(mygo.WindowOptions{
			Title:    "OpenCode GUI",
			Width:    1100,
			Height:   760,
			StateKey: "main",
			Content:  ui.View(st.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

func (st *state) view(c *ui.Context) {
	t := c.Theme()
	sidebarSel := st.active
	ui.Sidebar(c, &sidebarSel, func() {
		for _, id := range st.ids {
			ui.SidebarItem(c, id, nil, st.title[id])
		}
	})
	ui.Split(c, &st.sidebarWidth, func() {
		ui.Column(c).Fill().Padding(12).Gap(8).Background(t.Background).Children(func() {
			ui.Scroll(c).Fill().Children(func() {
				ui.Column(c).Fill().Gap(8).Children(func() {
					if st.busy {
						ui.Spinner(c)
					}
					if txt := st.replyText(); txt != "" {
						markdown.View(c, txt, &st.md)
					} else if !st.busy {
						ui.Text(c, "Type a prompt below.").TextColor(t.TextMuted)
					}
					if st.status != "" {
						ui.Text(c, st.status).TextColor(t.TextMuted)
					}
				})
			})
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.TextInput(c, &st.prompt).Placeholder("Prompt…").Fill().
					OnTextKey(func(k ui.TextKey) bool {
						if k.Key == ui.KeyEnter && k.Mods == 0 {
							st.send()
							return true
						}
						return false
					})
				if ui.PrimaryButton(c, "Send").Clicked() {
					st.send()
				}
			})
		})
	}, func() {
		ui.Column(c).Fill().Padding(12).Gap(8).Children(func() {
			ui.Text(c, "Replies appear here.").TextColor(t.TextMuted)
		})
	})
}

// replyText returns the active session's accumulated reply text.
func (st *state) replyText() string {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.replies[st.active]
}

// send submits the prompt for the active session on a goroutine.
func (st *state) send() {
	st.mu.Lock()
	if st.busy || st.prompt == "" || st.active == "" {
		st.mu.Unlock()
		return
	}
	text := st.prompt
	sid := st.active
	st.prompt = ""
	st.busy = true
	st.status = "waiting…"
	st.mu.Unlock()
	st.win.Update(func() {}) // repaint cleared prompt

	go func() {
		err := st.sessions.Prompt(context.Background(), sid, text)
		st.mu.Lock()
		st.busy = false
		if err != nil {
			st.status = "error: " + err.Error()
		} else {
			st.status = "sent; streaming reply…"
			go st.pollReplies(sid)
		}
		st.mu.Unlock()
		st.win.Update(func() {})
	}()
}

// pollReplies polls the session's messages until a new assistant text
// appears, appending it to the reply log. Replaced by the SSE bridge on
// day 2.
func (st *state) pollReplies(sid string) {
	base := len(st.replies[sid])
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		time.Sleep(1500 * time.Millisecond)
		msgs, err := st.sessions.Messages(context.Background(), sid)
		if err != nil {
			continue
		}
		text := ""
		for _, m := range msgs {
			if m.Info.Type != "assistant" {
				continue
			}
			for _, p := range m.Parts {
				if p.Type == "text" {
					text += p.Text + "\n\n"
				}
			}
		}
		if text != "" {
			st.mu.Lock()
			if st.replies[sid] == "" {
				st.replies[sid] = text
			} else if len(text) > base {
				st.replies[sid] = text
			}
			done := st.status == "sent; streaming reply…"
			st.status = ""
			st.mu.Unlock()
			if done {
				st.win.Update(func() {})
			}
			return
		}
	}
	st.mu.Lock()
	st.status = "no reply in time"
	st.mu.Unlock()
	st.win.Update(func() {})
}
