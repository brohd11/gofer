package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// TestResolveShell pins the ladder: what was typed outranks $SHELL, an unset $SHELL is the
// one case resolveShell itself rejects, and everything else — including a shell it has
// never heard of — is handed on to wrapperFor to judge.
func TestResolveShell(t *testing.T) {
	for _, tt := range []struct {
		name    string
		arg     string
		env     string
		want    string
		wantErr bool
	}{
		{name: "the argument alone", arg: "fish", want: "fish"},
		{name: "the environment alone", env: "/bin/zsh", want: "zsh"},
		{name: "the argument outranks the environment", arg: "bash", env: "/bin/zsh", want: "bash"},
		{name: "a full path is read as its basename", env: "/opt/homebrew/bin/fish", want: "fish"},
		{name: "case does not matter", arg: "ZSH", want: "zsh"},
		{name: "surrounding space does not matter", arg: "  bash  ", want: "bash"},
		// Not resolveShell's to reject: it found a name, and wrapperFor is the one that
		// knows which names exist.
		{name: "an unknown name still resolves", arg: "tcsh", want: "tcsh"},
		{name: "an unknown $SHELL still resolves", env: "/bin/tcsh", want: "tcsh"},
		{name: "nothing at all", wantErr: true},
		{name: "a blank $SHELL is not a shell", env: "   ", wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveShell(tt.arg, tt.env)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("resolveShell(%q, %q) = %q, want an error", tt.arg, tt.env, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveShell(%q, %q): %v", tt.arg, tt.env, err)
			}
			if got != tt.want {
				t.Fatalf("resolveShell(%q, %q) = %q, want %q", tt.arg, tt.env, got, tt.want)
			}
		})
	}
}

// TestWrapperFor checks the properties the wrappers have to hold whatever their syntax:
// each defines the function, sets the variable gofer reads, and does not call itself.
func TestWrapperFor(t *testing.T) {
	for _, shell := range supportedShells {
		t.Run(shell, func(t *testing.T) {
			body, err := wrapperFor(shell)
			if err != nil {
				t.Fatalf("wrapperFor(%q): %v", shell, err)
			}
			if !strings.Contains(body, cdFileEnv+"=") {
				t.Errorf("the %s wrapper never sets $%s:\n%s", shell, cdFileEnv, body)
			}
			if !strings.HasSuffix(body, "\n") {
				t.Errorf("the %s wrapper does not end in a newline, so an rc file would run it into the next line", shell)
			}
			// The one bug that would make the wrapper hang forever instead of failing
			// loudly: calling the function from inside itself.
			for _, line := range strings.Split(body, "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "gofer ") || trimmed == "gofer" {
					t.Errorf("the %s wrapper calls itself: %q", shell, trimmed)
				}
			}
		})
	}
}

// TestWrapperSyntaxIsNotShared guards the split that made fish a separate string in the
// first place — the two bodies share no syntax, so a paste from one into the other is a
// mistake that only shows up in someone's rc file.
func TestWrapperSyntaxIsNotShared(t *testing.T) {
	if !strings.HasPrefix(posixWrapper, "gofer() {") {
		t.Errorf("the posix wrapper no longer opens with a function definition:\n%s", posixWrapper)
	}
	if strings.Contains(posixWrapper, "set -l") {
		t.Errorf("fish syntax leaked into the posix wrapper:\n%s", posixWrapper)
	}
	if !strings.HasPrefix(fishWrapper, "function gofer\n") || !strings.HasSuffix(fishWrapper, "end\n") {
		t.Errorf("the fish wrapper is not a fish function:\n%s", fishWrapper)
	}
	if strings.Contains(fishWrapper, "local ") {
		t.Errorf("posix syntax leaked into the fish wrapper:\n%s", fishWrapper)
	}
	// bash and zsh are deliberately the same text; if they ever diverge the map is the
	// place that says so.
	if wrappers["bash"] != wrappers["zsh"] {
		t.Error("bash and zsh no longer share a wrapper — say why here")
	}
}

// TestWrapperForRejectsAnUnknownShell: the error has to name the alternatives, because the
// place it surfaces is an eval in an rc file with no usage text around it.
func TestWrapperForRejectsAnUnknownShell(t *testing.T) {
	_, err := wrapperFor("tcsh")
	if err == nil {
		t.Fatal("wrapperFor(\"tcsh\") = nil error, want one")
	}
	for _, shell := range supportedShells {
		if !strings.Contains(err.Error(), shell) {
			t.Errorf("the error does not name %q: %v", shell, err)
		}
	}
}

// TestRunFuncPrintsOnlyTheWrapper: the output is eval'd, so a banner or a trailing note
// would be executed rather than read.
func TestRunFuncPrintsOnlyTheWrapper(t *testing.T) {
	// An ambient $SHELL must not decide a test that names its shell.
	t.Setenv("SHELL", "/bin/bash")

	var out, errOut bytes.Buffer
	funcCmd.SetOut(&out)
	funcCmd.SetErr(&errOut)
	t.Cleanup(func() { funcCmd.SetOut(nil); funcCmd.SetErr(nil) })

	if err := runFunc(funcCmd, []string{"fish"}); err != nil {
		t.Fatalf("runFunc: %v", err)
	}
	if out.String() != fishWrapper {
		t.Fatalf("runFunc printed\n%s\nwant\n%s", out.String(), fishWrapper)
	}
	if errOut.Len() != 0 {
		t.Errorf("runFunc wrote to stderr: %q", errOut.String())
	}

	// No argument: $SHELL decides, and nothing else is added.
	out.Reset()
	if err := runFunc(funcCmd, nil); err != nil {
		t.Fatalf("runFunc with no argument: %v", err)
	}
	if out.String() != posixWrapper {
		t.Fatalf("runFunc with no argument printed\n%s\nwant the posix wrapper", out.String())
	}
}
