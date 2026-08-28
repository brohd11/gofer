package app

import (
	"reflect"
	"runtime"
	"testing"
)

// TestEditorArgv walks the whole ladder. The last case is the one the row's contract rests
// on: with nothing set it must still name an editor, because a menu row that resolves to
// nothing is a row that does nothing when picked.
func TestEditorArgv(t *testing.T) {
	fallback := "vi"
	if runtime.GOOS == "windows" {
		fallback = "notepad"
	}

	cases := []struct {
		why    string
		editor string
		visual string
		want   []string
	}{
		{"$EDITOR wins", "nano", "emacs", []string{"nano"}},
		{"$VISUAL answers when $EDITOR is unset", "", "emacs", []string{"emacs"}},
		{"flags survive as separate arguments", "code --wait", "", []string{"code", "--wait"}},
		{"a quoted path with a space stays one argument",
			`"/opt/my editor/edit" --flag`, "", []string{"/opt/my editor/edit", "--flag"}},
		// EDITOR= in a profile is how a shell user turns the setting off; it must fall
		// through rather than resolve to an empty command.
		{"whitespace-only counts as unset", "   ", "emacs", []string{"emacs"}},
		// An unterminated quote is the only thing shellquote.Split refuses. A typo in a
		// profile must not be the reason nothing opens.
		{"an unparseable value falls through", `editor "unfinished`, "emacs", []string{"emacs"}},
		{"nothing set falls back", "", "", []string{fallback}},
	}

	for _, tc := range cases {
		t.Setenv("EDITOR", tc.editor)
		t.Setenv("VISUAL", tc.visual)
		if got := editorArgv(); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("editorArgv() = %#v, want %#v — %s", got, tc.want, tc.why)
		}
	}
}
