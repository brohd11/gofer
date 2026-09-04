package app

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

// writeConfig materializes ~/.gofer/config.yml with body under a temp HOME and loads it.
func writeConfig(t *testing.T, body string) Config {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))
	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path, err := ConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("loading %q: %v", body, err)
	}
	return cfg
}

// TestMissingConfigIsTheDefaults: a fresh install needs no setup.
func TestMissingConfigIsTheDefaults(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg, DefaultConfig()) {
		t.Fatalf("missing config = %#v, want the defaults %#v", cfg, DefaultConfig())
	}
	if !cfg.Compact {
		t.Fatal("gofer's default density is compact")
	}
}

// TestPartialConfigKeepsTheDefaults is the trap this file exists to hold shut. Compact
// defaults to TRUE, so the zero Config is not the default Config: a load into a fresh
// struct would read a file that never mentions "compact" as compact: false, because YAML
// cannot tell an absent key from one written as false. LoadConfig loads OVER the defaults,
// so the key the user did not touch keeps its default.
func TestPartialConfigKeepsTheDefaults(t *testing.T) {
	cfg := writeConfig(t, "show_hidden: true\n")
	if !cfg.Compact {
		t.Fatal("a config that never mentions compact must keep the default (true)")
	}
	if !cfg.ShowHidden {
		t.Fatal("show_hidden: true should load as true")
	}
}

// TestConfigOverridesTheDefaults: what the file does say, wins.
func TestConfigOverridesTheDefaults(t *testing.T) {
	cfg := writeConfig(t, "compact: false\nshow_hidden: false\n")
	if cfg.Compact || cfg.ShowHidden {
		t.Fatalf("cfg = %#v, want both off", cfg)
	}
}

// TestMalformedConfig: a parse error surfaces, and the defaults come back whole rather
// than half-written by the failing unmarshal.
func TestMalformedConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))
	dir, _ := Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path, _ := ConfigPath()
	if err := os.WriteFile(path, []byte("compact: [not a bool\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig()
	if err == nil {
		t.Fatal("a malformed config should surface its parse error")
	}
	if !reflect.DeepEqual(cfg, DefaultConfig()) {
		t.Fatalf("cfg = %#v, want the defaults alongside the error", cfg)
	}
}

// TestEnsureConfigWritesTheSchema: the materialized file is where the settings are
// documented, so every key has to be in it — an omitted one is a setting the user cannot
// discover. It must also leave an existing file alone.
func TestEnsureConfigWritesTheSchema(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))
	path, err := EnsureConfig()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("EnsureConfig should have written the file: %v", err)
	}
	for _, key := range []string{"compact:", "show_hidden:"} {
		if !strings.Contains(string(raw), key) {
			t.Fatalf("a materialized config should show every key, %q is missing:\n%s", key, raw)
		}
	}

	if err := os.WriteFile(path, []byte("compact: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureConfig(); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile(path)
	if err != nil || string(raw) != "compact: false\n" {
		t.Fatalf("an existing config must not be rewritten, got %q (%v)", raw, err)
	}
}

// TestDefaultConfigRoundTrip: the materialized file is exactly marshal(DefaultConfig()), so
// writing it and reading it back has to land on DefaultConfig() again.
func TestDefaultConfigRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))
	if _, err := EnsureConfig(); err != nil {
		t.Fatal(err)
	}
	got, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, DefaultConfig()) {
		t.Fatalf("round trip = %#v, want %#v", got, DefaultConfig())
	}
}

// TestOptionsOverrideConfig: --all is a per-run override, and only when it was typed.
// Changed("all") is the only thing that can tell "never given" from "given as false".
func TestOptionsOverrideConfig(t *testing.T) {
	hidden := Config{ShowHidden: true}
	shown := Config{ShowHidden: false}
	for _, tt := range []struct {
		name string
		cfg  Config
		opts Options
		want bool
	}{
		{"config decides when the flag is absent", hidden, Options{}, true},
		{"config decides when the flag is absent (off)", shown, Options{}, false},
		{"--all overrides a config that hides", shown, Options{Hidden: true, HiddenSet: true}, true},
		{"an explicit false overrides a config that shows", hidden, Options{Hidden: false, HiddenSet: true}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := New("test", tt.cfg, tt.opts).ShowHidden; got != tt.want {
				t.Fatalf("ShowHidden = %v, want %v", got, tt.want)
			}
		})
	}
}
