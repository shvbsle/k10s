# k10s

🙀 A modern, pretty TUI for Kubernetes, tuned for AI Hyperscaler clusters. 

Built with Go + Bubble Tea.

[![asciicast](https://asciinema.org/a/ZWCHSzaX35zjdPpS5pEVmrad1.svg)](https://asciinema.org/a/ZWCHSzaX35zjdPpS5pEVmrad1)

## Features

- **Paginated Tables**: Browse pods, nodes, and namespaces with configurable page sizes
- **Vim Keybindings**: Navigate efficiently with `j/k`, `h/l`, `g/G`, and command mode with `:`
- **Command Mode**: Type `:` to enter command mode, then use commands like `pods`, `nodes`, `ns`, or `quit`
- **Customizable**: Configure page sizes and UI elements via `~/.k10s.conf`
- **Fast & Lightweight**: Built in Go with minimal dependencies

## Installation

### From Source

```bash
git clone https://github.com/shvbsle/k10s.git
cd k10s
make build
```

The binary will be available at `bin/k10s`.

### Homebrew (Coming Soon)

Once releases are published:

```bash
brew tap shvbsle/tap
brew install k10s
```

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

# Pagination style: "bubbles" (dots) or "verbose" (text like "Page 1/10")
# Default: bubbles
pagination_style=bubbles

# ASCII logo (between logo_start and logo_end)
logo_start
 /\_/\
( o.o )
 > Y <
logo_end
```

### Configuration Options

- `page_size`: Number of rows to display per page (default: 20)
- `pagination_style`: Pagination display style - `bubbles` for dot-based paginator or `verbose` for text like "Page 1/10" (default: bubbles)
- `logo_start`/`logo_end`: Custom ASCII art logo to display

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

### Linting

```bash
make lint
```

### Code Formatting

```bash
make fmt
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Releasing

See [RELEASING.md](RELEASING.md) for detailed instructions on creating releases.

## License

Apache 2.0 - see LICENSE file for details
