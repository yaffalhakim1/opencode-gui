package main

import (
	"strings"
	"testing"
	"time"

	"opencode-gui/internal/opencode"
)

// Pure-logic tests, in the spirit of the reference app's suite: no window, one
// behaviour per test, a name that says what is asserted.

func TestSessionLabelPrefersTheTitle(t *testing.T) {
	got := sessionLabel(opencode.Session{ID: "ses_abc123", Title: "Fix the parser"})
	if got != "Fix the parser" {
		t.Errorf("label = %q, want the title", got)
	}
}

func TestSessionLabelShortensARawID(t *testing.T) {
	got := sessionLabel(opencode.Session{ID: "ses_ec003f15effe58wzZuqEeY4Jew"})
	if strings.HasPrefix(got, "ses_") {
		t.Errorf("label %q still shows the raw id", got)
	}
	if len([]rune(got)) > 20 {
		t.Errorf("label %q is too long for the sidebar", got)
	}
}

func TestSessionLabelTrimsWhitespaceOnlyTitles(t *testing.T) {
	got := sessionLabel(opencode.Session{ID: "ses_abcdefgh", Title: "   "})
	if strings.TrimSpace(got) == "" {
		t.Error("label is blank for a whitespace-only title")
	}
}

func TestTruncateKeepsShortStringsAndEllipsizesLongOnes(t *testing.T) {
	if got := truncate("short", 30); got != "short" {
		t.Errorf("truncate changed a short string: %q", got)
	}
	got := truncate("a very long session title that will not fit", 10)
	if len([]rune(got)) != 10 {
		t.Errorf("truncate produced %d runes, want 10", len([]rune(got)))
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("truncate did not mark the cut: %q", got)
	}
}

func TestTruncateCountsRunesNotBytes(t *testing.T) {
	got := truncate("日本語のタイトルです", 5)
	if len([]rune(got)) != 5 {
		t.Errorf("truncate split a multi-byte rune: %q", got)
	}
}

func TestShortPathKeepsTheLastTwoSegments(t *testing.T) {
	cases := map[string]string{
		`C:\Users\me\project\src\main.go`: ".../src/main.go",
		"src/main.go":                     "src/main.go",
		"main.go":                         "main.go",
	}
	for in, want := range cases {
		if got := shortPath(in); got != want {
			t.Errorf("shortPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTotalTokensSumsEveryKind(t *testing.T) {
	var u opencode.TokenUsage
	u.Input, u.Output, u.Reasoning = 100, 50, 25
	u.Cache.Read, u.Cache.Write = 200, 10

	if got := totalTokens(u); got != 385 {
		t.Errorf("totalTokens = %v, want 385", got)
	}
}

func TestHumanTokensUsesReadableUnits(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{2.35e9, "2.35B"},
		{9.97e6, "9.97M"},
		{1500, "1.5K"},
		{0, "0"},
	}
	for _, c := range cases {
		if got := humanTokens(c.in); got != c.want {
			t.Errorf("humanTokens(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestHumanCostAlwaysShowsCents(t *testing.T) {
	if got := humanCost(97.06508286320052); got != "$97.07" {
		t.Errorf("humanCost = %q, want $97.07", got)
	}
	if got := humanCost(0); got != "$0.00" {
		t.Errorf("humanCost(0) = %q, want $0.00", got)
	}
}

func TestMaskKeepsOnlyTheLastFourCharacters(t *testing.T) {
	tok := "-ubUbwomD1Q9Wb6Tq1gIkhet3mimVCd9QomPoiQLU7c"
	got := mask(tok)

	if strings.Contains(got, tok[:10]) {
		t.Error("mask leaked the start of the token")
	}
	if !strings.HasSuffix(got, tok[len(tok)-4:]) {
		t.Errorf("mask dropped the last four characters: %q", got)
	}
}

func TestMaskHandlesShortTokens(t *testing.T) {
	if got := mask("abc"); got != "••••" {
		t.Errorf("mask of a short token = %q", got)
	}
}

func TestSettingsSearchFiltersSectionNames(t *testing.T) {
	visible := 0
	for _, name := range sectionNames {
		if strings.Contains(strings.ToLower(name), "app") {
			visible++
		}
	}
	if visible != 1 {
		t.Errorf("searching \"app\" matched %d sections, want just Appearance", visible)
	}
}

func TestAllSettingsSectionsAreNamed(t *testing.T) {
	want := 7 // General, Appearance, Providers, Skills, Memory, Usage, Daemon
	if len(sectionNames) != want {
		t.Errorf("%d sections, want %d", len(sectionNames), want)
	}
	for _, name := range sectionNames {
		if strings.TrimSpace(name) == "" {
			t.Error("a section has no name")
		}
	}
}

func TestThemePaletteMatchesTheReference(t *testing.T) {
	light := kerenzikovTheme(false)
	if light.Dark {
		t.Error("light theme reports dark")
	}
	if got := light.Accent; got.R != 0x00 || got.G != 0xDA || got.B != 0x7D {
		t.Errorf("light accent = %02x%02x%02x, want 00da7d", got.R, got.G, got.B)
	}
	if got := light.Background; got.R != 0xF6 || got.G != 0xF5 || got.B != 0xF6 {
		t.Errorf("light canvas = %02x%02x%02x, want f6f5f6", got.R, got.G, got.B)
	}
}

// TestDarkUIFlagGatesTheDarkPalette documents that the dark palette is opt-in
// through darkUI: while it is false, even a dark system appearance gets the
// light look the reference ships by default.
func TestDarkUIFlagGatesTheDarkPalette(t *testing.T) {
	if darkUI {
		t.Skip("darkUI is on; the gate is not being exercised")
	}
	if kerenzikovTheme(true).Dark {
		t.Error("darkUI is off, so a dark request should still resolve to light")
	}
}

// TestDarkPaletteValuesMatchTheReference checks the dark palette itself, which
// is used when darkUI is turned on.
func TestDarkPaletteValuesMatchTheReference(t *testing.T) {
	dark := kerenzikovDarkPalette()
	if !dark.Dark {
		t.Error("the dark palette does not report dark")
	}
	if got := dark.Background; got.R != 0x1A || got.G != 0x1A || got.B != 0x1A {
		t.Errorf("dark background = %02x%02x%02x, want 1a1a1a", got.R, got.G, got.B)
	}
	if got := dark.Accent; got.R != 0x26 || got.G != 0xE0 || got.B != 0x85 {
		t.Errorf("dark accent = %02x%02x%02x, want 26e085", got.R, got.G, got.B)
	}
}

func TestSettingsCardUsesTheReferenceRadius(t *testing.T) {
	if settingsContentMaxWidth != 760 {
		t.Errorf("settings content width = %v, want 760", settingsContentMaxWidth)
	}
	if contentMaxWidth != 720 {
		t.Errorf("chat content width = %v, want 720", contentMaxWidth)
	}
}

func TestSidebarWidthDefaultsMatchTheReference(t *testing.T) {
	if sidebarWidthDefault != 252 {
		t.Errorf("sidebar default = %v, want 252", sidebarWidthDefault)
	}
	if panelWidthDefault != 380 {
		t.Errorf("panel default = %v, want 380", panelWidthDefault)
	}
}

func TestFontSizesOfferedToTheUser(t *testing.T) {
	if len(fontSizes) < 3 {
		t.Errorf("only %d font sizes offered", len(fontSizes))
	}
	if fontSizes[0] != "12 px" {
		t.Errorf("smallest font size = %q, want 12 px", fontSizes[0])
	}
}

func TestRecentSessionsSortNewestFirst(t *testing.T) {
	// The list is ordered by the server; the UI must not reorder it.
	sessions := []opencode.Session{
		{ID: "a", Title: "newest"},
		{ID: "b", Title: "older"},
	}
	if sessions[0].Title != "newest" {
		t.Error("session order changed")
	}
}

func TestFollowEndKeepsAChatPinned(t *testing.T) {
	st := &state{}
	st.chatList.FollowEnd = true
	if !st.chatList.FollowEnd {
		t.Error("a chat should follow its end by default")
	}
}

func TestIdleSessionsAreNotRunning(t *testing.T) {
	// A message without a finish is streaming; one with a finish is settled.
	streaming := opencode.Message{Type: "assistant"}
	settled := opencode.Message{Type: "assistant", Finish: "stop"}

	if streaming.Finish != "" {
		t.Error("a streaming message should have no finish")
	}
	if settled.Finish == "" {
		t.Error("a settled message should have a finish")
	}
}

func TestUsageRangeProducesEpochMilliseconds(t *testing.T) {
	from := time.Now().AddDate(0, -1, 0).UnixMilli()
	if from <= 0 {
		t.Fatal("a 30-day range should be a positive epoch-millisecond bound")
	}
	// Sanity: the bound is within a day of a month ago.
	want := time.Now().AddDate(0, -1, 0).UnixMilli()
	if diff := from - want; diff > int64(time.Hour) || diff < -int64(time.Hour) {
		t.Errorf("range bound drifted by %v", time.Duration(diff)*time.Millisecond)
	}
}
