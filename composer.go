package main

import (
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"opencode-gui/internal/opencode"
)

// AccessMode is the composer's permission posture, mirroring the reference
// app's RuntimeMode options (waku waku-protocol model.rs).
type AccessMode struct {
	Label string
	Desc  string
}

var accessModes = []AccessMode{
	{"Plan", "Investigate and propose without changing files"},
	{"Supervised", "Ask before commands and file changes"},
	{"Auto-accept edits", "Auto-approve edits, ask before other actions"},
	{"Auto", "An AI reviewer approves routine actions; risky ones still ask"},
	{"Full access", "Allow commands and edits without prompts"},
}

// composer renders the reference app's composer: a rounded card holding the
// prompt, its staged attachments, any queued follow-ups, and a control row of
// model, access, agent, attach and send/stop.
func (st *state) composer(c *ui.Context, t *ui.Theme) {
	ui.Row(c).FillWidth().Justify(ui.Center).
		Padding(0, composerPadX, composerPadY).Children(func() {
		ui.Column(c).Width(contentMaxWidth).Gap(8).Children(func() {
			st.queuedList(c, t)
			st.composerCard(c, t)
		})
	})
}

// composerCard is the input card itself.
func (st *state) composerCard(c *ui.Context, t *ui.Theme) {
	st.mu.Lock()
	running := st.running
	st.mu.Unlock()

	box := ui.Column(c).Radius(composerRadius).Padding(composerPadInner, 10).
		Background(composerBg(t)).Border(1, t.Border).Gap(8)
	box.Children(func() {
		// Staged attachments: the framework's token field, which already
		// chips them, wraps them and gives each a remove button.
		if len(st.attachments) > 0 {
			ui.TokenField(c, &st.attachments, nil)
		}

		input := ui.TextInput(c, &st.prompt).
			Placeholder(promptPlaceholder(running)).FillWidth()
		input.OnSubmit(st.submit)

		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			st.modelControl(c, t)
			st.accessControl(c, t)
			st.agentControl(c, t)
			ui.Spacer(c)
			if ui.Button(c, "📎").Clicked() {
				go st.attachFile()
			}
			st.sendOrStop(c, t, running)
		})
	})
}

// promptPlaceholder changes while a turn runs, so a second Enter queues
// instead of steering (the reference app's queue_placeholder).
func promptPlaceholder(running bool) string {
	if running {
		return "Queue a follow-up…"
	}
	return "Ask OpenCode to build something…"
}

// modelControl is the model menu: the framework's menu button, which opens a
// searchable list of every model the server offers.
func (st *state) modelControl(c *ui.Context, t *ui.Theme) {
	st.mu.Lock()
	name := st.modelName
	models := st.models
	st.mu.Unlock()
	if name == "" {
		name = "Model"
	}

	ui.MenuButton(c, truncate(name, 24), func(m *ui.Menu) {
		if len(models) == 0 {
			m.Item("Loading models…").Disabled(true)
			return
		}
		shown := 0
		for _, mod := range models {
			if !mod.Enabled {
				continue
			}
			shown++
			if shown > 40 {
				break // a menu of 129 models is unusable; the picker lists all
			}
			if m.Item(mod.Name).Checked(mod.Name == name).Chosen() {
				st.chooseModel(mod)
			}
		}
		if shown == 0 {
			m.Item("No models available").Disabled(true)
		}
		m.Separator()
		if m.Item("Browse all models…").Chosen() {
			st.mu.Lock()
			st.modelPickerOpen = true
			st.mu.Unlock()
		}
	})
}

// accessControl is the permission posture menu.
func (st *state) accessControl(c *ui.Context, t *ui.Theme) {
	st.mu.Lock()
	mode := st.accessMode
	st.mu.Unlock()
	if mode < 0 || mode >= len(accessModes) {
		mode = 4
	}

	ui.MenuButton(c, accessModes[mode].Label, func(m *ui.Menu) {
		for i, a := range accessModes {
			if m.Item(a.Label).Checked(i == mode).Chosen() {
				st.mu.Lock()
				st.accessMode = i
				st.mu.Unlock()
			}
		}
	})
}

// agentControl is the agent menu, listing the registered agents.
func (st *state) agentControl(c *ui.Context, t *ui.Theme) {
	st.mu.Lock()
	agents := append([]opencode.Agent(nil), st.agents...)
	idx := st.agentIdx
	st.mu.Unlock()
	if len(agents) == 0 {
		return
	}

	ui.MenuButton(c, truncate(agents[idx%len(agents)].Name, 18), func(m *ui.Menu) {
		for i, a := range agents {
			if m.Item(a.Name).Checked(i == idx).Chosen() {
				st.chooseAgent(i)
			}
		}
	})
}

// chooseAgent switches the session's agent.
func (st *state) chooseAgent(i int) {
	st.mu.Lock()
	st.agentIdx = i
	agents := append([]opencode.Agent(nil), st.agents...)
	sid := st.active
	st.mu.Unlock()
	if i >= len(agents) {
		return
	}
	go func() { _ = st.sessions.SwitchAgent(st.ctx, sid, agents[i].ID) }()
}

// sendOrStop is the send arrow, with a Stop button beside it while a turn runs.
func (st *state) sendOrStop(c *ui.Context, t *ui.Theme, running bool) {
	st.mu.Lock()
	hasText := strings.TrimSpace(st.prompt) != "" || len(st.attachments) > 0
	sid := st.active
	st.mu.Unlock()

	if running {
		if ui.Button(c, "Stop").Clicked() {
			go func() { _ = st.sessions.Interrupt(st.ctx, sid) }()
		}
	}
	if ui.PrimaryButton(c, "↑").Height(sendHeight).Radius(sendRadius).
		Disabled(!hasText).Clicked() {
		st.submit()
	}
}

// queuedList shows follow-ups waiting for the running turn to end.
func (st *state) queuedList(c *ui.Context, t *ui.Theme) {
	st.mu.Lock()
	queued := append([]string(nil), st.queued...)
	st.mu.Unlock()
	if len(queued) == 0 {
		return
	}

	ui.Column(c).Radius(composerRadius).Padding(10, 10).
		Background(t.Surface).Border(1, t.Border).Gap(4).Children(func() {
		ui.Textf(c, "Queued (%d)", len(queued)).FontSize(11).TextColor(t.TextMuted)
		for i, msg := range queued {
			ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, truncate(msg, 72)).FontSize(12).Grow(1).MaxLines(1).Ellipsis("…")
				if ui.Button(c, "Edit").Clicked() {
					st.editQueued(i)
				}
				if ui.Button(c, "×").Clicked() {
					st.removeQueued(i)
				}
			})
		}
	})
}

// submit sends the prompt, or queues it as a follow-up while a turn runs.
func (st *state) submit() {
	st.mu.Lock()
	text := strings.TrimSpace(st.prompt)
	sid := st.active
	if text == "" && len(st.attachments) == 0 {
		st.mu.Unlock()
		return
	}
	if sid == "" {
		st.mu.Unlock()
		return
	}
	if st.running {
		st.queued = append(st.queued, text)
		st.prompt = ""
		st.mu.Unlock()
		st.win.Update(func() {})
		return
	}

	atts := st.attachments
	st.prompt = ""
	st.attachments = nil
	st.running = true
	st.status = "sending…"
	st.mu.Unlock()
	st.win.Update(func() {})

	in := opencode.PromptInput{Text: text, Delivery: opencode.DeliverySteer}
	for _, a := range atts {
		in.Files = append(in.Files, opencode.FileAttachment{URI: a, Name: a})
	}

	go func() {
		if err := st.sessions.Prompt(st.ctx, sid, in); err != nil {
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

// flushQueued sends the next queued follow-up once the turn settles.
func (st *state) flushQueued() {
	st.mu.Lock()
	if len(st.queued) == 0 || st.running || st.active == "" {
		st.mu.Unlock()
		return
	}
	next := st.queued[0]
	st.queued = st.queued[1:]
	sid := st.active
	st.running = true
	st.mu.Unlock()
	st.win.Update(func() {})

	go func() {
		in := opencode.PromptInput{Text: next, Delivery: opencode.DeliveryQueue}
		if err := st.sessions.Prompt(st.ctx, sid, in); err != nil {
			st.mu.Lock()
			st.running = false
			st.mu.Unlock()
			st.win.Update(func() {})
			return
		}
		st.refreshActive()
	}()
}

func (st *state) removeQueued(i int) {
	st.mu.Lock()
	if i < len(st.queued) {
		st.queued = append(st.queued[:i], st.queued[i+1:]...)
	}
	st.mu.Unlock()
	st.win.Update(func() {})
}

// editQueued moves a queued follow-up back into the prompt.
func (st *state) editQueued(i int) {
	st.mu.Lock()
	if i < len(st.queued) {
		st.prompt = st.queued[i]
		st.queued = append(st.queued[:i], st.queued[i+1:]...)
	}
	st.mu.Unlock()
	st.win.Update(func() {})
}

// attachFile opens a file dialog and stages the chosen files as paths.
func (st *state) attachFile() {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Title:    "Attach files",
		Multiple: true,
	})
	if err != nil || len(paths) == 0 {
		return
	}
	st.mu.Lock()
	st.attachments = append(st.attachments, paths...)
	st.mu.Unlock()
	st.win.Update(func() {})
}

// loadModels fetches the model list once.
func (st *state) loadModels() {
	go func() {
		models, err := st.sessions.Models(st.ctx)
		if err != nil {
			return
		}
		st.mu.Lock()
		st.models = models
		st.mu.Unlock()
		st.win.Update(func() {})
	}()
}

// modelPicker is the full searchable list, opened from the model menu. It
// uses the framework's combobox, which already filters as the user types.
func (st *state) modelPicker(c *ui.Context, t *ui.Theme) {
	st.mu.Lock()
	open := st.modelPickerOpen
	models := append([]opencode.Model(nil), st.models...)
	st.mu.Unlock()
	if !open {
		return
	}

	// Only the enabled models, as "name (provider)" so the combobox's text
	// filter matches either.
	labels := make([]string, 0, len(models))
	byLabel := make(map[string]opencode.Model, len(models))
	for _, m := range models {
		if !m.Enabled {
			continue
		}
		label := m.Name
		if _, dup := byLabel[label]; dup {
			label = m.Name + " (" + m.ProviderID + ")"
		}
		labels = append(labels, label)
		byLabel[label] = m
	}

	chosen := st.modelName
	ui.Modal(c, &st.modelPickerOpen, func() {
		ui.Column(c).Width(520).Gap(10).Children(func() {
			ui.Textf(c, "Models (%d)", len(labels)).FontSize(13.5).FontWeight(500)
			if ui.Combobox(c, &chosen, labels).Label("Model").AutoFocus().FillWidth().Changed() {
				if m, ok := byLabel[chosen]; ok {
					st.chooseModel(m)
				}
			}
			ui.Text(c, "Type to filter, then choose with Enter.").
				FontSize(12.5).TextColor(t.TextMuted)
		})
	})
}

// chooseModel switches the session to a model.
func (st *state) chooseModel(m opencode.Model) {
	st.mu.Lock()
	st.modelName = m.Name
	st.modelPickerOpen = false
	sid := st.active
	st.mu.Unlock()
	st.win.Update(func() {})

	go func() {
		ref := opencode.ModelRef{ID: m.ModelID, ProviderID: m.ProviderID}
		if len(m.Variants) > 0 {
			ref.Variant = m.Variants[len(m.Variants)-1].ID // the most effort
		}
		_ = st.sessions.SwitchModel(st.ctx, sid, ref)
	}()
}

// permissionDialog asks the user about a tool call waiting to run, using the
// framework's modal so the keyboard and the dimmed backdrop are handled for us.
func (st *state) permissionDialog(c *ui.Context, t *ui.Theme) {
	st.mu.Lock()
	p := st.pending
	open := p != nil
	st.mu.Unlock()
	if !open {
		return
	}

	ui.Modal(c, &open, func() {
		ui.Column(c).Width(460).Gap(10).Children(func() {
			ui.Text(c, "Run a tool?").FontSize(13.5).FontWeight(500)
			ui.Textf(c, "The agent wants to run %s", p.Action).
				FontSize(12.5).TextColor(t.TextMuted)
			if len(p.Resources) > 0 {
				ui.Text(c, truncate(strings.Join(p.Resources, ", "), 200)).
					FontSize(12).TextColor(t.TextMuted)
			}
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, "Reject").Clicked() {
					st.replyPermission(opencode.PermissionReject)
				}
				if ui.Button(c, "Allow once").Clicked() {
					st.replyPermission(opencode.PermissionOnce)
				}
				if ui.PrimaryButton(c, "Always allow").Clicked() {
					st.replyPermission(opencode.PermissionAlways)
				}
			})
		})
	})
	// A dismissed modal (Escape or a click outside) rejects the request.
	if !open {
		st.replyPermission(opencode.PermissionReject)
	}
}

// replyPermission answers the pending request and clears it.
func (st *state) replyPermission(d opencode.PermissionDecision) {
	st.mu.Lock()
	p := st.pending
	st.pending = nil
	st.mu.Unlock()
	if p == nil {
		return
	}
	st.win.Update(func() {})
	go func() { _ = st.sessions.ReplyPermission(st.ctx, p.SessionID, p.ID, d) }()
}

// refreshPermissions reads the pending permission requests.
func (st *state) refreshPermissions() {
	reqs, err := st.sessions.Permissions(st.ctx)
	if err != nil {
		return
	}
	st.mu.Lock()
	if len(reqs) > 0 {
		first := reqs[0]
		st.pending = &first
	} else {
		st.pending = nil
	}
	st.mu.Unlock()
	st.win.Update(func() {})
}

// statusLine is the small footer under the composer: workspace and branch.
func (st *state) statusLine(c *ui.Context, t *ui.Theme) {
	st.mu.Lock()
	branch := st.branch
	st.mu.Unlock()

	ui.Row(c).FillWidth().Gap(12).Padding(4, composerPadX).Children(func() {
		ui.Text(c, "Local").FontSize(12).TextColor(t.TextMuted)
		if branch != "" {
			ui.Textf(c, "⎇ %s", branch).FontSize(12).TextColor(t.TextMuted)
		}
	})
}
