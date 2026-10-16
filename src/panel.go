package src

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"opencode-gui/internal/opencode"
)

// Right-panel surfaces, as the reference app offers them (Browser, Terminal,
// Files, Review). Terminal is listed but marked unavailable until opencode
// exposes a pty, matching how that app disables it too.
const (
	surfFiles = iota
	surfReview
	surfBrowser
	surfTerminal
	surfFile
)

type surface struct {
	Name     string
	Desc     string
	Avail    bool
	Terminal bool
}

var surfaces = []surface{
	{"Files", "Browse and read workspace files", true, false},
	{"Review", "Review file changes", true, false},
	{"Browser", "Open a local app or URL", false, false},
	{"Terminal", "Start a shell in this workspace", false, true},
}

// panel is the right pane: the chosen surface, or the "Open a surface"
// chooser when none is chosen, as the reference app's.
func (st *state) panel(c *ui.Context, t *ui.Theme) {
	st.mu.Lock()
	chosen := st.surface
	diffs := append([]opencode.FileDiff(nil), st.diffs...)
	st.mu.Unlock()

	if chosen < 0 {
		st.surfaceChooser(c, t)
		return
	}

	ui.Column(c).FillWidth().FillHeight().Children(func() {
		st.surfaceTabs(c, t)
		ui.Divider(c)
		switch chosen {
		case surfReview:
			st.reviewSurface(c, t, diffs)
		case surfFiles:
			st.filesSurface(c, t)
		case surfFile:
			st.fileSurface(c, t)
		default:
			st.unavailableSurface(c, t, surfaces[chosen])
		}
	})
}

// surfaceChooser is the empty state: the four cards.
func (st *state) surfaceChooser(c *ui.Context, t *ui.Theme) {
	ui.Column(c).FillWidth().FillHeight().Center().Gap(6).Padding(24).Children(func() {
		ui.Text(c, "Open a surface").FontSize(13.5).FontWeight(500)
		ui.Text(c, "Choose what to show in the right panel").
			FontSize(12.5).TextColor(t.TextMuted)
		ui.Row(c).Gap(10).Padding(16, 0, 0).Wrap().Justify(ui.Center).Children(func() {
			for i, s := range surfaces {
				idx := i
				card := ui.Column(c).Width(170).Radius(13).Padding(14).Gap(4).
					Background(t.Surface).Border(1, t.Border)
				card.Children(func() {
					ui.Text(c, s.Name).FontSize(13.5).FontWeight(500)
					ui.Text(c, s.Desc).FontSize(12.5).TextColor(t.TextMuted)
				})
				if card.Hovered() {
					card.Background(t.SurfaceHover)
				}
				if card.Clicked() && s.Avail {
					st.mu.Lock()
					st.surface = idx
					st.mu.Unlock()
				}
			}
		})
	})
}

// surfaceTabs is the row of tabs across the top of the panel, using the
// framework's tab list so the arrows, Home and End move the choice.
func (st *state) surfaceTabs(c *ui.Context, t *ui.Theme) {
	st.mu.Lock()
	chosen := st.surface
	st.mu.Unlock()

	ui.Row(c).FillWidth().Gap(8).Padding(8, 10).AlignItems(ui.Center).Children(func() {
		// A file shown in the panel is the Files surface with a file open, so
		// the tab list keeps highlighting Files.
		tab := chosen
		if tab == surfFile {
			tab = surfFiles
		}
		ui.Tabs(c, &tab, "Files", "Review", "Browser", "Terminal").Grow(1)
		if tab != chosen && surfaces[tab].Avail {
			st.mu.Lock()
			st.surface = tab
			st.mu.Unlock()
			if tab == surfFiles {
				go st.loadWorkspaceFiles()
			}
		}
		if ui.Button(c, "×").Clicked() {
			st.mu.Lock()
			st.surface = -1
			st.rightCollapsed = true
			st.mu.Unlock()
		}
	})
}

// reviewSurface lists changed files and shows the chosen one's diff.
func (st *state) reviewSurface(c *ui.Context, t *ui.Theme, diffs []opencode.FileDiff) {
	if len(diffs) == 0 {
		ui.Column(c).FillWidth().Grow(1).Center().Children(func() {
			ui.Text(c, "No file changes in this session.").FontSize(12.5).TextColor(t.TextMuted)
		})
		return
	}

	st.mu.Lock()
	chosen := st.diffFile
	st.mu.Unlock()

	ui.Column(c).FillWidth().FillHeight().Gap(6).Padding(8).Children(func() {
		ui.List(c, nil, len(diffs), func(i int) {
			d := diffs[i]
			row := ui.Row(c).Radius(7).Padding(6, 8).Gap(8).AlignItems(ui.Center)
			row.Children(func() {
				ui.Text(c, statusMark(d.Status)).FontSize(12)
				ui.Text(c, shortPath(d.File)).Grow(1).FontSize(12).MaxLines(1).Ellipsis("…")
				ui.Textf(c, "+%d −%d", d.Additions, d.Deletions).
					FontSize(11).TextColor(t.TextMuted)
			})
			if row.Hovered() {
				row.Background(itemBg(t))
			}
			if d.File == chosen {
				row.Background(itemBg(t))
			}
			if row.Clicked() {
				st.mu.Lock()
				st.diffFile = d.File
				st.diffState = nil
				st.mu.Unlock()
			}
		}).MaxHeight(140)

		ui.Divider(c)
		st.diffView(c, t, diffs)
	})
}

// diffStats is the "+N −M · K changes" line the reference app shows above a
// review, from the diff viewer's own hunk statistics.
func diffStats(ds *editor.DiffState) string {
	add, del := ds.Stats()
	n := len(ds.Hunks())
	return fmt.Sprintf("+%d −%d · %d change%s", add, del, n, plural(n))
}

// plural is "s" unless n is one.
func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// statusMark is the one-character mark for a file's change kind.
func statusMark(status string) string {
	switch status {
	case "added":
		return "+"
	case "deleted":
		return "−"
	default:
		return "±"
	}
}

// filesSurface lists the workspace files as a tree, so directories open and
// close as the reference app's Files surface does.
func (st *state) filesSurface(c *ui.Context, t *ui.Theme) {
	st.mu.Lock()
	entries := append([]opencode.FSEntry(nil), st.fsEntries...)
	ready := st.filesReady
	st.mu.Unlock()

	if !ready {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(8).Children(func() {
			ui.Spinner(c)
			ui.Text(c, "Loading files…").FontSize(12.5).TextColor(t.TextMuted)
		})
		return
	}
	if len(entries) == 0 {
		ui.Text(c, "No files found.").FontSize(12.5).TextColor(t.TextMuted).Padding(8)
		return
	}

	ui.Tree(c, func() {
		for _, e := range entries {
			name := baseName(e.Path)
			if e.Type == "directory" {
				open := false
				ui.TreeItem(c, name, &open, nil)
				continue
			}
			if ui.TreeItem(c, name, nil, nil).Clicked() {
				go st.openFile(e.Path)
			}
		}
	}).FillWidth().Grow(1)
}

// openFile reads a file and shows it in the panel.
func (st *state) openFile(path string) {
	content, err := st.sessions.ReadFile(st.ctx, path)
	if err != nil {
		return
	}
	st.mu.Lock()
	st.openPath = path
	st.openText = content
	st.surface = surfFile
	st.mu.Unlock()
	st.win.Update(func() {})
}

// fileSurface shows one file's text.
func (st *state) fileSurface(c *ui.Context, t *ui.Theme) {
	st.mu.Lock()
	path := st.openPath
	text := st.openText
	st.mu.Unlock()

	ui.Column(c).FillWidth().FillHeight().Padding(8).Gap(6).Children(func() {
		ui.Text(c, shortPath(path)).FontSize(12.5).FontWeight(500).MaxLines(1).Ellipsis("…")
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Text(c, text).FontSize(12).Wrap().Selectable()
		})
	})
}

// baseName is the last segment of a path, with either separator.
func baseName(p string) string {
	i := strings.LastIndexAny(p, `/\`)
	if i < 0 {
		return p
	}
	return p[i+1:]
}

// diffView renders the chosen file's patch with the framework's review
// features: hunk statistics, next/previous change, and per-change revert.
func (st *state) diffView(c *ui.Context, t *ui.Theme, diffs []opencode.FileDiff) {
	st.mu.Lock()
	idx := 0
	for i, d := range diffs {
		if d.File == st.diffFile {
			idx = i
		}
	}
	if st.diffState == nil {
		if fs, err := editor.ParsePatch(diffs[idx].Patch); err == nil && len(fs) > 0 {
			ds := &editor.DiffState{HunkActions: true}
			ds.SetPatch(fs[0])
			st.diffState = ds
		}
	}
	ds := st.diffState
	st.mu.Unlock()

	if ds == nil {
		ui.Text(c, "Cannot render this patch.").FontSize(12.5).TextColor(t.TextMuted)
		return
	}

	// The change count and navigation, as a review tool shows them.
	cur, inside := ds.CurrentHunk()
	total := len(ds.Hunks())
	label := diffStats(ds)
	if cur >= 0 {
		pos := cur
		if !inside {
			pos = cur + 1 // the change the caret is before
		}
		label = fmt.Sprintf("%s · %d of %d", diffStats(ds), pos+1, total)
	}

	ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Padding(0, 0, 4).Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(t.TextMuted).Grow(1)
		if ui.Button(c, "↑").Label("Previous change").Disabled(total == 0).Clicked() {
			ds.GoToHunk(cur - 1)
		}
		if ui.Button(c, "↓").Label("Next change").Disabled(total == 0).Clicked() {
			ds.GoToHunk(cur + 1)
		}
	})

	editor.DiffView(c, ds).Grow(1).FillWidth()
}

// unavailableSurface is what an unavailable surface shows.
func (st *state) unavailableSurface(c *ui.Context, t *ui.Theme, s surface) {
	ui.Column(c).FillWidth().Grow(1).Center().Gap(4).Children(func() {
		ui.Text(c, s.Name+" unavailable").FontSize(13.5).FontWeight(500)
		ui.Text(c, "Open the "+s.Name+" surface again to start it").
			FontSize(12.5).TextColor(t.TextMuted)
	})
}

// loadWorkspaceFiles fills the Files surface from the workspace root.
func (st *state) loadWorkspaceFiles() {
	entries, err := st.sessions.ListDir(st.ctx, "")
	st.mu.Lock()
	defer st.mu.Unlock()
	st.filesReady = true
	if err != nil {
		return
	}
	// Directories first, then files, each alphabetical, as a file browser.
	var dirs, files []opencode.FSEntry
	for _, e := range entries {
		if e.Type == "directory" {
			dirs = append(dirs, e)
		} else {
			files = append(files, e)
		}
	}
	st.fsEntries = append(dirs, files...)
	st.win.Update(func() {})
}

// fileExt returns a lower-case extension without the dot.
func fileExt(name string) string {
	i := strings.LastIndex(name, ".")
	if i < 0 || i == len(name)-1 {
		return ""
	}
	return strings.ToLower(name[i+1:])
}
