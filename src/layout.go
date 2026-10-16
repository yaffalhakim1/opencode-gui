package src

import "math"

// Panel sizing, ported from the reference app (waku src/app.rs) so both apps
// shrink the panels the same way when the window narrows. Every function here
// is pure, so the layout rules are testable without a window.

const (
	sidebarMinWidth   = 180.0 // SIDEBAR_MIN_WIDTH
	sidebarMaxWidth   = 420.0 // SIDEBAR_MAX_WIDTH
	panelMinWidth     = 280.0 // RIGHT_PANEL_MIN_WIDTH
	panelMaxWidth     = 1000.0
	mainPanelMinWidth = 360.0 // MAIN_PANEL_MIN_WIDTH: the chat never goes below this
)

// sanitizePanelWidth clamps a remembered width to its bounds, falling back to
// the default when it is not a finite number (a corrupt saved state, a NaN).
func sanitizePanelWidth(width, def, min, max float32) float32 {
	if isFinite(width) {
		return clamp(width, min, max)
	}
	return def
}

// fittedPanelWidths returns the sidebar and right-panel widths that leave the
// chat at least mainPanelMinWidth, shrinking the right panel first and the
// sidebar second. Hidden panels take no room.
func fittedPanelWidths(viewportWidth float32, sidebarVisible, panelVisible bool, sidebarWidth, panelWidth float32) (float32, float32) {
	sidebarMin := float32(0)
	if sidebarVisible {
		sidebarMin = sidebarMinWidth
	}
	panelMin := float32(0)
	if panelVisible {
		panelMin = panelMinWidth
	}

	var sidebar, panel float32
	if sidebarVisible {
		sidebar = sanitizePanelWidth(sidebarWidth, sidebarWidthDefault, sidebarMinWidth, sidebarMaxWidth)
	}
	if panelVisible {
		panel = sanitizePanelWidth(panelWidth, panelWidthDefault, panelMinWidth, panelMaxWidth)
	}

	available := max32(viewportWidth-mainPanelMinWidth, 0)
	overflow := max32(sidebar+panel-available, 0)

	rightReduction := min32(overflow, max32(panel-panelMin, 0))
	panel -= rightReduction
	overflow -= rightReduction

	sidebarReduction := min32(overflow, max32(sidebar-sidebarMin, 0))
	sidebar -= sidebarReduction
	overflow -= sidebarReduction

	// A host may report a viewport smaller than the configured minimum during
	// a resize or display transition; this last step protects the layout.
	if overflow > 0 {
		rightReduction := min32(overflow, panel)
		panel -= rightReduction
		overflow -= rightReduction
		sidebar = max32(sidebar-overflow, 0)
	}

	return sidebar, panel
}

func clamp(v, lo, hi float32) float32 {
	return min32(max32(v, lo), hi)
}

func min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// isFinite reports whether v is a real number: not NaN and not infinite.
func isFinite(v float32) bool {
	return !math.IsNaN(float64(v)) && !math.IsInf(float64(v), 0)
}
