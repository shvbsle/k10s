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

    // Launch executes the plugin
    // Returns true to restart k10s TUI after plugin exits
    // Returns false to exit k10s entirely
    Launch() bool
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

func (k *KittenClimberPlugin) Launch() bool {
    game.LaunchGame()
    return true  // Return to k10s after game exits
}

func New() *KittenClimberPlugin {
    return &KittenClimberPlugin{}
}
```

## Plugin Guidelines

- **Naming**: Use kebab-case for plugin names (e.g., "my-plugin", "debug-tool")
- **Commands**: Choose short, memorable command aliases
- **Return Value**:
  - Return `true` if your plugin is temporary (games, viewers, utilities)
  - Return `false` if your plugin should exit k10s entirely
- **Error Handling**: Handle errors gracefully within your plugin
- **Dependencies**: Add required dependencies to `go.mod`
- **Testing**: Test your plugin in isolation before integration

## Plugin Types

### TUI-based Plugins

For plugins using Bubble Tea or other TUI frameworks:

```go
func (p *MyPlugin) Launch() bool {
    program := tea.NewProgram(myModel{})
    if _, err := program.Run(); err != nil {
        log.Printf("Error running plugin: %v", err)
    }
    return true  // Return to k10s
}
```

### CLI-based Plugins

For plugins that run command-line tools:

```go
func (p *MyPlugin) Launch() bool {
    cmd := exec.Command("kubectl", "debug", "...")
    output, err := cmd.CombinedOutput()
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    }
    fmt.Printf("%s\n", output)
    return true
}
```

### Utility Plugins

For plugins that perform quick actions:

```go
func (p *MyPlugin) Launch() bool {
    fmt.Println("Processing...")
    // Do some work
    fmt.Println("Done!")
    time.Sleep(2 * time.Second)  // Let user see output
    return true
}
```
