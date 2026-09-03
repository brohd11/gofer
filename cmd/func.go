package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// The wrapper functions below are the reason this command exists: a program cannot change
// its parent shell's directory, so gofer writes the folder it quit in to $GOFER_CD_FILE and
// a shell function does the cd. Printing that function rather than documenting it means an
// rc file carries one line — eval "$(gofer func zsh)" — and the wrapper travels with the
// binary instead of going stale in someone's dotfiles.
//
// This is the single source of truth for that text. README.md, rootCmd.Long and
// install.sh's post_install_note all point here rather than pasting their own copies.

// posixWrapper serves bash and zsh, which need no variations between them.
//
// Every line is load-bearing:
//
//   - `command gofer` keeps the function from calling itself, and is also the documented
//     way to bypass the wrapper for one run.
//   - The variable is `ret` and not `status` because zsh reserves `status` as a read-only
//     alias for $?.
//   - `return "$ret"` is not optional. The cd line is a test that fails whenever there is
//     nowhere to go, so without it the function would report failure after every browse
//     that stayed put — and would swallow a real error from `gofer update`.
//   - The environment variable, rather than --cd-file, is what lets the wrapper be
//     unconditional: it is invisible to argv, so `gofer config` and the rest pass straight
//     through. See cdFileEnv in root.go.
const posixWrapper = `gofer() {
  local tmp dir ret
  tmp="$(mktemp -t gofer-cd)"
  ` + cdFileEnv + `="$tmp" command gofer "$@"
  ret=$?
  dir="$(cat -- "$tmp" 2>/dev/null)"; rm -f -- "$tmp"
  [ -n "$dir" ] && [ "$dir" != "$PWD" ] && cd -- "$dir"
  return "$ret"
}
`

// fishWrapper is the same function in fish's syntax, which shares none of the above:
// `function ... end`, `set -l` for locals, $argv for "$@".
//
// `env` rather than `command`: it execs the real binary, so it dodges this function exactly
// as `command` does in a POSIX shell, and unlike fish's own VAR=value prefix it does not
// need fish 3.1. `set -l ret $status` must sit immediately after the call — anything in
// between would overwrite $status. The cd needs no `--` guard because gofer only ever
// writes an absolute path, and fish does not word-split a variable, so a path with spaces
// survives unquoted.
const fishWrapper = `function gofer
  set -l tmp (mktemp -t gofer-cd)
  env ` + cdFileEnv + `=$tmp gofer $argv
  set -l ret $status
  set -l dir (cat -- $tmp 2>/dev/null)
  rm -f -- $tmp
  if test -n "$dir"; and test "$dir" != "$PWD"
    cd $dir
  end
  return $ret
end
`

// wrappers maps a shell name to the function to print for it. supportedShells names the
// same set in the order the errors and the help text list them.
var (
	wrappers = map[string]string{
		"bash": posixWrapper,
		"zsh":  posixWrapper,
		"fish": fishWrapper,
	}
	supportedShells = []string{"bash", "zsh", "fish"}
)

var funcCmd = &cobra.Command{
	Use:   "func [shell]",
	Short: "Print the shell wrapper that makes gofer move your shell",
	Long: `func prints the shell function that follows gofer out of a browse — a program
cannot change its parent shell's directory, so gofer records the folder you quit in and the
function does the cd.

Add one line to your rc file and the wrapper updates itself whenever gofer does:

  eval "$(gofer func zsh)"

bash, zsh and fish are supported. "gofer func" with no argument reads $SHELL, which is your
login shell and not necessarily the one reading the rc file — naming the shell is the
sturdier form.

Only a browse records a folder, so "gofer config" and the rest pass through the wrapper
without moving your shell. So does a crash. To bypass it for one run: "command gofer".`,
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: false,
	RunE:          runFunc,
}

func init() {
	rootCmd.AddCommand(funcCmd)
}

// runFunc writes the wrapper and nothing else: the output is eval'd, so a stray banner
// would be executed. OutOrStdout rather than fmt.Print for the same reason the tests need
// it — the destination has to be substitutable.
func runFunc(cmd *cobra.Command, args []string) error {
	arg := ""
	if len(args) > 0 {
		arg = args[0]
	}
	name, err := resolveShell(arg, os.Getenv("SHELL"))
	if err != nil {
		return err
	}
	body, err := wrapperFor(name)
	if err != nil {
		return err
	}
	fmt.Fprint(cmd.OutOrStdout(), body)
	return nil
}

// resolveShell names the shell to print for: what was typed, otherwise the basename of
// $SHELL. It does not judge the name — wrapperFor does that, so an unrecognized $SHELL and
// an unrecognized argument fail the same way — but it does report the case where there is
// no name to judge at all, since "unknown shell" would be a lie there.
func resolveShell(arg, shellEnv string) (string, error) {
	if s := strings.ToLower(strings.TrimSpace(arg)); s != "" {
		return s, nil
	}
	if s := strings.TrimSpace(shellEnv); s != "" {
		return strings.ToLower(filepath.Base(s)), nil
	}
	return "", fmt.Errorf("no shell given and $SHELL is not set: name one of %s, as in `gofer func zsh`",
		strings.Join(supportedShells, ", "))
}

// wrapperFor is the name to text lookup. The error names the whole supported set, so a typo
// inside an eval fails loudly with the fix in it.
func wrapperFor(shell string) (string, error) {
	body, ok := wrappers[shell]
	if !ok {
		return "", fmt.Errorf("unknown shell %q: gofer func supports %s",
			shell, strings.Join(supportedShells, ", "))
	}
	return body, nil
}
