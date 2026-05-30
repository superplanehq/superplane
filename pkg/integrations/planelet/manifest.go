package planelet

import (
	"sync"
)

type Manifest struct {
	ID          string            `json:"id"`
	Label       string            `json:"label"`
	Icon        string            `json:"icon,omitempty"`
	IconURL     string            `json:"iconUrl,omitempty"`
	Description string            `json:"description,omitempty"`
	Actions     []ActionManifest  `json:"actions"`
	Triggers    []TriggerManifest `json:"triggers"`
}

type ActionManifest struct {
	ID          string              `json:"id"`
	Label       string              `json:"label"`
	Icon        string              `json:"icon,omitempty"`
	IconURL     string              `json:"iconUrl,omitempty"`
	Description string              `json:"description,omitempty"`
	Parameters  []ParameterManifest `json:"parameters"`
}

type TriggerManifest struct {
	ID          string              `json:"id"`
	Label       string              `json:"label"`
	Icon        string              `json:"icon,omitempty"`
	IconURL     string              `json:"iconUrl,omitempty"`
	Description string              `json:"description,omitempty"`
	Parameters  []ParameterManifest `json:"parameters"`
}

type ParameterManifest struct {
	ID          string           `json:"id"`
	Label       string           `json:"label"`
	Type        string           `json:"type"`
	Description string           `json:"description,omitempty"`
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
