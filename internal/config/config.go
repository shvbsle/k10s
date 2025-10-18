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
	DefaultPageSize = 20
	DefaultLogo     = ` /\_/\
( o.o )
 > Y <`
)

type Config struct {
	PageSize int
	Logo     string
}

func Load() (*Config, error) {
	cfg := &Config{
		PageSize: DefaultPageSize,
		Logo:     DefaultLogo,
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
	defer file.Close()

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
		}
	}

	return cfg, nil
}

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
