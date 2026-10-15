package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"

	"opencode-gui/internal/app"
	"opencode-gui/internal/daemon"
	"opencode-gui/internal/opencode"
)

// message is one chat row: a user prompt or an assistant reply.
type message struct {
	ID   string
	Role string // "user" | "assistant"
	Text string
	Done bool
}

// state is what the window shows.
type state struct {
	sessions *app.Sessions
	win      *mygo.Window
	mu       sync.Mutex

	sessionsList []opencode.Session
	active       string // active session ID
	messages     []message
	diffs        []opencode.FileDiff

	prompt  string
	status  string
	running bool
	tab     int // inspector tab: 0 diff, 1 files

	sidebarWidth  float32
	chatList      ui.ListState
	md            markdown.Options
	diffState     *editor.DiffState
	daemon        *daemon.Daemon
	daemonURL     string
	diffFile      string
	ctx           context.Context
	cancelSession context.CancelFunc
}

func main() {
	st := &state{
		sessions:     &app.Sessions{},
		sidebarWidth: 240,
		ctx:          context.Background(),
	}
	st.chatList.FollowEnd = true
	st.chatList.FollowAnimation = &ui.Smooth
	st.chatList.Key = func(i int) any {
		st.mu.Lock()
		defer st.mu.Unlock()
		if i < len(st.messages) {
			return st.messages[i].ID
		}
		return i
	}

	mygo.App.WhenReady(func() {
		if err := st.sessions.Start(st.ctx); err != nil {
			log.Printf("opencode: %v", err)
			st.status = err.Error()
		} else {
			st.loadSessions()
			st.startDaemon(st.ctx)
		}
		st.win = mygo.NewWindow(mygo.WindowOptions{
			Title:     "OpenCode GUI",
			Width:     1280,
			Height:    820,
			MinWidth:  900,
			MinHeight: 560,
			StateKey:  "main",
			Content:   ui.View(st.view),
		})
		st.startWatch()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// loadSessions refreshes the session list (blocking; call from a goroutine
// or before the window opens).
func (st *state) loadSessions() {
	list, err := st.sessions.List(st.ctx)
	st.mu.Lock()
	defer st.mu.Unlock()
	if err != nil {
		st.status = "sessions: " + err.Error()
		return
	}
	st.sessionsList = list
	if st.active == "" && len(list) > 0 {
		st.active = list[0].ID
	}
}

// startWatch streams events and refreshes the active session's messages.
func (st *state) startWatch() {
	go func() {
		for {
			st.mu.Lock()
			sid := st.active
			st.mu.Unlock()
			if sid == "" {
				time.Sleep(time.Second)
				continue
			}
			st.sessions.Watch(st.ctx, sid, func() { st.refreshActive() })
			time.Sleep(500 * time.Millisecond)
		}
	}()
}

// refreshActive re-reads the active session's messages and diffs.
func (st *state) refreshActive() {
	st.mu.Lock()
	sid := st.active
	st.mu.Unlock()
	if sid == "" {
		return
	}
	msgs, err := st.sessions.Messages(st.ctx, sid)
	if err != nil {
		return
	}
	diffs, _ := st.sessions.Diff(st.ctx, sid)

	st.mu.Lock()
	converted := make([]message, 0, len(msgs))
	running := false
	for _, m := range msgs {
		role := "user"
		done := true
		switch m.Info.Type {
		case "assistant":
			role = "assistant"
		case "idle", "error":
			continue
		}
		if m.Info.Type == "assistant" {
			// An assistant message without a finish is still streaming.
			if m.Info.Finish == "" {
				running = true
				done = false
			}
		}
		var b strings.Builder
		for _, p := range m.Parts {
			if p.Type == "text" && p.Text != "" {
				if b.Len() > 0 {
					b.WriteString("\n\n")
				}
				b.WriteString(p.Text)
			}
		}
		if b.Len() == 0 && role == "assistant" {
			continue
		}
		converted = append(converted, message{ID: m.Info.ID, Role: role, Text: b.String(), Done: done})
	}
	st.messages = converted
	st.diffs = diffs
	st.running = running
	st.status = ""
	st.mu.Unlock()

	if st.daemon != nil {
		st.daemon.Broadcast("session.updated", map[string]any{"sessionID": sid})
	}
	st.win.Update(func() {})
}

func (st *state) view(c *ui.Context) {
	t := c.Theme()

	ui.Row(c).Fill().Children(func() {
		// ── Sidebar ──────────────────────────────────────────────
		st.sidebar(c)

		// ── Chat ─────────────────────────────────────────────────
		ui.Column(c).Grow(1).Fill().Children(func() {
			st.toolbar(c, t)
			st.chat(c, t)
			st.composer(c)
		})

		// ── Inspector ────────────────────────────────────────────
		if st.tab >= 0 {
			ui.Column(c).Width(420).Fill().Background(t.Surface).Children(func() {
				st.inspector(c, t)
			})
		}
	})
}

// sidebar lists sessions and actions.
func (st *state) sidebar(c *ui.Context) {
	st.mu.Lock()
	sel := st.active
	sessions := append([]opencode.Session(nil), st.sessionsList...)
	st.mu.Unlock()

	ui.Sidebar(c, &sel, func() {
		ui.SidebarSection(c, "Sessions", nil, func() {
			for _, s := range sessions {
				title := s.Title
				if title == "" {
					title = s.ID
				}
				ui.SidebarItem(c, s.ID, nil, title)
			}
		})
		ui.SidebarSection(c, "Actions", nil, func() {
			if ui.SidebarItem(c, "+new", nil, "New session").Clicked() {
				go st.newSession()
			}
		})
		ui.SidebarSection(c, "Daemon", nil, func() {
			clients := 0
			if st.daemon != nil {
				clients = st.daemon.Clients()
			}
			ui.SidebarItem(c, "daemon", nil, fmt.Sprintf("Phones: %d", clients))
			if ui.SidebarItem(c, "copyurl", nil, "Copy connect URL").Clicked() {
				mygo.Clipboard.WriteText(st.daemonURL)
			}
		})
	}).Width(st.sidebarWidth)

	if sel != "" && sel != st.activeID() {
		st.mu.Lock()
		st.active = sel
		st.messages = nil
		st.diffs = nil
		st.mu.Unlock()
		st.refreshActive()
	}
}

// newSession creates a session and makes it active.
func (st *state) newSession() {
	sess, err := st.sessions.New(st.ctx, "Session")
	if err != nil {
		return
	}
	st.mu.Lock()
	st.sessionsList = append([]opencode.Session{sess}, st.sessionsList...)
	st.active = sess.ID
	st.messages = nil
	st.diffs = nil
	st.mu.Unlock()
	st.win.Update(func() {})
}

// activeID reads the active session ID.
func (st *state) activeID() string {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.active
}

// toolbar shows the project, model, run state and controls.
func (st *state) toolbar(c *ui.Context, t *ui.Theme) {
	st.mu.Lock()
	running := st.running
	status := st.status
	tab := st.tab
	st.mu.Unlock()

	ui.Toolbar(c, func() {
		ui.Text(c, "OpenCode").Bold()
		ui.Text(c, fmt.Sprintf(":%d", st.sessions.Port())).TextColor(t.TextMuted)
		if status != "" {
			ui.Text(c, status).TextColor(t.TextMuted)
		}
		ui.Spacer(c)
		if running {
			ui.Spinner(c)
			if ui.Button(c, "Stop").Clicked() {
				st.mu.Lock()
				sid := st.active
				st.mu.Unlock()
				go func() { _ = st.sessions.Interrupt(st.ctx, sid) }()
			}
		}
		labels := []string{"Diff", "None"}
		if ui.Segmented(c, &tab, labels...).Changed() {
			st.mu.Lock()
			st.tab = tab
			st.mu.Unlock()
		}
	})
}

// chat renders the message list.
func (st *state) chat(c *ui.Context, t *ui.Theme) {
	st.mu.Lock()
	msgs := append([]message(nil), st.messages...)
	st.mu.Unlock()

	ui.List(c, &st.chatList, len(msgs), func(i int) {
		m := msgs[i]
		ui.Column(c).Gap(4).Padding(8, 12).Children(func() {
			label := "You"
			if m.Role == "assistant" {
				label = "Assistant"
			}
			ui.Text(c, label).FontSize(11).Bold().TextColor(t.TextMuted)
			if m.Role == "assistant" {
				if m.Done {
					markdown.View(c, m.Text, &st.md)
				} else {
					markdown.Stream(c, m.Text, &st.md)
				}
			} else {
				ui.Text(c, m.Text).Selectable()
			}
		})
	}).Grow(1).Gap(8).Children(func() {
		if len(msgs) == 0 {
			ui.Text(c, "Type a prompt below to start.").TextColor(t.TextMuted).Padding(16)
		}
	})
}

// composer is the prompt input row.
func (st *state) composer(c *ui.Context) {
	ui.Row(c).Gap(8).Padding(12).AlignItems(ui.Center).Children(func() {
		input := ui.TextInput(c, &st.prompt).Placeholder("Ask opencode…").Grow(1)
		send := st.send
		input.OnSubmit(send)
		if ui.PrimaryButton(c, "Send").Clicked() {
			send()
		}
	})
}

// send submits the current prompt for the active session.
func (st *state) send() {
	st.mu.Lock()
	text := strings.TrimSpace(st.prompt)
	sid := st.active
	if text == "" || sid == "" || st.running {
		st.mu.Unlock()
		return
	}
	st.prompt = ""
	st.running = true
	st.status = "sending…"
	st.mu.Unlock()
	st.win.Update(func() {})

	go func() {
		if err := st.sessions.Prompt(st.ctx, sid, text); err != nil {
			st.mu.Lock()
			st.running = false
			st.status = "prompt: " + err.Error()
			st.mu.Unlock()
			st.win.Update(func() {})
			return
		}
		st.refreshActive()
	}()
}

// inspector shows the changed files of the active session and the diff of
// the chosen one.
func (st *state) inspector(c *ui.Context, t *ui.Theme) {
	st.mu.Lock()
	diffs := append([]opencode.FileDiff(nil), st.diffs...)
	chosen := st.diffFile
	st.mu.Unlock()

	ui.Textf(c, "Changes (%d)", len(diffs)).Bold().Padding(12, 12, 4)
	if len(diffs) == 0 {
		ui.Text(c, "No changes in this session.").TextColor(t.TextMuted).Padding(12)
		return
	}

	// File list, newest state first.
	ui.List(c, nil, len(diffs), func(i int) {
		d := diffs[i]
		row := ui.Row(c).Gap(8).Padding(4, 12).AlignItems(ui.Center)
		row.Children(func() {
			mark := "±"
			switch d.Status {
			case "added":
				mark = "+"
			case "deleted":
				mark = "−"
			}
			ui.Text(c, mark).Bold()
			ui.Text(c, shortPath(d.File)).Grow(1)
			ui.Textf(c, "+%d −%d", d.Additions, d.Deletions).FontSize(11).TextColor(t.TextMuted)
		})
		if row.Clicked() {
			st.mu.Lock()
			st.diffFile = d.File
			st.diffState = nil // rebuild below
			st.mu.Unlock()
		}
		if d.File == chosen {
			row.Background(t.Selection)
		}
	}).MaxHeight(160)

	// The diff of the chosen file.
	st.mu.Lock()
	idx := 0
	for i, d := range diffs {
		if d.File == st.diffFile {
			idx = i
		}
	}
	if st.diffState == nil {
		if fs, err := editor.ParsePatch(diffs[idx].Patch); err == nil && len(fs) > 0 {
			ds := &editor.DiffState{}
			ds.SetPatch(fs[0])
			st.diffState = ds
		}
	}
	ds := st.diffState
	st.mu.Unlock()

	ui.Divider(c)
	if ds != nil {
		editor.DiffView(c, ds).Grow(1).FillWidth()
	} else {
		ui.Text(c, "Cannot render this patch.").TextColor(t.TextMuted).Padding(12)
	}
}

// shortPath trims a long path to its last two segments.
func shortPath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	parts := strings.Split(p, "/")
	if len(parts) <= 2 {
		return p
	}
	return ".../" + strings.Join(parts[len(parts)-2:], "/")
}
