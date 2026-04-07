# Delivery

How to build and run the spawn editor.

See also [[architecture]], [[tests]].

## Build

Plain Go build, no external toolchain required.

```bash
go build -o spawn-editor .
```

## Run

Default:

```bash
./spawn-editor
```

Custom mod path:

```bash
./spawn-editor -mod "/path/to/mod/root"
```

## Submodule in TMT

This repo is a submodule of ovitente/tmt at `tools/spawn-editor/`.

## lat.md policy

Update lat.md when features, keybindings, or architecture change. Skip for cosmetic fixes.
