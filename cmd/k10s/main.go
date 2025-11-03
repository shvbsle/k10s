package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/adrg/xdg"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/shvbsle/k10s/internal/config"
	"github.com/shvbsle/k10s/internal/k8s"
	"github.com/shvbsle/k10s/internal/plugins"
	"github.com/shvbsle/k10s/internal/plugins/kitten"
	"github.com/shvbsle/k10s/internal/tui"
)

func setupLogging() error {
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

	pluginRegistry := plugins.NewRegistry()
	pluginRegistry.Register(kitten.New())
	log.Printf("Loaded %d plugins", len(pluginRegistry.List()))

	log.Printf("Starting TUI...")

	for {
		p := tea.NewProgram(
			tui.New(cfg, client, pluginRegistry),
			tea.WithAltScreen(),
			tea.WithMouseCellMotion(),
		)

		finalModel, err := p.Run()
		if err != nil {
			log.Fatal(err)
		}

		if finalModel == nil {
			break
		}

		model, ok := finalModel.(tui.Model)
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
