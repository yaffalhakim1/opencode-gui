package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"

	"opencode-gui/internal/daemon"
	"opencode-gui/internal/opencode"
)

// settingsContentMaxWidth is the settings column's width (SETTINGS_CONTENT_MAX_WIDTH).
const settingsContentMaxWidth = 760

// Settings section identifiers, in nav order.
const (
	secGeneral = iota
	secAppearance
	secProviders
	secSkills
	secMemory
	secUsage
	secDaemon
)

var sectionNames = []string{
	"General", "Appearance", "Providers", "Skills", "Memory", "Usage", "Daemon",
}

// settings is the Settings page state.
type settings struct {
	open    bool
	section int
	search  string

	// Appearance.
	theme    string
	uiFont   string
	codeFont string

	// Skills.
	skills      []opencode.Skill
	skillsReady bool
	skillQuery  string
	skillFilter string // the chosen filter label
	skillSel    int

	// Usage.
	usage      opencode.Stats
	usageReady bool
	usageRange int // 0: 30 days, 1: 7 days, 2: all

	// Daemon.
	daemonPort   string
	daemonExpose bool
}

var fontSizes = []string{"12 px", "13 px", "14 px", "15 px", "16 px", "18 px"}

// openSettings shows the page and loads what the section needs.
func (st *state) openSettings() {
	st.set.section = secGeneral
	st.set.open = true
	st.set.theme = "System"
	st.set.uiFont = "14 px"
	st.set.codeFont = "14 px"
	st.set.daemonPort = "9680"
	st.set.skillFilter = "All skills"
	st.set.daemonExpose = true
	st.loadSkills()
}

// loadSkills fetches the skill list once.
func (st *state) loadSkills() {
	go func() {
		skills, err := st.sessions.Skills(st.ctx)
		if err != nil {
			st.set.skillsReady = true
			st.win.Update(func() {})
			return
		}
		sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })
		st.set.skills = skills
		st.set.skillsReady = true
		st.win.Update(func() {})
	}()
}

// loadUsage fetches aggregate stats for the chosen range.
func (st *state) loadUsage() {
	st.set.usageReady = false
	var from int64
	switch st.set.usageRange {
	case 0:
		from = time.Now().AddDate(0, -1, 0).UnixMilli()
	case 1:
		from = time.Now().AddDate(0, 0, -7).UnixMilli()
	}
	to := time.Now().UnixMilli()
	go func() {
		stats, err := st.sessions.Usage(st.ctx, from, to)
		if err == nil {
			st.set.usage = stats
		}
		st.set.usageReady = true
		st.win.Update(func() {})
	}()
}

// settingsView is the whole Settings page: nav on the left, section on the
// right, as the reference app's.
func (st *state) settingsView(c *ui.Context, t *ui.Theme) {
	ui.Row(c).FillWidth().FillHeight().Children(func() {
		st.settingsNav(c, t)
		ui.Column(c).Grow(1).FillHeight().Background(t.Background).Children(func() {
			ui.Scroll(c).Grow(1).Children(func() {
				ui.Column(c).FillWidth().Padding(32, 32).Gap(15).Children(func() {
					ui.Column(c).MaxWidth(settingsContentMaxWidth).Children(func() {
						ui.Column(c).Gap(15).Children(func() {
							switch st.set.section {
							case secGeneral:
								st.sectionGeneral(c, t)
							case secAppearance:
								st.sectionAppearance(c, t)
							case secProviders:
								st.sectionProviders(c, t)
							case secSkills:
								st.sectionSkills(c, t)
							case secMemory:
								st.sectionMemory(c, t)
							case secUsage:
								st.sectionUsage(c, t)
							case secDaemon:
								st.sectionDaemon(c, t)
							}
						})
					})
				})
			})
		})
	})
}

// settingsNav is the left rail: Back, search, and the section list.
func (st *state) settingsNav(c *ui.Context, t *ui.Theme) {
	ui.Column(c).Width(sidebarWidthDefault).FillHeight().
		Background(sidebarColor(t)).
		BorderWidth(0, 1, 0, 0).BorderColor(t.Border).
		Children(func() {
			ui.Column(c).Gap(8).Padding(12).Children(func() {
				if ui.Button(c, "← Back").FillWidth().Clicked() {
					st.set.open = false
				}
				ui.SearchField(c, &st.set.search).Label("Search Settings").FillWidth()
			})
			ui.Column(c).Gap(2).Padding(6, 8).Children(func() {
				for i, name := range sectionNames {
					if st.set.search != "" &&
						!strings.Contains(strings.ToLower(name), strings.ToLower(st.set.search)) {
						continue
					}
					row := ui.Row(c).Height(actionRowHeight).Radius(7).
						Padding(0, 10).AlignItems(ui.Center)
					row.Children(func() {
						ui.Text(c, name).Grow(1).MaxLines(1)
					})
					if row.Hovered() {
						row.Background(t.SurfaceHover)
					}
					if i == st.set.section {
						row.Background(t.Surface)
					}
					if row.Clicked() {
						st.set.section = i
						if i == secUsage && !st.set.usageReady {
							st.loadUsage()
						}
					}
				}
			})
		})
}

// settingsCard is the raised card the sections use for their rows.
func settingsCard(c *ui.Context, t *ui.Theme, fn func()) {
	ui.Column(c).FillWidth().Radius(13).Padding(20, 14).
		Background(t.Surface).Children(fn)
}

// settingsRow is one row of a settings card: the label and its description on
// the left, the control on the right, as the reference app lays them out.
//
// It is a plain row rather than ui.Form/ui.Field: a Form puts labels in their
// own right-aligned column (macOS style), which is a different shape from the
// reference. The control keeps its own accessible name through .Label().
func settingsRow(c *ui.Context, t *ui.Theme, title, desc string, control func()) {
	ui.Row(c).FillWidth().Gap(16).AlignItems(ui.Center).Padding(10, 0).Children(func() {
		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(13.5).FontWeight(500)
			if desc != "" {
				ui.Text(c, desc).FontSize(12.5).TextColor(t.TextMuted)
			}
		})
		control()
	})
}

// labelledSwitch is a switch named for assistive technology, which ui.Switch
// requires since it draws no text of its own.
func labelledSwitch(c *ui.Context, on *bool, label string) ui.Element {
	return ui.Switch(c, on).Label(label)
}

// sectionHeader is the page title, as "General".
func sectionHeader(c *ui.Context, title string) {
	ui.Text(c, title).FontSize(18).FontWeight(500)
}

func (st *state) sectionGeneral(c *ui.Context, t *ui.Theme) {
	sectionHeader(c, "General")
	settingsCard(c, t, func() {
		settingsRow(c, t, "Local by default",
			"Projects, conversations, and settings are stored on this computer", func() {})
		ui.Divider(c)
		settingsRow(c, t, "Automatic updates",
			"Check for new versions in the background and offer to install them", func() {
				on := true
				ui.Switch(c, &on).Label("Automatic updates")
			})
	})
	settingsCard(c, t, func() {
		settingsRow(c, t, "opencode service",
			fmt.Sprintf("Connected on port %d", st.sessions.Port()), func() {
				ui.Badge(c, "Healthy")
			})
	})
}

func (st *state) sectionAppearance(c *ui.Context, t *ui.Theme) {
	sectionHeader(c, "Appearance")
	settingsCard(c, t, func() {
		settingsRow(c, t, "Theme", "Choose between system, light, or dark themes", func() {
			ui.Select(c, &st.set.theme, []string{"System", "Light", "Dark"}).Width(160)
		})
		ui.Divider(c)
		settingsRow(c, t, "Language", "Choose the language used throughout the app", func() {
			lang := "System"
			ui.Select(c, &lang, []string{"System", "English", "Indonesian"}).Width(160)
		})
		ui.Divider(c)
		settingsRow(c, t, "UI font size", "Text size across the interface and messages", func() {
			ui.Select(c, &st.set.uiFont, fontSizes).Width(160)
		})
		ui.Divider(c)
		settingsRow(c, t, "Code font size", "Text size in the file editor, diffs, and code blocks", func() {
			ui.Select(c, &st.set.codeFont, fontSizes).Width(160)
		})
	})
}

func (st *state) sectionProviders(c *ui.Context, t *ui.Theme) {
	sectionHeader(c, "Providers")
	agents, _ := st.sessions.Agents(st.ctx)
	settingsCard(c, t, func() {
		if len(agents) == 0 {
			ui.Text(c, "No providers connected.").TextColor(t.TextMuted)
			return
		}
		for i, a := range agents {
			if i > 0 {
				ui.Divider(c)
			}
			settingsRow(c, t, a.Name, providerSubtitle(a), func() {
				if a.Model != nil {
					ui.Badge(c, a.Model.ID)
				}
			})
		}
	})
}

// providerSubtitle describes an agent's mode.
func providerSubtitle(a opencode.Agent) string {
	model := "default model"
	if a.Model != nil && a.Model.ID != "" {
		model = a.Model.ID
	}
	return "Agent · " + model
}

func (st *state) sectionMemory(c *ui.Context, t *ui.Theme) {
	sectionHeader(c, "Memory")
	settingsCard(c, t, func() {
		ui.Text(c, "Project instructions").FontWeight(500)
		ui.Text(c, "opencode reads AGENTS.md from the project root and applies it to every session.").
			FontSize(12).TextColor(t.TextMuted)
		ui.Row(c).Gap(8).Padding(12, 0, 0).Children(func() {
			if ui.Button(c, "Open AGENTS.md").Clicked() {
				mygo.Shell.OpenPath("AGENTS.md")
			}
		})
	})
}

func (st *state) sectionDaemon(c *ui.Context, t *ui.Theme) {
	sectionHeader(c, "Daemon")

	settingsCard(c, t, func() {
		settingsRow(c, t, "Expose managed daemon",
			"Listen beyond this machine so another client can connect", func() {
				ui.Switch(c, &st.set.daemonExpose).Label("Expose managed daemon")
			})
	})

	settingsCard(c, t, func() {
		ui.Text(c, "Listener access").FontWeight(500)
		ui.Text(c, "Clients reach this daemon over your Tailscale network.").
			FontSize(12).TextColor(t.TextMuted)
		ui.Row(c).FillWidth().Gap(16).AlignItems(ui.Center).Padding(10, 0).Children(func() {
			ui.Column(c).Grow(1).Gap(2).Children(func() {
				ui.Text(c, "Port")
				ui.Text(c, "The daemon listens on this port.").FontSize(12).TextColor(t.TextMuted)
			})
			ui.TextInput(c, &st.set.daemonPort).Width(160)
		})
	})

	settingsCard(c, t, func() {
		ui.Text(c, "Connection details").FontWeight(500)
		ui.Text(c, "Enter these in the phone app.").FontSize(12).TextColor(t.TextMuted)

		ui.Row(c).FillWidth().Gap(16).AlignItems(ui.Center).Padding(10, 0).Children(func() {
			ui.Column(c).Width(120).Children(func() { ui.Text(c, "WebSocket URL") })
			ui.Text(c, st.daemonURL).Grow(1).FontSize(12).MaxLines(1).Ellipsis("…")
			if ui.Button(c, "Copy").Clicked() {
				mygo.Clipboard.WriteText(st.daemonURL)
			}
		})
		ui.Divider(c)
		ui.Row(c).FillWidth().Gap(16).AlignItems(ui.Center).Padding(10, 0).Children(func() {
			ui.Column(c).Width(120).Children(func() { ui.Text(c, "Token") })
			tok := ""
			if st.daemon != nil {
				tok = mask(st.daemon.Token())
			}
			ui.Text(c, tok).Grow(1).FontSize(12).MaxLines(1).Ellipsis("…")
			if ui.Button(c, "Copy token").Clicked() && st.daemon != nil {
				mygo.Clipboard.WriteText(st.daemon.Token())
			}
		})
		ui.Row(c).Padding(10, 0, 0).Children(func() {
			ui.Text(c, "The token grants full access to sessions on this machine. Keep it private.").
				FontSize(12).TextColor(t.Warning)
		})
	})

	settingsCard(c, t, func() {
		ui.Text(c, "Reach it from anywhere with Tailscale").FontWeight(500)
		ui.Text(c, "Tailscale lets your phone reach this machine from any network, with no port forwarding.").
			FontSize(12).TextColor(t.TextMuted)
		ip := tailscaleHost()
		ui.Row(c).Gap(8).Padding(12, 0, 0).AlignItems(ui.Center).Children(func() {
			if ip != "" {
				ui.Badge(c, ip)
			} else {
				ui.Text(c, "Tailscale not detected").FontSize(12).TextColor(t.TextMuted)
			}
		})
	})
}

// mask hides all but the last four characters of a token.
func mask(s string) string {
	if len(s) <= 4 {
		return "••••"
	}
	return strings.Repeat("•", 24) + s[len(s)-4:]
}

// tailscaleHost returns this machine's Tailscale address, or "".
func tailscaleHost() string {
	if ip := daemon.TailscaleIPv4(); ip != "" {
		return ip
	}
	return daemon.TailscaleName()
}

func (st *state) sectionSkills(c *ui.Context, t *ui.Theme) {
	sectionHeader(c, "Skills")
	if !st.set.skillsReady {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c)
			ui.Text(c, "Loading skills…").TextColor(t.TextMuted)
		})
		return
	}

	ui.Row(c).FillWidth().FillHeight().Gap(16).Children(func() {
		// Middle list.
		ui.Column(c).Width(300).FillHeight().Children(func() {
			ui.SearchField(c, &st.set.skillQuery).Label("Search skills").FillWidth()
			ui.Select(c, &st.set.skillFilter, []string{"All skills", "User skills", "Shared skills"}).
				FillWidth()
			ui.Textf(c, "%d skills", len(st.set.skills)).
				FontSize(12).TextColor(t.TextMuted).Padding(8, 0)
			ui.Scroll(c).Grow(1).Children(func() {
				ui.Column(c).Gap(2).Children(func() {
					for i, s := range st.set.skills {
						if !skillMatches(s, st.set.skillQuery, st.set.skillFilter) {
							continue
						}
						row := ui.Row(c).Height(48).Radius(7).Padding(6, 10).Gap(8).AlignItems(ui.Center)
						row.Children(func() {
							ui.Column(c).Grow(1).Gap(1).Children(func() {
								ui.Text(c, s.Name).MaxLines(1)
								ui.Text(c, s.Description).FontSize(11).TextColor(t.TextMuted).
									MaxLines(1).Ellipsis("…")
							})
						})
						if row.Hovered() {
							row.Background(t.SurfaceHover)
						}
						if i == st.set.skillSel {
							row.Background(t.Surface)
						}
						if row.Clicked() {
							st.set.skillSel = i
						}
					}
				})
			})
		})

		// Detail.
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Gap(12).Children(func() {
				st.skillDetail(c, t)
			})
		})
	})
}

// skillMatches reports whether a skill passes the search and the filter. A
// built-in skill lives under /builtin/; the rest come from disk.
func skillMatches(s opencode.Skill, query, filter string) bool {
	if query != "" &&
		!strings.Contains(strings.ToLower(s.Name), strings.ToLower(query)) &&
		!strings.Contains(strings.ToLower(s.Description), strings.ToLower(query)) {
		return false
	}
	builtin := strings.Contains(s.Path, "/builtin/")
	switch filter {
	case "User skills":
		return !builtin
	case "Shared skills":
		return builtin
	default: // All skills
		return true
	}
}

// skillDetail shows the chosen skill.
func (st *state) skillDetail(c *ui.Context, t *ui.Theme) {
	if st.set.skillSel >= len(st.set.skills) {
		ui.Text(c, "Choose a skill.").TextColor(t.TextMuted)
		return
	}
	s := st.set.skills[st.set.skillSel]

	ui.Row(c).FillWidth().Gap(12).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).Gap(2).Children(func() {
			ui.Text(c, s.Name).FontSize(18).FontWeight(500)
			ui.Text(c, "Available in every project").FontSize(12).TextColor(t.TextMuted)
		})
		on := s.Autoinvoke
		ui.Switch(c, &on).Label(s.Name + " auto-invoke")
	})

	if s.Description != "" {
		ui.Text(c, s.Description).FontSize(13)
	}

	settingsCard(c, t, func() {
		ui.Row(c).FillWidth().Gap(16).Padding(4, 0).Children(func() {
			ui.Column(c).Width(120).Children(func() { ui.Text(c, "Invoke").FontSize(12).TextColor(t.TextMuted) })
			ui.Textf(c, "/%s", s.ID).FontSize(12)
		})
		ui.Divider(c)
		ui.Row(c).FillWidth().Gap(16).Padding(4, 0).Children(func() {
			ui.Column(c).Width(120).Children(func() { ui.Text(c, "Path").FontSize(12).TextColor(t.TextMuted) })
			ui.Text(c, s.Path).FontSize(12).Grow(1).MaxLines(1).Ellipsis("…")
		})
	})

	ui.Row(c).Gap(8).Children(func() {
		if ui.Button(c, "Open SKILL.md").Clicked() {
			mygo.Shell.OpenPath(s.Path)
		}
		if ui.Button(c, "Show in File Manager").Clicked() {
			mygo.Shell.ShowItemInFolder(s.Path)
		}
		if ui.Button(c, "Copy Path").Clicked() {
			mygo.Clipboard.WriteText(s.Path)
		}
	})

	ui.Text(c, "SKILL.md").FontSize(11).TextColor(t.TextMuted)
	ui.Text(c, s.Content).FontSize(13).Wrap()
}

func (st *state) sectionUsage(c *ui.Context, t *ui.Theme) {
	sectionHeader(c, "Usage")
	if !st.set.usageReady {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c)
			ui.Text(c, "Loading usage…").TextColor(t.TextMuted)
		})
		return
	}
	u := st.set.usage

	// Range chooser.
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Segmented(c, &st.set.usageRange, "30 days", "7 days", "All")
		if ui.Button(c, "Refresh").Clicked() {
			st.loadUsage()
		}
	})

	// Headline stats.
	settingsCard(c, t, func() {
		ui.Text(c, "Processed tokens").FontSize(11).TextColor(t.TextMuted)
		ui.Text(c, humanTokens(totalTokens(u.Tokens))).FontSize(30).FontWeight(500)
		ui.Textf(c, "Across %d sessions · %s", u.Sessions, humanCost(u.Cost)).
			FontSize(12).TextColor(t.TextMuted)
	})

	// The five-stat row.
	settingsCard(c, t, func() {
		ui.Row(c).FillWidth().Gap(24).Children(func() {
			stat(c, t, "Input", humanTokens(u.Tokens.Input), "")
			stat(c, t, "Cached read", humanTokens(u.Tokens.Cache.Read), "")
			stat(c, t, "Output", humanTokens(u.Tokens.Output), "")
			stat(c, t, "Reasoning", humanTokens(u.Tokens.Reasoning), "")
			stat(c, t, "Cost", humanCost(u.Cost), "")
		})
	})

	// Daily activity: an area chart of steps per day, as the reference app's
	// "Daily processed tokens" plot.
	if len(u.Activity) > 1 {
		settingsCard(c, t, func() {
			ui.Text(c, "Daily activity").FontSize(13.5).FontWeight(500)
			ui.Text(c, "Steps per day across the range").
				FontSize(12.5).TextColor(t.TextMuted)
			activityChart(c, t, u.Activity)
		})
	}

	// Per-model breakdown, as a table: the framework sorts, resizes and moves
	// the columns, and reads the rows to assistive technology.
	settingsCard(c, t, func() {
		ui.Text(c, "Breakdown").FontSize(13.5).FontWeight(500)
		total := totalTokens(u.Tokens)
		cols := []ui.TableColumn{
			{Title: "Model"},
			{Title: "Cost", Width: 90, Align: ui.End},
			{Title: "Share", Width: 70, Align: ui.End},
			{Title: "Tokens", Width: 90, Align: ui.End},
		}
		ui.Table(c, nil, cols, len(u.Models), func(row, col int) {
			m := u.Models[row]
			switch col {
			case 0:
				ui.Text(c, m.Model.ID).SingleLine()
			case 1:
				ui.Text(c, humanCost(m.Cost))
			case 2:
				share := "—"
				if total > 0 {
					share = fmt.Sprintf("%.1f%%", totalTokens(m.Tokens)/total*100)
				}
				ui.Text(c, share)
			case 3:
				ui.Text(c, humanTokens(totalTokens(m.Tokens)))
			}
		}).Grow(1).Height(240).Label("Usage by model")
	})
}

// activityChart draws steps per day as a filled area with a line on top, the
// shape the reference app's usage plot has.
func activityChart(c *ui.Context, t *ui.Theme, days []opencode.Activity) {
	values := make([]float64, len(days))
	labels := make([]string, len(days))
	for i, d := range days {
		values[i] = float64(d.Steps)
		labels[i] = d.Date
	}

	_, hi := chart.Extent(values...)
	y := chart.NewLinear(0, hi).Nice(4)
	ticks := y.Ticks(4)

	// Fractions along x for each day, and along y for each value.
	n := len(values)
	xs := make([]float32, n)
	ys := make([]float32, n)
	for i, v := range values {
		if n > 1 {
			xs[i] = float32(i) / float32(n-1)
		}
		ys[i] = y.At(v)
	}

	chart.Plot(c, chart.Axes{Left: ticks}, func(area ui.Element) {
		area.Draw(func(p *ui.Painter, r ui.Rect) {
			for _, tick := range ticks { // grid lines
				p.Line(r.X, chart.Y(r, tick.At), r.X+r.W, chart.Y(r, tick.At), 1, t.Border)
			}
			// The filled area, then the line over it.
			p.FillPath(chart.Area(r, xs, nil, ys, chart.CurveMonotone), t.Accent.Alpha(0.15))
			p.StrokePath(chart.Line(r, xs, ys, chart.CurveMonotone), 2, t.Accent)
		})
	}).Label("Daily activity").Height(180).Gap(6)
}

// stat is one label + value + sub-line in the usage row.
func stat(c *ui.Context, t *ui.Theme, label, value, sub string) {
	ui.Column(c).Grow(1).Gap(2).Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(t.TextMuted)
		ui.Text(c, value).FontSize(18).FontWeight(500)
		if sub != "" {
			ui.Text(c, sub).FontSize(11).TextColor(t.TextMuted)
		}
	})
}

// totalTokens sums every kind of token use.
func totalTokens(t opencode.TokenUsage) float64 {
	return t.Input + t.Output + t.Reasoning + t.Cache.Read + t.Cache.Write
}

// humanTokens formats a token count as 1.2M, 3.4B, or a plain number.
func humanTokens(n float64) string {
	switch {
	case n >= 1e9:
		return fmt.Sprintf("%.2fB", n/1e9)
	case n >= 1e6:
		return fmt.Sprintf("%.2fM", n/1e6)
	case n >= 1e3:
		return fmt.Sprintf("%.1fK", n/1e3)
	default:
		return fmt.Sprintf("%.0f", n)
	}
}

// humanCost formats a dollar amount.
func humanCost(n float64) string {
	return fmt.Sprintf("$%.2f", n)
}
