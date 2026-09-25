package app

import (
	"os"
	"path/filepath"

	"github.com/brohd11/bubblestack"
	"github.com/brohd11/bubblestack/components"
	"github.com/brohd11/bubblestack/core"
	"github.com/brohd11/bubblestack/sysopen"
)

// Run launches the gofer TUI and, on a clean exit, writes the final directory to
// opts.CDFile. A failed run leaves the file untouched so the shell stays put.
func Run(version string, opts Options) error {
	// Best-effort: the file documents the schema, and LoadConfig falls back to defaults.
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
		TerminalAction:       func(dir string) core.Action { return sysopen.TerminalInlineFor("gofer", dir) },
		TerminalWindowAction: func(dir string) core.Action { return sysopen.Terminal(dir) },
		OpenDirAction:        func(dir string) core.Action { return sysopen.Path(dir, false) },
	})
	if err != nil {
		return err
	}
	return writeCDFile(opts.CDFile, c.Dir)
}

// refreshAction rebuilds the tab root, which re-reads Ctx.Dir from disk.
func refreshAction(sh *core.Shared) core.Action {
	return core.Seq(core.SetStatus("re-read "+filepath.Base(Of(sh).Dir)), core.RefreshRoots())
}

// writeCDFile records dir for the shell wrapper (0o600). An empty path is a no-op.
func writeCDFile(path, dir string) error {
	if path == "" {
		return nil
	}
	return os.WriteFile(path, []byte(dir+"\n"), 0o600)
}
