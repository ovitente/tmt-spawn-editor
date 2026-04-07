# spawn-editor

TUI tool for editing spawn missions in Terminator: Dark Fate — Defiance mods.

This repository is a **submodule of [TMT](https://github.com/ovitente/tmt)**. It lives at `tools/spawn-editor/` in the parent repo. See `CLAUDE.md` and `lat.md/` for details.

## Setup

After cloning, activate the project git hooks:

```bash
git config core.hooksPath .githooks
```

The pre-commit hook runs `lat check` if [lat.md](https://www.npmjs.com/package/lat.md) is installed.

## Build

```bash
go build -o spawn-editor .
./spawn-editor
```
