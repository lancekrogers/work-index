package config

import (
	"fmt"
	"os"
	"slices"

	"gopkg.in/yaml.v3"
)

// SyncConfig represents sync-config.yaml.
type SyncConfig struct {
	Owners           []string `yaml:"owners"`
	IncludeForksFrom []string `yaml:"include_forks_from"`
	ExcludeForksFrom []string `yaml:"exclude_forks_from"`
}

// AllowsForks returns true if the given owner's forks should be included.
func (c *SyncConfig) AllowsForks(owner string) bool {
	return slices.Contains(c.IncludeForksFrom, owner)
}

// IsExcludedParent returns true if forks from this upstream owner should be dropped.
func (c *SyncConfig) IsExcludedParent(parentOwner string) bool {
	if parentOwner == "" {
		return false
	}
	return slices.Contains(c.ExcludeForksFrom, parentOwner)
}

// LoadSyncConfig reads and parses sync-config.yaml.
func LoadSyncConfig(path string) (*SyncConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read sync config: %w", err)
	}
	var cfg SyncConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse sync config: %w", err)
	}
	if len(cfg.Owners) == 0 {
		return nil, fmt.Errorf("sync config has no owners")
	}
	return &cfg, nil
}
