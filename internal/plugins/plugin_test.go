package plugins

import (
	"testing"
)

type mockPlugin struct {
	name        string
	description string
	commands    []string
}

func (m *mockPlugin) Name() string {
	return m.name
}

func (m *mockPlugin) Description() string {
	return m.description
}

func (m *mockPlugin) Commands() []string {
	return m.commands
}

func (m *mockPlugin) Launch() (bool, error) {
	return true, nil
}

func TestRegistryRegister(t *testing.T) {
	registry := NewRegistry()

	plugin1 := &mockPlugin{
		name:        "test-plugin",
		description: "A test plugin",
		commands:    []string{"test", "tp"},
	}

	registry.Register(plugin1)

	if len(registry.List()) != 1 {
		t.Errorf("Expected 1 plugin, got %d", len(registry.List()))
	}
}

func TestRegistryGet(t *testing.T) {
	registry := NewRegistry()

	plugin1 := &mockPlugin{
		name:        "test-plugin",
		description: "A test plugin",
		commands:    []string{"test"},
	}

	registry.Register(plugin1)

	retrieved, ok := registry.Get("test-plugin")
	if !ok {
		t.Error("Expected to find plugin 'test-plugin'")
	}

	if retrieved.Name() != "test-plugin" {
		t.Errorf("Expected plugin name 'test-plugin', got '%s'", retrieved.Name())
	}

	_, ok = registry.Get("non-existent")
	if ok {
		t.Error("Expected not to find non-existent plugin")
	}
}

func TestRegistryGetByCommand(t *testing.T) {
	registry := NewRegistry()

	plugin1 := &mockPlugin{
		name:        "test-plugin",
		description: "A test plugin",
		commands:    []string{"test", "tp"},
	}

	registry.Register(plugin1)

	retrieved, ok := registry.GetByCommand("test")
	if !ok {
		t.Error("Expected to find plugin by command 'test'")
	}

	if retrieved.Name() != "test-plugin" {
		t.Errorf("Expected plugin name 'test-plugin', got '%s'", retrieved.Name())
	}

	retrieved, ok = registry.GetByCommand("tp")
	if !ok {
		t.Error("Expected to find plugin by command 'tp'")
	}

	if retrieved.Name() != "test-plugin" {
		t.Errorf("Expected plugin name 'test-plugin', got '%s'", retrieved.Name())
	}

	_, ok = registry.GetByCommand("non-existent")
	if ok {
		t.Error("Expected not to find plugin by non-existent command")
	}
}

func TestRegistryCommandCollision(t *testing.T) {
	registry := NewRegistry()

	plugin1 := &mockPlugin{
		name:        "plugin1",
		description: "First plugin",
		commands:    []string{"shared"},
	}

	plugin2 := &mockPlugin{
		name:        "plugin2",
		description: "Second plugin",
		commands:    []string{"shared"},
	}

	registry.Register(plugin1)
	registry.Register(plugin2)

	retrieved, ok := registry.GetByCommand("shared")
	if !ok {
		t.Error("Expected to find plugin by command 'shared'")
	}

	if retrieved.Name() != "plugin2" {
		t.Errorf("Expected command 'shared' to be overwritten by 'plugin2', got '%s'", retrieved.Name())
	}
}

func TestRegistryCommandSuggestions(t *testing.T) {
	registry := NewRegistry()

	plugin1 := &mockPlugin{
		name:        "plugin1",
		description: "First plugin",
		commands:    []string{"cmd1", "cmd2"},
	}

	plugin2 := &mockPlugin{
		name:        "plugin2",
		description: "Second plugin",
		commands:    []string{"cmd3"},
	}

	registry.Register(plugin1)
	registry.Register(plugin2)

	suggestions := registry.CommandSuggestions()

	if len(suggestions) != 3 {
		t.Errorf("Expected 3 command suggestions, got %d", len(suggestions))
	}

	commandSet := make(map[string]bool)
	for _, cmd := range suggestions {
		commandSet[cmd] = true
	}

	expectedCommands := []string{"cmd1", "cmd2", "cmd3"}
	for _, cmd := range expectedCommands {
		if !commandSet[cmd] {
			t.Errorf("Expected command '%s' in suggestions", cmd)
		}
	}
}

func TestRegistryList(t *testing.T) {
	registry := NewRegistry()

	plugin1 := &mockPlugin{name: "plugin1", description: "First", commands: []string{"cmd1"}}
	plugin2 := &mockPlugin{name: "plugin2", description: "Second", commands: []string{"cmd2"}}
	plugin3 := &mockPlugin{name: "plugin3", description: "Third", commands: []string{"cmd3"}}

	registry.Register(plugin1)
	registry.Register(plugin2)
	registry.Register(plugin3)

	plugins := registry.List()

	if len(plugins) != 3 {
		t.Errorf("Expected 3 plugins, got %d", len(plugins))
	}

	if plugins[0].Name() != "plugin1" {
		t.Errorf("Expected first plugin to be 'plugin1', got '%s'", plugins[0].Name())
	}

	if plugins[1].Name() != "plugin2" {
		t.Errorf("Expected second plugin to be 'plugin2', got '%s'", plugins[1].Name())
	}

	if plugins[2].Name() != "plugin3" {
		t.Errorf("Expected third plugin to be 'plugin3', got '%s'", plugins[2].Name())
	}
}
