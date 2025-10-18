# k10s (kittens)
🙀 A modern, scalable TUI for Kubernetes, tuned for AI/ML hyperscaler clusters with 100k+ nodes.

Built with Go + Bubble Tea.

## Why k10s?

Traditional Kubernetes CLI tools like `kubectl` and `k9s` don't scale well for massive AI hyperscaler clusters:

- **kubectl**: No pagination by default, requires verbose commands for large-scale operations
- **k9s**: Lacks pagination, making it impossible to navigate through millions of pods efficiently

**k10s** solves these problems with:
- ✅ Built-in pagination for all resource types
- ✅ Vim-like keybindings for fast navigation
- ✅ Optimized for clusters with 100k+ nodes
- ✅ Beautiful, minimal TUI interface
- ✅ Configurable page sizes and UI elements

## Features

- **Paginated Tables**: Browse pods, nodes, and namespaces with configurable page sizes
- **Vim Keybindings**: Navigate efficiently with `j/k`, `h/l`, `g/G`, and command mode with `:`
- **Command Mode**: Type `:` to enter command mode, then use commands like `pods`, `nodes`, `ns`, or `quit`
- **ASCII Logo**: Customizable cat logo displayed in the TUI
- **Configuration**: Store preferences in `~/.k10s.conf`
- **Fast & Lightweight**: Built in Go with minimal dependencies

## Installation

### From Source

```bash
git clone https://github.com/shvbsle/k10s.git
cd k10s
make build
```

The binary will be available at `bin/k10s`.

### Running

```bash
# Run directly
bin/k10s

# Or use make
make run
```

## Usage

### Keybindings

#### Normal Mode
- `j` or `↓`: Move down in the table
- `k` or `↑`: Move up in the table
- `h` or `←` or `PgUp`: Previous page
- `l` or `→` or `PgDown`: Next page
- `g`: Jump to top of table
- `G`: Jump to bottom of table
- `:`: Enter command mode
- `q`: Quit k10s

#### Command Mode
- Type a command and press `Enter` to execute
- Press `Esc` to cancel and return to normal mode

### Commands

When in command mode (press `:`), you can use:

- `pods` or `po`: Show all pods across all namespaces
- `nodes` or `no`: Show all nodes in the cluster
- `namespaces` or `ns`: Show all namespaces
- `quit` or `q`: Exit k10s

## Configuration

k10s reads configuration from `~/.k10s.conf`. On first run, a default config file is created automatically.

### Example Configuration

```conf
# k10s configuration file
# Number of items per page in table views
page_size=20

# ASCII logo (between logo_start and logo_end)
logo_start
 /\_/\
( o.o )
 > Y <
logo_end
```

### Configuration Options

- `page_size`: Number of rows to display per page (default: 20)
- `logo_start`/`logo_end`: Custom ASCII art logo to display

## Architecture

```
k10s/
├── cmd/k10s/           # Main application entry point
│   └── main.go
├── internal/
│   ├── config/         # Configuration handling
│   │   └── config.go
│   ├── k8s/            # Kubernetes client wrapper
│   │   ├── client.go
│   │   └── utils.go
│   └── tui/            # Terminal UI components
│       └── model.go    # Bubble Tea model
├── bin/                # Compiled binaries
├── Makefile
├── go.mod
└── README.md
```

## Technology Stack

- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)**: TUI framework
- **[Bubbles](https://github.com/charmbracelet/bubbles)**: TUI components (table, paginator, textinput)
- **[Lip Gloss](https://github.com/charmbracelet/lipgloss)**: Terminal styling
- **[client-go](https://github.com/kubernetes/client-go)**: Official Kubernetes Go client

## Development

### Prerequisites

- Go 1.24 or later
- Access to a Kubernetes cluster (via `~/.kube/config` or in-cluster config)

### Building

```bash
make build
```

### Running

```bash
make run
```

### Testing

```bash
make test
```

### Code Formatting

```bash
make fmt
```

## Roadmap

Future features planned for k10s:

- [ ] Search/filter functionality
- [ ] Support for more resource types (deployments, services, etc.)
- [ ] Resource details view
- [ ] Log streaming
- [ ] Multi-cluster support
- [ ] Custom column configuration
- [ ] Export data to CSV/JSON
- [ ] Real-time resource updates
- [ ] Resource editing capabilities
- [ ] Custom themes and color schemes

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see LICENSE file for details

## Credits

Built with love for the Kubernetes community, especially those running massive AI/ML infrastructure.

Powered by:
- [Charm](https://charm.sh/) - For the amazing TUI libraries
- [Kubernetes](https://kubernetes.io/) - The cloud-native platform

---

**k10s** - Because your cluster deserves better tools. 🐱
