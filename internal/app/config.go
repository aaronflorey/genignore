package app

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

const configRelativePath = ".config/genignore/config.toml"

var userHomeDir = os.UserHomeDir

type Config struct {
	Defaults ConfigDefaults `mapstructure:"defaults" toml:"defaults"`
	Runtime  ConfigRuntime  `mapstructure:"runtime" toml:"runtime"`
}

type ConfigDefaults struct {
	Providers   []string `mapstructure:"providers" toml:"providers"`
	IgnoreRules []string `mapstructure:"ignore_rules" toml:"ignore_rules"`
}

type ConfigRuntime struct {
	Offline bool `mapstructure:"offline" toml:"offline"`
}

func LoadConfig() (Config, error) {
	home, err := userHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("resolve config home: %w", err)
	}

	path := filepath.Join(home, configRelativePath)
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("stat config file %s: %w", path, err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config file %s: %w", path, err)
	}

	var cfg Config
	decoder := toml.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("invalid config file %s: %w", path, err)
	}

	return cfg, nil
}
