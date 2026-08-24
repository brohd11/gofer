package cmd

import "testing"

// TestResolveCDFile pins the ladder: anything typed outranks the environment, and a blank
// environment value is not a path. The blank case is the one that matters in practice — a
// wrapper function always sets GOFER_CD_FILE, so emptying it for one command is the only
// way to browse without moving the shell.
func TestResolveCDFile(t *testing.T) {
	for _, tt := range []struct {
		name        string
		flag        string
		flagChanged bool
		env         string
		want        string
	}{
		{name: "nothing given", want: ""},
		{name: "the environment alone", env: "/tmp/a", want: "/tmp/a"},
		{name: "the flag alone", flag: "/tmp/b", flagChanged: true, want: "/tmp/b"},
		{
			name: "the flag outranks the environment",
			flag: "/tmp/b", flagChanged: true,
			env:  "/tmp/a",
			want: "/tmp/b",
		},
		{
			// A stray `export GOFER_CD_FILE=` must not make gofer write a file named "".
			name: "a blank environment value is not a path",
			env:  "   ",
			want: "",
		},
		{
			// An empty flag typed on purpose says the same thing, and must not fall back
			// to an environment the wrapper set.
			name: "an explicitly empty flag beats the environment",
			flag: "", flagChanged: true,
			env:  "/tmp/a",
			want: "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Always set, never merely read: an ambient GOFER_CD_FILE in the developer's
			// own shell (which the wrapper puts there) would otherwise decide the
			// "nothing given" case.
			t.Setenv(cdFileEnv, tt.env)
			if got := resolveCDFile(tt.flag, tt.flagChanged); got != tt.want {
				t.Fatalf("resolveCDFile(%q, %v) = %q, want %q", tt.flag, tt.flagChanged, got, tt.want)
			}
		})
	}
}
