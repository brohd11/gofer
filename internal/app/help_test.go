package app

import (
	"strings"
	"testing"

	"github.com/brohd11/bubblestack/components"
	"github.com/brohd11/bubblestack/core"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// TestHelpKeyToggles: "?" opens the page and "?" closes it again — the toggle asked for,
// which a plain pushed screen does not give you (its own keys never see the second press).
func TestHelpKeyToggles(t *testing.T) {
	root := tree(t)
	model, _, _ := newBrowseRouter(t, root)
	question := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}

	model, _ = model.Update(question)
	if _, ok := model.(core.Router).Top().(*components.DocScreen); !ok {
		t.Fatalf("? should open the help page, got %T", model.(core.Router).Top())
	}
	model, _ = model.Update(question)
	if _, ok := model.(core.Router).Top().(*browseScreen); !ok {
		t.Fatalf("? again should close it, got %T", model.(core.Router).Top())
	}

	// esc closes it too, the way it closes anything else.
	model, _ = model.Update(question)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if _, ok := model.(core.Router).Top().(*browseScreen); !ok {
		t.Fatalf("esc should close the help page, got %T", model.(core.Router).Top())
	}
}

// TestHelpPageIsTheCompleteReference pins the split the bar and the page make: the bar
// names only the way in, so every key gofer binds has to be written on the page or it is
// written nowhere.
func TestHelpPageIsTheCompleteReference(t *testing.T) {
	root := tree(t)
	s, sh := newBrowse(t, root, false)

	help := s.helpText()
	for _, want := range []string{
		"enter", "esc", "backspace", "/", // navigation
		"dot files", ".", // the key that moved off the bar
		"alt+r", "row density",
		"t", "terminal", "ctrl+t", "file manager",
		"a", "actions", "q", "quit", "?",
		"~/.gofer/config.yml", "GOFER_CD_FILE", // what a keys page cannot say but a reader wants
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("the ? page is the only place gofer's keys are written; missing %q:\n%s", want, help)
		}
	}
	if strings.Contains(help, "⌥") {
		t.Fatalf("alt chords should be spelled \"alt+\" throughout, not ⌥:\n%s", help)
	}

	// And the bar keeps the pointer and nothing else gofer owns.
	bar := s.HelpView(sh)
	if !strings.Contains(bar, "?") {
		t.Fatalf("the bar should point at the page:\n%s", bar)
	}
	for _, gone := range []string{"dot files", "row density", "actions"} {
		if strings.Contains(bar, gone) {
			t.Fatalf("%q belongs on the page, not the bar:\n%s", gone, bar)
		}
	}
}

// TestHelpReachableWhileFiltering: alt+? outranks the capture gate, because a help page you
// cannot open mid-filter is one you cannot open when you most need it.
func TestHelpReachableWhileFiltering(t *testing.T) {
	root := tree(t)
	model, s, _ := newBrowseRouter(t, root)

	s.panel.List().SetFilterState(list.Filtering)
	if !s.Filtering() {
		t.Fatal("setup: the panel should be capturing")
	}
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?"), Alt: true})
	if _, ok := model.(core.Router).Top().(*components.DocScreen); !ok {
		t.Fatalf("alt+? should open the page while filtering, got %T", model.(core.Router).Top())
	}
}

// TestBareHelpKeyIsTextWhileFiltering is the other half: "?" is a typable character, so a
// live query must keep it.
func TestBareHelpKeyIsTextWhileFiltering(t *testing.T) {
	root := tree(t)
	model, s, _ := newBrowseRouter(t, root)

	s.panel.List().SetFilterState(list.Filtering)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	if _, ok := model.(core.Router).Top().(*browseScreen); !ok {
		t.Fatalf("a bare ? must not be stolen from a filter query, got %T", model.(core.Router).Top())
	}
}
