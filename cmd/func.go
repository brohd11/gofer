package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// A program cannot change its parent shell's directory, so gofer writes the folder it quit in
// to $GOFER_CD_FILE and a shell wrapper does the cd. Printing the wrapper keeps it in step
// with the binary: eval "$(gofer func zsh)". This is the single source of that text.

// posixWrapper serves bash and zsh. `command gofer` bypasses the function; the variable is
// `ret` because zsh reserves `status`; `return "$ret"` keeps the cd test from masking gofer's
// own exit status. The env var (not --cd-file) keeps argv untouched for subcommands.
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

// fishWrapper is the fish version. `env` execs the real binary like `command` does (without
// needing fish 3.1); `set -l ret $status` must directly follow the call. gofer writes an
// absolute path and fish does not word-split, so the cd needs no quoting or `--`.
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

// runFunc writes the wrapper and nothing else: the output is eval'd.
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

// resolveShell returns the typed shell, else $SHELL's basename. wrapperFor judges the name;
// this only errors when there is no name at all.
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
