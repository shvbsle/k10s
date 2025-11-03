package main

import (
	"fmt"
	"time"

	"github.com/shvbsle/k10s/internal/plugins"
)

// SimplePlugin demonstrates the plugin interface implementation.
type SimplePlugin struct{}

func (s *SimplePlugin) Name() string {
	return "example"
}

func (s *SimplePlugin) Description() string {
	return "Example plugin demonstrating the plugin API"
}

func (s *SimplePlugin) Commands() []string {
	return []string{"example", "demo"}
}

func (s *SimplePlugin) Launch() error {
	fmt.Println("=================================")
	fmt.Println("    Example Plugin Launched!")
	fmt.Println("=================================")
	fmt.Println()
	fmt.Println("This is a simple plugin example.")
	fmt.Println("In a real plugin, you would:")
	fmt.Println("  - Launch a TUI with Bubble Tea")
	fmt.Println("  - Run a game with termloop")
	fmt.Println("  - Execute interactive tools")
	fmt.Println("  - Display custom dashboards")
	fmt.Println()
	fmt.Println("Returning to k10s in 3 seconds...")

	time.Sleep(3 * time.Second)
	return nil
}

// Verify interface implementation at compile time
var _ plugins.Plugin = (*SimplePlugin)(nil)

func main() {
	plugin := &SimplePlugin{}

	fmt.Printf("Plugin Name: %s\n", plugin.Name())
	fmt.Printf("Description: %s\n", plugin.Description())
	fmt.Printf("Commands: %v\n", plugin.Commands())
	fmt.Println()

	if err := plugin.Launch(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
