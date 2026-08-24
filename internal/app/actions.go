package app

import (
	"github.com/brohd11/bubblestack/components"
	"github.com/brohd11/bubblestack/core"
)

// actionsMenu is the small Actions picker opened with "a": the shared bubblestack menu
// (theme, self-update, refresh), where gofer's Refresh is a re-read of the folder on
// screen — the same thing the global Refresh key fires. No docs pages are compiled in, so
// the menu shows no "? Docs" row.
func actionsMenu(sh *core.Shared) *components.PickerScreen {
	return components.NewActionsMenu(selfUpdateHooks(Of(sh).Version),
		"re-read the current folder", refreshAction, nil)
}
