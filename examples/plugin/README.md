# Plugin Example

This directory contains a working plugin example that compiles against the k10s plugin API.

## Purpose

Unlike documentation examples in markdown, this code is compiled and tested, ensuring it stays up-to-date with the plugin interface.

## Running the Example

```bash
cd examples/plugin
go run simple_plugin.go
```

Expected output:
```
Plugin Name: example
Description: Example plugin demonstrating the plugin API
Commands: [example demo]

=================================
    Example Plugin Launched!
=================================

This is a simple plugin example.
In a real plugin, you would:
  - Launch a TUI with Bubble Tea
  - Run a game with termloop
  - Execute interactive tools
  - Display custom dashboards

Returning to k10s in 3 seconds...
```

## Creating Your Own Plugin

1. Copy this example as a template
2. Implement the four interface methods:
   - `Name()` - Unique identifier
   - `Description()` - Human-readable description
   - `Commands()` - Command aliases
   - `Launch()` - Plugin logic
3. Register in `cmd/k10s/main.go`:
   ```go
   pluginRegistry.Register(yourplugin.New())
   ```
4. Build k10s and test with `:yourcommand`

See the main README and `internal/plugins/README.md` for complete documentation.
