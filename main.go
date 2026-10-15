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
	filter        string
	modelName     string
	ctx           context.Context
	cancelSession context.CancelFunc
}

func main() {
	st := &state{
		sessions:     &app.Sessions{},
		sidebarWidth: sidebarWidthDefault,
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
			go func() { time.Sleep(time.Second); st.loadModel() }()
		}
		st.win = mygo.NewWindow(mygo.WindowOptions{
			Title:     "OpenCode GUI",
			Width:     1280,
			Height:    800,
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

// loadModel sets the model name shown in the composer chip.
func (st *state) loadModel() {
	agents, err := st.sessions.Agents(st.ctx)
	if err != nil {
		return
	}
	for _, a := range agents {
		if a.Model != nil && a.Model.ID != "" {
			st.mu.Lock()
			st.modelName = a.Model.ID
			st.mu.Unlock()
			st.win.Update(func() {})
			return
		}
	}
}

func (st *state) view(c *ui.Context) {
	t := kerenzikovTheme(c.Theme().Dark)
	c.SetTheme(t)

	ui.Row(c).FillWidth().FillHeight().AlignItems(ui.Stretch).Children(func() {
		// ── Sidebar ──────────────────────────────────────────────
		st.sidebar(c, t)

		// ── Chat ─────────────────────────────────────────────────
		ui.Column(c).Grow(1).FillHeight().Children(func() {
			st.toolbar(c, t)
			st.chat(c, t)
			st.composer(c, t)
		})

		// ── Right panel ──────────────────────────────────────────
		if st.tab >= 0 {
			ui.Column(c).Width(panelWidthDefault).FillHeight().
				Background(t.Background).
				BorderWidth(0, 0, 0, 1).BorderColor(t.Border).
				Children(func() {
					st.panel(c, t)
				})
		}
	})
}

// sidebar lists sessions with a search field and time groups, as the
// reference app's source list.
func (st *state) sidebar(c *ui.Context, t *ui.Theme) {
	st.mu.Lock()
	sel := st.active
	sessions := append([]opencode.Session(nil), st.sessionsList...)
	query := st.filter
	st.mu.Unlock()

	ui.Column(c).Width(st.sidebarWidth).FillHeight().
		Background(sidebarColor(t)).
		BorderWidth(0, 1, 0, 0).BorderColor(t.Border).
		Children(func() {
			ui.Column(c).Gap(8).Padding(12).Children(func() {
				if ui.PrimaryButton(c, "New session").FillWidth().Clicked() {
					go st.newSession()
				}
				ui.TextInput(c, &st.filter).Placeholder("Filter sessions").FillWidth()
			})

			ui.Scroll(c).Grow(1).Children(func() {
				ui.Column(c).Gap(2).Padding(6, 8, 12).Children(func() {
					shown := 0
					for _, s := range sessions {
						title := sessionLabel(s)
						if query != "" && !strings.Contains(strings.ToLower(title), strings.ToLower(query)) {
							continue
						}
						shown++
						row := ui.Row(c).Height(sessionRowHeight).Radius(7).
							Padding(0, 10).Gap(8).AlignItems(ui.Center)
						row.Children(func() {
							ui.Text(c, truncate(title, 30)).Grow(1).MaxLines(1).Ellipsis("…")
							if s.ID == sel {
								ui.Text(c, "•").TextColor(t.Accent)
							}
						})
						if row.Hovered() {
							row.Background(t.SurfaceHover)
						}
						if s.ID == sel {
							row.Background(t.Surface)
						}
						if row.Clicked() {
							st.selectSession(s.ID)
						}
					}
					if shown == 0 {
						ui.Text(c, "No sessions").TextColor(t.TextMuted).Padding(8)
					}
				})
			})

			ui.Column(c).Gap(6).Padding(12).
				BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
				clients := 0
				if st.daemon != nil {
					clients = st.daemon.Clients()
				}
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "Daemon").FontSize(11).TextColor(t.TextMuted)
					ui.Badge(c, fmt.Sprintf("%d phone(s)", clients))
				})
				if ui.Button(c, "Copy connect URL").FillWidth().Clicked() {
					mygo.Clipboard.WriteText(st.daemonURL)
					st.status = "connect URL copied"
				}
			})
		})
}

// selectSession makes a session active and loads it.
func (st *state) selectSession(id string) {
	st.mu.Lock()
	if id == st.active {
		st.mu.Unlock()
		return
	}
	st.active = id
	st.messages = nil
	st.diffs = nil
	st.diffState = nil
	st.mu.Unlock()
	st.refreshActive()
}

// truncate shortens s to n runes with an ellipsis.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// sessionLabel is a session's title, or a short stand-in for a session that
// has none (a raw ses_… id reads as noise in the list).
func sessionLabel(s opencode.Session) string {
	if t := strings.TrimSpace(s.Title); t != "" {
		return t
	}
	id := strings.TrimPrefix(s.ID, "ses_")
	if len(id) > 8 {
		id = id[:8]
	}
	return "Session " + id
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

// toolbar is the quiet top bar: title on the left, status and controls on
// the right.
func (st *state) toolbar(c *ui.Context, t *ui.Theme) {
	st.mu.Lock()
	running := st.running
	status := st.status
	tab := st.tab
	active := st.active
	st.mu.Unlock()

	ui.Row(c).Gap(10).Padding(10, 16).AlignItems(ui.Center).Height(48).
		BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
		title := "New task"
		st.mu.Lock()
		for _, s := range st.sessionsList {
			if s.ID == active && s.Title != "" {
				title = s.Title
			}
		}
		st.mu.Unlock()
		ui.Text(c, truncate(title, 48)).Bold().MaxLines(1)
		ui.Spacer(c)
		if running {
			ui.Spinner(c)
			if ui.Button(c, "Stop").Clicked() {
				go func() { _ = st.sessions.Interrupt(st.ctx, active) }()
			}
		}
		if status != "" {
			ui.Text(c, status).FontSize(12).TextColor(t.TextMuted).MaxLines(1)
		}
		labels := []string{"Panel", "Hidden"}
		ui.Segmented(c, &tab, labels...).Changed()
		if tab != st.tab {
			st.mu.Lock()
			st.tab = tab
			st.mu.Unlock()
		}
	})
}

// chat renders the message list, or the centered empty state.
func (st *state) chat(c *ui.Context, t *ui.Theme) {
	st.mu.Lock()
	msgs := append([]message(nil), st.messages...)
	st.mu.Unlock()

	if len(msgs) == 0 {
		ui.Column(c).Grow(1).FillWidth().Center().Gap(8).Children(func() {
			ui.Text(c, "*").FontSize(26).TextColor(t.Accent)
			ui.Text(c, "What should we build?").FontSize(22).Bold()
			ui.Text(c, "Type below, or ask on your phone.").
				FontSize(14).TextColor(t.TextMuted)
		})
		return
	}

	ui.List(c, &st.chatList, len(msgs), func(i int) {
		m := msgs[i]
		contentColumn(c, func() {
			if m.Role == "assistant" {
				ui.Column(c).FillWidth().Padding(10, 20).Children(func() {
					if m.Done {
						markdown.View(c, m.Text, &st.md)
					} else {
						markdown.Stream(c, m.Text, &st.md)
					}
				})
				return
			}
			// The user's own message: a right-aligned rounded bubble.
			ui.Row(c).FillWidth().Justify(ui.End).Padding(10, 20).Children(func() {
				ui.Column(c).Radius(14).Padding(10, 14).MaxWidth(560).
					Background(t.Surface).Border(1, t.Border).Children(func() {
					ui.Text(c, m.Text).Wrap().Selectable().FillWidth()
				})
			})
		})
	}).Grow(1).Gap(4).Justify(ui.End)
}

// composer is the large rounded input card with the model/access chips,
// measured from the reference app: 20px side padding, 8px bottom, a 720
// wide column, 12px radius, 12px inner padding.
func (st *state) composer(c *ui.Context, t *ui.Theme) {
	ui.Row(c).FillWidth().Justify(ui.Center).
		Padding(0, composerPadX, composerPadY).Children(func() {
		ui.Column(c).Width(contentMaxWidth).Children(func() {
			box := ui.Column(c).Radius(composerRadius).Padding(composerPadInner).
				Background(t.Background).Border(1, t.Border).Gap(9)
			box.Children(func() {
				input := ui.TextInput(c, &st.prompt).
					Placeholder("Do anything…").FillWidth()
				input.OnSubmit(st.send)

				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					model := "deepseek-v4-1-flash"
					if st.modelName != "" {
						model = st.modelName
					}
					chip(c, t).Children(func() {
						ui.Text(c, model).FontSize(12).TextColor(t.TextMuted)
					})
					chip(c, t).Children(func() {
						ui.Text(c, "Full access").FontSize(12).TextColor(t.TextMuted)
					})
					ui.Spacer(c)
					if ui.PrimaryButton(c, "↑").Height(sendHeight).
						Radius(sendRadius).Clicked() {
						st.send()
					}
				})
			})
		})
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

// panel is the right pane: the session's changes, or the "open a surface"
// chooser when there are none.
func (st *state) panel(c *ui.Context, t *ui.Theme) {
	st.mu.Lock()
	diffs := append([]opencode.FileDiff(nil), st.diffs...)
	chosen := st.diffFile
	st.mu.Unlock()

	if len(diffs) == 0 {
		ui.Column(c).FillWidth().FillHeight().Padding(32).Gap(4).
			AlignItems(ui.Start).Children(func() {
			ui.Text(c, "Open a surface").Bold()
			ui.Text(c, "Choose what to show in this panel").
				FontSize(12).TextColor(t.TextMuted)
			ui.Row(c).Gap(10).Padding(20, 0, 0).Children(func() {
				tile(c, t, "Files", "Browse the workspace")
				tile(c, t, "Review", "Review file changes")
			})
		})
		return
	}

	ui.Column(c).FillWidth().FillHeight().Gap(8).Padding(12).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Textf(c, "Changes (%d)", len(diffs)).Bold()
		})
		ui.List(c, nil, len(diffs), func(i int) {
			d := diffs[i]
			row := ui.Row(c).Gap(8).Padding(6, 10).Radius(8).AlignItems(ui.Center)
			row.Children(func() {
				mark := "±"
				if d.Status == "added" {
					mark = "+"
				} else if d.Status == "deleted" {
					mark = "−"
				}
				ui.Text(c, mark).Bold()
				ui.Text(c, shortPath(d.File)).Grow(1)
				ui.Textf(c, "+%d −%d", d.Additions, d.Deletions).
					FontSize(11).TextColor(t.TextMuted)
			})
			if row.Clicked() {
				st.mu.Lock()
				st.diffFile = d.File
				st.diffState = nil
				st.mu.Unlock()
			}
			if d.File == chosen {
				row.Background(t.Surface)
			}
		}).MaxHeight(140)

		ui.Divider(c)

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

		if ds != nil {
			editor.DiffView(c, ds).Grow(1).FillWidth()
		} else {
			ui.Text(c, "Cannot render this patch.").TextColor(t.TextMuted)
		}
	})
}

// tile is one choice of the surface chooser.
func tile(c *ui.Context, t *ui.Theme, title, desc string) {
	ui.Column(c).Width(170).Radius(14).Padding(14).Gap(4).
		Background(t.Surface).Border(1, t.Border).Children(func() {
		ui.Text(c, title).Bold()
		ui.Text(c, desc).FontSize(12).TextColor(t.TextMuted)
	})
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
