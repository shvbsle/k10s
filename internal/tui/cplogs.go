package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/shvbsle/k10s/internal/k8s"
	"github.com/shvbsle/k10s/internal/log"
)

type logsCopiedMsg struct {
	success bool
	message string
}

// executeCplogsCommand copies or writes container logs.
// Usage: :cplogs [all] [path]
func (m *Model) executeCplogsCommand(args []string) tea.Cmd {
	// Validate we're in logs view
	if m.currentGVR.Resource != k8s.ResourceLogs {
		return m.showCommandError("cplogs only works in logs view")
	}

	if len(m.logLines) == 0 {
		return m.showCommandError("no logs to copy")
	}

	// Parse arguments
	copyAll := false
	filePath := ""

	for _, arg := range args {
		if arg == "all" {
			copyAll = true
		} else {
			filePath = arg
		}
	}

	var logsToProcess []k8s.LogLine
	var scope string

	if copyAll {
		logsToProcess = m.logLines
		scope = "all"
	} else {
		logsToProcess = m.getCurrentPageLogs()
		scope = "current page"
	}

	if len(logsToProcess) == 0 {
		return m.showCommandError("no logs on current page")
	}

	formattedLogs := m.formatLogs(logsToProcess)

	if filePath != "" {
		// Write to file (runs in goroutine)
		return func() tea.Msg {
			err := m.writeLogsToFile(formattedLogs, filePath)
			if err != nil {
				log.G().Error("failed to write logs to file", "file_path", filePath, "error", err)
				return logsCopiedMsg{
					success: false,
					message: fmt.Sprintf("failed to write to file: %v", err),
				}
			}
			log.G().Info("wrote logs to file", "lines", len(logsToProcess), "scope", scope, "file_path", filePath)
			return logsCopiedMsg{
				success: true,
				message: fmt.Sprintf("Wrote %d lines (%s) to %s", len(logsToProcess), scope, filePath),
			}
		}
	}

	// Copy to clipboard via OSC 52 (works over SSH, no xsel/xclip needed)
	lineCount := len(logsToProcess)
	log.G().Info("copying logs to clipboard via OSC 52", "lines", lineCount, "scope", scope)
	return tea.Batch(
		tea.SetClipboard(formattedLogs),
		func() tea.Msg {
			return logsCopiedMsg{
				success: true,
				message: fmt.Sprintf("Copied %d lines (%s) to clipboard", lineCount, scope),
			}
		},
	)
}

func (m *Model) getCurrentPageLogs() []k8s.LogLine {
	start := m.paginator.Page * m.paginator.PerPage
	end := min(start+m.paginator.PerPage, len(m.logLines))

	if start >= len(m.logLines) {
		return []k8s.LogLine{}
	}

	return m.logLines[start:end]
}

func (m *Model) formatLogs(logLines []k8s.LogLine) string {
	var b strings.Builder

	for _, logLine := range logLines {
		if m.logView.ShowTimestamps && logLine.Timestamp != "" {
			fmt.Fprintf(&b, "[%s] %s\n", logLine.Timestamp, logLine.Content)
		} else {
			fmt.Fprintf(&b, "%s\n", logLine.Content)
		}
	}

	return b.String()
}

func (m *Model) writeLogsToFile(content string, filePath string) error {
	if strings.HasPrefix(filePath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		filePath = filepath.Join(homeDir, filePath[2:])
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// copySelectedLogLines copies the currently selected log lines to the clipboard
// using OSC 52 (works over SSH without xsel/xclip).
func (m *Model) copySelectedLogLines() tea.Cmd {
	selected := m.logViewport.SelectedLines()
	if len(selected) == 0 {
		return m.showCommandError("no lines selected")
	}

	formattedLogs := m.formatLogs(selected)
	lineCount := len(selected)
	log.G().Info("copying selected logs to clipboard via OSC 52", "lines", lineCount)

	// Clear selection after copy
	m.logViewport.ClearSelection()

	return tea.Batch(
		tea.SetClipboard(formattedLogs),
		func() tea.Msg {
			return logsCopiedMsg{
				success: true,
				message: fmt.Sprintf("Yanked %d selected lines to clipboard", lineCount),
			}
		},
	)
}
