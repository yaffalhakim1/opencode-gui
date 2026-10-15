package main

import (
	"math"
	"testing"
)

// These mirror the reference app's layout tests (waku src/app/tests.rs), with
// the same inputs and the same expected numbers, so both apps shrink panels
// identically.

func TestPanelWidthsPreserveMainContentWhenTheWindowNarrows(t *testing.T) {
	sidebar, panel := fittedPanelWidths(980, true, true, 420, 720)

	if sidebar != 340 {
		t.Errorf("sidebar = %v, want 340", sidebar)
	}
	if panel != 280 {
		t.Errorf("right panel = %v, want 280", panel)
	}
	if got := 980 - sidebar - panel; got != 360 {
		t.Errorf("chat width = %v, want 360", got)
	}
}

func TestHiddenPanelsDoNotConsumeLayoutWidth(t *testing.T) {
	sidebar, panel := fittedPanelWidths(980, false, true, 420, 720)

	if sidebar != 0 {
		t.Errorf("hidden sidebar took %v, want 0", sidebar)
	}
	if panel != 620 {
		t.Errorf("right panel = %v, want 620", panel)
	}
}

func TestWideWindowKeepsRequestedWidths(t *testing.T) {
	sidebar, panel := fittedPanelWidths(1600, true, true, sidebarWidthDefault, panelWidthDefault)

	if sidebar != sidebarWidthDefault {
		t.Errorf("sidebar = %v, want %v", sidebar, float32(sidebarWidthDefault))
	}
	if panel != panelWidthDefault {
		t.Errorf("right panel = %v, want %v", panel, float32(panelWidthDefault))
	}
}

func TestRightPanelShrinksBeforeTheSidebar(t *testing.T) {
	// 900 - 360 = 540 available for 252 + 460 = 712: overflow 172 comes out of
	// the right panel first, which has room to 280.
	sidebar, panel := fittedPanelWidths(900, true, true, sidebarWidthDefault, panelWidthDefault)

	if sidebar != sidebarWidthDefault {
		t.Errorf("sidebar shrank to %v, want %v kept", sidebar, float32(sidebarWidthDefault))
	}
	if panel != 460-172 {
		t.Errorf("right panel = %v, want %v", panel, float32(460-172))
	}
}

func TestSidebarShrinksOnlyAfterTheRightPanelHitsItsMinimum(t *testing.T) {
	// At 880 the panels have 520. Right panel gives its full 180 (460→280),
	// then the sidebar gives the remaining 12 (252→240).
	sidebar, panel := fittedPanelWidths(880, true, true, sidebarWidthDefault, panelWidthDefault)

	if panel != panelMinWidth {
		t.Errorf("right panel = %v, want its minimum %v", panel, float32(panelMinWidth))
	}
	if sidebar != 240 {
		t.Errorf("sidebar = %v, want 240", sidebar)
	}
	if sidebar < sidebarMinWidth {
		t.Errorf("sidebar %v went below its minimum %v", sidebar, float32(sidebarMinWidth))
	}
}

func TestPanelWidthsSurviveATinyViewport(t *testing.T) {
	sidebar, panel := fittedPanelWidths(320, true, true, sidebarWidthDefault, panelWidthDefault)

	if sidebar < 0 || panel < 0 {
		t.Fatalf("negative widths: sidebar %v, panel %v", sidebar, panel)
	}
	if sidebar+panel > 320 {
		t.Errorf("panels %v exceed the 320 viewport", sidebar+panel)
	}
}

func TestSanitizePanelWidthClampsAndFallsBack(t *testing.T) {
	if got := sanitizePanelWidth(500, 252, 180, 420); got != 420 {
		t.Errorf("above max = %v, want 420", got)
	}
	if got := sanitizePanelWidth(100, 252, 180, 420); got != 180 {
		t.Errorf("below min = %v, want 180", got)
	}
	if got := sanitizePanelWidth(float32(math.NaN()), 252, 180, 420); got != 252 {
		t.Errorf("NaN = %v, want the default 252", got)
	}
	if got := sanitizePanelWidth(float32(math.Inf(1)), 252, 180, 420); got != 252 {
		t.Errorf("+Inf = %v, want the default 252", got)
	}
}
