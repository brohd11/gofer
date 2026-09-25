package app

import (
	"io/fs"
	"strings"

	"github.com/brohd11/bubblestack/core"
)

// Options is the launch selection the CLI resolves. HiddenSet records whether --all was
// typed, so --all=false can override show_hidden: true.
type Options struct {
	Dir       string // resolved absolute start directory
	CDFile    string // where to record the directory gofer quit in ($GOFER_CD_FILE, or --cd-file)
	Hidden    bool   // --all
	HiddenSet bool
}

// Ctx is gofer's app context. Dir is the directory on screen; Run reads it after
// bubblestack.Run returns to write the cd file.
type Ctx struct {
	// ListCompact is the session density for standard menus, independent of
	// Compact, which configures the file panel.
	ListCompact bool

	Dir        string
	Version    string
	Compact    bool
	ShowHidden bool
}

// New builds the context from the config and launch options (--all overrides for this run
// only). In-app toggles are never written back to the config.
func New(version string, cfg Config, opts Options) *Ctx {
	c := &Ctx{
		Dir:        opts.Dir,
		Version:    version,
		Compact:    cfg.Compact,
		ShowHidden: cfg.ShowHidden,
	}
	if opts.HiddenSet {
		c.ShowHidden = opts.Hidden
	}
	return c
}

// Of recovers the gofer context from a Shared. Screens call c := app.Of(sh).
func Of(sh *core.Shared) *Ctx { return core.App[Ctx](sh) }

// Receive rebuilds the tab root on a theme change; the root holds no unsaved state.
func (c *Ctx) Receive(sh *core.Shared, payload any) core.Action {
	return core.OnThemeChange(payload)
}

// include is the panel's Include hook, hiding dotfiles unless toggled. It reads the ctx so
// the toggle applies on the next Refresh without a rebuild.
func (c *Ctx) include(_ string, d fs.DirEntry) bool {
	return c.ShowHidden || !strings.HasPrefix(d.Name(), ".")
}

// ListDensity opts standard lists into the app-wide session preference.
func (c *Ctx) ListDensity() *bool { return &c.ListCompact }
