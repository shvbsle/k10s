package plugins

import (
	"sync"

	"github.com/shvbsle/k10s/internal/log"
)

// Plugin represents an extension that adds functionality to k10s.
// Plugins are registered in the Registry and invoked via commands.
// k10s automatically returns to the main TUI after the plugin exits.
type Plugin interface {
	// Name returns the unique identifier (kebab-case recommended).
	Name() string

	// Description is shown in help text and plugin listings.
	Description() string

	// Commands returns aliases that trigger this plugin (e.g., ["play", "game", "kitten"]).
	Commands() []string

	// Launch executes the plugin. k10s returns to the main TUI on exit.
	Launch() error
}

type Registry struct {
	mu             sync.RWMutex
	plugins        map[string]Plugin
	commandMap     map[string]Plugin
	orderedPlugins []Plugin
}

func NewRegistry() *Registry {
	return &Registry{
		plugins:        make(map[string]Plugin),
		commandMap:     make(map[string]Plugin),
		orderedPlugins: make([]Plugin, 0),
	}
}

func (r *Registry) Register(p Plugin) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[p.Name()]; exists {
		log.G().Warn("plugin already registered", "plugin", p.Name())
	}

	r.plugins[p.Name()] = p
	r.orderedPlugins = append(r.orderedPlugins, p)

	for _, cmd := range p.Commands() {
		if existingPlugin, exists := r.commandMap[cmd]; exists {
			log.G().Warn("command collision",
				"command", cmd,
				"existing_plugin", existingPlugin.Name(),
				"new_plugin", p.Name())
		}
		r.commandMap[cmd] = p
	}
}

func (r *Registry) Get(name string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.plugins[name]
	return p, ok
}

func (r *Registry) GetByCommand(cmd string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.commandMap[cmd]
	return p, ok
}

func (r *Registry) List() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.orderedPlugins
}

func (r *Registry) CommandSuggestions() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	suggestions := make([]string, 0, len(r.commandMap))
	for cmd := range r.commandMap {
		suggestions = append(suggestions, cmd)
	}
	return suggestions
}
