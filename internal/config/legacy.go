package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const configFileNameLegacy = ".k10s.conf"

// Load reads the k10s configuration from ~/.k10s.conf. If the file doesn't
// exist or cannot be read, it returns a Config with default values.
func loadConfigLegacy() (*Config, error) {
	cfg := &Config{
		MaxPageSize:     DefaultAutoPageSize,
		LogTailLines:    DefaultLogTailLines,
		Logo:            DefaultLogo,
		PaginationStyle: PaginationStyleBubbles,
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg, nil // Return defaults if can't get home dir
	}

	configPath := filepath.Join(home, configFileNameLegacy)
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
		// TODO: rename to max_page_size
		case "page_size":
			if strings.ToLower(value) == "auto" {
				cfg.MaxPageSize = DefaultAutoPageSize
			} else if size, err := strconv.Atoi(value); err == nil && size > 0 {
				cfg.MaxPageSize = size
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
