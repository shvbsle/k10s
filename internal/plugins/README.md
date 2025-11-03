# k10s Plugins

This directory contains the plugin system for k10s.

## Structure

- `plugin.go` - Core plugin interface and registry
- `kitten/` - Built-in Kitten Climber game plugin (reference implementation)
- `[your-plugin]/` - Add your custom plugins here

## Plugin Interface

Plugins implement a simple interface that provides metadata and a launch function:

```go
type Plugin interface {
    // Name returns unique identifier for this plugin (kebab-case recommended)
    Name() string

    // Description returns human-readable description
    Description() string

    // Commands returns command aliases that trigger this plugin
    Commands() []string

    // Launch executes the plugin and returns an error if it fails
    Launch() error
}
```

## Registry

The registry manages plugin discovery and command routing:

```go
// Create registry
registry := plugins.NewRegistry()

// Register plugins
registry.Register(myplugin.New())

// Lookup by name
plugin, ok := registry.Get("my-plugin")

// Lookup by command
plugin, ok := registry.GetByCommand("mycommand")

// Get all plugin commands for autocomplete
commands := registry.CommandSuggestions()
```

## Creating a Plugin

See the main [README.md](../../README.md#plugins) for detailed step-by-step instructions.

## Example: Kitten Climber

The `kitten/` directory contains a complete reference implementation of a plugin that launches the Kitten Climber platformer game.

```go
package kitten

import "github.com/shvbsle/k10s/internal/game"

type KittenClimberPlugin struct{}

func (k *KittenClimberPlugin) Name() string {
    return "kitten-climber"
}

func (k *KittenClimberPlugin) Description() string {
    return "Kitten Climber - An infinite runner platformer game"
}

func (k *KittenClimberPlugin) Commands() []string {
    return []string{"play", "game", "kitten"}
}

func (k *KittenClimberPlugin) Launch() error {
    game.LaunchGame()
    return nil  // Return to k10s after game exits
}

func New() *KittenClimberPlugin {
    return &KittenClimberPlugin{}
}
```

## Plugin Guidelines

- **Naming**: Use kebab-case for plugin names (e.g., "my-plugin", "debug-tool")
- **Commands**: Choose short, memorable command aliases
- **Return Value**: Return `nil` on success or an `error` if the plugin fails
- **Error Handling**: Handle errors gracefully within your plugin
- **Dependencies**: Add required dependencies to `go.mod`
- **Testing**: Test your plugin in isolation before integration

## Plugin Types

### TUI-based Plugins

For plugins using Bubble Tea or other TUI frameworks:

```go
func (p *MyPlugin) Launch() error {
    program := tea.NewProgram(myModel{})
    if _, err := program.Run(); err != nil {
        return fmt.Errorf("error running plugin: %w", err)
    }
    return nil  // Return to k10s
}
```

### CLI-based Plugins

For plugins that run command-line tools:

```go
func (p *MyPlugin) Launch() error {
    cmd := exec.Command("kubectl", "debug", "...")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("command failed: %w", err)
    }
    fmt.Printf("%s\n", output)
    return nil
}
```

### Utility Plugins

For plugins that perform quick actions:

```go
func (p *MyPlugin) Launch() error {
    fmt.Println("Processing...")
    // Do some work
    fmt.Println("Done!")
    time.Sleep(2 * time.Second)  // Let user see output
    return nil
}
```
