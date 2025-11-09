package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shvbsle/k10s/internal/config"
	"github.com/shvbsle/k10s/internal/k8s"
	"github.com/shvbsle/k10s/internal/log"
	"github.com/shvbsle/k10s/internal/plugins"
	"github.com/shvbsle/k10s/internal/plugins/kitten"
	"github.com/shvbsle/k10s/internal/tui"
)

func main() {
	// Parse CLI flags
	logLevelFlag := *flag.String("log-level", "info", "Set log level (debug, info, warn, error)")
	flag.Parse()

	// Load config first to get log path preference
	if err := config.CreateDefaultConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not create default config: %v\n", err)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	loggerConfig := &log.LoggerConfiguration{
		LogLevel: parseLogLevel(logLevelFlag),
	}

	if logPath, err := getLogPath(cfg.LogFilePath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not setup logging: %v\n", err)
	} else {
		f, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not open log file: %v\n", err)
			return
		}

		// assign the file writer to the logger configuration
		loggerConfig.Writer = f

		// remember to cleanup the file handle before exiting the program
		defer func() {
			if closeErr := f.Close(); closeErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not setup logging: %v\n", err)
			}
		}()
	}

	logger := log.NewLogger(loggerConfig)
	log.SetDefault(logger)
	logger.Info("k10s logging initialized", "config", loggerConfig)

	logger.Info("k10s starting", "version", tui.Version)
	logger.Info("configuration loaded", "max_page_size", cfg.MaxPageSize)

	// Don't exit on failure, let TUI handle it
	client, err := k8s.NewClient()
	if err != nil {
		logger.Warn("could not initialize Kubernetes client", "error", err)
		logger.Info("starting k10s in disconnected mode")
	} else if client != nil && client.IsConnected() {
		if info, err := client.GetClusterInfo(); err == nil {
			logger.Info("connected to cluster", "cluster", info.Cluster, "context", info.Context)
		} else {
			logger.Warn("could not get cluster info", "error", err)
			logger.Info("connected to Kubernetes API")
		}
	}

	// Initialize plugin registry
	pluginRegistry := plugins.NewRegistry()
	pluginRegistry.Register(kitten.New())
	logger.Info("loaded plugins", "count", len(pluginRegistry.List()))

	// Works even if client is nil or disconnected
	logger.Info("starting TUI")

	for {
		p := tea.NewProgram(
			tui.New(cfg, client, pluginRegistry),
			tea.WithAltScreen(),
			tea.WithMouseCellMotion(),
		)

		finalModel, err := p.Run()
		if err != nil {
			logger.Error("TUI error", "error", err)
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

		logger.Info("launching plugin", "plugin", plugin.Name())
		if err := plugin.Launch(); err != nil {
			logger.Error("plugin launch failed", "plugin", plugin.Name(), "error", err)
		}

		logger.Info("returning to k10s TUI")
	}

	logger.Info("k10s exiting")
}
