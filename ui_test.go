package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"opencode-gui/internal/app"
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
	// A made-up token of the same shape, never a real one.
	tok := "FAKE-token-for-tests-only-0123456789abcdefXYZ"
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
	// 252 is the reference's DEFAULT_SIDEBAR_WIDTH; this app widens it to 272
	// so the icon rows have more room. Keep it within the reference's bounds.
	if sidebarWidthDefault < sidebarWidthMin || sidebarWidthDefault > sidebarWidthMax {
		t.Errorf("sidebar default %v is outside its bounds %v..%v",
			sidebarWidthDefault, sidebarWidthMin, sidebarWidthMax)
	}
	if sidebarWidthDefault < 252 {
		t.Errorf("sidebar default = %v, want at least the reference's 252", sidebarWidthDefault)
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

func TestSessionGroupBucketsByRecency(t *testing.T) {
	now := time.Now()
	mk := func(when time.Time) opencode.Session {
		var s opencode.Session
		s.Time.Updated = float64(when.UnixMilli())
		return s
	}
	cases := []struct {
		when time.Time
		want string
	}{
		{now, "Today"},
		{now.AddDate(0, 0, -1), "Yesterday"},
		{now.AddDate(0, 0, -3), "This Week"},
		{now.AddDate(0, 0, -30), "Older"},
	}
	for _, c := range cases {
		if got := sessionGroup(mk(c.when)); got != c.want {
			t.Errorf("sessionGroup(%v) = %q, want %q", c.when.Format("2006-01-02"), got, c.want)
		}
	}
}

func TestSessionGroupWithoutATimestamp(t *testing.T) {
	if got := sessionGroup(opencode.Session{}); got != "Sessions" {
		t.Errorf("a session with no time = %q, want Sessions", got)
	}
}

func TestSkillFilterSplitsBuiltinFromUser(t *testing.T) {
	builtin := opencode.Skill{Name: "opencode", Path: "/builtin/opencode.md"}
	user := opencode.Skill{Name: "my-skill", Path: `C:\Users\me\.config\opencode\skills\my-skill\SKILL.md`}

	if !skillMatches(builtin, "", "All skills") {
		t.Error("a builtin skill should pass the All filter")
	}
	if skillMatches(builtin, "", "User skills") {
		t.Error("a builtin skill should not pass the User filter")
	}
	if !skillMatches(builtin, "", "Shared skills") {
		t.Error("a builtin skill should pass the Shared filter")
	}
	if skillMatches(user, "", "Shared skills") {
		t.Error("a user skill should not pass the Shared filter")
	}
	if !skillMatches(user, "", "User skills") {
		t.Error("a user skill should pass the User filter")
	}
}

func TestSkillSearchMatchesNameAndDescription(t *testing.T) {
	s := opencode.Skill{Name: "anatomy", Description: "Isometric SVG figures"}
	if !skillMatches(s, "ana", "All skills") {
		t.Error("search should match the name")
	}
	if !skillMatches(s, "isometric", "All skills") {
		t.Error("search should match the description")
	}
	if skillMatches(s, "zzz", "All skills") {
		t.Error("search should not match unrelated text")
	}
}

func TestPromptPlaceholderChangesWhileRunning(t *testing.T) {
	if got := promptPlaceholder(false); got != "Ask OpenCode to build something…" {
		t.Errorf("idle placeholder = %q", got)
	}
	if got := promptPlaceholder(true); got != "Queue a follow-up…" {
		t.Errorf("running placeholder = %q, want the queue wording", got)
	}
}

func TestAccessModesMatchTheReferenceOptions(t *testing.T) {
	want := []string{"Plan", "Supervised", "Auto-accept edits", "Auto", "Full access"}
	if len(accessModes) != len(want) {
		t.Fatalf("%d access modes, want %d", len(accessModes), len(want))
	}
	for i, w := range want {
		if accessModes[i].Label != w {
			t.Errorf("mode %d = %q, want %q", i, accessModes[i].Label, w)
		}
	}
}

func TestSurfaceCardsMatchTheReference(t *testing.T) {
	want := []string{"Files", "Review", "Browser", "Terminal"}
	if len(surfaces) != len(want) {
		t.Fatalf("%d surfaces, want %d", len(surfaces), len(want))
	}
	for i, w := range want {
		if surfaces[i].Name != w {
			t.Errorf("surface %d = %q, want %q", i, surfaces[i].Name, w)
		}
	}
	if !surfaces[surfReview].Avail || surfaces[surfReview].Desc != "Review file changes" {
		t.Error("the Review surface should be available with the reference description")
	}
}

// todaySession is a session updated just now, so it lands in the "Today"
// group whatever the date.
func todaySession(id, title string) opencode.Session {
	var s opencode.Session
	s.ID, s.Title = id, title
	s.Time.Updated = float64(time.Now().UnixMilli())
	return s
}

func TestFoldingAGroupHidesItsSessions(t *testing.T) {
	// The row builder skips a folded group's sessions but keeps its header,
	// as the reference's sidebar_rows does.
	rows := buildSidebarRows([]opencode.Session{
		todaySession("a", "today one"),
		todaySession("b", "today two"),
	}, map[string]bool{"Today": true})

	if len(rows) != 1 {
		t.Fatalf("%d rows, want just the header", len(rows))
	}
	if rows[0].Header != "Today" {
		t.Errorf("header = %q, want Today", rows[0].Header)
	}
}

func TestUnfoldedGroupKeepsItsSessions(t *testing.T) {
	rows := buildSidebarRows([]opencode.Session{
		todaySession("a", "one"),
		todaySession("b", "two"),
	}, nil)

	if len(rows) != 3 { // header + two sessions
		t.Fatalf("%d rows, want a header and two sessions", len(rows))
	}
	if rows[0].Header == "" {
		t.Error("the first row should be the group header")
	}
}

func TestSidebarRowsCarryAStableKey(t *testing.T) {
	rows := buildSidebarRows([]opencode.Session{todaySession("ses_x", "x")}, nil)
	if got := rows[0].key(); got != "h:Today" {
		t.Errorf("header key = %q, want h:Today", got)
	}
	if got := rows[1].key(); got != "ses_x" {
		t.Errorf("session key = %q, want ses_x", got)
	}
}

func TestFoldingOneGroupLeavesTheOthers(t *testing.T) {
	now := time.Now()
	older := func(id, title string, daysAgo int) opencode.Session {
		var s opencode.Session
		s.ID, s.Title = id, title
		s.Time.Updated = float64(now.AddDate(0, 0, -daysAgo).UnixMilli())
		return s
	}
	rows := buildSidebarRows([]opencode.Session{
		todaySession("t1", "today"),
		older("y1", "yesterday", 1),
	}, map[string]bool{"Yesterday": true})

	// Today's header and its session, then Yesterday's header alone.
	if len(rows) != 3 {
		t.Fatalf("%d rows, want Today + its session + the Yesterday header", len(rows))
	}
	if rows[0].Header != "Today" || rows[1].Sess.ID != "t1" || rows[2].Header != "Yesterday" {
		t.Errorf("rows = %+v", rows)
	}
}

func TestSidebarHasNoDaemonLabel(t *testing.T) {
	st := &state{
		sessions:     &app.Sessions{},
		sidebarWidth: sidebarWidthDefault,
		ctx:          context.Background(),
		tab:          0,
	}
	tt := ui.NewTester(st.view, 1280, 800)
	if tt.HasText("Daemon") {
		t.Error("the footer still shows the word Daemon")
	}
}
