// Command gofer is a small TUI file explorer built on bubblestack's FilePanel. With the
// shell wrapper ($GOFER_CD_FILE) your shell follows the directory you quit in.
package main

import "github.com/brohd11/gofer/cmd"

func main() {
	cmd.Execute()
}
