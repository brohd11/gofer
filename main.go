// Command gofer is a small TUI file explorer: one directory at a time, enter walks into a
// folder and raises a menu on a file, and backspace walks back out. It opens wherever you
// launch it, and --cd-file hands the directory you quit in back to a shell wrapper so the
// explorer can move your shell with you. It is the thinnest possible consumer of
// bubblestack's components.FilePanel — the same panel gote's folder view uses.
package main

import "github.com/brohd11/gofer/cmd"

func main() {
	cmd.Execute()
}
