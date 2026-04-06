# Spawn Editor

TUI tool for editing spawn missions in Terminator: Dark Fate — Defiance mods.

## Architecture
Go + Bubble Tea. See `lat.md/` for full knowledge base.

## Key files
- `main.go` — entry point
- `model.go` — TUI model
- `keys.go` — keybindings
- `styles.go` — Lip Gloss styles

## Build & Run
```bash
go build -o spawn-editor .
./spawn-editor
```

## Git Workflow
Direct to main for all changes.

## lat.md policy
Update `lat.md/` when adding features, changing keybindings, or making architectural decisions.
