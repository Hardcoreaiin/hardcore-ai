# HARDCOREAI TUI

A small, fast terminal prompt for HARDCOREAI.

## Run

```sh
go mod tidy
go run .
```

## Non-TUI Smoke Flow

Create a deterministic STM32 blink project through the embedded tools without
starting Bubble Tea:

```sh
go run ./scripts/stm_blink_flow -skip-build
go run ./scripts/stm_blink_flow
```

## Controls

- `enter`: send
- `pgup` / `pgdown`: scroll
- `esc` / `ctrl+c`: quit

The app uses Bubble Tea for the event loop and Lip Gloss for lightweight
Markdown-style rendering.
