package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// DefaultPageSize is the default number of items to display per page.
	DefaultPageSize = 20
	// DefaultLogTailLines is the default number of log lines to fetch.
	DefaultLogTailLines = 100
	// DefaultLogo is the default ASCII art logo displayed in the TUI header.
	DefaultLogo = ` /\_/\
( o.o )
 > Y <`
	// DefaultPaginationStyle is the default pagination display style.
	DefaultPaginationStyle = "bubbles"
)

// PaginationStyle represents the style of pagination display
type PaginationStyle string

const (
	// PaginationStyleBubbles uses the bubbles paginator component (dots)
	PaginationStyleBubbles PaginationStyle = "bubbles"
	// PaginationStyleVerbose uses text like "Page 1/10"
	PaginationStyleVerbose PaginationStyle = "verbose"
)

// Config holds the user configuration for k10s, including display preferences
// like page size and the ASCII logo to show in the header.
type Config struct {
	PageSize        int
	LogTailLines    int
	Logo            string
	PaginationStyle PaginationStyle
	LogFilePath     string // Custom log file path (empty means use XDG default)
}

// Load reads the k10s configuration from ~/.k10s.conf. If the file doesn't
// exist or cannot be read, it returns a Config with default values.
func Load() (*Config, error) {
	cfg := &Config{
		PageSize:        DefaultPageSize,
		LogTailLines:    DefaultLogTailLines,
		Logo:            DefaultLogo,
		PaginationStyle: PaginationStyleBubbles,
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg, nil // Return defaults if can't get home dir
	}

	configPath := filepath.Join(home, ".k10s.conf")
	file, err := os.Open(configPath)
	if err != nil {
		// Config file doesn't exist, return defaults
		return cfg, nil
	}
	defer func() {
		_ = file.Close() // Ignore close error on read-only file
	}()

	scanner := bufio.NewScanner(file)
	var logoLines []string
	inLogo := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if strings.HasPrefix(trimmed, "logo_start") {
			inLogo = true
			logoLines = []string{}
			continue
		}

		if strings.HasPrefix(trimmed, "logo_end") {
			inLogo = false
			cfg.Logo = strings.Join(logoLines, "\n")
			continue
		}

		if inLogo {
			logoLines = append(logoLines, line)
			continue
		}

		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "page_size":
			if size, err := strconv.Atoi(value); err == nil && size > 0 {
				cfg.PageSize = size
			}
		case "log_tail_lines":
			if lines, err := strconv.Atoi(value); err == nil && lines > 0 {
				cfg.LogTailLines = lines
			}
		case "pagination_style":
			switch value {
			case "bubbles":
				cfg.PaginationStyle = PaginationStyleBubbles
			case "verbose":
				cfg.PaginationStyle = PaginationStyleVerbose
			}
		case "k10s_log_path":
			// Accept the value as-is, will be validated in setupLogging
			cfg.LogFilePath = value
		}
	}

	return cfg, nil
}

// CreateDefaultConfig creates a default configuration file at ~/.k10s.conf
// if it doesn't already exist. It does not overwrite existing configurations.
func CreateDefaultConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(home, ".k10s.conf")

	// Don't overwrite existing config
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	defaultConfig := `# k10s configuration file
# Number of items per page in table views
page_size=20

# Number of log lines to fetch when viewing container logs
log_tail_lines=100

# Pagination style: "bubbles" (dots) or "verbose" (text like "Page 1/10")
# Default: bubbles
pagination_style=bubbles

# Log file path for k10s internal logs
# If commented out or empty, logs will be stored in the default XDG state directory:
#   - macOS: ~/Library/Application Support/k10s/k10s.log
#   - Linux: ~/.local/state/k10s/k10s.log
# You can override this with a custom path (supports ~ for home directory)
# Example: k10s_log_path=/var/log/k10s.log
# k10s_log_path=

# ASCII logo (between logo_start and logo_end)
logo_start
 /\_/\
( o.o )
 > Y <
logo_end
`

	return os.WriteFile(configPath, []byte(defaultConfig), 0644)
}

func (c *Config) String() string {
	return fmt.Sprintf("PageSize: %d\nLogo:\n%s", c.PageSize, c.Logo)
}

func GetPluginDataDir(pluginName string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get user home directory: %w", err)
	}

	pluginDir := filepath.Join(homeDir, ".k10s", "plugins", pluginName)
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return "", fmt.Errorf("could not create plugin data directory: %w", err)
	}

	return pluginDir, nil
}
