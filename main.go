package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

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
}

func main() {
	modPath := flag.String("mod", defaultModPath, "path to mod root")
	flag.Parse()

	var profs []ProfileState
	for _, p := range profileDefs {
		dir := filepath.Join(*modPath, p.RelPath)
		paths, err := CollectSwtFiles(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: cannot read %s: %v\n", p.Label, err)
			continue
		}
		orig := append([]string(nil), paths...)
		sorted := append([]string(nil), paths...)
		SortSwtFiles(sorted)
		if len(paths) == 0 {
			fmt.Fprintf(os.Stderr, "Warning: no .swt files in %s\n", p.Label)
			continue
		}
		profs = append(profs, ProfileState{
			label:         p.Label,
			dir:           dir,
			files:         sorted,
			filesOriginal: orig,
			sorted:        true,
		})
	}

	if len(profs) == 0 {
		fmt.Fprintln(os.Stderr, "No spawn files found")
		os.Exit(1)
	}

	model := NewModel(profs)
	prog := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
