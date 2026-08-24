package app

import (
	"os"
	"path/filepath"

	"github.com/brohd11/goutil/configdir"
)

// Config is the parsed ~/.gofer/config.yml: gofer's two view preferences, and nothing else.
// A missing file yields the defaults, so a fresh install needs no setup.
//
// Nothing is omitempty: the written file is the only place the schema is visible, so every
// key appears even when its value is the zero one. A config that showed a key only once it
// differed from the default would be a setting the user cannot discover.
type Config struct {
	Compact    bool `yaml:"compact"`     // one row per entry; false gives each a name and a size line
	ShowHidden bool `yaml:"show_hidden"` // list dot files (--all turns them on for one run)
}

// DefaultConfig is what a missing ~/.gofer/config.yml means, and — since EnsureConfig
// writes exactly this — what a fresh one says. Compact is on: a listing is something you
// scan down, and one row an entry is what makes that reading rather than scrolling. The
// standard rows are a deliberate choice you make with alt+r or this key.
func DefaultConfig() Config { return Config{Compact: true} }

// Dir is ~/.gofer, gofer's config home. The ~/.<app> convention itself is
// goutil/configdir's; this pins gofer's own name.
func Dir() (string, error) {
	return configdir.Dir("gofer")
}

// ConfigPath is ~/.gofer/config.yml — what LoadConfig reads, SaveConfig writes, and
// `gofer config` opens.
func ConfigPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yml"), nil
}

// EnsureConfig returns the config path, materializing a defaults file first when none
// exists. `gofer config` on a fresh install should open the real schema to edit, not an
// empty buffer that gives no hint what belongs in it.
func EnsureConfig() (string, error) {
	path, err := ConfigPath()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := SaveConfig(DefaultConfig()); err != nil {
			return "", err
		}
	}
	return path, nil
}

// LoadConfig reads ~/.gofer/config.yml. A missing file is not an error — it returns the
// defaults; a malformed one returns the parse error alongside them.
//
// The load runs OVER DefaultConfig(), and that is load-bearing rather than tidy: Compact
// defaults to true, so the zero Config is not the default Config. Unmarshalling into a
// fresh struct would make a file that sets only show_hidden silently turn the density off,
// because "absent" and "false" are the same thing in YAML.
func LoadConfig() (Config, error) {
	cfg := DefaultConfig()
	path, err := ConfigPath()
	if err != nil {
		return cfg, err
	}
	if err := configdir.Load(path, &cfg); err != nil {
		// A half-parsed config is worse than none: rebuild the defaults rather than
		// returning whatever the failing unmarshal happened to have written.
		return DefaultConfig(), err
	}
	return cfg, nil
}

// SaveConfig writes the complete gofer config atomically — a failed write cannot truncate a
// working config. The atomic-write mechanics are goutil/configdir's.
func SaveConfig(cfg Config) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	return configdir.SaveAtomic(dir, "config.yml", cfg)
}
