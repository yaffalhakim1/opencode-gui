package src

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui/editor"

	"github.com/egoist/mygo/ui"

	"opencode-gui/internal/app"
	"opencode-gui/internal/opencode"
)

// newTestState is a state with the settings page open on a section.
func newTestState(section int) *state {
	st := &state{
		sessions:     &app.Sessions{},
		sidebarWidth: sidebarWidthDefault,
		ctx:          context.Background(),
		tab:          1,
	}
	st.set.open = true
	st.set.section = section
	st.set.theme = "System"
	st.set.uiFont = "14 px"
	st.set.codeFont = "14 px"
	st.set.daemonPort = "9680"
	return st
}

// TestSettingsSectionsRender builds every section without a window and
// checks the expected copy is on screen.
func TestSettingsSectionsRender(t *testing.T) {
	cases := []struct {
		section int
		want    string
	}{
		{secGeneral, "Local by default"},
		{secGeneral, "Automatic updates"},
		{secAppearance, "Theme"},
		{secAppearance, "UI font size"},
		{secProviders, "Providers"},
		{secMemory, "Project instructions"},
		{secDaemon, "Expose managed daemon"},
		{secDaemon, "Connection details"},
		{secDaemon, "Reach it from anywhere with Tailscale"},
	}
	for _, tc := range cases {
		st := newTestState(tc.section)
		tt := ui.NewTester(st.view, 1280, 800)
		if !tt.HasText(tc.want) {
			t.Errorf("section %d: %q not found in %q",
				tc.section, tc.want, tt.Texts())
		}
	}
}

// TestSkillsSectionRendersDetail checks the skills list and detail pane.
func TestSkillsSectionRendersDetail(t *testing.T) {
	st := newTestState(secSkills)
	st.set.skillsReady = true
	st.set.skills = []opencode.Skill{
		{ID: "anatomy", Name: "anatomy", Description: "Isometric SVG figures",
			Path: "/skills/anatomy/SKILL.md", Content: "# Anatomy\n\nMakes figures."},
	}
	st.set.skillSel = 0

	tt := ui.NewTester(st.view, 1280, 800)
	for _, want := range []string{"anatomy", "Isometric SVG figures", "/anatomy", "SKILL.md"} {
		if !tt.HasText(want) {
			t.Errorf("%q not found in %q", want, tt.Texts())
		}
	}
}

// TestUsageSectionRendersStats checks the usage numbers are formatted.
func TestUsageSectionRendersStats(t *testing.T) {
	st := newTestState(secUsage)
	st.set.usageReady = true
	st.set.usage = opencode.Stats{Sessions: 206}
	st.set.usage.Tokens.Input = 2.29e9
	st.set.usage.Tokens.Output = 9.97e6
	st.set.usage.Models = []opencode.ModelUsage{
		{Model: opencode.ModelRef{ID: "deepseek-v4-1-flash"}, Cost: 138.29},
	}

	tt := ui.NewTester(st.view, 1280, 800)
	for _, want := range []string{"2.29B", "$138.29", "deepseek-v4-1-flash", "Breakdown"} {
		if !tt.HasText(want) {
			t.Errorf("%q not found in %q", want, tt.Texts())
		}
	}
}

func TestHumanFormatters(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{2.35e9, "2.35B"},
		{9.97e6, "9.97M"},
		{1500, "1.5K"},
		{42, "42"},
	}
	for _, c := range cases {
		if got := humanTokens(c.in); got != c.want {
			t.Errorf("humanTokens(%v) = %q, want %q", c.in, got, c.want)
		}
	}
	if got := humanCost(138.29); got != "$138.29" {
		t.Errorf("humanCost = %q", got)
	}
}

func TestMaskHidesAllButLastFour(t *testing.T) {
	got := mask("abcdefghijklmnop")
	if got == "abcdefghijklmnop" {
		t.Fatal("token was not masked")
	}
	if got[len(got)-4:] != "mnop" {
		t.Errorf("mask kept %q, want the last four", got[len(got)-4:])
	}
}

func TestNavBackClosesSettings(t *testing.T) {
	st := newTestState(secGeneral)
	tt := ui.NewTester(st.view, 1280, 800)
	if err := tt.Click("← Back"); err != nil {
		t.Fatalf("click Back: %v", err)
	}
	if st.set.open {
		t.Error("Back did not close settings")
	}
}

// TestEverySwitchHasAnAccessibleName checks the labels ui.Switch requires:
// it draws no text of its own, so a switch without one is unnamed for
// assistive technology, which is an accessibility failure.
func TestEverySwitchHasAnAccessibleName(t *testing.T) {
	src, err := os.ReadFile("settings.go")
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(string(src), "\n") {
		if !strings.Contains(line, "ui.Switch(") {
			continue
		}
		if !strings.Contains(line, ".Label(") {
			t.Errorf("settings.go:%d: a switch has no Label: %s", i+1, strings.TrimSpace(line))
		}
	}
}

// TestBreakdownUsesATable checks the usage breakdown is a framework table,
// which sorts, resizes and reads its rows, rather than hand-built rows.
func TestBreakdownUsesATable(t *testing.T) {
	src, err := os.ReadFile("settings.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "ui.Table(") {
		t.Error("the usage breakdown should use ui.Table")
	}
	if !strings.Contains(string(src), "ui.TableColumn{") {
		t.Error("the breakdown table should declare its columns")
	}
}

// TestFilesSurfaceUsesATree checks the file browser is a framework tree.
func TestFilesSurfaceUsesATree(t *testing.T) {
	src, err := os.ReadFile("panel.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "ui.Tree(") {
		t.Error("the Files surface should use ui.Tree")
	}
}

// TestDiffViewerEnablesReviewFeatures checks the review surface turns on the
// viewer's hunk actions and reports its statistics.
func TestDiffViewerEnablesReviewFeatures(t *testing.T) {
	src, err := os.ReadFile("panel.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(src)
	for _, want := range []string{"HunkActions: true", "ds.Stats()", "ds.GoToHunk(", "ds.Hunks()"} {
		if !strings.Contains(s, want) {
			t.Errorf("the diff viewer does not use %s", want)
		}
	}
}

func TestDiffStatsLine(t *testing.T) {
	var ds editor.DiffState
	ds.SetPatch(editor.FilePatch{})
	got := diffStats(&ds)
	if !strings.Contains(got, "+") || !strings.Contains(got, "−") {
		t.Errorf("diffStats = %q, want added and deleted counts", got)
	}
}

func TestPlural(t *testing.T) {
	if plural(1) != "" {
		t.Error("one change should not be plural")
	}
	if plural(0) != "s" || plural(2) != "s" {
		t.Error("zero and many changes should be plural")
	}
}
