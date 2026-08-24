package cmd

import (
	"github.com/brohd11/gofer/internal/app"
	"github.com/brohd11/goutil/configcmd"
)

var configCmd = configcmd.NewCommand(configcmd.Options{
	Path: app.ConfigPath,
	Dir:  app.Dir,
	Ensure: func() error {
		_, err := app.EnsureConfig()
		return err
	},
})

func init() {
	rootCmd.AddCommand(configCmd)
}
