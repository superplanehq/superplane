package gitserver

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Registry maps slugs to canvas IDs.
type Registry struct {
	mu       sync.RWMutex
	mappings map[string]*SlugToCanvasMapping
	reposDir string
}

// NewRegistry scans the repos directory to build the slug -> canvas mapping.
func NewRegistry(reposDir string) *Registry {
	r := &Registry{
		mappings: make(map[string]*SlugToCanvasMapping),
		reposDir: reposDir,
	}
	r.scan()
	return r
}

// Register adds or updates a slug mapping.
func (r *Registry) Register(slug string, mapping *SlugToCanvasMapping) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mappings[slug] = mapping
}

// Resolve returns the canvas mapping for a slug.
func (r *Registry) Resolve(slug string, _ string) (*SlugToCanvasMapping, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	m, ok := r.mappings[slug]
	if !ok {
		return nil, fmt.Errorf("no canvas registered for slug %q", slug)
	}
	return m, nil
}

// scan reads .superplane.yaml from each bare repo to populate mappings.
func (r *Registry) scan() {
	entries, err := os.ReadDir(r.reposDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if len(name) < 5 || name[len(name)-4:] != ".git" {
			continue
		}

		slug := name[:len(name)-4]
		configPath := filepath.Join(r.reposDir, name, "superplane.yaml")

		data, err := os.ReadFile(configPath)
		if err != nil {
			// Try reading from the tree itself
			continue
		}

		var cfg struct {
			CanvasID string `yaml:"canvasId"`
			OrgID    string `yaml:"orgId"`
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			continue
		}

		if cfg.CanvasID != "" {
			r.mappings[slug] = &SlugToCanvasMapping{
				CanvasID: cfg.CanvasID,
				OrgID:    cfg.OrgID,
			}
		}
	}
}
