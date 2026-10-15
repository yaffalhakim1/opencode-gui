package main

import "github.com/egoist/mygo/ui"

// darkUI toggles the graphite look. The reference (Kerenzikov light) is the
// default; set this true for its dark palette.
const darkUI = false

// kerenzikovTheme maps the Kerenzikov/waku palette (graphite surfaces,
// green accent reserved for meaning) onto MyGo's Theme. Colors are taken
// from that project's theme.rs so the app matches its look.
func kerenzikovTheme(dark bool) *ui.Theme {
	if !darkUI {
		dark = false
	}
	t := *ui.LightTheme()
	t.Dark = false

	t.Background = ui.Hex("#F6F5F6") // canvas
	t.Surface = ui.Hex("#ECECEC")    // raised
	t.SurfaceHover = ui.Hex("#E4E4E4")
	t.SurfacePressed = ui.Hex("#DEDEDE")
	t.Border = ui.Hex("#E6E6E6")
	t.Text = ui.Hex("#242424")
	t.TextMuted = ui.Hex("#858585")
	t.Accent = ui.Hex("#00DA7D") // brand green
	t.AccentHover = ui.Hex("#00C470")
	t.AccentPressed = ui.Hex("#00AE63")
	t.AccentText = ui.Hex("#0A1A12")
	t.Danger = ui.Hex("#C64A42")
	t.Warning = ui.Hex("#A66B20")
	t.Success = ui.Hex("#2F8F52")
	t.Selection = ui.Hex("#BBD7FA")
	t.Focus = ui.Hex("#00DA7D")
	t.Inverse = ui.Hex("#202227")
	t.InverseText = ui.Hex("#F8F8F9")
	t.Scrollbar = ui.Hex("#C9C9C9")
	t.ScrollbarWidth = 8
	t.Radius = 10 // Kerenzikov's rounded cards
	t.Spacing = 4
	t.FontSize = 14

	if dark {
		t.Dark = true
		t.Background = ui.Hex("#1A1A1A")
		t.Surface = ui.Hex("#232323")
		t.SurfaceHover = ui.Hex("#2A2A2A")
		t.SurfacePressed = ui.Hex("#303030")
		t.Border = ui.Hex("#2E2E2E")
		t.Text = ui.Hex("#E2E2E2")
		t.TextMuted = ui.Hex("#7D7D7D")
		t.Accent = ui.Hex("#26E085")
		t.AccentHover = ui.Hex("#35EA92")
		t.AccentPressed = ui.Hex("#1FCE79")
		t.AccentText = ui.Hex("#08211A")
		t.Danger = ui.Hex("#E2726A")
		t.Warning = ui.Hex("#E0B36A")
		t.Success = ui.Hex("#62C987")
		t.Selection = ui.Hex("#3B62A8")
		t.Focus = ui.Hex("#26E085")
		t.Inverse = ui.Hex("#E7E9EC")
		t.InverseText = ui.Hex("#17181C")
		t.Scrollbar = ui.Hex("#3A3A3A")
	}
	return &t
}

// Card and layout helpers that give Kerenzikov's look: soft raised cards,
// generous radii, quiet borders.

// sidebarColor is the source list's own surface, a touch distinct from the
// canvas, as the reference app's sidebar is.
func sidebarColor(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#181818")
	}
	return ui.Hex("#F3F3F3")
}

// card styles a container as a Kerenzikov panel.
func card(c *ui.Context, t *ui.Theme) ui.Element {
	return ui.Column(c).Radius(16).Padding(16).
		Background(t.Surface).
		Border(1, t.Border)
}

// pill styles a small rounded chip, as the model/effort/access chips.
func pill(c *ui.Context, t *ui.Theme) ui.Element {
	return ui.Row(c).Radius(999).Padding(4, 10).Gap(6).
		Background(t.Background).Border(1, t.Border)
}
