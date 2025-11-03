package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/adrg/xdg"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/shvbsle/k10s/internal/config"
	"github.com/shvbsle/k10s/internal/k8s"
	"github.com/shvbsle/k10s/internal/tui"
)

// setupLogging configures logging to write to the XDG state directory at
// ~/.local/state/k10s/k10s.log (or platform equivalent). It sets up a
// structured logger using slog. Returns the log file handle and an error
// if the log file cannot be created or opened.
func setupLogging() (*os.File, error) {
	// Use XDG state directory for cross-platform log storage
	logPath, err := xdg.StateFile("k10s/k10s.log")
	if err != nil {
		return nil, fmt.Errorf("could not get log path: %w", err)
	}

	f, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("could not open log file: %w", err)
	}

	// Create a JSON handler for structured logging
	handler := slog.NewJSONHandler(f, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("k10s logging initialized", "log_path", logPath)
	return f, nil
}

func main() {
	logFile, err := setupLogging()
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

	if err := config.CreateDefaultConfig(); err != nil {
		slog.Warn("could not create default config", "error", err)
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}
	slog.Info("configuration loaded", "page_size", cfg.PageSize)

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

	// Works even if client is nil or disconnected
	slog.Info("starting TUI")
	m := tui.New(cfg, client)

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		slog.Error("TUI error", "error", err)
		os.Exit(1)
	}
}
