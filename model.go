package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Navigation levels
type Level int

const (
	LevelFiles   Level = iota // browse .swt files
	LevelEntries              // browse spawn entries in a file
	LevelEdit                 // edit fields of a spawn entry
)

const (
	entryFilterUnit = iota
	entryFilterOwner
	entryFilterZone
	entryFilterType
)

var entryFilterLabels = []string{"Unit", "Owner", "Zone", "Type"}

type ProfileState struct {
	label         string
	dir           string
	files         []string // full paths (current order)
	filesOriginal []string // original order from disk
	sorted        bool
}

const (
	leftPanelPadLeft  = 1
	leftPanelPadRight = 0
)

type Model struct {
	profiles      []ProfileState
	activeProfile int

	level Level

	// File browser
	fileCursor int
	fileScroll int
	currentSwt *SwtFile // loaded when entering LevelEntries

	// Entry browser
	entryCursor int
	entryScroll int

	entryFilterActive  bool
	entryFilterTyping  bool
	entryFilterInput   string
	entryFilterIndices []int
	entryFilterCursor  int
	entryFilterScroll  int
	entryFilterMode    int

	// Field editor
	fieldCursor      int
	fieldEditing     bool
	fieldInput       string
	fieldInputCursor int

	// Droplist
	dropActive bool
	dropItems  []string
	dropCursor int
	dropScroll int

	// Filter
	filterActive  bool
	filterTyping  bool
	filterInput   string
	filterIndices []int
	filterCursor  int
	filterScroll  int

	width     int
	height    int
	statusMsg string

	confirmDelete bool
	confirmQuit   bool
}

type saveResultMsg struct{ err error }

func NewModel(profiles []ProfileState) Model {
	return Model{
		profiles: profiles,
		level:    LevelFiles,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m *Model) prof() *ProfileState {
	return &m.profiles[m.activeProfile]
}

func (m *Model) currentEntry() *SpawnEntry {
	if m.currentSwt == nil || m.entryCursor < 0 || m.entryCursor >= len(m.currentSwt.Entries) {
		return nil
	}
	return &m.currentSwt.Entries[m.entryCursor]
}

func (m *Model) restoreCurrentEntry() {
	entry := m.currentEntry()
	if entry == nil {
		return
	}
	if entry.Added {
		idx := m.entryCursor
		m.currentSwt.Entries = append(m.currentSwt.Entries[:idx], m.currentSwt.Entries[idx+1:]...)
		if m.entryCursor >= len(m.currentSwt.Entries) && m.entryCursor > 0 {
			m.entryCursor--
		}
		m.currentSwt.RecalcDirty()
		return
	}
	entry.Params = entry.Original
	entry.TriggerName = entry.OriginalTriggerName
	m.currentSwt.RecalcDirty()
}

func (m *Model) visibleFiles() []int {
	if m.filterActive {
		return m.filterIndices
	}
	p := m.prof()
	indices := make([]int, len(p.files))
	for i := range p.files {
		indices[i] = i
	}
	return indices
}

func (m *Model) visibleEntries() []int {
	if m.entryFilterActive {
		return m.entryFilterIndices
	}
	if m.currentSwt == nil {
		return nil
	}
	indices := make([]int, len(m.currentSwt.Entries))
	for i := range m.currentSwt.Entries {
		indices[i] = i
	}
	return indices
}

func (m *Model) updateFileFilter() {
	p := m.prof()
	m.filterIndices = nil
	for i, f := range p.files {
		name := strings.ToLower(filepath.Base(f))
		if m.filterInput == "" || fuzzyMatch(name, m.filterInput) {
			m.filterIndices = append(m.filterIndices, i)
		}
	}
	m.filterCursor = 0
	m.filterScroll = 0
	if len(m.filterIndices) > 0 {
		m.fileCursor = m.filterIndices[0]
	}
}

func entryFilterParamIndex(mode int) int {
	switch mode {
	case entryFilterOwner:
		return 6
	case entryFilterZone:
		return 5
	case entryFilterType:
		return 2
	default:
		return 3
	}
}

func (m *Model) updateEntryFilter() {
	if m.currentSwt == nil {
		return
	}
	paramIdx := entryFilterParamIndex(m.entryFilterMode)
	m.entryFilterIndices = nil
	for i, e := range m.currentSwt.Entries {
		val := strings.ToLower(e.Params[paramIdx])
		if m.entryFilterInput == "" || fuzzyMatch(val, m.entryFilterInput) {
			m.entryFilterIndices = append(m.entryFilterIndices, i)
		}
	}
	m.entryFilterCursor = 0
	m.entryFilterScroll = 0
	if len(m.entryFilterIndices) > 0 {
		m.entryCursor = m.entryFilterIndices[0]
	}
}

func fuzzyMatch(s, query string) bool {
	s = strings.ToLower(s)
	query = strings.ToLower(query)
	qi := 0
	for i := 0; i < len(s) && qi < len(query); i++ {
		if s[i] == query[qi] {
			qi++
		}
	}
	return qi == len(query)
}

func (m *Model) clearFilter() {
	m.filterActive = false
	m.filterTyping = false
	m.filterInput = ""
	m.filterIndices = nil
	m.filterCursor = 0
	m.filterScroll = 0
}

func (m *Model) clearEntryFilter() {
	m.entryFilterActive = false
	m.entryFilterTyping = false
	m.entryFilterInput = ""
	m.entryFilterIndices = nil
	m.entryFilterCursor = 0
	m.entryFilterScroll = 0
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case saveResultMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.statusMsg = "Saved!"
			if m.currentSwt != nil {
				for i := range m.currentSwt.Entries {
					m.currentSwt.Entries[i].Original = m.currentSwt.Entries[i].Params
					m.currentSwt.Entries[i].OriginalTriggerName = m.currentSwt.Entries[i].TriggerName
					m.currentSwt.Entries[i].Added = false
				}
				m.currentSwt.DeletedActionGUIDs = nil
				m.currentSwt.dirty = false
			}
		}
		return m, nil

	case tea.KeyMsg:
		if m.statusMsg != "" && !key.Matches(msg, keys.Save) {
			m.statusMsg = ""
		}

		if m.confirmDelete {
			switch msg.String() {
			case "y", "Y":
				if m.currentSwt != nil && m.entryCursor < len(m.currentSwt.Entries) {
					if err := DeleteSpawnEntry(m.currentSwt, m.entryCursor); err != nil {
						m.statusMsg = fmt.Sprintf("Error: %v", err)
					}
					if m.entryCursor >= len(m.currentSwt.Entries) && m.entryCursor > 0 {
						m.entryCursor--
					}
					if len(m.currentSwt.Entries) == 0 {
						m.level = LevelFiles
						m.currentSwt = nil
					}
				}
			}
			m.confirmDelete = false
			return m, nil
		}

		if m.confirmQuit {
			if msg.String() == "y" || msg.String() == "Y" {
				return m, tea.Quit
			}
			m.confirmQuit = false
			m.statusMsg = ""
			return m, nil
		}

		if m.filterActive && m.filterTyping {
			return m.updateFilterTyping(msg)
		}

		if m.entryFilterActive && m.entryFilterTyping {
			return m.updateEntryFilterTyping(msg)
		}

		if m.dropActive {
			return m.updateDroplist(msg)
		}

		if m.fieldEditing {
			return m.updateFieldInput(msg)
		}

		if key.Matches(msg, keys.Save) && m.currentSwt != nil {
			return m, m.saveCmd()
		}

		if key.Matches(msg, keys.Restore) && m.level != LevelFiles {
			m.restoreCurrentEntry()
			return m, nil
		}

		if key.Matches(msg, keys.Profile) && m.level == LevelFiles && len(m.profiles) > 1 {
			m.activeProfile = (m.activeProfile + 1) % len(m.profiles)
			m.fileCursor = 0
			m.fileScroll = 0
			m.currentSwt = nil
			m.clearFilter()
			m.clearEntryFilter()
			return m, nil
		}

		if key.Matches(msg, keys.Quit) {
			if m.level == LevelEdit {
				m.level = LevelEntries
				return m, nil
			}
			if m.level == LevelEntries {
				m.level = LevelFiles
				m.currentSwt = nil
				m.clearEntryFilter()
				return m, nil
			}
			if m.currentSwt != nil && m.currentSwt.dirty {
				m.confirmQuit = true
				m.statusMsg = "Unsaved changes! Y to quit, any key to cancel"
				return m, nil
			}
			return m, tea.Quit
		}

		if key.Matches(msg, keys.Escape) {
			if m.filterActive && !m.filterTyping {
				m.clearFilter()
				return m, nil
			}
			if m.entryFilterActive && !m.entryFilterTyping {
				m.clearEntryFilter()
				return m, nil
			}
			if m.level == LevelEdit {
				m.level = LevelEntries
				return m, nil
			}
			if m.level == LevelEntries {
				m.level = LevelFiles
				m.currentSwt = nil
				m.clearEntryFilter()
				return m, nil
			}
			return m, nil
		}

		switch m.level {
		case LevelFiles:
			return m.updateFiles(msg)
		case LevelEntries:
			return m.updateEntries(msg)
		case LevelEdit:
			return m.updateEditNav(msg)
		}
	}
	return m, nil
}

func (m Model) updateFiles(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	vis := m.visibleFiles()
	switch {
	case key.Matches(msg, keys.Up):
		if m.filterActive {
			if m.filterCursor > 0 {
				m.filterCursor--
				m.fileCursor = m.filterIndices[m.filterCursor]
			}
		} else if m.fileCursor > 0 {
			m.fileCursor--
		}

	case key.Matches(msg, keys.Down):
		if m.filterActive {
			if m.filterCursor < len(m.filterIndices)-1 {
				m.filterCursor++
				m.fileCursor = m.filterIndices[m.filterCursor]
			}
		} else if m.fileCursor < len(vis)-1 {
			m.fileCursor++
		}

	case key.Matches(msg, keys.Enter):
		p := m.prof()
		if m.fileCursor >= 0 && m.fileCursor < len(p.files) {
			swt, err := ParseSwtFile(p.files[m.fileCursor])
			if err != nil {
				m.statusMsg = fmt.Sprintf("Error: %v", err)
			} else {
				m.currentSwt = swt
				m.entryCursor = 0
				m.entryScroll = 0
				m.clearEntryFilter()
				m.level = LevelEntries
			}
		}

	case key.Matches(msg, keys.Find):
		m.filterActive = true
		m.filterTyping = true
		m.filterInput = ""
		m.updateFileFilter()

	case key.Matches(msg, keys.SortToggle):
		p := m.prof()
		selected := ""
		if m.fileCursor >= 0 && m.fileCursor < len(p.files) {
			selected = p.files[m.fileCursor]
		}
		if p.sorted {
			p.files = append([]string(nil), p.filesOriginal...)
			p.sorted = false
		} else {
			p.files = append([]string(nil), p.filesOriginal...)
			SortSwtFiles(p.files)
			p.sorted = true
		}
		if m.filterActive {
			m.updateFileFilter()
			if selected != "" {
				for i, idx := range m.filterIndices {
					if p.files[idx] == selected {
						m.filterCursor = i
						m.fileCursor = idx
						break
					}
				}
			}
		} else if selected != "" {
			for i, path := range p.files {
				if path == selected {
					m.fileCursor = i
					break
				}
			}
		}
	}
	return m, nil
}

func (m Model) updateEntries(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.currentSwt == nil {
		return m, nil
	}
	switch {
	case key.Matches(msg, keys.Up):
		if m.entryFilterActive {
			if m.entryFilterCursor > 0 {
				m.entryFilterCursor--
				m.entryCursor = m.entryFilterIndices[m.entryFilterCursor]
			}
		} else if m.entryCursor > 0 {
			m.entryCursor--
		}
	case key.Matches(msg, keys.Down):
		if m.entryFilterActive {
			if m.entryFilterCursor < len(m.entryFilterIndices)-1 {
				m.entryFilterCursor++
				m.entryCursor = m.entryFilterIndices[m.entryFilterCursor]
			}
		} else if m.entryCursor < len(m.currentSwt.Entries)-1 {
			m.entryCursor++
		}
	case key.Matches(msg, keys.Edit), key.Matches(msg, keys.Enter):
		if len(m.currentSwt.Entries) > 0 {
			m.level = LevelEdit
			m.fieldCursor = 0
		}
	case key.Matches(msg, keys.AddEntry):
		entry, err := AddSpawnEntry(m.currentSwt)
		if err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", err)
		} else {
			_ = entry
			m.entryCursor = len(m.currentSwt.Entries) - 1
			m.level = LevelEdit
			m.fieldCursor = 0
		}
	case key.Matches(msg, keys.Duplicate):
		if len(m.currentSwt.Entries) > 0 {
			source := m.currentSwt.Entries[m.entryCursor]
			dup, err := DuplicateSpawnEntry(m.currentSwt, source)
			if err != nil {
				m.statusMsg = fmt.Sprintf("Error: %v", err)
				return m, nil
			}
			insertAt := m.entryCursor + 1
			if insertAt > len(m.currentSwt.Entries) {
				insertAt = len(m.currentSwt.Entries)
			}
			m.currentSwt.Entries = append(m.currentSwt.Entries[:insertAt], append([]SpawnEntry{dup}, m.currentSwt.Entries[insertAt:]...)...)
			m.entryCursor = insertAt
			m.currentSwt.RecalcDirty()
		}
	case key.Matches(msg, keys.Delete):
		if len(m.currentSwt.Entries) > 0 {
			e := m.currentEntry()
			m.confirmDelete = true
			m.statusMsg = fmt.Sprintf("Delete %s [%s]? Y/N", e.Unit(), e.EntityType())
		}
	case key.Matches(msg, keys.Find):
		m.entryFilterActive = true
		m.entryFilterTyping = true
		m.entryFilterInput = ""
		m.entryFilterMode = entryFilterUnit
		m.updateEntryFilter()
	}
	return m, nil
}

func (m Model) updateEditNav(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	entry := m.currentEntry()
	if entry == nil {
		return m, nil
	}
	switch {
	case key.Matches(msg, keys.Up):
		if m.fieldCursor > 0 {
			m.fieldCursor--
		}
	case key.Matches(msg, keys.Down):
		if m.fieldCursor < paramCount { // 12 params + trigger
			m.fieldCursor++
		}
	case key.Matches(msg, keys.Enter):
		if m.fieldCursor == 2 { // Type droplist
			m.dropItems = CollectCandidates(m.currentSwt.Entries, 2, entityTypes...)
			if len(m.dropItems) > 0 {
				m.dropActive = true
				m.dropCursor = 0
				m.dropScroll = 0
				return m, nil
			}
		}
		if m.fieldCursor == 5 { // Zone droplist
			m.dropItems = CollectCandidates(m.currentSwt.Entries, 5)
			if len(m.dropItems) > 0 {
				m.dropActive = true
				m.dropCursor = 0
				m.dropScroll = 0
				return m, nil
			}
		}
		if m.fieldCursor == 6 { // Owner droplist
			m.dropItems = CollectCandidates(m.currentSwt.Entries, 6, "player")
			m.dropActive = true
			m.dropCursor = 0
			m.dropScroll = 0
			return m, nil
		}
		// Text input for other fields
		idx := m.fieldCursor
		if idx < paramCount {
			m.fieldInput = entry.Params[idx]
		} else {
			m.fieldInput = entry.TriggerName
		}
		m.fieldInputCursor = len(m.fieldInput)
		m.fieldEditing = true
	}
	return m, nil
}

func (m Model) updateFieldInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Escape):
		m.fieldEditing = false
	case key.Matches(msg, keys.Enter):
		entry := m.currentEntry()
		if entry != nil {
			if m.fieldCursor < paramCount {
				entry.Params[m.fieldCursor] = m.fieldInput
			} else {
				entry.TriggerName = m.fieldInput
			}
			m.currentSwt.RecalcDirty()
		}
		m.fieldEditing = false
	case msg.Type == tea.KeyBackspace:
		if m.fieldInputCursor > 0 {
			m.fieldInput = m.fieldInput[:m.fieldInputCursor-1] + m.fieldInput[m.fieldInputCursor:]
			m.fieldInputCursor--
		}
	case msg.Type == tea.KeyLeft:
		if m.fieldInputCursor > 0 {
			m.fieldInputCursor--
		}
	case msg.Type == tea.KeyRight:
		if m.fieldInputCursor < len(m.fieldInput) {
			m.fieldInputCursor++
		}
	case msg.Type == tea.KeyRunes:
		ch := string(msg.Runes)
		m.fieldInput = m.fieldInput[:m.fieldInputCursor] + ch + m.fieldInput[m.fieldInputCursor:]
		m.fieldInputCursor += len(ch)
	}
	return m, nil
}

func (m Model) updateDroplist(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Escape):
		m.dropActive = false
	case key.Matches(msg, keys.Up):
		if m.dropCursor > 0 {
			m.dropCursor--
		}
	case key.Matches(msg, keys.Down):
		if m.dropCursor < len(m.dropItems)-1 {
			m.dropCursor++
		}
	case key.Matches(msg, keys.Enter):
		if m.dropCursor < len(m.dropItems) {
			entry := m.currentEntry()
			if entry != nil && m.fieldCursor < paramCount {
				entry.Params[m.fieldCursor] = m.dropItems[m.dropCursor]
				m.currentSwt.RecalcDirty()
			}
		}
		m.dropActive = false
	}
	return m, nil
}

func (m Model) updateFilterTyping(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Escape):
		m.clearFilter()
	case key.Matches(msg, keys.Enter):
		if len(m.filterIndices) > 0 {
			m.filterTyping = false
		}
	case msg.Type == tea.KeyBackspace:
		if len(m.filterInput) > 0 {
			m.filterInput = m.filterInput[:len(m.filterInput)-1]
			m.updateFileFilter()
		}
	case msg.Type == tea.KeyRunes:
		m.filterInput += string(msg.Runes)
		m.updateFileFilter()
	}
	return m, nil
}

func (m Model) updateEntryFilterTyping(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Escape):
		m.clearEntryFilter()

	case key.Matches(msg, keys.Enter):
		if len(m.entryFilterIndices) > 0 {
			m.entryFilterTyping = false
		}

	case key.Matches(msg, keys.Tab):
		m.entryFilterMode = (m.entryFilterMode + 1) % len(entryFilterLabels)
		m.updateEntryFilter()

	case msg.Type == tea.KeyBackspace:
		if len(m.entryFilterInput) > 0 {
			m.entryFilterInput = m.entryFilterInput[:len(m.entryFilterInput)-1]
			m.updateEntryFilter()
		}

	case msg.Type == tea.KeyRunes:
		m.entryFilterInput += string(msg.Runes)
		m.updateEntryFilter()
	}
	return m, nil
}

func (m Model) saveCmd() tea.Cmd {
	swt := m.currentSwt
	return func() tea.Msg {
		err := SaveSwtFile(swt)
		return saveResultMsg{err: err}
	}
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	title := appTitleStyle.Render("Spawn Editor")

	contentHeight := m.height - 7
	if contentHeight < 5 {
		contentHeight = 5
	}

	if m.level == LevelFiles {
		contentWidth := m.width - 4
		panel := m.renderFiles(contentWidth, contentHeight)
		rendered := activePanelStyle.Width(contentWidth).Height(contentHeight).Render(panel)
		status := m.renderStatus(lipgloss.Width(rendered))
		return lipgloss.JoinVertical(lipgloss.Left, title, rendered, status)
	}

	totalWidth := m.width
	leftWidth := (totalWidth - 9) * 3 / 5
	rightWidth := totalWidth - 9 - leftWidth
	if leftWidth < 30 {
		leftWidth = 30
	}
	if rightWidth < 25 {
		rightWidth = 25
	}
	if m.currentSwt != nil {
		maxPanelWidth := totalWidth - 9 - 25
		desiredLeft := desiredEntriesWidth(m.currentSwt.Entries, m.currentSwt.Name, len(m.currentSwt.Entries), 30, maxPanelWidth)
		if desiredLeft < leftWidth {
			leftWidth = desiredLeft
			rightWidth = totalWidth - 9 - leftWidth
			if rightWidth < 25 {
				rightWidth = 25
				leftWidth = totalWidth - 9 - rightWidth
			}
		}
	}

	leftPanel := m.renderEntries(leftWidth, contentHeight)
	var rightPanel string
	if m.level == LevelEdit {
		rightPanel = m.renderEditFields(rightWidth, contentHeight)
	} else {
		rightPanel = m.renderEntryPreview(rightWidth, contentHeight)
	}
	leftPanel = padToHeight(leftPanel, contentHeight)
	rightPanel = padToHeight(rightPanel, contentHeight)

	leftStyle := panelStyle.Width(leftWidth).Height(contentHeight).PaddingLeft(leftPanelPadLeft).PaddingRight(leftPanelPadRight)
	rightStyle := panelStyle.Width(rightWidth).Height(contentHeight)
	if m.level == LevelEdit || m.dropActive || m.fieldEditing {
		rightStyle = activePanelStyle.Width(rightWidth).Height(contentHeight)
	} else {
		leftStyle = activePanelStyle.Width(leftWidth).Height(contentHeight)
	}

	leftRendered := leftStyle.Render(leftPanel)
	rightRendered := rightStyle.Render(rightPanel)
	leftHeight := lipgloss.Height(leftRendered)
	rightHeight := lipgloss.Height(rightRendered)
	if rightHeight < leftHeight {
		rightRendered += strings.Repeat("\n", leftHeight-rightHeight)
	} else if leftHeight < rightHeight {
		leftRendered += strings.Repeat("\n", rightHeight-leftHeight)
	}

	panels := lipgloss.JoinHorizontal(lipgloss.Top,
		leftRendered,
		" ",
		rightRendered,
	)

	status := m.renderStatus(lipgloss.Width(panels))
	return lipgloss.JoinVertical(lipgloss.Left, title, panels, status)
}

func (m Model) renderFiles(width, height int) string {
	var sb strings.Builder
	p := m.prof()
	textWidth := width - 2

	if m.filterActive {
		header := titleStyle.Render("Find") + helpSepStyle.Render(" | ") + titleStyle.Render("Files")
		count := fmt.Sprintf("%d", len(m.filterIndices)) + helpSepStyle.Render("/") + fmt.Sprintf("%d", len(p.files))
		pad := textWidth - lipgloss.Width(header) - lipgloss.Width(count)
		if pad < 1 {
			pad = 1
		}
		sb.WriteString(header + strings.Repeat(" ", pad) + count)
		sb.WriteString("\n")
		if m.filterTyping {
			sb.WriteString(searchInputStyle.Render(m.filterInput + "█"))
		} else {
			sb.WriteString(checkedStyle.Render(m.filterInput + " ✓"))
		}
		sb.WriteString("\n")
	} else {
		title := "Files"
		count := fmt.Sprintf("%d", len(p.files))
		pad := textWidth - len(title) - len(count)
		if pad < 1 {
			pad = 1
		}
		sb.WriteString(titleStyle.Render(title) + strings.Repeat(" ", pad) + titleStyle.Render(count))
		sb.WriteString("\n\n")
	}

	vis := m.visibleFiles()
	visibleHeight := height - 3
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	cursor := m.fileCursor
	scroll := &m.fileScroll
	if m.filterActive {
		cursor = m.filterCursor
		scroll = &m.filterScroll
	}
	if cursor < *scroll {
		*scroll = cursor
	}
	if cursor >= *scroll+visibleHeight {
		*scroll = cursor - visibleHeight + 1
	}

	for vi := *scroll; vi < len(vis) && vi < *scroll+visibleHeight; vi++ {
		idx := vis[vi]
		name := filepath.Base(p.files[idx])
		if len(name) > textWidth-2 {
			name = name[:textWidth-5] + "..."
		}

		isSelected := false
		if m.filterActive {
			isSelected = vi == m.filterCursor
		} else {
			isSelected = vi == m.fileCursor
		}

		if isSelected {
			sb.WriteString(selectorPrefix.Render("▶ ") + reorderHighlight.Render(name) + "\n")
		} else {
			sb.WriteString("  " + name + "\n")
		}
	}

	return sb.String()
}

func (m Model) renderEntries(width, height int) string {
	var sb strings.Builder
	textWidth := width - 2 - leftPanelPadLeft - leftPanelPadRight

	if m.currentSwt == nil {
		return "No file loaded"
	}

	if m.entryFilterActive {
		modeLabel := entryFilterLabels[m.entryFilterMode]
		header := titleStyle.Render("Find") + helpSepStyle.Render(" | ") + titleStyle.Render(modeLabel)
		count := fmt.Sprintf("%d", len(m.entryFilterIndices)) + helpSepStyle.Render("/") + fmt.Sprintf("%d", len(m.currentSwt.Entries))
		pad := textWidth - lipgloss.Width(header) - lipgloss.Width(count)
		if pad < 1 {
			pad = 1
		}
		sb.WriteString(header + strings.Repeat(" ", pad) + count)
		sb.WriteString("\n")
		if m.entryFilterTyping {
			sb.WriteString(searchInputStyle.Render(m.entryFilterInput + "█"))
		} else {
			sb.WriteString(checkedStyle.Render(m.entryFilterInput + " ✓"))
		}
		sb.WriteString("\n")
	} else {
		title := m.currentSwt.Name
		if m.currentSwt.dirty {
			title += dirtyStyle.Render(" [modified]")
		}
		count := fmt.Sprintf("%d", len(m.currentSwt.Entries))
		pad := textWidth - lipgloss.Width(titleStyle.Render(title)) - lipgloss.Width(titleStyle.Render(count))
		if pad < 1 {
			pad = 1
		}
		sb.WriteString(titleStyle.Render(title) + strings.Repeat(" ", pad) + titleStyle.Render(count))
		sb.WriteString("\n\n")
	}

	entries := m.currentSwt.Entries
	headerLines := 2
	tableHeaderLines := 2
	visibleHeight := height - headerLines - tableHeaderLines
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	vis := m.visibleEntries()
	cursor := m.entryCursor
	scroll := &m.entryScroll
	if m.entryFilterActive {
		cursor = m.entryFilterCursor
		scroll = &m.entryFilterScroll
	}
	if cursor < *scroll {
		*scroll = cursor
	}
	if cursor >= *scroll+visibleHeight {
		*scroll = cursor - visibleHeight + 1
	}

	rowWidth := textWidth - 2
	if rowWidth < 10 {
		rowWidth = 10
	}
	sep := " | "
	sepWidth := 3

	unitWidth, presetWidth, ownerWidth, zoneWidth, typeWidth := entryColumnWidths(entries, rowWidth, sepWidth)

	headerRow := formatHeaderRow(unitWidth, presetWidth, ownerWidth, zoneWidth, typeWidth)
	sb.WriteString("  " + headerRow + "\n")

	lineRow := formatHeaderRuleRow(unitWidth, presetWidth, ownerWidth, zoneWidth, typeWidth)
	sb.WriteString("  " + lineRow + "\n")

	for vi := *scroll; vi < len(vis) && vi < *scroll+visibleHeight; vi++ {
		idx := vis[vi]
		e := entries[idx]
		unit := e.Unit()
		if unit == "" {
			unit = "(empty)"
		}
		preset := e.Params[4]
		owner := e.Owner()
		zone := e.Zone()
		typeLabel := ""
		if e.EntityType() != "" {
			typeLabel = "[" + e.EntityType() + "]"
		}

		rowPlain := formatEntryRow(unit, preset, owner, zone, typeLabel, unitWidth, presetWidth, ownerWidth, zoneWidth, typeWidth, sep)

		modified := e.Modified()
		gutter := " "
		if modified {
			gutter = activeTabStyle.Render("·")
		}

		isSelected := false
		if m.entryFilterActive {
			isSelected = vi == m.entryFilterCursor
		} else {
			isSelected = idx == m.entryCursor
		}
		if isSelected {
			sb.WriteString(selectorPrefix.Render("▶ ") + reorderHighlight.Render(rowPlain) + "\n")
		} else {
			row := styleRowSeparators(rowPlain)
			sb.WriteString(gutter + " " + row + "\n")
		}
	}

	return sb.String()
}

func formatEntryRow(unit, preset, owner, zone, typ string, unitWidth, presetWidth, ownerWidth, zoneWidth, typeWidth int, sep string) string {
	row := fmt.Sprintf(
		"%-*s%s%-*s%s%-*s%s%-*s%s%-*s",
		unitWidth, clampText(unit, unitWidth),
		sep,
		presetWidth, clampText(preset, presetWidth),
		sep,
		ownerWidth, clampText(owner, ownerWidth),
		sep,
		zoneWidth, clampText(zone, zoneWidth),
		sep,
		typeWidth, clampText(typ, typeWidth),
	)
	return strings.TrimRight(row, " ")
}

func formatHeaderRow(unitWidth, presetWidth, ownerWidth, zoneWidth, typeWidth int) string {
	sep := " " + headerSepStyle.Render("|") + " "
	return headerCell("Unit", unitWidth) +
		sep + headerCell("Preset", presetWidth) +
		sep + headerCell("Owner", ownerWidth) +
		sep + headerCell("Zone", zoneWidth) +
		sep + headerCell("Type", typeWidth)
}

func formatHeaderRuleRow(unitWidth, presetWidth, ownerWidth, zoneWidth, typeWidth int) string {
	sep := " " + headerSepStyle.Render("|") + " "
	return headerRuleCell(unitWidth) +
		sep + headerRuleCell(presetWidth) +
		sep + headerRuleCell(ownerWidth) +
		sep + headerRuleCell(zoneWidth) +
		sep + headerRuleCell(typeWidth)
}

func headerCell(label string, width int) string {
	if width <= 0 {
		return ""
	}
	text := fmt.Sprintf("%-*s", width, clampText(label, width))
	return headerTextStyle.Render(text)
}

func headerRuleCell(width int) string {
	if width <= 0 {
		return ""
	}
	return headerTextStyle.Render(strings.Repeat("-", width))
}

func entryColumnWidths(entries []SpawnEntry, rowWidth int, sepWidth int) (int, int, int, int, int) {
	unitWidth := len("Unit")
	presetWidth := len("Preset")
	ownerWidth := len("Owner")
	zoneWidth := len("Zone")
	typeWidth := len("Type")

	for _, e := range entries {
		if l := len(e.Unit()); l > unitWidth {
			unitWidth = l
		}
		if l := len(e.Params[4]); l > presetWidth {
			presetWidth = l
		}
		if l := len(e.Owner()); l > ownerWidth {
			ownerWidth = l
		}
		if l := len(e.Zone()); l > zoneWidth {
			zoneWidth = l
		}
		if e.EntityType() != "" {
			if l := len("[" + e.EntityType() + "]"); l > typeWidth {
				typeWidth = l
			}
		}
	}

	if rowWidth < 10 {
		rowWidth = 10
	}
	maxTotal := rowWidth - sepWidth*4
	if maxTotal < 5 {
		maxTotal = 5
	}

	widths := []int{unitWidth, presetWidth, ownerWidth, zoneWidth, typeWidth}
	minWidth := 4
	for totalWidths(widths) > maxTotal {
		idx := maxWidthIndex(widths, minWidth)
		if idx == -1 {
			break
		}
		widths[idx]--
	}

	return widths[0], widths[1], widths[2], widths[3], widths[4]
}

func totalWidths(widths []int) int {
	total := 0
	for _, w := range widths {
		total += w
	}
	return total
}

func maxWidthIndex(widths []int, minWidth int) int {
	maxIdx := -1
	maxVal := 0
	for i, w := range widths {
		if w > maxVal && w > minWidth {
			maxVal = w
			maxIdx = i
		}
	}
	return maxIdx
}

func styleRowSeparators(row string) string {
	return strings.ReplaceAll(row, " | ", " "+headerSepStyle.Render("|")+" ")
}

func desiredEntriesWidth(entries []SpawnEntry, title string, count int, minWidth int, maxWidth int) int {
	if minWidth < 1 {
		minWidth = 1
	}
	if maxWidth < minWidth {
		maxWidth = minWidth
	}

	sepWidth := 3
	unitWidth, presetWidth, ownerWidth, zoneWidth, typeWidth := entryColumnWidths(entries, maxWidth-4, sepWidth)
	rowWidth := unitWidth + presetWidth + ownerWidth + zoneWidth + typeWidth + sepWidth*4

	rowTextWidth := rowWidth + 2
	countLabel := fmt.Sprintf("%d", count)
	minHeaderText := len(title) + len(countLabel) + 1
	if rowTextWidth < minHeaderText {
		rowTextWidth = minHeaderText
	}

	width := rowTextWidth + 2
	if width < minWidth {
		return minWidth
	}
	if width > maxWidth {
		return maxWidth
	}
	return width
}

func clampText(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if len(s) > width {
		if width <= 3 {
			return s[:width]
		}
		return s[:width-3] + "..."
	}
	return s
}

func (m Model) renderEntryPreview(width, height int) string {
	var sb strings.Builder
	entry := m.currentEntry()
	if entry == nil {
		return "No entry selected"
	}

	unitName := entry.Unit()
	if unitName == "" {
		unitName = "(empty)"
	}
	sb.WriteString(titleStyle.Render("Spawn: " + unitName))
	sb.WriteString("\n\n\n\n")

	for i := 0; i < paramCount; i++ {
		label := fieldLabels[i]
		value := entry.Params[i]
		if value == "" {
			value = commentStyle.Render("—")
		}
		sb.WriteString(commentStyle.Render(fmt.Sprintf("%-12s", label)) + " " + value + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(commentStyle.Render(fmt.Sprintf("%-12s", "Trigger")) + " " + entry.TriggerName)

	_ = width
	return padToHeight(sb.String(), height)
}

func (m Model) renderEditFields(width, height int) string {
	var sb strings.Builder

	entry := m.currentEntry()
	if entry == nil {
		return "No entry selected"
	}

	unitName := entry.Unit()
	if unitName == "" {
		unitName = "(new)"
	}
	sb.WriteString(titleStyle.Render("Edit: " + unitName))
	sb.WriteString("\n\n\n\n")

	for i := 0; i < paramCount; i++ {
		label := fieldLabels[i]
		value := entry.Params[i]

		if m.fieldEditing && m.fieldCursor == i {
			before := m.fieldInput[:m.fieldInputCursor]
			after := m.fieldInput[m.fieldInputCursor:]
			sb.WriteString(selectorPrefix.Render("▶ ") + commentStyle.Render(fmt.Sprintf("%-12s", label)) + " " + searchInputStyle.Render(before+"█"+after) + "\n")
			continue
		}

		if m.dropActive && m.fieldCursor == i {
			sb.WriteString(selectorPrefix.Render("▶ ") + commentStyle.Render(fmt.Sprintf("%-12s", label)) + "\n")
			for di, item := range m.dropItems {
				if di == m.dropCursor {
					sb.WriteString("    " + selectorPrefix.Render("▶ ") + reorderHighlight.Render(item) + "\n")
				} else {
					sb.WriteString("      " + item + "\n")
				}
			}
			continue
		}

		prefix := "  "
		if i == m.fieldCursor {
			prefix = selectorPrefix.Render("▶ ")
		}

		displayVal := value
		if displayVal == "" {
			displayVal = commentStyle.Render("—")
		}

		sb.WriteString(prefix + commentStyle.Render(fmt.Sprintf("%-12s", label)) + " " + displayVal + "\n")
	}

	// Trigger field
	sb.WriteString("\n")
	prefix := "  "
	if m.fieldCursor == paramCount {
		prefix = selectorPrefix.Render("▶ ")
	}
	if m.fieldEditing && m.fieldCursor == paramCount {
		before := m.fieldInput[:m.fieldInputCursor]
		after := m.fieldInput[m.fieldInputCursor:]
		sb.WriteString(prefix + commentStyle.Render(fmt.Sprintf("%-12s", "Trigger")) + " " + searchInputStyle.Render(before+"█"+after) + "\n")
	} else {
		triggerVal := entry.TriggerName
		sb.WriteString(prefix + commentStyle.Render(fmt.Sprintf("%-12s", "Trigger")) + " " + triggerVal + "\n")
	}

	_ = width
	return padToHeight(sb.String(), height)
}

func padToHeight(s string, height int) string {
	if height <= 0 {
		return s
	}
	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	lines := lipgloss.Height(s)
	if lines >= height {
		return s
	}
	return s + strings.Repeat("\n", height-lines)
}

func (m Model) renderStatus(totalWidth int) string {
	var left string

	if m.statusMsg != "" {
		left = reorderHighlight.Render(m.statusMsg)
	} else if m.filterActive && m.filterTyping {
		left = helpLine(h("type", "filter"), h("Enter", "apply"), h("Esc", "cancel"))
	} else if m.entryFilterActive && m.entryFilterTyping {
		left = helpLine(h("type", "filter"), h("Enter", "apply"), h("Tab", "mode"), h("Esc", "cancel"))
	} else if m.entryFilterActive {
		left = helpLine(h("j/k", "navigate"), h("Enter", "edit"), h("Esc", "clear filter"))
	} else if m.fieldEditing {
		left = helpLine(h("type", "edit"), h("Enter", "confirm"), h("Esc", "cancel"))
	} else if m.dropActive {
		left = helpLine(h("j/k", "select"), h("Enter", "confirm"), h("Esc", "cancel"))
	} else {
		switch m.level {
		case LevelFiles:
			if m.filterActive {
				left = helpLine(h("j/k", "navigate"), h("Enter", "open"), h("r", "sort"), h("Esc", "clear filter"))
			} else {
				hp := activeTabStyle.Render("p") + helpDescStyle.Render(": profile")
				left = helpLine(h("j/k", "navigate"), h("Enter", "open"), h("f", "find"), h("r", "sort"), hp, h("q", "quit"))
			}
		case LevelEntries:
			left = helpLine(h("j/k", "navigate"), h("e", "edit"), h("a", "add"), h("c", "copy"), h("f", "find"), h("R", "restore"), h("d", "delete"), h("s", "save"), h("Esc", "back"))
		case LevelEdit:
			left = helpLine(h("j/k", "navigate"), h("Enter", "edit field"), h("R", "restore"), h("s", "save"), h("Esc", "back"))
		}
	}

	right := m.renderProfileTabs()

	styleWidth := totalWidth - 2
	textWidth := styleWidth - 2
	if textWidth < 40 {
		textWidth = 40
	}

	leftLen := lipgloss.Width(left)
	rightLen := lipgloss.Width(right)
	gap := textWidth - leftLen - rightLen
	if gap < 1 {
		gap = 1
	}

	content := left + strings.Repeat(" ", gap) + right
	return statusBarStyle.Width(styleWidth).Render(content)
}

func (m Model) renderProfileTabs() string {
	if len(m.profiles) <= 1 {
		return ""
	}
	var tabs []string
	for i, p := range m.profiles {
		label := p.label
		if i == m.activeProfile {
			tabs = append(tabs, activeTabStyle.Render(label))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(label))
		}
	}
	return strings.Join(tabs, helpSepStyle.Render(" | "))
}

func h(k, desc string) string {
	return helpKeyStyle.Render(k) + helpDescStyle.Render(": "+desc)
}

func helpLine(entries ...string) string {
	return strings.Join(entries, "  ")
}
