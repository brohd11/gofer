package app

import (
	"os"
	"path/filepath"
	"reflect"
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

// keyMsg is a bare rune keypress, the form the screen's own letter keys arrive in.
func keyMsg(k string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
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
	s.Update(sh, keyMsg("d"))

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
	s.Update(sh, keyMsg("x"))
	if Of(sh).Dir != root {
		t.Fatalf("x should walk to the parent; Dir = %q", Of(sh).Dir)
	}
}

// TestUnclamped: the launch directory is a starting point, not a floor.
func TestUnclamped(t *testing.T) {
	root := tree(t)
	s, sh := newBrowse(t, root, false)
	s.Update(sh, keyMsg("x"))
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

// TestFileMenuItems: the editor row leads on a text file and is absent on a binary, and
// every row is pickable. The order is the assertion that matters — editing is the likelier
// verb on a file gofer can read, so it must be the row the cursor opens on.
func TestFileMenuItems(t *testing.T) {
	dir := t.TempDir()
	text := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(text, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "image.png")
	if err := os.WriteFile(binary, []byte("\x89PNG\r\n\x1a\n\x00\x00"), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		path string
		want []string
	}{
		{"notes.txt", text, []string{"Open in text editor", "Open in default app"}},
		{"image.png", binary, []string{"Open in default app"}},
		// A path that cannot be read is not text, so it gets the shorter menu rather than a
		// row that would launch an editor on nothing.
		{"gone.txt", filepath.Join(dir, "gone.txt"), []string{"Open in default app"}},
	}
	for _, tc := range cases {
		items := fileMenuItems(components.FileEntry{Name: tc.name, Path: tc.path, Dir: dir})
		var labels []string
		for _, it := range items {
			labels = append(labels, it.Label)
			if it.Pick == nil {
				t.Fatalf("%s: row %q must be pickable", tc.name, it.Label)
			}
		}
		if !reflect.DeepEqual(labels, tc.want) {
			t.Errorf("%s menu = %v, want %v", tc.name, labels, tc.want)
		}
	}
}

// TestEditorCloseRefreshes: the message the ExecProcess callback returns re-reads the folder
// and names the file on the status line. Without the re-read the size column would still
// show what the file was before the editor wrote it.
func TestEditorCloseRefreshes(t *testing.T) {
	root := tree(t)
	s, sh := newBrowse(t, root, false)

	if err := os.WriteFile(filepath.Join(root, "fresh.txt"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, act := s.Update(sh, editorClosedMsg{name: "notes.txt"})

	if !hasRow(rowTitles(s.panel.List()), "fresh.txt") {
		t.Error("closing the editor should re-read the folder")
	}
	if act.Msg == nil {
		t.Error("closing the editor should say so on the status line")
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

// TestDirPickRaisesMenu: enter on a FOLDER opens its menu rather than walking in. That is
// the whole point of moving the walk onto d — a folder used to be the one row with no verbs.
func TestDirPickRaisesMenu(t *testing.T) {
	root := tree(t)
	model, s, sh := newBrowseRouter(t, root)

	selectRow(t, s.panel.List(), "sub/")
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if _, ok := model.(core.Router).Top().(*components.MenuScreen); !ok {
		t.Fatalf("picking a folder should push a MenuScreen, got %T", model.(core.Router).Top())
	}
	if Of(sh).Dir != root {
		t.Fatalf("enter must not walk; Dir = %q, want %q", Of(sh).Dir, root)
	}
}

// TestUpRowStillWalks: ".." is the exception — enter on it walks up, because the way out of
// a folder is a navigation affordance and not a folder you act on.
func TestUpRowStillWalks(t *testing.T) {
	root := tree(t)
	model, s, sh := newBrowseRouter(t, root)

	selectRow(t, s.panel.List(), "sub/")
	s.Update(sh, keyMsg("d"))
	selectRow(t, s.panel.List(), "..")
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if _, ok := model.(core.Router).Top().(*components.MenuScreen); ok {
		t.Fatal("the \"..\" row should walk, not raise a menu")
	}
	if Of(sh).Dir != root {
		t.Fatalf("enter on \"..\" should walk to the parent; Dir = %q", Of(sh).Dir)
	}
}

// TestDescendOnUpRow: d means "into whatever the cursor is on", and on ".." that is the
// parent. It is a screen key rather than the panel's OnKey precisely so this row works —
// OnKey reports unhandled on it, and it is the row an unclamped explorer lands on.
func TestDescendOnUpRow(t *testing.T) {
	root := tree(t)
	s, sh := newBrowse(t, root, false)

	selectRow(t, s.panel.List(), "sub/")
	s.Update(sh, keyMsg("d"))
	selectRow(t, s.panel.List(), "..")
	s.Update(sh, keyMsg("d"))

	if Of(sh).Dir != root {
		t.Fatalf("d on \"..\" should walk to the parent; Dir = %q", Of(sh).Dir)
	}
}

// TestDescendIgnoresFiles: there is nothing to descend into on a file, and d must not be a
// second way to open its menu — that is enter's job.
func TestDescendIgnoresFiles(t *testing.T) {
	root := tree(t)
	s, sh := newBrowse(t, root, false)

	selectRow(t, s.panel.List(), "notes.txt")
	s.Update(sh, keyMsg("d"))
	if Of(sh).Dir != root {
		t.Fatalf("d on a file should do nothing; Dir = %q", Of(sh).Dir)
	}
}

// TestDirMenuItems: the folder menu's rows and their order. "Open folder" leads because it
// is the second press of what used to be one — the two below it are the new reach, since
// the global t/ctrl+t only ever act on the folder you are already in.
func TestDirMenuItems(t *testing.T) {
	root := tree(t)
	s, _ := newBrowse(t, root, false)

	var labels []string
	for _, it := range s.dirMenuItems(components.FileEntry{Name: "sub", Path: filepath.Join(root, "sub"), Dir: root, IsDir: true}) {
		labels = append(labels, it.Label)
		if it.Pick == nil {
			t.Fatalf("row %q must be pickable", it.Label)
		}
	}
	want := []string{"Open folder", "Open in file manager", "Terminal here"}
	if !reflect.DeepEqual(labels, want) {
		t.Errorf("folder menu = %v, want %v", labels, want)
	}
}

// TestDirMenuOpenFolderWalks: the first row is the one that keeps enter cheap — it lands in
// the folder the menu was raised over.
func TestDirMenuOpenFolderWalks(t *testing.T) {
	root := tree(t)
	s, sh := newBrowse(t, root, false)

	items := s.dirMenuItems(components.FileEntry{Name: "sub", Path: filepath.Join(root, "sub"), Dir: root, IsDir: true})
	items[0].Pick(sh)

	if want := filepath.Join(root, "sub"); Of(sh).Dir != want {
		t.Fatalf("Open folder should walk in; Dir = %q, want %q", Of(sh).Dir, want)
	}
}

// TestFolderKeysAreOwnedByTheScreen: a live /-filter owns every character, so neither folder
// key may be taken back out of the query.
func TestFolderKeysAreOwnedByTheScreen(t *testing.T) {
	root := tree(t)
	s, sh := newBrowse(t, root, false)

	s.Update(sh, keyMsg("/"))
	if !s.Filtering() {
		t.Fatal("/ should open the filter")
	}
	before := Of(sh).Dir
	s.Update(sh, keyMsg("d"))
	s.Update(sh, keyMsg("x"))
	if Of(sh).Dir != before {
		t.Fatalf("d and x must be typed into a live filter, not acted on; Dir = %q", Of(sh).Dir)
	}
}

// TestFolderKeysSurviveTheRouter is the load-bearing test for the choice of x: it is
// core.Keys.NextTab's second keycode, and it only reaches the panel because switchTab is a
// no-op below two tabs and reports the key unhandled. Every other folder-key test drives
// the screen directly and would pass even if the router ate it, so this one goes through
// the real router — and will fail the day gofer grows a second tab, which is exactly when
// somebody needs to be told.
func TestFolderKeysSurviveTheRouter(t *testing.T) {
	root := tree(t)
	model, s, sh := newBrowseRouter(t, root)

	selectRow(t, s.panel.List(), "sub/")
	model, _ = model.Update(keyMsg("d"))
	sub := filepath.Join(root, "sub")
	if Of(sh).Dir != sub {
		t.Fatalf("d should reach the screen through the router; Dir = %q, want %q", Of(sh).Dir, sub)
	}

	model.Update(keyMsg("x"))
	if Of(sh).Dir != root {
		t.Fatalf("x should reach the panel through the router; Dir = %q, want %q", Of(sh).Dir, root)
	}
}

// clickFolder presses button over the "sub/" row at an ABSOLUTE terminal cell, the way a
// real one arrives: the router and ModularScreen do the translation to pane-local
// coordinates between here and the panel, and driving the panel directly would skip both.
func clickFolder(t *testing.T, model tea.Model, s *browseScreen, button tea.MouseButton) tea.Model {
	t.Helper()
	selectRow(t, s.panel.List(), "sub/")
	row, ok := s.panel.RowY(s.panel.List().Index())
	if !ok {
		t.Fatal("the sub/ row should be on-page")
	}
	model, _ = model.Update(tea.MouseMsg{
		Action: tea.MouseActionPress, Button: button, X: 5, Y: s.sh.BodyY() + row,
	})
	return model
}

// TestLeftClickEntersFolder: the mouse mirrors the keys — a left click is d, so it walks
// rather than raising the menu the way it used to.
func TestLeftClickEntersFolder(t *testing.T) {
	root := tree(t)
	model, s, sh := newBrowseRouter(t, root)

	model = clickFolder(t, model, s, tea.MouseButtonLeft)

	if _, ok := model.(core.Router).Top().(*components.MenuScreen); ok {
		t.Fatal("a left click on a folder should walk, not raise the menu")
	}
	if want := filepath.Join(root, "sub"); Of(sh).Dir != want {
		t.Fatalf("Dir = %q, want %q", Of(sh).Dir, want)
	}
}

// TestRightClickRaisesFolderMenu: the other button is enter, so it raises the same menu in
// the same place — anchored on the row it landed on, since the click selects first.
func TestRightClickRaisesFolderMenu(t *testing.T) {
	root := tree(t)
	model, s, sh := newBrowseRouter(t, root)

	model = clickFolder(t, model, s, tea.MouseButtonRight)

	if _, ok := model.(core.Router).Top().(*components.MenuScreen); !ok {
		t.Fatalf("a right click on a folder should push a MenuScreen, got %T", model.(core.Router).Top())
	}
	if Of(sh).Dir != root {
		t.Fatalf("a right click must not walk; Dir = %q", Of(sh).Dir)
	}
}
