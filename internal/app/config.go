package app

import (
	"path/filepath"

	"github.com/brohd11/goutil/configdir"
)

// Config is ~/.gofer/config.yml. No field is omitempty, so the written file shows every key.
type Config struct {
	Compact    bool `yaml:"compact"`     // one row per entry; false adds a size line
	ShowHidden bool `yaml:"show_hidden"` // list dot files (--all turns them on for one run)
}

const configName = "config.yml"

// DefaultConfig is what a missing config means, and what EnsureConfig writes.
func DefaultConfig() Config { return Config{Compact: true} }

// Dir is ~/.gofer.
func Dir() (string, error) { return configdir.Dir("gofer") }

// ConfigPath is ~/.gofer/config.yml.
func ConfigPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configName), nil
}

// EnsureConfig returns the config path, writing the defaults first when it is missing, so
// `gofer config` opens the real schema rather than an empty buffer.
func EnsureConfig() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	if _, err := configdir.Ensure(dir, configName, DefaultConfig()); err != nil {
		return "", err
	}
	return filepath.Join(dir, configName), nil
}

// LoadConfig reads the config over DefaultConfig so absent keys keep their defaults. A
// missing file is not an error; a malformed one returns the defaults and the parse error.
func LoadConfig() (Config, error) {
	cfg := DefaultConfig()
	path, err := ConfigPath()
	if err != nil {
		return cfg, err
	}
	if err := configdir.Load(path, &cfg); err != nil {
		return DefaultConfig(), err
	}
	return cfg, nil
}

// SaveConfig writes the whole config atomically.
func SaveConfig(cfg Config) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	return configdir.SaveAtomic(dir, configName, cfg)
}
