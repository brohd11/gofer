package app

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/brohd11/bubblestack/core"
)

// Options is the launch selection the CLI resolves (see cmd.runRoot). HiddenSet carries
// whether --all was actually typed, rather than treating false as "unset": a bool flag has
// no other way to say "the user asked for false", and without it a config with
// show_hidden: true could never be turned off for one run.
type Options struct {
	Dir       string // resolved absolute start directory
	CDFile    string // --cd-file: where to record the directory gofer quit in
	Hidden    bool   // --all
	HiddenSet bool
}

// Ctx is gofer's app context, stored on core.Shared.App and recovered with Of. It is
// almost nothing: the directory the panel is currently listing, the two view preferences,
// and the version the self-update flow checks against.
//
// Dir is the whole point of keeping a context at all. The panel owns navigation, but
// Run has to read the final directory AFTER bubblestack.Run returns — that is what
// --cd-file writes — and this is the only object that outlives the program's UI.
type Ctx struct {
	Dir        string
	Version    string
	Compact    bool
	ShowHidden bool
}

// New builds the context from the config and the launch options: the config supplies both
// view preferences, and --all overrides one of them for this run only. The directory is
// taken on trust — an unreadable one renders as an empty listing rather than a launch
// error, which is what FilePanel does with it anyway.
//
// Neither preference is written back when the in-app keys flip it. The config is where the
// session STARTS, not a record of where it ended: a density flipped for one deep folder is
// not a decision about every future launch.
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

// Receive handles app-level broadcasts: a theme change rebuilds the tab root so its list
// re-bakes the new palette (the router-drawn chrome repaints on its own). gofer's root
// holds no buffer or unsaved state, so the wholesale rebuild core.OnThemeChange asks for
// is safe here in a way it is not in gote.
func (c *Ctx) Receive(sh *core.Shared, payload any) core.Action {
	return core.OnThemeChange(payload)
}

// include is the panel's Include hook. It closes over the ctx rather than over a bool, so
// the "." toggle takes effect on the next Refresh with no panel rebuild.
//
// Everything else is listed: gofer is a file explorer, not a document picker, so it has no
// business deciding a file is uninteresting. Hidden entries are the one exception, and only
// because a listing that opens on .DS_Store and .git is a worse default than one you can
// widen with a keystroke.
func (c *Ctx) include(_ string, d fs.DirEntry) bool {
	return c.ShowHidden || !strings.HasPrefix(d.Name(), ".")
}

// shortHome renders a path with the home directory as "~", the form a directory is worth
// reading as in a breadcrumb. An unrelatable path is returned unchanged.
func shortHome(path, home string) string {
	if home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	rel, err := filepath.Rel(home, path)
	// A path outside home relativizes to something starting with "..", which is longer and
	// less readable than the absolute path it came from.
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return path
	}
	return "~" + string(filepath.Separator) + rel
}
