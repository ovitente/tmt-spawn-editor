package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const spawnActionName = "a_spawnUnitGroupToZone"
const paramCount = 12

// Field labels for the 12 parameters.
var fieldLabels = [paramCount]string{
	"Group Id", "Param 2", "Type", "Unit", "Preset",
	"Zone", "Owner", "Param 8", "Param 9", "Param 10",
	"Param 11", "Param 12",
}

// Entity types for Type field toggle.
var entityTypes = []string{"squad", "car", "tank", "helicopter"}

// XML structures matching .swt format.

type swtRoot struct {
	XMLName    xml.Name     `xml:"Root"`
	InnerAttrs []xml.Attr   `xml:",any,attr"`
	Variables  []swtVar     `xml:"Variable"`
	Triggers   []swtTrigger `xml:"Trigger"`
}

type swtVar struct {
	XMLName xml.Name `xml:"Variable"`
	Attrs   []xml.Attr `xml:",any,attr"`
}

type swtTrigger struct {
	XMLName xml.Name    `xml:"Trigger"`
	Attrs   []xml.Attr  `xml:",any,attr"`
	GUID    string      `xml:"guid,attr"`
	Name    string      `xml:"Name"`
	Actions []swtAction `xml:"Action"`
	Inner   string      `xml:",innerxml"`
}

type swtAction struct {
	XMLName  xml.Name `xml:"Action"`
	GUID     string   `xml:"guid,attr"`
	Disabled string   `xml:"disabled,attr"`
	Name     string   `xml:"Name"`
	Params   []string `xml:"Param"`
}

// SpawnEntry is a parsed spawn action for the UI.
type SpawnEntry struct {
	TriggerGUID string
	TriggerName string
	ActionGUID  string
	Disabled    string
	Params      [paramCount]string // normalized (prm=? → "")
	Original    [paramCount]string // snapshot for dirty check
}

// Convenience accessors.
func (e *SpawnEntry) EntityType() string { return e.Params[2] }
func (e *SpawnEntry) Unit() string       { return e.Params[3] }
func (e *SpawnEntry) Zone() string       { return e.Params[5] }
func (e *SpawnEntry) Owner() string      { return e.Params[6] }

func (e *SpawnEntry) Modified() bool {
	return e.Params != e.Original
}

// SwtFile holds parsed data for one .swt file.
type SwtFile struct {
	Path    string
	Name    string // basename
	Entries []SpawnEntry
	dirty   bool
}

func (f *SwtFile) RecalcDirty() {
	f.dirty = false
	for _, e := range f.Entries {
		if e.Modified() {
			f.dirty = true
			return
		}
	}
}

// ParseSwtFile reads and parses spawn actions from a .swt file.
func ParseSwtFile(path string) (*SwtFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var root swtRoot
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
	}

	var entries []SpawnEntry
	for _, trigger := range root.Triggers {
		for _, action := range trigger.Actions {
			if strings.TrimSpace(action.Name) != spawnActionName {
				continue
			}
			params := normalizeParams(action.Params)
			entries = append(entries, SpawnEntry{
				TriggerGUID: trigger.GUID,
				TriggerName: trigger.Name,
				ActionGUID:  action.GUID,
				Disabled:    action.Disabled,
				Params:      params,
				Original:    params,
			})
		}
	}

	return &SwtFile{
		Path:    path,
		Name:    filepath.Base(path),
		Entries: entries,
	}, nil
}

// SaveSwtFile writes spawn changes back to the .swt file using targeted replacement.
func SaveSwtFile(sf *SwtFile) error {
	data, err := os.ReadFile(sf.Path)
	if err != nil {
		return err
	}

	content := string(data)

	for _, entry := range sf.Entries {
		if !entry.Modified() {
			continue
		}
		// Find the action block by GUID and replace params
		content, err = replaceActionParams(content, entry.ActionGUID, entry.Params)
		if err != nil {
			return fmt.Errorf("save %s guid=%s: %w", sf.Name, entry.ActionGUID, err)
		}
	}

	return os.WriteFile(sf.Path, []byte(content), 0o644)
}

// replaceActionParams finds an Action by GUID and replaces its Param elements.
func replaceActionParams(content, guid string, params [paramCount]string) (string, error) {
	// Find <Action guid="XXX"
	marker := fmt.Sprintf(`guid="%s"`, guid)
	idx := strings.Index(content, marker)
	if idx == -1 {
		return content, fmt.Errorf("action guid=%s not found", guid)
	}

	// Find the opening <Action before the guid
	actionStart := strings.LastIndex(content[:idx], "<Action")
	if actionStart == -1 {
		return content, fmt.Errorf("action tag not found for guid=%s", guid)
	}

	// Find </Action> after
	actionEnd := strings.Index(content[actionStart:], "</Action>")
	if actionEnd == -1 {
		return content, fmt.Errorf("closing </Action> not found for guid=%s", guid)
	}
	actionEnd += actionStart + len("</Action>")

	// Build new action block
	oldBlock := content[actionStart:actionEnd]
	// Extract the opening tag line (preserves attributes)
	nameEnd := strings.Index(oldBlock, "</Name>")
	if nameEnd == -1 {
		return content, fmt.Errorf("Name tag not found in action guid=%s", guid)
	}
	header := oldBlock[:nameEnd+len("</Name>")]

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n")
	for _, p := range params {
		val := p
		if val == "" {
			val = "prm=?"
		}
		sb.WriteString("<Param>" + xmlEscape(val) + "</Param>\n")
	}
	sb.WriteString("</Action>")

	return content[:actionStart] + sb.String() + content[actionEnd:], nil
}

// AddSpawnEntry adds a new spawn action to the first trigger in the file.
func AddSpawnEntry(sf *SwtFile) (*SpawnEntry, error) {
	data, err := os.ReadFile(sf.Path)
	if err != nil {
		return nil, err
	}

	var root swtRoot
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, err
	}

	if len(root.Triggers) == 0 {
		return nil, fmt.Errorf("no triggers in %s", sf.Name)
	}

	guid := nextGUID(&root)
	triggerGUID := root.Triggers[0].GUID
	triggerName := root.Triggers[0].Name

	// Insert new action XML before </Trigger> of first trigger
	content := string(data)
	triggerMarker := fmt.Sprintf(`guid="%s"`, triggerGUID)
	triggerIdx := strings.Index(content, triggerMarker)
	if triggerIdx == -1 {
		return nil, fmt.Errorf("trigger guid=%s not found", triggerGUID)
	}

	// Find </Trigger> after this trigger
	closeTrigger := strings.Index(content[triggerIdx:], "</Trigger>")
	if closeTrigger == -1 {
		return nil, fmt.Errorf("closing </Trigger> not found")
	}
	insertPos := triggerIdx + closeTrigger

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<Action guid="%s" disabled="0"><Name>%s</Name>`, guid, spawnActionName))
	sb.WriteString("\n")
	for i := 0; i < paramCount; i++ {
		sb.WriteString("<Param>prm=?</Param>\n")
	}
	sb.WriteString("</Action>\n")

	newContent := content[:insertPos] + sb.String() + content[insertPos:]
	if err := os.WriteFile(sf.Path, []byte(newContent), 0o644); err != nil {
		return nil, err
	}

	var params [paramCount]string
	entry := SpawnEntry{
		TriggerGUID: triggerGUID,
		TriggerName: triggerName,
		ActionGUID:  guid,
		Disabled:    "0",
		Params:      params,
		Original:    params,
	}
	sf.Entries = append(sf.Entries, entry)
	return &entry, nil
}

// DeleteSpawnEntry removes an action from the file by GUID.
func DeleteSpawnEntry(sf *SwtFile, idx int) error {
	if idx < 0 || idx >= len(sf.Entries) {
		return fmt.Errorf("index out of range")
	}
	entry := sf.Entries[idx]

	data, err := os.ReadFile(sf.Path)
	if err != nil {
		return err
	}

	content := string(data)
	marker := fmt.Sprintf(`guid="%s"`, entry.ActionGUID)
	guidIdx := strings.Index(content, marker)
	if guidIdx == -1 {
		return fmt.Errorf("action guid=%s not found", entry.ActionGUID)
	}

	actionStart := strings.LastIndex(content[:guidIdx], "<Action")
	if actionStart == -1 {
		return fmt.Errorf("action tag not found for guid=%s", entry.ActionGUID)
	}

	actionEnd := strings.Index(content[actionStart:], "</Action>")
	if actionEnd == -1 {
		return fmt.Errorf("closing </Action> not found")
	}
	actionEnd += actionStart + len("</Action>")

	// Also eat trailing newline
	if actionEnd < len(content) && content[actionEnd] == '\n' {
		actionEnd++
	}

	newContent := content[:actionStart] + content[actionEnd:]
	if err := os.WriteFile(sf.Path, []byte(newContent), 0o644); err != nil {
		return err
	}

	sf.Entries = append(sf.Entries[:idx], sf.Entries[idx+1:]...)
	return nil
}

// CollectSwtFiles returns all .swt file paths in a directory.
func CollectSwtFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".swt") {
			paths = append(paths, filepath.Join(dir, e.Name()))
		}
	}
	return paths, nil
}

// CollectCandidates extracts unique values from a specific param index across all entries.
func CollectCandidates(entries []SpawnEntry, paramIdx int, always ...string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, a := range always {
		if !seen[a] {
			seen[a] = true
			result = append(result, a)
		}
	}
	for _, e := range entries {
		val := e.Params[paramIdx]
		if val != "" && !seen[val] {
			seen[val] = true
			result = append(result, val)
		}
	}
	return result
}

func normalizeParams(raw []string) [paramCount]string {
	var result [paramCount]string
	for i := 0; i < paramCount; i++ {
		if i < len(raw) {
			val := strings.TrimSpace(raw[i])
			if strings.EqualFold(val, "prm=?") {
				val = ""
			}
			result[i] = val
		}
	}
	return result
}

func nextGUID(root *swtRoot) string {
	maxVal := 0
	foundNumeric := false
	for _, trigger := range root.Triggers {
		if v, ok := parseNumericGUID(trigger.GUID); ok && v > maxVal {
			maxVal = v
			foundNumeric = true
		}
		for _, action := range trigger.Actions {
			if v, ok := parseNumericGUID(action.GUID); ok && v > maxVal {
				maxVal = v
				foundNumeric = true
			}
		}
	}
	if foundNumeric {
		return fmt.Sprintf("%d", maxVal+1)
	}
	return fmt.Sprintf("%d", 10000)
}

func parseNumericGUID(s string) (int, bool) {
	s = strings.TrimSpace(s)
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return v, true
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
