package config

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

const (
	configFileName = ".k10s.toml"
)

const (
	// DefaultAutoPageSize indicates that page size should use all available screen space.
	// This is the default behavior when no page_size is configured.
	DefaultAutoPageSize = 0
	// DefaultLogTailLines is the default number of log lines to fetch.
	DefaultLogTailLines = 100
	// DefaultLogo is the default ASCII art logo displayed in the TUI header.
	DefaultLogo = " /\\_/\\\n( o.o )\n> Y <"
	// DefaultPaginationStyle is the default pagination display style.
	DefaultPaginationStyle = PaginationStyleBubbles
	// DefaultLogFilePath is the default path for k10s logs.
	DefaultLogFilePath = "k10s.log"
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
	MaxPageSize     int             `toml:"page_size" comment:"Number of items per page in table views.\nSet to \"auto\" to use all available screen space, or set to a specific number."`
	LogTailLines    int             `toml:"log_tail_lines" comment:"Number of log lines to fetch when viewing container logs."`
	Logo            string          `toml:"logo_string,multiline" comment:"ASCII logo to display in the header."`
	LogFilePath     string          `toml:"k10s_log_path" comment:"Logging path for k10s. (eg. /var/log/k10s.log)"`
	PaginationStyle PaginationStyle `toml:"pagination_style"`
}

// Load reads the k10s configuration file from the default path. If the file
// doesn't exist or cannot be read, it returns a Config with default values.
func Load() (*Config, error) {
	c, err := loadConfig()
	if err == nil {
		return c, nil
	}

	// if we failed to load the latest config format, try to load the legacy
	// config.
	c, err = loadConfigLegacy()
	if err == nil {
		// try to backfill legacy data into the current config.
		return c, createIfNotExists(*c)
	}

	// if we can't load anything, then return defaults.
	c = &Config{
		MaxPageSize:     DefaultAutoPageSize,
		LogTailLines:    DefaultLogTailLines,
		Logo:            DefaultLogo,
		PaginationStyle: PaginationStyleBubbles,
		LogFilePath:     DefaultLogFilePath,
	}

	return c, createIfNotExists(*c)
}

func loadConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(filepath.Join(home, configFileName))
	if err != nil {
		return nil, err
	}
	defer file.Close() // nolint:errcheck

	var cfg Config
	return &cfg, toml.NewDecoder(file).Decode(&cfg)
}

// createIfNotExists creates a configuration file based on given config template
// if it doesn't already exist.
func createIfNotExists(config Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(home, configFileName)

	// Don't overwrite existing config
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	configData, err := toml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, configData, 0644)
}

func (c *Config) String() string {
	data, _ := toml.Marshal(c)
	return string(data)
}
