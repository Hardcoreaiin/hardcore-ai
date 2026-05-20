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

### File Upload Picker
- `h` / `l` or `left` / `right` arrows: navigate directories
- `j` / `k` or `up` / `down` arrows: scroll files and directories
- `space`: toggle selection on a PDF file
- `y` or `enter`: confirm selection and start async background RAG ingestion
- `esc` or `q`: close file picker

## Slash Commands

Type these in the chat input to manage the TUI state:
- `/upload`: open the Ranger-style file browser to select and index PDF manuals into the local RAG database.
- `/clear-db`: completely wipe and re-initialize the local RAG database.
- `/quit`: exit the TUI application.
- `/help`: show description of all available commands.

The app uses Bubble Tea for the event loop and Lip Gloss for lightweight Markdown-style rendering.
