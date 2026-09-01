package app

import (
	"os/exec"

	"github.com/brohd11/bubblestack/components"
	"github.com/brohd11/bubblestack/core"
	"github.com/brohd11/bubblestack/sysopen"
	"github.com/brohd11/goutil/textfile"

	tea "github.com/charmbracelet/bubbletea"
)

// pickFile is the panel's OnSelect: a file raises a context menu over its own row.
// Directories never reach here — walking into one is the panel's own business (OnOpenDir is
// left nil), which is the split the component was built around.
//
// A menu rather than acting directly, even with one row in it: "what happens to this file"
// is a list that wants to grow (reveal in the file manager, copy the path, a terminal
// here), and a menu that appears once there are two verbs is a behavior change, where a
// menu that starts with one is a list that gets longer.
func (s *browseScreen) pickFile(sh *core.Shared, e components.FileEntry) core.Action {
	return core.Push(components.NewMenu(components.MenuOpts{
		Title:  e.Name,
		Items:  fileMenuItems(e),
		Anchor: s.rowAnchor(sh),
	}))
}

// fileMenuItems is the file menu's rows. Each owns its own dismissal (the MenuItem
// convention), so a pick pops the menu and then acts.
//
// The editor row is FIRST and conditional: on a file gofer can tell is text, editing it is
// the likelier verb, and the row the cursor already sits on is the one you want. It is
// omitted rather than disabled on a PNG or a binary — a permanently grey row teaches
// nothing the file's own name does not already say.
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

// pickDir is the panel's OnOpenDir: enter (or a click) on a FOLDER raises its menu instead
// of walking into it. Walking is d's job, and a folder that could only be walked into was a
// row with no verbs at all — "open this one in the file manager" and "put a shell in this
// one" had no key, because the global t/T/ctrl+t are DirLocator keys and only ever act on
// the folder you are already IN (browseScreen.LocateDir), never the one under the cursor.
//
// The ".." row is the exception and reports unhandled, so the panel walks up as it always
// did. The way out of a folder is a navigation affordance, not a folder you act on — the
// same reason it is kept out of a /-filter (fileItem.FilterValue).
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

// dirMenuItems is the folder menu's rows, fileMenuItems' counterpart. Each owns its own
// dismissal, as every MenuItem does.
//
// "Open folder" is first and is the row that keeps enter cheap: the key that used to walk
// in now takes two presses to do it, and this is the second one.
func (s *browseScreen) dirMenuItems(e components.FileEntry) []components.MenuItem {
	return []components.MenuItem{{
		Label: "Open folder",
		// SetDir runs as this Seq's argument, so the panel has re-listed before the Pop is
		// applied. Nothing renders between the two — the same single-tick ordering
		// openInEditor relies on below — but the walk really does happen first.
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
		// The inline form, matching what bare t does: a shell for a two-command detour
		// shouldn't cost a window. The Pop runs first so the terminal is restored onto the
		// listing rather than onto a menu stranded over it.
		Pick: func(*core.Shared) core.Action {
			return core.Seq(core.Pop(), sysopen.TerminalInline(e.Path))
		},
	}}
}

// openInEditor hands this process's terminal to $EDITOR on e, and takes it back when the
// editor exits — the user returns to the folder and the row they left, rather than to a
// detached window they now have to close.
//
// tea.ExecProcess is the same mechanism sysopen.TerminalInline uses for the t key
// (bubblestack/sysopen/sysopen.go). It is spelled out here rather than called through
// TerminalInline because that helper reports "terminal at <dir> closed", which is the wrong
// sentence for an editor, and because gofer is the only app that wants this — the framework
// keeps its surface.
//
// The Pop runs first and synchronously (core.Seq applies control messages in the same
// tick), so the menu is gone before the screen is handed over; what the terminal is
// restored onto is the folder listing, with no overlay stranded on top of it.
func openInEditor(e components.FileEntry) core.Action {
	argv := editorArgv()
	cmd := exec.Command(argv[0], append(argv[1:], e.Path)...)
	// The editor inherits the listing directory, so a relative path typed inside it (a
	// :split, a shell-out) resolves against the folder on screen.
	cmd.Dir = e.Dir
	return core.Seq(core.Pop(), core.Async(tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return core.SetStatusAndLog(argv[0] + " " + e.Name + ": " + err.Error()).Msg
		}
		return editorClosedMsg{name: e.Name}
	})))
}

// editorClosedMsg lands on the browse screen when the child editor exits cleanly. It is a
// message rather than a status Action because the panel has to be re-read as well, and only
// the screen holds it.
type editorClosedMsg struct{ name string }

// rowAnchor puts the menu over the selected row. The panel is column 0, row 0 of a one-slot
// layout, so its outer top-left is (0, BodyY) — the panel itself owns the rest of the math
// (its frame, a live filter line, and the row height, all of which move with the density).
// A row scrolled off-page cannot happen for the SELECTED row, but the fallback keeps the
// menu on screen rather than at (0,0) if it ever does.
func (s *browseScreen) rowAnchor(sh *core.Shared) components.MenuAnchor {
	if a, ok := s.panel.RowAnchor(s.panel.List().Index(), 0, sh.BodyY()); ok {
		return a
	}
	return components.AnchorAt(0, sh.BodyY())
}
