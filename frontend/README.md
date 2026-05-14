# HARDCOREAI TUI

A small, fast terminal prompt for HARDCOREAI.

## Run

```sh
go mod tidy
go run .
```

## Controls

- `enter`: send
- `pgup` / `pgdown`: scroll
- `esc` / `ctrl+c`: quit

The app uses Bubble Tea for the event loop and Lip Gloss for lightweight
Markdown-style rendering.
