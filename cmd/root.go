package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/brohd11/gofer/internal/app"

	"github.com/spf13/cobra"
)

// cdFileEnv is the shell wrapper's channel. A wrapper function cannot put --cd-file in
// argv: it has no way to know whether what follows is a directory or a subcommand, and
// `gofer --cd-file=… config` fails with "unknown flag" because the flag belongs to the root
// command alone (deliberately — it means nothing to `config` or `update`). The environment
// carries the same request without touching argv, so it is invisible to every subcommand
// and stays correct as more are added.
const cdFileEnv = "GOFER_CD_FILE"

// version is the binary version; defaults to "dev" for a plain `go build`. The makefile
// stamps it via -X ldflags, matching the sibling tools.
var version = "dev"

var (
	cdFile string
	all    bool
)

var rootCmd = &cobra.Command{
	Use:   "gofer [dir]",
	Short: "Browse a directory (TUI)",
	Long: `gofer opens a file explorer rooted at a directory: enter walks into a folder and
raises a menu on a file, backspace walks back out, and there is no floor — you can walk
all the way to the filesystem root.

  gofer            # current directory
  gofer /path      # an explicit start

The view preferences (row density, dot files) live in ~/.gofer/config.yml — edit them
with "gofer config". Press ? inside gofer for the keys.

gofer writes the directory you quit in to $GOFER_CD_FILE (or --cd-file), so a shell
wrapper can follow you out:

  gofer() {
    local tmp dir ret
    tmp="$(mktemp -t gofer-cd)"
    GOFER_CD_FILE="$tmp" command gofer "$@"
    ret=$?
    dir="$(cat -- "$tmp" 2>/dev/null)"; rm -f -- "$tmp"
    [ -n "$dir" ] && [ "$dir" != "$PWD" ] && cd -- "$dir"
    return "$ret"
  }

Only a browse writes the file, so "gofer config" and the rest pass through the wrapper
without moving your shell.`,
	Version:       version,
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: false,
	RunE:          runRoot,
}

func init() {
	rootCmd.SetVersionTemplate("gofer {{.Version}}\n")
	rootCmd.Flags().StringVar(&cdFile, "cd-file", "",
		"write the directory gofer quit in to this file (for a shell wrapper to cd into)")
	// The real default is the ladder resolveCDFile walks, not the empty string the flag
	// holds — which pflag suppresses anyway as a zero value. DefValue is only ever the
	// string cobra renders in "(default %s)", so rewriting it states that ladder where a
	// reader looks.
	rootCmd.Flags().Lookup("cd-file").DefValue = "$" + cdFileEnv
	rootCmd.Flags().BoolVarP(&all, "all", "a", false,
		"show hidden files for this run, whatever the config says (\".\" toggles it live)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// runRoot resolves the optional start directory (default: cwd) to an absolute path and
// launches the TUI.
//
// Changed("all") rather than the flag's value: --all overrides the config's show_hidden,
// and a bool flag that was never typed is indistinguishable from one typed as false. Only
// the CLI can tell them apart, so it is the CLI that answers the question.
func runRoot(cmd *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	return app.Run(version, app.Options{
		Dir:       abs,
		CDFile:    resolveCDFile(cdFile, cmd.Flags().Changed("cd-file")),
		Hidden:    all,
		HiddenSet: cmd.Flags().Changed("all"),
	})
}

// resolveCDFile picks where the finishing directory is recorded: the flag when it was
// actually typed, otherwise $GOFER_CD_FILE, otherwise nowhere. Anything typed outranks the
// environment, which is the ladder the sibling tools use for their own variables.
//
// A blank value is not a path, so a stray `export GOFER_CD_FILE=` cannot make gofer write a
// file named "". It is not the way to opt out of a WRAPPER, though — the wrapper sets the
// variable on the command line and so overrides whatever the outer environment said; the way
// past a shell function is `command gofer`.
//
// Unlike gote's depth there is nothing here to malform, so there is no error to report: any
// non-blank string is a path, and a path that cannot be written surfaces on exit as the
// write's own error.
func resolveCDFile(flagValue string, flagChanged bool) string {
	if flagChanged {
		return flagValue
	}
	return strings.TrimSpace(os.Getenv(cdFileEnv))
}
