package app

import (
	"github.com/brohd11/bubblestack/components"
	"github.com/brohd11/bubblestack/core"
	"github.com/brohd11/bubblestack/sysopen"
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
func fileMenuItems(e components.FileEntry) []components.MenuItem {
	return []components.MenuItem{
		{
			Label: "Open in default app",
			Pick: func(*core.Shared) core.Action {
				return core.Seq(core.Pop(), sysopen.Path(e.Path, false))
			},
		},
	}
}

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
