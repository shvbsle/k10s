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
	Logo            string
	PaginationStyle PaginationStyle
}

// Load reads the k10s configuration from ~/.k10s.conf. If the file doesn't
// exist or cannot be read, it returns a Config with default values.
func Load() (*Config, error) {
	cfg := &Config{
		PageSize:        DefaultPageSize,
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
		case "pagination_style":
			switch value {
			case "bubbles":
				cfg.PaginationStyle = PaginationStyleBubbles
			case "verbose":
				cfg.PaginationStyle = PaginationStyleVerbose
			}
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

# Pagination style: "bubbles" (dots) or "verbose" (text like "Page 1/10")
# Default: bubbles
pagination_style=bubbles

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
