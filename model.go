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

type ProfileState struct {
	label string
	dir   string
	files []string // full paths
}

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
				}
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

		if m.dropActive {
			return m.updateDroplist(msg)
		}

		if m.fieldEditing {
			return m.updateFieldInput(msg)
		}

		if key.Matches(msg, keys.Save) && m.currentSwt != nil {
			return m, m.saveCmd()
		}

		if key.Matches(msg, keys.Profile) && m.level == LevelFiles && len(m.profiles) > 1 {
			m.activeProfile = (m.activeProfile + 1) % len(m.profiles)
			m.fileCursor = 0
			m.fileScroll = 0
			m.currentSwt = nil
			m.clearFilter()
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
			if m.level == LevelEdit {
				m.level = LevelEntries
				return m, nil
			}
			if m.level == LevelEntries {
				m.level = LevelFiles
				m.currentSwt = nil
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
				m.level = LevelEntries
			}
		}

	case key.Matches(msg, keys.Find):
		m.filterActive = true
		m.filterTyping = true
		m.filterInput = ""
		m.updateFileFilter()
	}
	return m, nil
}

func (m Model) updateEntries(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.currentSwt == nil {
		return m, nil
	}
	switch {
	case key.Matches(msg, keys.Up):
		if m.entryCursor > 0 {
			m.entryCursor--
		}
	case key.Matches(msg, keys.Down):
		if m.entryCursor < len(m.currentSwt.Entries)-1 {
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
	case key.Matches(msg, keys.Delete):
		if len(m.currentSwt.Entries) > 0 {
			e := m.currentEntry()
			m.confirmDelete = true
			m.statusMsg = fmt.Sprintf("Delete %s [%s]? Y/N", e.Unit(), e.EntityType())
		}
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
	case key.Matches(msg, keys.Space):
		if m.fieldCursor == 2 { // Type toggle
			current := entry.Params[2]
			for i, t := range entityTypes {
				if t == current {
					entry.Params[2] = entityTypes[(i+1)%len(entityTypes)]
					m.currentSwt.RecalcDirty()
					return m, nil
				}
			}
			entry.Params[2] = entityTypes[0]
			m.currentSwt.RecalcDirty()
		}
	case key.Matches(msg, keys.Enter):
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

	contentWidth := m.width - 4
	contentHeight := m.height - 6
	if contentHeight < 5 {
		contentHeight = 5
	}

	title := appTitleStyle.Render("Spawn Editor")

	var panel string
	switch m.level {
	case LevelFiles:
		panel = m.renderFiles(contentWidth, contentHeight)
	case LevelEntries:
		panel = m.renderEntries(contentWidth, contentHeight)
	case LevelEdit:
		panel = m.renderEditFields(contentWidth, contentHeight)
	}

	panelStyle := activePanelStyle.Width(contentWidth).Height(contentHeight)
	rendered := panelStyle.Render(panel)

	status := m.renderStatus(lipgloss.Width(rendered))

	return lipgloss.JoinVertical(lipgloss.Left, title, rendered, status)
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
	textWidth := width - 2

	if m.currentSwt == nil {
		return "No file loaded"
	}

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

	entries := m.currentSwt.Entries
	detailLines := 0
	if len(entries) > 0 {
		detailLines = 3
	}
	visibleHeight := height - 3 - detailLines
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	if m.entryCursor < m.entryScroll {
		m.entryScroll = m.entryCursor
	}
	if m.entryCursor >= m.entryScroll+visibleHeight {
		m.entryScroll = m.entryCursor - visibleHeight + 1
	}

	for i := m.entryScroll; i < len(entries) && i < m.entryScroll+visibleHeight; i++ {
		e := entries[i]
		display := e.Unit()
		if display == "" {
			display = "(empty)"
		}
		eType := e.EntityType()
		if eType != "" {
			display += " [" + eType + "]"
		}
		if len(display) > textWidth-2 {
			display = display[:textWidth-5] + "..."
		}

		modified := e.Modified()
		gutter := " "
		if modified {
			gutter = activeTabStyle.Render("·")
		}

		if i == m.entryCursor {
			sb.WriteString(selectorPrefix.Render("▶ ") + reorderHighlight.Render(display) + "\n")
		} else {
			sb.WriteString(gutter + " " + display + "\n")
		}
	}

	if len(entries) > 0 && m.entryCursor < len(entries) {
		e := entries[m.entryCursor]
		rendered := len(entries)
		if rendered > visibleHeight {
			rendered = visibleHeight
		}
		showing := rendered
		if m.entryScroll > 0 {
			showing = visibleHeight
		}
		remaining := visibleHeight - showing
		for i := 0; i < remaining; i++ {
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
		if e.Zone() != "" {
			sb.WriteString(commentStyle.Render("  Zone: " + e.Zone()) + "\n")
		}
		if e.Owner() != "" {
			sb.WriteString(commentStyle.Render("  Owner: " + e.Owner()))
		}
	}

	return sb.String()
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
	sb.WriteString("\n\n")

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
			maxShow := 5
			if maxShow > len(m.dropItems) {
				maxShow = len(m.dropItems)
			}
			scroll := 0
			if m.dropCursor >= maxShow {
				scroll = m.dropCursor - maxShow + 1
			}
			for di := scroll; di < len(m.dropItems) && di < scroll+maxShow; di++ {
				item := m.dropItems[di]
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

	return sb.String()
}

func (m Model) renderStatus(totalWidth int) string {
	var left string

	if m.statusMsg != "" {
		left = reorderHighlight.Render(m.statusMsg)
	} else if m.filterActive && m.filterTyping {
		left = helpLine(h("type", "filter"), h("Enter", "apply"), h("Esc", "cancel"))
	} else if m.fieldEditing {
		left = helpLine(h("type", "edit"), h("Enter", "confirm"), h("Esc", "cancel"))
	} else if m.dropActive {
		left = helpLine(h("j/k", "select"), h("Enter", "confirm"), h("Esc", "cancel"))
	} else {
		switch m.level {
		case LevelFiles:
			if m.filterActive {
				left = helpLine(h("j/k", "navigate"), h("Enter", "open"), h("Esc", "clear filter"))
			} else {
				hp := activeTabStyle.Render("p") + helpDescStyle.Render(": profile")
				left = helpLine(h("j/k", "navigate"), h("Enter", "open"), h("f", "find"), hp, h("q", "quit"))
			}
		case LevelEntries:
			left = helpLine(h("j/k", "navigate"), h("e", "edit"), h("a", "add"), h("d", "delete"), h("s", "save"), h("Esc", "back"))
		case LevelEdit:
			left = helpLine(h("j/k", "navigate"), h("Enter", "edit field"), h("Space", "toggle type"), h("s", "save"), h("Esc", "back"))
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
