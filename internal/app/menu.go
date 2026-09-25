package app

import (
	"github.com/brohd11/bubblestack/components"
	"github.com/brohd11/bubblestack/core"
	"github.com/brohd11/bubblestack/sysopen"
	"github.com/brohd11/goutil/executil"
	"github.com/brohd11/goutil/textfile"

	tea "charm.land/bubbletea/v2"
)

// pickFile is the panel's OnSelect: a file raises a context menu over its row.
func (s *browseScreen) pickFile(sh *core.Shared, e components.FileEntry) core.Action {
	return core.Push(components.NewMenu(components.MenuOpts{
		Title:  e.Name,
		Items:  fileMenuItems(e),
		Anchor: s.rowAnchor(sh),
	}))
}

// fileMenuItems are the file menu's rows. The editor row comes first and only for text files.
func fileMenuItems(e components.FileEntry) []components.MenuItem {
	var items []components.MenuItem
	if textfile.IsText(e.Path) {
		items = append(items, components.MenuItem{
			Label: "Open in text editor",
			Pick:  func(*core.Shared) core.Action { return openInEditor(e) },
		})
	}
	return append(items, components.MenuItem{
		Label: "Open in default app",
		Pick: func(*core.Shared) core.Action {
			return core.Seq(core.Pop(), sysopen.Path(e.Path, false))
		},
	})
}

// pickDir is the panel's OnOpenDir: enter on a folder raises its menu (d walks in). The ".."
// row reports unhandled so the panel walks up.
func (s *browseScreen) pickDir(sh *core.Shared, e components.FileEntry) (core.Action, bool) {
	if e.Up {
		return core.Action{}, false
	}
	return core.Push(components.NewMenu(components.MenuOpts{
		Title:  e.Name,
		Items:  s.dirMenuItems(e),
		Anchor: s.rowAnchor(sh),
	})), true
}

// dirMenuItems are the folder menu's rows; "Open folder" comes first.
func (s *browseScreen) dirMenuItems(e components.FileEntry) []components.MenuItem {
	return []components.MenuItem{{
		Label: "Open folder",
		// SetDir runs before the Pop is applied, in the same tick.
		Pick: func(sh *core.Shared) core.Action {
			return core.Seq(core.Pop(), s.panel.SetDir(sh, e.Path))
		},
	}, {
		Label: "Open in file manager",
		// reveal is false: the target is a directory, so it opens as one rather than being
		// highlighted inside its parent.
		Pick: func(*core.Shared) core.Action {
			return core.Seq(core.Pop(), sysopen.Path(e.Path, false))
		},
	}, {
		Label: "Terminal here",
		// Inline, like bare t. Pop first so the terminal returns to the listing.
		Pick: func(*core.Shared) core.Action {
			return core.Seq(core.Pop(), sysopen.TerminalInlineFor("gofer", e.Path))
		},
	}}
}

// openInEditor hands the terminal to the editor on e and takes it back when it exits.
// The menu is popped first so nothing is left stranded on top of the restored listing.
func openInEditor(e components.FileEntry) core.Action {
	argv := editorArgv()
	cmd, err := executil.Command(append(argv, e.Path)...)
	if err != nil {
		return core.StatusErr(err)
	}
	cmd.Dir = e.Dir // relative paths typed in the editor resolve against the folder on screen
	return core.Seq(core.Pop(), core.Async(tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return core.SetStatusAndLog(argv[0] + " " + e.Name + ": " + err.Error()).Msg
		}
		return editorClosedMsg{name: e.Name}
	})))
}

// editorClosedMsg tells the browse screen the editor exited cleanly, so it re-reads the panel.
type editorClosedMsg struct{ name string }

// rowAnchor puts the menu over the selected row; the panel is the only slot, at (0, BodyY).
func (s *browseScreen) rowAnchor(sh *core.Shared) components.MenuAnchor {
	if a, ok := s.panel.RowAnchor(s.panel.List().Index(), 0, sh.BodyY()); ok {
		return a
	}
	return components.AnchorAt(0, sh.BodyY())
}
