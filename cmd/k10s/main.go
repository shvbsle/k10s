package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shvbsle/k10s/internal/config"
	"github.com/shvbsle/k10s/internal/k8s"
	"github.com/shvbsle/k10s/internal/tui"
)

func main() {
	// Create default config file if it doesn't exist
	if err := config.CreateDefaultConfig(); err != nil {
		log.Printf("Warning: could not create default config: %v", err)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize Kubernetes client
	client, err := k8s.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize Kubernetes client: %v\n", err)
		fmt.Fprintf(os.Stderr, "Make sure you have a valid kubeconfig file or are running in a cluster.\n")
		os.Exit(1)
	}

	// Initialize TUI
	m := tui.New(cfg, client)

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
