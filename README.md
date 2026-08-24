# gofer

A small TUI file explorer: one directory at a time, folders first. Enter walks into a
folder and raises a menu on a file, backspace walks back out.

```
gofer            # the current directory
gofer /path      # an explicit start
```

## Follow it out — `--cd-file`

A program cannot change its parent shell's directory, so gofer writes the directory you
quit in and lets a shell function do the `cd`. Add this to your rc file:

```sh
gofer() {
  tmp="$(mktemp -t gofer-cd)"
  command gofer --cd-file="$tmp" "$@"
  dir="$(cat -- "$tmp" 2>/dev/null)"; rm -f -- "$tmp"
  [ -n "$dir" ] && [ "$dir" != "$PWD" ] && cd -- "$dir"
}
```

`command gofer` is what keeps the function from calling itself. The file is written only on
a clean exit, so a crash leaves your shell where it was.

## Keys

| key | |
|---|---|
| `enter` / click | walk into a folder, or open the menu on a file |
| `backspace` | up a folder (the `..` row does the same) |
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