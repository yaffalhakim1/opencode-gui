package main

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"opencode-gui/internal/app"
)

// The view runs without a window in tests, which click and type as a user
// would.
func TestView(t *testing.T) {
	st := &state{
		sessions: &app.Sessions{},
		title:    map[string]string{},
		replies:  map[string]string{},
		ids:      []string{"ses_test"},
		active:   "ses_test",
	}
	st.replies["ses_test"] = "hello reply"
	tt := ui.NewTester(st.view, 1100, 760)
	if !tt.HasText("hello reply") {
		t.Errorf("reply text missing, texts %q", tt.Texts())
	}
	if err := tt.Click("Send"); err != nil {
		t.Fatal(err)
	}
	if st.busy {
		t.Error("empty prompt should not set busy")
	}
}
