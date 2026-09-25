package app

import (
	"fmt"
	"strings"

	"github.com/brohd11/bubblestack/components"
	"github.com/brohd11/bubblestack/core"

	"charm.land/bubbles/v2/key"
)

// helpScreen is the "?" page listing every shortcut. "?" toggles it; esc closes it.
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

// helpText renders the complete key reference (the bar only shows "? more"). Rows come from
// live bindings so rebinding keeps the page accurate.
func (s *browseScreen) helpText() string {
	var b strings.Builder
	// The two notes lead rather than close the page: they are the questions a keys list
	// cannot answer, and they must not sit below a fold on a short terminal.
	b.WriteString("the view below starts from ~/.gofer/config.yml — edit it with 'gofer config'\n")
	b.WriteString("$GOFER_CD_FILE records the folder you quit in, so a shell wrapper can follow you out\n")
	// The mouse is a note, not a row: the sections are built from key.Bindings.
	b.WriteString("left click opens a row (a folder by entering it); right click is the menu, as enter is\n\n")
	// One blank line between sections so the page fits a short terminal.
	writeSection := func(name string, binds []key.Binding) {
		b.WriteString(name + "\n")
		for _, kb := range binds {
			h := kb.Help()
			fmt.Fprintf(&b, "  %-12s %s\n", h.Key, h.Desc)
		}
		b.WriteString("\n")
	}
	// The up key is read off the panel, which owns it.
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
	// These act on the folder on screen, not the launch directory.
	writeSection("this folder", []key.Binding{
		core.Hint("re-read it", core.Keys.Refresh),
		core.Hint("terminal here", core.Keys.Terminal),
		core.Hint("terminal window here", core.Keys.TerminalWindow),
		core.Hint("open in the file manager", core.Keys.OpenDir),
	})
	writeSection("general", []key.Binding{
		core.FullHint("actions (theme, update, refresh)", core.Keys.Actions),
		core.Hint("quit", core.Keys.Quit, key.NewBinding(key.WithKeys("ctrl+c"))),
		core.Hint("this page (? again or esc closes it)", helpKey),
	})
	return strings.TrimRight(b.String(), "\n") + "\n"
}
