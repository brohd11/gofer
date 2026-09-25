package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/brohd11/gofer/internal/app"

	"github.com/spf13/cobra"
)

// cdFileEnv is the shell wrapper's channel. A wrapper cannot put --cd-file in argv (the flag
// is root-only, so `gofer --cd-file=… config` fails); the environment reaches the root
// without touching argv.
const cdFileEnv = "GOFER_CD_FILE"

// version is stamped by the makefile via -X ldflags; "dev" for a plain go build.
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
wrapper can follow you out. "gofer func" prints that wrapper, so an rc file carries one
line and the wrapper stays current with the binary:

  eval "$(gofer func zsh)"    # bash, zsh and fish; "gofer func" alone reads $SHELL

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
	// Show the fallback ladder resolveCDFile walks rather than the empty default.
	rootCmd.Flags().Lookup("cd-file").DefValue = "$" + cdFileEnv
	rootCmd.Flags().BoolVarP(&all, "all", "a", false,
		"show hidden files for this run, whatever the config says (\".\" toggles it live)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// runRoot resolves the optional start directory (default: cwd) and launches the TUI.
// Changed("all") lets --all=false override the config's show_hidden.
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

// resolveCDFile returns the flag when typed, else $GOFER_CD_FILE, else "". A blank value is
// treated as unset.
func resolveCDFile(flagValue string, flagChanged bool) string {
	if flagChanged {
		return flagValue
	}
	return strings.TrimSpace(os.Getenv(cdFileEnv))
}
