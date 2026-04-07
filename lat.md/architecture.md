# Architecture

TUI tool for editing spawn missions in Terminator: Dark Fate — Defiance mods. Go + Bubble Tea.

See also [[delivery]], [[environments]], [[tests]].

## System shape

Standalone TUI tool. Same stack as tmt-music-changer: Go + Bubble Tea + Lip Gloss.

## Source files

Core files and responsibilities.

| File | Responsibility |
|------|---------------|
| [[main.go]] | Entry point, launches TUI |
| [[model.go]] | Bubble Tea Model — state, Update, View |
| [[keys.go]] | Key bindings |
| [[styles.go]] | Lip Gloss styles |
| [[swt.go]] | SWT parse, sanitize, save, and data helpers |

## UI behavior

- Levels: file list → entry list → edit fields.
- Two-panel layout in entry mode: left list, right preview/edit.
- Panel height alignment: both panels must render to exactly `contentHeight` lines before `panelStyle.Height(contentHeight)` is applied. Lip Gloss' `Height` only pads, it does not truncate — any overflow (trailing `\n` after the last row, wrapped long title, etc.) makes one panel taller than the other and the shorter one "floats up". `MaxHeight` is NOT a fix: it truncates the border too and eats the bottom edge of the box. Instead, clamp panel content via `clampPanelLines(s, contentHeight)` in `model.go`, which trims trailing newlines and caps the line count before rendering. Any new panel renderer must go through the same clamp. Reusable pattern for sibling tools (e.g. theme-changer).
- Spawn changes are staged in memory and written on explicit save.
- Entry list supports filtering by Unit/Owner/Zone/Type with Tab mode switch.
- Entry list columns size to content width with "|" separators and header dividers.
- Droplists for Type, Zone, Owner; Type list seeded from known values.

## Keybindings

- Files: `f` find, `r` sort toggle, `p` profile switch.
- Entries: `f` find, `c` duplicate, `R` restore, `a` add, `d` delete.
- Edit: `Enter` edit field, `R` restore entry, `s` save.
- `r` toggle file sorting (original vs numeric).
- `c` duplicate selected spawn entry.
- `R` restore selected spawn entry to last saved state.

## Data model

- `SwtFile` holds path, entries, and dirty state.
- `SpawnEntry` tracks params, original snapshot, and Added flag.
- Dirty computed from param diffs or Added entries; restore resets to original.

## Save flow

- Load: read file bytes, sanitize invalid UTF-8 and illegal XML refs, parse.
- Save: apply deletes, trigger name changes, added actions, then param updates.
- Disk writes only on explicit save.

## Ownership boundaries

Which lat.md files are authoritative for which concerns.

- Architecture decisions: this file
- Delivery pipeline: [[delivery]]
- Config and runtime: [[environments]]
- Test strategy: [[tests]]
