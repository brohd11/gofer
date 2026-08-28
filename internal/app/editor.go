package app

import (
	"os"
	"runtime"
	"strings"

	"github.com/brohd11/goutil/shellquote"
)

// editorArgv is the command that opens a file for editing: $EDITOR, then $VISUAL, then the
// editor every system has. The result is never empty — the menu row's whole contract is
// that it opens something, and a row that quietly does nothing on a machine with no $EDITOR
// is worse than one that lands you in vi.
//
// That is the deliberate difference from goutil/configcmd's runEditor, which refuses with
// "neither EDITOR nor VISUAL is set". A CLI can print that and exit; a TUI menu row cannot.
//
// The variable is parsed with shellquote.Split rather than split on whitespace, so
// "code --wait" arrives as two arguments and "/path with spaces/editor" stays one. An
// unterminated quote — Split's only error — falls through to the next rung rather than
// failing the pick, on the same reasoning: a typo in a shell profile must not be the reason
// nothing opens.
func editorArgv() []string {
	// An unset variable and one set to whitespace are both absent: EDITOR= in a profile is
	// how a shell user turns a setting off (the convention goutil/envopt already follows).
	for _, name := range []string{"EDITOR", "VISUAL"} {
		raw := strings.TrimSpace(os.Getenv(name))
		if raw == "" {
			continue
		}
		if argv, err := shellquote.Split(raw); err == nil && len(argv) > 0 {
			return argv
		}
	}
	return []string{fallbackEditor()}
}

// fallbackEditor is the editor present on a machine that names none: vi is in POSIX, and
// windows has neither it nor a $EDITOR convention.
func fallbackEditor() string {
	if runtime.GOOS == "windows" {
		return "notepad"
	}
	return "vi"
}
