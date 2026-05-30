package plugin

import (
	"sync"
)

type Manifest struct {
	Name        string           `json:"name"`
	Label       string           `json:"label"`
	Icon        string           `json:"icon"`
	Description string           `json:"description"`
	Actions     []ActionManifest `json:"actions"`
}

type ActionManifest struct {
	Name        string          `json:"name"`
	Label       string          `json:"label"`
	Description string          `json:"description"`
	Fields      []FieldManifest `json:"fields"`
}

type FieldManifest struct {
	Name        string           `json:"name"`
	Label       string           `json:"label"`
	Type        string           `json:"type"`
	Description string           `json:"description"`
	Required    bool             `json:"required"`
	Default     any              `json:"default,omitempty"`
	Options     []OptionManifest `json:"options,omitempty"`
}

type OptionManifest struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

var (
	cachedManifest   *Manifest
	cachedManifestMu sync.RWMutex
)

func getCachedManifest() *Manifest {
	cachedManifestMu.RLock()
	defer cachedManifestMu.RUnlock()
	return cachedManifest
}

func setCachedManifest(m *Manifest) {
	cachedManifestMu.Lock()
	defer cachedManifestMu.Unlock()
	cachedManifest = m
}
