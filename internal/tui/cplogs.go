package tui

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/shvbsle/k10s/internal/k8s"
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
			// Treat as file path
			filePath = arg
		}
	}

	return func() tea.Msg {
		// Get the logs to copy/write
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
			return logsCopiedMsg{
				success: false,
				message: "no logs on current page",
			}
		}

		// Format the logs
		formattedLogs := m.formatLogs(logsToProcess)

		// Either copy to clipboard or write to file
		if filePath == "" {
			// Copy to clipboard
			err := clipboard.WriteAll(formattedLogs)
			if err != nil {
				slog.Error("failed to copy logs to clipboard", "error", err)
				return logsCopiedMsg{
					success: false,
					message: fmt.Sprintf("failed to copy to clipboard: %v", err),
				}
			}

			slog.Info("copied logs to clipboard", "lines", len(logsToProcess), "scope", scope)
			return logsCopiedMsg{
				success: true,
				message: fmt.Sprintf("Copied %d lines (%s) to clipboard", len(logsToProcess), scope),
			}
		} else {
			// Write to file
			err := m.writeLogsToFile(formattedLogs, filePath)
			if err != nil {
				slog.Error("failed to write logs to file", "file_path", filePath, "error", err)
				return logsCopiedMsg{
					success: false,
					message: fmt.Sprintf("failed to write to file: %v", err),
				}
			}

			slog.Info("wrote logs to file", "lines", len(logsToProcess), "scope", scope, "file_path", filePath)
			return logsCopiedMsg{
				success: true,
				message: fmt.Sprintf("Wrote %d lines (%s) to %s", len(logsToProcess), scope, filePath),
			}
		}
	}
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
			b.WriteString(fmt.Sprintf("[%s] %s\n", logLine.Timestamp, logLine.Content))
		} else {
			b.WriteString(fmt.Sprintf("%s\n", logLine.Content))
		}
	}

	return b.String()
}

func (m *Model) writeLogsToFile(content string, filePath string) error {
	// Expand ~ to home directory if present
	if strings.HasPrefix(filePath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		filePath = filepath.Join(homeDir, filePath[2:])
	}

	// Create parent directories if they don't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Write the file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
