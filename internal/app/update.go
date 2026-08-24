package app

import (
	"github.com/brohd11/bubblestack/components"
	"github.com/brohd11/bubblestack/core"
	bsupdate "github.com/brohd11/bubblestack/selfupdate"

	tea "github.com/charmbracelet/bubbletea"
)

// selfUpdateRepo is gofer's own GitHub repo slug, passed to the shared self-update bridge.
const selfUpdateRepo = "brohd11/gofer"

// selfUpdateHooks builds the shared self-update flow's hook set for gofer. The
// goutil↔components wiring lives in the bubblestack/selfupdate bridge, which every app in
// the monorepo shares.
func selfUpdateHooks(version string) components.SelfUpdateHooks {
	return bsupdate.Hooks("gofer", selfUpdateRepo, version)
}

// SelfUpdateCheckCmd is the app-level startup command (wired onto bubblestack Config.Init):
// it checks gofer's own repo for a newer release off the UI thread and, only when an update
// is available, writes an "update available" line to the shared status line and log.
// Anything else (up to date, dev build, fetch error) is silent.
func SelfUpdateCheckCmd(sh *core.Shared) tea.Cmd {
	return components.SelfUpdateCheckCmd(selfUpdateHooks(Of(sh).Version))
}
