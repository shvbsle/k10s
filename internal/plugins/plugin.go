package plugins

import (
	"log"
	"sync"
)

type Plugin interface {
	Name() string
	Description() string
	Commands() []string
	Launch() (bool, error)
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

	if existing, exists := r.plugins[p.Name()]; exists {
		log.Printf("Warning: plugin '%s' already registered, overwriting with new instance", p.Name())
		_ = existing
	}

	r.plugins[p.Name()] = p
	r.orderedPlugins = append(r.orderedPlugins, p)

	for _, cmd := range p.Commands() {
		if existingPlugin, exists := r.commandMap[cmd]; exists {
			log.Printf("Warning: command '%s' already registered by plugin '%s', overwriting with plugin '%s'",
				cmd, existingPlugin.Name(), p.Name())
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
