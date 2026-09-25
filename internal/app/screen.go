package app

import (
	"os"

	"github.com/brohd11/bubblestack/components"
	"github.com/brohd11/bubblestack/core"
	"github.com/brohd11/goutil/strutil"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// The screen's own keys, intercepted only when nothing is capturing. densityKey is alt+r
// because alt+d/f move by words in the editors.
var (
	hiddenKey  = key.NewBinding(key.WithKeys("."), key.WithHelp(".", "show or hide dot files"))
	densityKey = key.NewBinding(key.WithKeys("alt+r"), key.WithHelp("alt+r", "row density"))
	// d and x are bare letters for the most-used keys. The arrows are the lists' only
	// pagination keys. x is also core.Keys.NextTab, which the router ignores with one tab;
	// adding a second tab would take it back.
	descendKey = key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "into the folder under the cursor"))
	upKey      = key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "up a folder"))
	// alt+? is the modified alias that summons the page from anywhere, the capture gate
	// included; the bare "?" is the one the bar advertises.
	helpKey = key.NewBinding(key.WithKeys("?", "alt+?"), key.WithHelp("?", "more"))
)

// browseScreen is gofer's only screen: a FilePanel in a one-slot ModularScreen. It adds
// DirLocator (global keys act on the folder on screen), Crumber (the full path in the
// breadcrumb) and Ctx.Dir tracking for the cd file.
type browseScreen struct {
	modular *components.ModularScreen
	panel   *components.FilePanel
	home    string
	sh      *core.Shared
}

var _ core.Screen = (*browseScreen)(nil)
var _ core.Filterer = (*browseScreen)(nil)
var _ core.DirLocator = (*browseScreen)(nil)
var _ core.Crumber = (*browseScreen)(nil)

// NewBrowseScreen builds the panel over the ctx's starting directory and wraps it in the
// one-slot layout.
func NewBrowseScreen(sh *core.Shared) core.Screen {
	c := Of(sh)
	s := &browseScreen{}
	s.home, _ = os.UserHomeDir()
	s.panel = components.NewFilePanel(components.FilePanelOpts{
		Dir: c.Dir,
		// No Root: gofer is a file explorer, so ".." goes all the way up. gote clamps
		// because its explorer must not leave the scan the rest of the app knows about.
		Border: true,
		// A listing you scan down is the one place file-type color earns its keep.
		Colors: components.FileColorsAll,
		// The starting density is the config's (compact by default); alt+r flips it for
		// this session without writing the choice back.
		Compact:    c.Compact,
		DensityKey: densityKey,
		Include:    c.include,
		// x rather than backspace, which is also core.Keys.Back.
		UpKey:    upKey,
		OnSelect: s.pickFile,
		// Enter on a folder raises its menu; pickDir returns unhandled for "..".
		OnOpenDir: s.pickDir,
		OnDir:     func(sh *core.Shared, dir string) core.Action { Of(sh).Dir = dir; return core.Action{} },
		OnError: func(_ *core.Shared, err error) core.Action {
			return core.Push(components.CreatePopup("open folder", err.Error(), core.Pop()))
		},
	})
	// ExpandH pads the slot out to the terminal width: the framed panel is already as wide
	// as its allocation, but the flag costs nothing and keeps a ragged edge impossible.
	s.modular = components.NewModularScreen(
		[][]components.Slot{{{Panel: s.panel, Weight: 1, ExpandH: true}}},
		// The bar shows only "?"; every key is on the help page.
		components.ModularOpts{Help: []key.Binding{helpKey}},
	)
	return s
}

func (s *browseScreen) Init(sh *core.Shared) tea.Cmd {
	s.sh = sh
	return s.modular.Init(sh)
}

// Update claims the screen's own keys, gated on Filtering so a /-query keeps its
// characters. The hidden toggle is a screen key because the panel's OnKey reports unhandled
// on the ".." row.
func (s *browseScreen) Update(sh *core.Shared, msg tea.Msg) (core.Screen, core.Action) {
	// The editor returned the terminal: re-read the folder (sizes may have changed).
	if m, ok := msg.(editorClosedMsg); ok {
		s.panel.Refresh()
		return s, core.SetStatus(m.name + " closed")
	}
	if km, ok := msg.(tea.KeyPressMsg); ok && km.String() == "alt+?" {
		// A modified chord produces no text, so help stays reachable while filtering.
		return s, core.Push(s.helpScreen())
	}
	if km, ok := msg.(tea.KeyPressMsg); ok && !s.modular.Filtering() {
		switch k := km.String(); {
		case core.MatchKey(k, core.Keys.Actions):
			return s, core.Push(actionsMenu(sh))
		case core.MatchKey(k, hiddenKey):
			return s, s.toggleHidden(sh)
		case core.MatchKey(k, descendKey):
			return s, s.descend(sh)
		case core.MatchKey(k, helpKey):
			return s, core.Push(s.helpScreen())
		}
	}
	_, act := s.modular.Update(sh, msg)
	return s, act
}

// descend walks into the folder under the cursor; on ".." it walks up, on a file it does
// nothing.
func (s *browseScreen) descend(sh *core.Shared) core.Action {
	e, ok := s.panel.Selected()
	if !ok || !e.IsDir {
		return core.Action{}
	}
	return s.panel.SetDir(sh, e.Path)
}

// toggleHidden shows or hides dot files and re-reads the folder in place — the panel keeps
// its cursor, so widening the listing does not lose your place in it.
func (s *browseScreen) toggleHidden(sh *core.Shared) core.Action {
	c := Of(sh)
	c.ShowHidden = !c.ShowHidden
	s.panel.Refresh()
	if c.ShowHidden {
		return core.SetStatus("hidden files shown")
	}
	return core.SetStatus("hidden files off")
}

func (s *browseScreen) View(sh *core.Shared) string { return s.modular.View(sh) }

func (s *browseScreen) SetSize(sh *core.Shared, width, bodyHeight int) {
	s.sh = sh
	s.modular.SetSize(sh, width, bodyHeight)
}

func (s *browseScreen) HelpView(sh *core.Shared) string { return s.modular.HelpView(sh) }

// Filtering proxies the panel's capture state: the router must leave its global single-key
// shortcuts alone while the list is filtering.
func (s *browseScreen) Filtering() bool { return s.modular.Filtering() }

// LocateDir advertises the folder on screen to the router's global terminal and open-dir
// keys, so t/T/ctrl+t act on where you are browsing rather than where you launched.
func (s *browseScreen) LocateDir() (string, bool) {
	if s.sh == nil {
		return "", false
	}
	return Of(s.sh).Dir, true
}

// CrumbLabel names the current directory; short and long are the same string.
func (s *browseScreen) CrumbLabel(bool) string {
	if s.sh == nil {
		return "gofer"
	}
	return strutil.ContractHome(Of(s.sh).Dir, s.home)
}
