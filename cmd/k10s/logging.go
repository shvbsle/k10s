package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/adrg/xdg"
)

// parseLogLevel parses a log level string and returns the corresponding slog.Level.
// Supports: debug, info, warn, error (case-insensitive).
// Returns slog.LevelInfo if the level string is invalid.
func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// getLogPath determines the log file path to use.
// Priority: customPath (from config) > XDG default path
// If customPath is invalid, falls back to XDG path.
func getLogPath(customPath string) (string, error) {
	// If custom path is provided, try to use it
	if customPath != "" {
		// Expand ~ to home directory
		if strings.HasPrefix(customPath, "~/") {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				customPath = strings.Replace(customPath, "~", homeDir, 1)
			}
		}

		// Try to create parent directories if they don't exist
		var dir string
		if lastSlash := strings.LastIndex(customPath, "/"); lastSlash > 0 {
			dir = customPath[:lastSlash]
		} else {
			// No directory separator, use current directory
			dir = "."
		}

		if err := os.MkdirAll(dir, 0755); err == nil {
			// Test if we can write to this location
			testFile, err := os.OpenFile(customPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
			if err == nil {
				_ = testFile.Close() // Ignore close error on test file
				return customPath, nil
			}
		}

		// If custom path fails, we'll fall through to XDG default
		fmt.Fprintf(os.Stderr, "Warning: could not use custom log path %s, falling back to XDG default\n", customPath)
	}

	// Use XDG state directory for cross-platform log storage
	logPath, err := xdg.StateFile("k10s/k10s.log")
	if err != nil {
		return "", fmt.Errorf("could not get log path: %w", err)
	}

	return logPath, nil
}
