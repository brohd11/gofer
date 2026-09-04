package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestWriteCDFile: the file is what a shell wrapper cds into, so it holds exactly one
// path and a newline — `cat` of it has to be usable unquoted-ish in `cd -- "$(...)"`.
func TestWriteCDFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cd")

	if err := writeCDFile(path, "/some/where"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "/some/where\n" {
		t.Fatalf("cd file = %q, want the path and a newline", raw)
	}
	if info, err := os.Stat(path); err != nil {
		t.Fatal(err)
	} else if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("cd file mode = %v, want 0600", info.Mode().Perm())
	}
}

// TestWriteCDFileUnset: no flag, no file. This is the common path — the flag only makes
// sense from inside the wrapper function.
func TestWriteCDFileUnset(t *testing.T) {
	if err := writeCDFile("", "/some/where"); err != nil {
		t.Fatalf("an unset --cd-file should be a no-op, got %v", err)
	}
}

// TestShortHome: the breadcrumb reads paths the way a shell prompt does, and leaves alone
// what it cannot shorten.
func TestShortHome(t *testing.T) {
	home := "/Users/x"
	for _, tt := range []struct{ in, want string }{
		{"/Users/x", "~"},
		{filepath.Join(home, "main", "go"), filepath.Join("~", "main", "go")},
		{"/etc", "/etc"},
		{"/Users/xylophone", "/Users/xylophone"}, // a prefix is not a parent
	} {
		if got := shortHome(tt.in, home); got != tt.want {
			t.Errorf("shortHome(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
	if got := shortHome("/etc", ""); got != "/etc" {
		t.Errorf("no home should leave the path alone, got %q", got)
	}
}

// TestCDFileFollowsTheBrowse joins the two halves: the panel moves the ctx as you walk, and
// the ctx is what the cd file is written from after the program exits. That seam is the
// whole feature, and each half passing on its own would not prove it.
func TestCDFileFollowsTheBrowse(t *testing.T) {
	root := tree(t)
	s, sh := newBrowse(t, root, false)
	selectRow(t, s.panel.List(), "sub/")
	s.Update(sh, keyMsg("d"))

	path := filepath.Join(t.TempDir(), "cd")
	if err := writeCDFile(path, Of(sh).Dir); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "sub") + "\n"; string(raw) != want {
		t.Fatalf("cd file = %q, want the folder browsed to (%q)", raw, want)
	}
}
