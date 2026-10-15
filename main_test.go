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
	tt := ui.NewTester(st.view, 1280, 800)

	// Collapse the left panel.
	if err := tt.Click("◧"); err != nil {
		t.Fatalf("click left toggle: %v", err)
	}
	if !st.leftCollapsed {
		t.Error("left panel did not collapse")
	}
	if !tt.HasText("▣") {
		t.Errorf("left glyph did not change, texts %q", tt.Texts())
	}

	// Collapse the right panel.
	if err := tt.Click("◨"); err != nil {
		t.Fatalf("click right toggle: %v", err)
	}
	if !st.rightCollapsed {
		t.Error("right panel did not collapse")
	}

	// Expand both again.
	tt2 := ui.NewTester(st.view, 1280, 800)
	if err := tt2.Click("▣"); err != nil {
		t.Fatalf("expand: %v", err)
	}
	if st.leftCollapsed {
		t.Error("left panel did not expand")
	}
}
