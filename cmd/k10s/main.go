package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/adrg/xdg"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/shvbsle/k10s/internal/config"
	"github.com/shvbsle/k10s/internal/k8s"
	"github.com/shvbsle/k10s/internal/plugins"
	"github.com/shvbsle/k10s/internal/plugins/kitten"
	"github.com/shvbsle/k10s/internal/tui"
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

// setupLogging configures logging to write to the specified log file.
// It sets up a structured logger using slog. Returns the log file handle
// and an error if the log file cannot be created or opened.
func setupLogging(logLevel slog.Level, customLogPath string) (*os.File, error) {
	logPath, err := getLogPath(customLogPath)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("could not open log file: %w", err)
	}

	// Create a JSON handler for structured logging
	handler := slog.NewJSONHandler(f, &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("k10s logging initialized", "log_path", logPath, "log_level", logLevel.String())
	return f, nil
}

func main() {
	// Parse CLI flags
	logLevelFlag := flag.String("log-level", "", "Set log level (debug, info, warn, error). Defaults to info.")
	flag.Parse()

	// Determine log level from flag (defaults to info if empty)
	logLevel := parseLogLevel(*logLevelFlag)

	// Load config first to get log path preference
	if err := config.CreateDefaultConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not create default config: %v\n", err)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup logging with custom path from config (if specified)
	logFile, err := setupLogging(logLevel, cfg.LogFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not setup logging: %v\n", err)
	} else if logFile != nil {
		defer func() {
			if closeErr := logFile.Close(); closeErr != nil {
				slog.Error("failed to close log file", "error", closeErr)
			}
		}()
	}

	slog.Info("k10s starting", "version", tui.Version)
	slog.Info("configuration loaded", "max_page_size", cfg.MaxPageSize)

	// Don't exit on failure, let TUI handle it
	client, err := k8s.NewClient()
	if err != nil {
		slog.Warn("could not initialize Kubernetes client", "error", err)
		slog.Info("starting k10s in disconnected mode")
	} else if client != nil && client.IsConnected() {
		if info, err := client.GetClusterInfo(); err == nil {
			slog.Info("connected to cluster", "cluster", info.Cluster, "context", info.Context)
		} else {
			slog.Warn("could not get cluster info", "error", err)
			slog.Info("connected to Kubernetes API")
		}
	}

	// Initialize plugin registry
	pluginRegistry := plugins.NewRegistry()
	pluginRegistry.Register(kitten.New())
	slog.Info("loaded plugins", "count", len(pluginRegistry.List()))

	// Works even if client is nil or disconnected
	slog.Info("starting TUI")

	for {
		p := tea.NewProgram(
			tui.New(cfg, client, pluginRegistry),
			tea.WithAltScreen(),
			tea.WithMouseCellMotion(),
		)

		finalModel, err := p.Run()
		if err != nil {
			slog.Error("TUI error", "error", err)
			os.Exit(1)
		}

		if finalModel == nil {
			break
		}

		model, ok := finalModel.(*tui.Model)
		if !ok {
			break
		}

		plugin := model.GetPluginToLaunch()
		if plugin == nil {
			break
		}

		slog.Info("launching plugin", "plugin", plugin.Name())
		if err := plugin.Launch(); err != nil {
			slog.Error("plugin launch failed", "plugin", plugin.Name(), "error", err)
		}

		slog.Info("returning to k10s TUI")
	}

	slog.Info("k10s exiting")
}
