package app

import (
	"fmt"
	"strings"

	"github.com/brohd11/bubblestack/components"
	"github.com/brohd11/bubblestack/core"

	"charm.land/bubbles/v2/key"
)

// helpScreen is the pushed "?" page: a scrollable list of gofer's shortcuts, grouped.
// OnKey pops on "?" so the key toggles — press it to open the page, press it again to
// close — while esc closes it the way it closes anything else.
func (s *browseScreen) helpScreen() *components.DocScreen {
	return components.NewDocScreen(components.DocOpts{
		Title: "gofer · shortcuts",
		Crumb: "help",
		Render: func(int) string {
			return s.helpText()
		},
		OnKey: func(_ *core.Shared, k string) (core.Action, bool) {
			if core.MatchKey(k, helpKey) {
				return core.Pop(), true
			}
			return core.Action{}, false
		},
	})
}

// helpText renders the page's body. This is the COMPLETE reference, not the overflow from a
// bar that lists the common keys: the bar carries only "? more", so anything not written
// here is written nowhere.
//
// Every row is built from a live binding rather than a copied string, so rebinding any of
// them reaches this page instead of leaving it quietly stale — including the panel's own
// up key, which the component owns and could change under us.
//
// The key column is 12 wide, which is roomier than the widest entry needs ("ctrl+t") — the
// slack is there so a longer binding can be added without every row below it shifting. Every
// label spells its modifier out ("alt+r", not "⌥r") so the page reads in one notation.
func (s *browseScreen) helpText() string {
	var b strings.Builder
	// The two notes lead rather than close the page: they are the questions a keys list
	// cannot answer, and they must not sit below a fold on a short terminal.
	b.WriteString("the view below starts from ~/.gofer/config.yml — edit it with 'gofer config'\n")
	b.WriteString("$GOFER_CD_FILE records the folder you quit in, so a shell wrapper can follow you out\n")
	// The mouse cannot go in a section below: those rows are built from key.Bindings, and a
	// button is not one. It belongs with the notes anyway — it is the same split the keys
	// make, said once, rather than two more rows repeating them.
	b.WriteString("left click opens a row (a folder by entering it); right click is the menu, as enter is\n\n")
	// One blank between sections, none under a heading: the whole page has to fit a short
	// terminal without scrolling, and a heading over its own indented rows reads as a group
	// with or without the gap.
	writeSection := func(name string, binds []key.Binding) {
		b.WriteString(name + "\n")
		for _, kb := range binds {
			h := kb.Help()
			fmt.Fprintf(&b, "  %-12s %s\n", h.Key, h.Desc)
		}
		b.WriteString("\n")
	}
	// Moving through the tree and acting on a row are two different keys, and the rows say
	// so in that order: d and x are what you hold the folder open with, enter is what you
	// do to the thing under the cursor. The up key is read off the panel rather than
	// written here, since the component owns it and could change it under us.
	writeSection("navigation", []key.Binding{
		descendKey,
		core.Hint("up a folder (the \"..\" row does the same)", s.panel.UpKey()),
		core.Hint("the menu on this row, folder or file", core.Keys.Select),
		core.Hint("close a menu or this page", core.Keys.Back),
		core.Hint("filter this folder", s.panel.List().KeyMap.Filter),
		core.Hint("top/bottom", core.Keys.Top, core.Keys.Bottom),
	})
	writeSection("view", []key.Binding{
		hiddenKey,
		densityKey,
	})
	// These four follow the browse: they act on the folder ON SCREEN, not the one gofer was
	// launched in (browseScreen.LocateDir). Worth a section of its own, since nothing else
	// on screen says so.
	writeSection("this folder", []key.Binding{
		core.Hint("re-read it", core.Keys.Refresh),
		core.Hint("terminal here", core.Keys.Terminal),
		core.Hint("terminal window here", core.Keys.TerminalWindow),
		core.Hint("open in the file manager", core.Keys.OpenDir),
	})
	writeSection("general", []key.Binding{
		actionsKey,
		core.Hint("quit", core.Keys.Quit, key.NewBinding(key.WithKeys("ctrl+c"))),
		core.Hint("this page (? again or esc closes it)", helpKey),
	})
	return strings.TrimRight(b.String(), "\n") + "\n"
}
