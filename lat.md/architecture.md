# Architecture

TUI tool for editing spawn missions in Terminator: Dark Fate — Defiance mods. Go + Bubble Tea.

See also [[delivery]], [[environments]], [[tests]].

## System shape

Standalone TUI tool. Same stack as tmt-music-changer: Go + Bubble Tea + Lip Gloss.

## Source files

Scaffolded, ready for development.

| File | Responsibility |
|------|---------------|
| [[main.go]] | Entry point, launches TUI |
| [[model.go]] | Bubble Tea Model — state, Update, View |
| [[keys.go]] | Key bindings |
| [[styles.go]] | Lip Gloss styles |

## UI behavior

- Two-panel layout in entry mode: left list, right preview/edit.
- Spawn changes are staged in memory and written on explicit save.
- Entry list supports filtering by Unit/Owner/Zone/Type with Tab mode switch.
- Entry list columns size to content width with "|" separators and header dividers.

## Keybindings

- `r` toggle file sorting (original vs numeric).
- `c` duplicate selected spawn entry.
- `R` restore selected spawn entry to last saved state.

## Ownership boundaries

Which lat.md files are authoritative for which concerns.

- Architecture decisions: this file
- Delivery pipeline: [[delivery]]
- Config and runtime: [[environments]]
- Test strategy: [[tests]]
