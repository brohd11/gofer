package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/brohd11/bubblestack/components"
	"github.com/brohd11/bubblestack/core"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// tree builds root/{sub/{deep.txt}, notes.txt, .hidden} and returns root.
func tree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	for rel, body := range map[string]string{
		"notes.txt":    "hello\n",
		".hidden":      "x\n",
		"sub/deep.txt": "deep\n",
	} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// newBrowse builds the screen directly — enough for anything that does not push. The
// defaults are gofer's own, so a test says only what it is actually about.
func newBrowse(t *testing.T, root string, showHidden bool) (*browseScreen, *core.Shared) {
	t.Helper()
	cfg := DefaultConfig()
	cfg.ShowHidden = showHidden
	return newBrowseCfg(t, cfg, Options{Dir: root})
}

// newBrowseCfg is newBrowse for a specific config and launch options — the two inputs
// New reconciles.
func newBrowseCfg(t *testing.T, cfg Config, opts Options) (*browseScreen, *core.Shared) {
	t.Helper()
	sh := core.NewShared(New("test", cfg, opts))
	s := NewBrowseScreen(sh).(*browseScreen)
	s.Init(sh)
	s.SetSize(sh, 100, 30)
	return s, sh
}

// newBrowseRouter builds the same screen through the real router. Anything that pushes a
// screen needs this path: Update hands back the navigation Action but cannot apply it.
func newBrowseRouter(t *testing.T, root string) (tea.Model, *browseScreen, *core.Shared) {
	t.Helper()
	sh := core.NewShared(New("test", DefaultConfig(), Options{Dir: root}))
	r := core.NewRouter(sh, []core.TabEntry{{Title: "Files", New: NewBrowseScreen}})
	r.Init()
	var model tea.Model = r
	model, _ = model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return model, model.(core.Router).Top().(*browseScreen), sh
}

func rowTitles(l *list.Model) []string {
	out := []string{}
	for _, it := range l.VisibleItems() {
		if t, ok := it.(interface{ Title() string }); ok {
			out = append(out, t.Title())
		}
	}
	return out
}

func hasRow(rows []string, name string) bool {
	for _, r := range rows {
		if r == name {
			return true
		}
	}
	return false
}

func selectRow(t *testing.T, l *list.Model, name string) {
	t.Helper()
	for i, r := range rowTitles(l) {
		if r == name {
			l.Select(i)
			return
		}
	}
	t.Fatalf("no row %q in %v", name, rowTitles(l))
}

// TestBrowseWalks: the screen opens on the launch directory, enter walks into a folder, and
// the three things that track the directory — the ctx, the breadcrumb and the directory the
// global terminal keys act on — all follow.
func TestBrowseWalks(t *testing.T) {
	root := tree(t)
	s, sh := newBrowse(t, root, false)

	rows := rowTitles(s.panel.List())
	if !hasRow(rows, "sub/") || !hasRow(rows, "notes.txt") {
		t.Fatalf("the launch directory should be listed, got %v", rows)
	}
	if dir, ok := s.LocateDir(); !ok || dir != root {
		t.Fatalf("LocateDir = %q,%v, want the launch directory", dir, ok)
	}

	selectRow(t, s.panel.List(), "sub/")
	s.Update(sh, tea.KeyMsg{Type: tea.KeyEnter})

	sub := filepath.Join(root, "sub")
	if Of(sh).Dir != sub {
		t.Fatalf("Ctx.Dir = %q, want %q", Of(sh).Dir, sub)
	}
	if dir, _ := s.LocateDir(); dir != sub {
		t.Fatalf("LocateDir = %q, want the folder on screen %q", dir, sub)
	}
	if crumb := s.CrumbLabel(false); crumb == "" || !hasRow([]string{crumb}, shortHome(sub, s.home)) {
		t.Fatalf("CrumbLabel = %q, want the current directory", crumb)
	}
	if rows := rowTitles(s.panel.List()); !hasRow(rows, "..") || !hasRow(rows, "deep.txt") {
		t.Fatalf("inside sub the rows should be .. and deep.txt, got %v", rows)
	}

	// And back out — gofer is unclamped, so there is always a way up.
	s.Update(sh, tea.KeyMsg{Type: tea.KeyBackspace})
	if Of(sh).Dir != root {
		t.Fatalf("backspace should walk to the parent; Dir = %q", Of(sh).Dir)
	}
}

// TestUnclamped: the launch directory is a starting point, not a floor.
func TestUnclamped(t *testing.T) {
	root := tree(t)
	s, sh := newBrowse(t, root, false)
	s.Update(sh, tea.KeyMsg{Type: tea.KeyBackspace})
	if want := filepath.Dir(root); Of(sh).Dir != want {
		t.Fatalf("Dir = %q, want %q — gofer has no floor", Of(sh).Dir, want)
	}
}

// TestFilePickRaisesMenu: enter on a file opens the context menu rather than acting, so the
// verb list can grow without the behavior changing.
func TestFilePickRaisesMenu(t *testing.T) {
	root := tree(t)
	model, s, _ := newBrowseRouter(t, root)

	selectRow(t, s.panel.List(), "notes.txt")
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if _, ok := model.(core.Router).Top().(*components.MenuScreen); !ok {
		t.Fatalf("picking a file should push a MenuScreen, got %T", model.(core.Router).Top())
	}
}

// TestFileMenuItems: one row for now, and it owns its own dismissal.
func TestFileMenuItems(t *testing.T) {
	items := fileMenuItems(components.FileEntry{Name: "notes.txt", Path: "/tmp/notes.txt"})
	if len(items) != 1 || items[0].Label != "Open in default app" {
		t.Fatalf("file menu = %+v, want one open row", items)
	}
	if items[0].Pick == nil {
		t.Fatal("the open row must be pickable")
	}
}

// TestHiddenToggle: "." widens the listing and narrows it again. Claimed from the panel's
// row-key hook, which is what makes a bare "." safe beside a /-filter.
func TestHiddenToggle(t *testing.T) {
	root := tree(t)
	s, sh := newBrowse(t, root, false)

	if hasRow(rowTitles(s.panel.List()), ".hidden") {
		t.Fatal("dot files should be off by default")
	}
	s.Update(sh, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(".")})
	if !hasRow(rowTitles(s.panel.List()), ".hidden") {
		t.Fatalf("\".\" should show dot files, got %v", rowTitles(s.panel.List()))
	}
	s.Update(sh, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(".")})
	if hasRow(rowTitles(s.panel.List()), ".hidden") {
		t.Fatal("\".\" should hide them again")
	}
}

// TestShowHiddenFromFlag: -a is the same switch, thrown at launch.
func TestShowHiddenFromFlag(t *testing.T) {
	root := tree(t)
	s, _ := newBrowse(t, root, true)
	if !hasRow(rowTitles(s.panel.List()), ".hidden") {
		t.Fatalf("--all should list dot files from the start, got %v", rowTitles(s.panel.List()))
	}
}

// TestConfigDecidesTheView: both view preferences come from the config, and the panel is
// built from them rather than from a constant.
func TestConfigDecidesTheView(t *testing.T) {
	root := tree(t)
	s, _ := newBrowseCfg(t, Config{Compact: false, ShowHidden: true}, Options{Dir: root})

	if s.panel.Compact() {
		t.Fatal("compact: false should open on the standard rows")
	}
	if !hasRow(rowTitles(s.panel.List()), ".hidden") {
		t.Fatalf("show_hidden: true should list dot files, got %v", rowTitles(s.panel.List()))
	}
}

// TestDensityKey: alt+r flips the row height without moving the listing.
func TestDensityKey(t *testing.T) {
	root := tree(t)
	s, sh := newBrowse(t, root, false)
	if !s.panel.Compact() {
		t.Fatal("gofer should start on the compact rows (DefaultConfig)")
	}
	s.Update(sh, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r"), Alt: true})
	if s.panel.Compact() {
		t.Fatal("alt+r should flip the density")
	}
	if Of(sh).Dir != root {
		t.Fatal("the flip must not move the listing")
	}
}
