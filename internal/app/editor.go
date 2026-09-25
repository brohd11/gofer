package app

import (
	"os"
	"runtime"
	"strings"

	"github.com/brohd11/goutil/shellquote"
)

// editorArgv returns $EDITOR, then $VISUAL, then vi (notepad on Windows). It never fails: a
// malformed variable falls through to the next.
func editorArgv() []string {
	for _, name := range []string{"EDITOR", "VISUAL"} {
		raw := strings.TrimSpace(os.Getenv(name))
		if raw == "" {
			continue
		}
		if argv, err := shellquote.Split(raw); err == nil && len(argv) > 0 {
			return argv
		}
	}
	if runtime.GOOS == "windows" {
		return []string{"notepad"}
	}
	return []string{"vi"}
}
