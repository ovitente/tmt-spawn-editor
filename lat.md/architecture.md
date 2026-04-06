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

## Ownership boundaries

Which lat.md files are authoritative for which concerns.

- Architecture decisions: this file
- Delivery pipeline: [[delivery]]
- Config and runtime: [[environments]]
- Test strategy: [[tests]]
