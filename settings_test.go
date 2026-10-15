package main

import (
	"context"
	"testing"

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
