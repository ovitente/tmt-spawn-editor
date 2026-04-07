package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const defaultModPath = `/home/det/play/Terminator Overhaul`

type Profile struct {
	Label   string
	RelPath string
}

var profileDefs = []Profile{
	{Label: "Original", RelPath: "basis/spawns"},
	{Label: "Resistance", RelPath: "dlc/Resistance/basis/spawns"},
	{Label: "Legion", RelPath: "dlc/Legion/basis/spawns"},
}

// ModEntry describes a mod listed in the picker modal.
type ModEntry struct {
	Path string // absolute directory path
	Name string // display name taken from mod.json "name" field
}

// LoadProfiles scans modPath for known profile subdirs and returns whichever
// contain .swt files. Empty slice means this mod has no spawn data.
func LoadProfiles(modPath string) []ProfileState {
	var profs []ProfileState
	for _, p := range profileDefs {
		dir := filepath.Join(modPath, p.RelPath)
		paths, err := CollectSwtFiles(dir)
		if err != nil || len(paths) == 0 {
			continue
		}
		orig := append([]string(nil), paths...)
		sorted := append([]string(nil), paths...)
		SortSwtFiles(sorted)
		profs = append(profs, ProfileState{
			label:         p.Label,
			dir:           dir,
			files:         sorted,
			filesOriginal: orig,
			sorted:        true,
		})
	}
	return profs
}

// ScanMods lists subdirectories of modsDir that contain a mod.json file.
// The display name is taken from the "name" field in mod.json; missing or
// empty name falls back to the folder basename. Sorted case-insensitively.
func ScanMods(modsDir string) []ModEntry {
	entries, err := os.ReadDir(modsDir)
	if err != nil {
		return nil
	}
	var mods []ModEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		modPath := filepath.Join(modsDir, e.Name())
		jsonPath := filepath.Join(modPath, "mod.json")
		data, err := os.ReadFile(jsonPath)
		if err != nil {
			continue
		}
		var meta struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(data, &meta)
		name := strings.TrimSpace(meta.Name)
		if name == "" {
			name = e.Name()
		}
		mods = append(mods, ModEntry{Path: modPath, Name: name})
	}
	sort.SliceStable(mods, func(i, j int) bool {
		return strings.ToLower(mods[i].Name) < strings.ToLower(mods[j].Name)
	})
	return mods
}

// LookupModName returns the display name for modPath from its mod.json,
// falling back to the folder basename.
func LookupModName(modPath string) string {
	data, err := os.ReadFile(filepath.Join(modPath, "mod.json"))
	if err == nil {
		var meta struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(data, &meta)
		if name := strings.TrimSpace(meta.Name); name != "" {
			return name
		}
	}
	return filepath.Base(modPath)
}

func main() {
	modPath := flag.String("mod", defaultModPath, "path to mod root")
	flag.Parse()

	abs, err := filepath.Abs(*modPath)
	if err == nil {
		*modPath = abs
	}
	// Resolve symlinks so ScanMods' parent scan finds siblings even if the
	// mod was passed via a symlink pointing into the mods directory.
	if resolved, err := filepath.EvalSymlinks(*modPath); err == nil {
		*modPath = resolved
	}

	profs := LoadProfiles(*modPath)
	if len(profs) == 0 {
		fmt.Fprintln(os.Stderr, "No spawn files found")
		os.Exit(1)
	}

	modsDir := filepath.Dir(*modPath)
	modName := LookupModName(*modPath)

	model := NewModel(profs, modsDir, *modPath, modName)
	prog := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
