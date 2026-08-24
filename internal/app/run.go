package app

import (
	"os"
	"path/filepath"

	"github.com/brohd11/bubblestack"
	"github.com/brohd11/bubblestack/components"
	"github.com/brohd11/bubblestack/core"
	"github.com/brohd11/bubblestack/sysopen"
)

// Run launches the gofer TUI over the resolved start directory and, on a clean exit, hands
// the directory the user quit in to opts.CDFile.
//
// The write happens HERE, after bubblestack.Run returns, because that is the only seam
// there is: Run gives back an error and nothing else, so the finishing directory has to
// come off the context we built and still hold. It is also the right place — a program
// that failed leaves the file untouched, so the shell wrapper reads nothing and stays
// where it was rather than following a crash somewhere unexpected.
//
// One tab, so bubblestack draws no tab strip; a status line for feedback (the hidden-files
// toggle, a self-update note); no header or output pane. The breadcrumb carries the current
// path, which is the only persistent chrome a file explorer actually needs.
func Run(version string, opts Options) error {
	// Materialized before the load, and best-effort: the file is where the schema is
	// documented, so a user who never runs `gofer config` should still end up with one to
	// read. A failed write (a read-only home) is not a reason to refuse to start, and
	// LoadConfig treats the still-missing file as the defaults anyway.
	_, _ = EnsureConfig()
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	c := New(version, cfg, opts)
	err = bubblestack.Run(bubblestack.Config{
		App:    c,
		Status: components.NewStatusLine(),
		Tabs: []bubblestack.TabEntry{
			{Title: "Files", New: NewBrowseScreen},
		},
		Init:                 SelfUpdateCheckCmd,
		RefreshAction:        refreshAction,
		TerminalAction:       func(dir string) core.Action { return sysopen.TerminalInline(dir) },
		TerminalWindowAction: func(dir string) core.Action { return sysopen.Terminal(dir) },
		OpenDirAction:        func(dir string) core.Action { return sysopen.Path(dir, false) },
	})
	if err != nil {
		return err
	}
	return writeCDFile(opts.CDFile, c.Dir)
}

// refreshAction is the global Refresh key and the Actions menu's Refresh row: rebuild the
// tab root, which rebuilds the panel over Ctx.Dir and so re-reads the folder from disk.
// Going through RefreshRoots rather than reaching for the live panel is what keeps this a
// package-level function the menu can hold without a screen.
func refreshAction(sh *core.Shared) core.Action {
	return core.Seq(core.SetStatus("re-read "+filepath.Base(Of(sh).Dir)), core.RefreshRoots())
}

// writeCDFile records dir for a shell wrapper to cd into. An empty path means the flag was
// not given and there is nothing to do — the overwhelmingly common case, since the flag
// only makes sense from inside the wrapper function.
//
// 0o600 rather than 0o644: the file names a directory the user was just browsing, it lives
// wherever mktemp put it, and nothing else has any business reading it.
func writeCDFile(path, dir string) error {
	if path == "" {
		return nil
	}
	return os.WriteFile(path, []byte(dir+"\n"), 0o600)
}
