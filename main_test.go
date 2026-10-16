package main

import (
	"context"
	"testing"

	"github.com/egoist/mygo/ui"

	"opencode-gui/internal/app"
)

// The view runs without a window in tests, which click and type as a user
// would.
func TestViewRendersMessages(t *testing.T) {
	st := &state{
		sessions:     &app.Sessions{},
		sidebarWidth: 240,
		ctx:          context.Background(),
		tab:          1,
		messages: []message{
			{ID: "m1", Role: "user", Text: "hello there", Done: true},
			{ID: "m2", Role: "assistant", Text: "hi!", Done: true},
		},
	}
	st.chatList.Key = func(i int) any { return st.messages[i].ID }

	tt := ui.NewTester(st.view, 1280, 820)
	if !tt.HasText("hello there") {
		t.Errorf("user message missing, texts %q", tt.Texts())
	}
	if !tt.HasText("hi!") {
		t.Errorf("assistant message missing, texts %q", tt.Texts())
	}
}

func TestSubmitIgnoresEmptyPrompt(t *testing.T) {
	st := &state{
		sessions: &app.Sessions{},
		ctx:      context.Background(),
		tab:      1,
	}
	st.prompt = "   "
	st.submit() // must not panic or block
	if st.running {
		t.Error("empty prompt should not start a run")
	}
}

func TestPanelToggles(t *testing.T) {
	st := &state{
		sessions:     &app.Sessions{},
		sidebarWidth: sidebarWidthDefault,
		ctx:          context.Background(),
		tab:          0,
	}
	// The toggles are icon-only buttons, named for assistive technology, so
	// they are reached by their labels.
	tt := ui.NewTester(st.view, 1280, 800)

	if err := tt.Click("Toggle left panel"); err != nil {
		t.Fatalf("click left toggle: %v", err)
	}
	if !st.leftCollapsed {
		t.Error("left panel did not collapse")
	}

	// With the sidebar gone, the toolbar shows the button that brings it back.
	tt2 := ui.NewTester(st.view, 1280, 800)
	if err := tt2.Click("Show left panel"); err != nil {
		t.Fatalf("click show-left: %v", err)
	}
	if st.leftCollapsed {
		t.Error("left panel did not expand")
	}

	// Collapse the right panel.
	tt3 := ui.NewTester(st.view, 1280, 800)
	if err := tt3.Click("Toggle right panel"); err != nil {
		t.Fatalf("click right toggle: %v", err)
	}
	if !st.rightCollapsed {
		t.Error("right panel did not collapse")
	}
}

// TestSidebarRowsAreIconsNotFilledButtons checks the sidebar's actions follow
// the reference: a plain icon+label row, not a filled accent button.
func TestSidebarRowsAreIconsNotFilledButtons(t *testing.T) {
	st := &state{
		sessions:     &app.Sessions{},
		sidebarWidth: sidebarWidthDefault,
		ctx:          context.Background(),
		tab:          0,
	}
	tt := ui.NewTester(st.view, 1280, 800)

	if tt.HasText("New session") {
		t.Error("the sidebar still shows the old filled \"New session\" button")
	}
	if !tt.HasText("New task") {
		t.Errorf("the sidebar should show the \"New task\" action row, texts %q", tt.Texts())
	}
	if !tt.HasText("Filter tasks") {
		t.Error("the filter should be labelled \"Filter tasks\"")
	}
}
