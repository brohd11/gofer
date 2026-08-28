package app

import (
	"os"

	"github.com/brohd11/bubblestack/components"
	"github.com/brohd11/bubblestack/core"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// The screen's own keys. The two bare ones are intercepted only when nothing is capturing
// (see Update), so a /-filter never loses a character to them; densityKey is the panel's
// own and carries a modifier, alt+r rather than alt+d/f — the sibling apps' editor moves by
// words on those, and the chords should not mean two things across the monorepo.
var (
	actionsKey = key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "actions (theme, update, refresh)"))
	hiddenKey  = key.NewBinding(key.WithKeys("."), key.WithHelp(".", "show or hide dot files"))
	densityKey = key.NewBinding(key.WithKeys("alt+r"), key.WithHelp("alt+r", "row density"))
	// alt+? is the modified alias that summons the page from anywhere, the capture gate
	// included; the bare "?" is the one the bar advertises.
	helpKey = key.NewBinding(key.WithKeys("?", "alt+?"), key.WithHelp("?", "more"))
)

// browseScreen is gofer's only screen: one components.FilePanel in a one-slot
// ModularScreen. The shell around the panel exists for the three things a layout host
// cannot answer for a directory that MOVES —
//
//   - DirLocator, so the global terminal/open-dir keys act on the folder on screen rather
//     than the one gofer was launched in (ModularOpts.Dir is a fixed string);
//   - Crumber, so the bar above the panel carries the full path the panel's border legend
//     only shows the base name of;
//   - the ctx's Dir, which is what the cd file records after the program exits.
//
// ModularScreen rather than hosting the panel bare: it already translates mouse presses to
// pane-local coordinates (which is what the panel's click math expects), composes the help
// bar from PanelHelp, and holds focus. One slot is a legal grid.
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
		// The starting density is the config's (compact by default); alt+r flips it for
		// this session without writing the choice back.
		Compact:    c.Compact,
		DensityKey: densityKey,
		Include:    c.include,
		OnSelect:   s.pickFile,
		OnDir:      func(sh *core.Shared, dir string) core.Action { Of(sh).Dir = dir; return core.Action{} },
		OnError: func(_ *core.Shared, err error) core.Action {
			return core.Push(components.CreatePopup("open folder", err.Error(), core.Pop()))
		},
	})
	// ExpandH pads the slot out to the terminal width: the framed panel is already as wide
	// as its allocation, but the flag costs nothing and keeps a ragged edge impossible.
	s.modular = components.NewModularScreen(
		[][]components.Slot{{{Panel: s.panel, Weight: 1, ExpandH: true}}},
		// One entry on the bar, and it is the pointer at all the others: every key gofer has
		// is written on the ? page, so the bar names the way in rather than reprinting a
		// couple of them beside the framework's own back/select hints.
		components.ModularOpts{Help: []key.Binding{helpKey}},
	)
	return s
}

func (s *browseScreen) Init(sh *core.Shared) tea.Cmd {
	s.sh = sh
	return s.modular.Init(sh)
}

// Update claims the screen's own keys, then delegates. Both are bare letters, so both are
// gated on Filtering: a live /-query owns every character, and a screen that took one back
// would eat it out of the search.
//
// The hidden toggle is a SCREEN key rather than the panel's OnKey hook, even though it only
// concerns the panel: OnKey is typed to the entry under the cursor and reports unhandled on
// the ".." row, which is exactly the row an unclamped explorer opens with. A view setting
// must not depend on where the cursor happens to be.
func (s *browseScreen) Update(sh *core.Shared, msg tea.Msg) (core.Screen, core.Action) {
	// The editor had the terminal and has given it back. Re-read the folder: the file it
	// just wrote is a different size, and on a standard-density row that number is on
	// screen. Refresh keeps the cursor, so the user lands back on the row they edited.
	if m, ok := msg.(editorClosedMsg); ok {
		s.panel.Refresh()
		return s, core.SetStatus(m.name + " closed")
	}
	if km, ok := msg.(tea.KeyMsg); ok && km.String() == "alt+?" {
		// The one key that outranks the capture gate: a modified chord produces no text, so
		// taking it from a live filter costs the query nothing, and a help page you cannot
		// reach while filtering is a help page you cannot reach when you most need it.
		return s, core.Push(s.helpScreen())
	}
	if km, ok := msg.(tea.KeyMsg); ok && !s.modular.Filtering() {
		switch k := km.String(); {
		case core.MatchKey(k, actionsKey):
			return s, core.Push(actionsMenu(sh))
		case core.MatchKey(k, hiddenKey):
			return s, s.toggleHidden(sh)
		case core.MatchKey(k, helpKey):
			return s, core.Push(s.helpScreen())
		}
	}
	_, act := s.modular.Update(sh, msg)
	return s, act
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

// CrumbLabel names the current directory in the router's breadcrumb bar. Short and long are
// the same string: the bar's own truncation is better than a second one here, and the
// panel's border legend already carries the base name when the path is cut.
func (s *browseScreen) CrumbLabel(bool) string {
	if s.sh == nil {
		return "gofer"
	}
	return shortHome(Of(s.sh).Dir, s.home)
}
