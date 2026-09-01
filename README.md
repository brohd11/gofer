# gofer

A small TUI file explorer: one directory at a time, folders first. Enter walks into a
folder and raises a menu on a file, backspace walks back out.

```
gofer            # the current directory
gofer /path      # an explicit start
```

## Follow it out — `$GOFER_CD_FILE`

A program cannot change its parent shell's directory, so gofer writes the directory you
quit in and lets a shell function do the `cd`. Add this to your rc file:

```sh
gofer() {
  local tmp dir ret
  tmp="$(mktemp -t gofer-cd)"
  GOFER_CD_FILE="$tmp" command gofer "$@"
  ret=$?
  dir="$(cat -- "$tmp" 2>/dev/null)"; rm -f -- "$tmp"
  [ -n "$dir" ] && [ "$dir" != "$PWD" ] && cd -- "$dir"
  return "$ret"
}
```

`command gofer` keeps the function from calling itself, and is also how you bypass the
wrapper for one run — `command gofer .` browses without moving your shell.

The environment variable, rather than a flag, is what lets the wrapper be unconditional:
`gofer config` and the other subcommands pass straight through it, and since only a browse
writes the file, they leave your shell where it was. So does a crash.

`return "$ret"` is not optional: the `cd` line is a test that fails whenever there is
nowhere to go, so without it the function would report failure after every browse that
stayed put — and would swallow a real error from `gofer update`. The variable is `ret` and
not `status` because zsh reserves `status` as a read-only alias for `$?`.

`--cd-file <path>` is the same thing typed directly, for a one-off outside the wrapper; a
typed flag wins over the variable.

## Keys

| key | |
|---|---|
| `d` / left click | into the folder under the cursor |
| `x` | up a folder (the `..` row does the same) |
| `enter` / right click | the menu on this row, folder or file |
| `/` | filter the current folder |
| `.` | show or hide dot files (`-a` shows them for one run) |
| `alt+r` | switch row density — one line per entry, or name plus size |
| `?` | the full key list (`?` again or `esc` closes it) |
| `t` / `T` | a terminal, or a terminal window, in the folder on screen |
| `ctrl+t` | open the folder on screen in the file manager |
| `a` | actions — theme, self-update, refresh |
| `r` | re-read the folder |
| `q` | quit |

The terminal and file-manager keys follow you: they act on the folder you are looking at,
not the one you launched in. The bar carries only `? more`; everything above is on that page.

The menu `enter` opens on a file offers `Open in default app`, and — on a file gofer can
read as text — `Open in text editor` above it. The editor is `$EDITOR`, then `$VISUAL`, then
`vi`; it borrows this terminal rather than opening a window, so closing it puts you back on
the same folder and the same row.

The mouse follows the same split as the keys: left click opens a row — a folder by walking
into it, a file by its menu — and right click is the menu on any row. Right-clicking again
closes it. `ctrl+g` turns the mouse off when you want the terminal's own text selection back.

On a folder the menu offers `Open folder`, `Open in file manager` and `Terminal here`. Those
last two are the reason a folder has a menu at all: `t` and `ctrl+t` act on the folder you
are already in, so without them there was no way to aim either at the folder under the
cursor. `..` is the exception — `enter` on it just walks up.

## Config

`~/.gofer/config.yml` holds start-up settings. `gofer config` edits it with `$EDITOR`.

```yaml
compact: true       # one row per entry; false gives each a name and a size line
show_hidden: false  # list dot files
```

`alt+r` and `.` change those for the session without writing them The
config is only for start-up. `-a` overrides `show_hidden` for one run.

## Install

```
curl -fsSL https://raw.githubusercontent.com/brohd11/gofer/main/install.sh | sh
```

Windows (PowerShell):

```
irm https://raw.githubusercontent.com/brohd11/gofer/main/install.ps1 | iex
```

`gofer update` checks for a newer release and installs it over the running binary.