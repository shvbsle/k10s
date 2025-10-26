package main

import (
	"fmt"
	"log"
	"os"

	"github.com/adrg/xdg"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/shvbsle/k10s/internal/config"
	"github.com/shvbsle/k10s/internal/game"
	"github.com/shvbsle/k10s/internal/k8s"
	"github.com/shvbsle/k10s/internal/tui"
)

// setupLogging configures logging to write to the XDG state directory at
// ~/.local/state/k10s/k10s.log (or platform equivalent). Returns an error
// if the log file cannot be created or opened.
func setupLogging() error {
	// Use XDG state directory for cross-platform log storage
	logPath, err := xdg.StateFile("k10s/k10s.log")
	if err != nil {
		return fmt.Errorf("could not get log path: %w", err)
	}

	f, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("could not open log file: %w", err)
	}

	log.SetOutput(f)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	return nil
}

func main() {
	if err := setupLogging(); err != nil {
		log.Printf("Warning: could not setup logging: %v", err)
	}

	log.Printf("k10s %s starting...", tui.Version)

	if err := config.CreateDefaultConfig(); err != nil {
		log.Printf("Warning: could not create default config: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Configuration loaded (page_size=%d)", cfg.PageSize)

	// Don't exit on failure, let TUI handle it
	client, err := k8s.NewClient()
	if err != nil {
		log.Printf("Warning: could not initialize Kubernetes client: %v", err)
		log.Printf("Starting k10s in disconnected mode")
	} else if client != nil && client.IsConnected() {
		if info, err := client.GetClusterInfo(); err == nil {
			log.Printf("Connected to cluster: %s (context: %s)", info.Cluster, info.Context)
		} else {
			log.Printf("Warning: could not get cluster info: %v", err)
			log.Printf("Connected to Kubernetes API")
		}
	}

	// Works even if client is nil or disconnected
	log.Printf("Starting TUI...")
	m := tui.New(cfg, client)

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	finalModel, err := p.Run()
	if err != nil {
		log.Fatal(err)
	}

	// Check if we should launch the game
	if finalModel != nil {
		if model, ok := finalModel.(tui.Model); ok && model.ShouldLaunchGame() {
			log.Printf("Launching Kitten Climber game...")
			game.LaunchGame()
		}
	}
}
