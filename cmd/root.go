package cmd

import (
	"os"
	"path/filepath"

	"github.com/brohd11/gofer/internal/app"

	"github.com/spf13/cobra"
)

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

--cd-file writes the directory you quit in, so a shell wrapper can follow you out:

  gofer() {
    tmp="$(mktemp -t gofer-cd)"
    command gofer --cd-file="$tmp" "$@"
    dir="$(cat -- "$tmp" 2>/dev/null)"; rm -f -- "$tmp"
    [ -n "$dir" ] && [ "$dir" != "$PWD" ] && cd -- "$dir"
  }`,
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
		CDFile:    cdFile,
		Hidden:    all,
		HiddenSet: cmd.Flags().Changed("all"),
	})
}
