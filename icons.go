package main

import (
	_ "embed"

	"github.com/egoist/mygo/ui"
)

// Icons, copied from the reference app's assets (assets/icons/*.svg, Fluent
// and Lucide) so the shapes match. Parse once at startup, never in a view.

//go:embed assets/icons/compose.svg
var composeSVG []byte

//go:embed assets/icons/panel-left.svg
var panelLeftSVG []byte

//go:embed assets/icons/settings.svg
var settingsSVG []byte

//go:embed assets/icons/folder.svg
var folderSVG []byte

//go:embed assets/icons/copy.svg
var copySVG []byte

//go:embed assets/icons/list.svg
var listSVG []byte

//go:embed assets/icons/file-diff.svg
var fileDiffSVG []byte

//go:embed assets/icons/globe.svg
var globeSVG []byte

//go:embed assets/icons/terminal.svg
var terminalSVG []byte

//go:embed assets/icons/chevron-right.svg
var chevronRightSVG []byte

//go:embed assets/icons/chevron-down.svg
var chevronDownSVG []byte

//go:embed assets/icons/arrow-left.svg
var arrowLeftSVG []byte

//go:embed assets/icons/arrow-right.svg
var arrowRightSVG []byte

//go:embed assets/icons/external-link.svg
var externalLinkSVG []byte

//go:embed assets/icons/list-filter.svg
var listFilterSVG []byte

var (
	iconCompose      = ui.MustParseSVG(composeSVG)
	iconPanelLeft    = ui.MustParseSVG(panelLeftSVG)
	iconSettings     = ui.MustParseSVG(settingsSVG)
	iconFolder       = ui.MustParseSVG(folderSVG)
	iconCopy         = ui.MustParseSVG(copySVG)
	iconList         = ui.MustParseSVG(listSVG)
	iconFileDiff     = ui.MustParseSVG(fileDiffSVG)
	iconGlobe        = ui.MustParseSVG(globeSVG)
	iconTerminal     = ui.MustParseSVG(terminalSVG)
	iconChevronRight = ui.MustParseSVG(chevronRightSVG)
	iconChevronDown  = ui.MustParseSVG(chevronDownSVG)
	iconArrowLeft    = ui.MustParseSVG(arrowLeftSVG)
	iconArrowRight   = ui.MustParseSVG(arrowRightSVG)
	iconExternalLink = ui.MustParseSVG(externalLinkSVG)
	iconListFilter   = ui.MustParseSVG(listFilterSVG)
)

// surfaceIcon is the icon a right-panel surface shows, as the reference's
// RightPanelSurface::icon_path does.
func surfaceIcon(name string) *ui.SVG {
	switch name {
	case "Files":
		return iconFolder
	case "Review":
		return iconFileDiff
	case "Browser":
		return iconGlobe
	case "Terminal":
		return iconTerminal
	default:
		return iconList
	}
}

// actionIconButton is a borderless icon-only button, as the reference's title
// bar and sidebar use: the icon takes the text colour, and Label names it for
// assistive technology (an icon alone is otherwise invisible to it).
func actionIconButton(c *ui.Context, s *ui.SVG, label string) ui.Element {
	el := ui.Button(c, "")
	el.Label(label)
	el.Children(func() {
		ui.Icon(c, s).Size(14, 14)
	})
	return el
}

// iconRow is a sidebar action row: a 20-wide icon slot and a label, as the
// reference's render_sidebar_action_row (h32, px4, gap10, icon 14, text 13).
func iconRow(c *ui.Context, t *ui.Theme, s *ui.SVG, label string) ui.Element {
	row := ui.Row(c).FillWidth().Height(sidebarActionHeight).Radius(7).
		Padding(0, 4).Gap(10).AlignItems(ui.Center)
	row.Children(func() {
		ui.Row(c).Width(20).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
			ui.Icon(c, s).Size(14, 14).TextColor(t.TextMuted)
		})
		ui.Text(c, label).FontSize(13).TextColor(t.TextMuted).Grow(1).
			MaxLines(1).Ellipsis("…")
	})
	return row
}
