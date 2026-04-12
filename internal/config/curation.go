package config

import (
	"fmt"
	"os"
	"slices"

	"gopkg.in/yaml.v3"
)

// CatalogEntry is the per-project editorial metadata in curation.yaml.
type CatalogEntry struct {
	Category     string `yaml:"category"`
	Summary      string `yaml:"summary,omitempty"`
	WhyItMatters string `yaml:"why_it_matters,omitempty"`
	Status       string `yaml:"status,omitempty"` // active, maintained, complete, archived, experiment
	Priority     int    `yaml:"priority,omitempty"` // higher = appears first within category. 0 = default (sort by date).
}

// Curation represents the full curation.yaml file.
type Curation struct {
	Excluded []string                 `yaml:"excluded"`
	Todo     []string                 `yaml:"todo"`
	Catalog  map[string]*CatalogEntry `yaml:"catalog"`
}

// IsExcluded checks if a repo ID (org/name) is in the excluded list.
func (c *Curation) IsExcluded(id string) bool {
	return slices.Contains(c.Excluded, id)
}

// IsTodo checks if a repo ID (org/name) is in the todo list.
func (c *Curation) IsTodo(id string) bool {
	return slices.Contains(c.Todo, id)
}

// CatalogEntryFor returns the catalog entry for a repo ID, or nil if not cataloged.
func (c *Curation) CatalogEntryFor(id string) *CatalogEntry {
	if c.Catalog == nil {
		return nil
	}
	return c.Catalog[id]
}

// ComputeStatus determines the curation status for a given repo ID.
// Priority: catalog > excluded > todo > unreviewed.
func (c *Curation) ComputeStatus(id string) string {
	if c.CatalogEntryFor(id) != nil {
		return "included"
	}
	if c.IsExcluded(id) {
		return "excluded"
	}
	if c.IsTodo(id) {
		return "todo"
	}
	return "unreviewed"
}

// LoadCuration reads and parses curation.yaml.
func LoadCuration(path string) (*Curation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read curation: %w", err)
	}
	var cur Curation
	if err := yaml.Unmarshal(data, &cur); err != nil {
		return nil, fmt.Errorf("parse curation: %w", err)
	}
	if cur.Catalog == nil {
		cur.Catalog = make(map[string]*CatalogEntry)
	}
	return &cur, nil
}

// AddCatalogEntry appends a new catalog entry and writes back to disk.
func AddCatalogEntry(path, id, category, summary string) error {
	cur, err := LoadCuration(path)
	if err != nil {
		return err
	}

	if cur.CatalogEntryFor(id) != nil {
		return fmt.Errorf("%s is already in the catalog", id)
	}

	// Remove from excluded/todo if present.
	cur.Excluded = removeFromSlice(cur.Excluded, id)
	cur.Todo = removeFromSlice(cur.Todo, id)

	entry := &CatalogEntry{Category: category}
	if summary != "" {
		entry.Summary = summary
	}
	cur.Catalog[id] = entry

	return writeCuration(path, cur)
}

// AddExcluded appends a repo ID to the excluded list and writes back to disk.
func AddExcluded(path, id string) error {
	cur, err := LoadCuration(path)
	if err != nil {
		return err
	}

	if cur.IsExcluded(id) {
		return fmt.Errorf("%s is already excluded", id)
	}

	// Remove from todo if present. Remove from catalog if present.
	cur.Todo = removeFromSlice(cur.Todo, id)
	delete(cur.Catalog, id)

	cur.Excluded = append(cur.Excluded, id)
	return writeCuration(path, cur)
}

func writeCuration(path string, cur *Curation) error {
	data, err := yaml.Marshal(cur)
	if err != nil {
		return fmt.Errorf("marshal curation: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func removeFromSlice(s []string, v string) []string {
	out := make([]string, 0, len(s))
	for _, item := range s {
		if item != v {
			out = append(out, item)
		}
	}
	return out
}
