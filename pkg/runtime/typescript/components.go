package typescript

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const ComponentsDirEnv = "TYPESCRIPT_COMPONENTS_DIR"

type ComponentManifest struct {
	Name           string                `json:"name"`
	Label          string                `json:"label"`
	Description    string                `json:"description"`
	Documentation  string                `json:"documentation"`
	File           string                `json:"file,omitempty"`
	Icon           string                `json:"icon"`
	Color          string                `json:"color"`
	Configuration  []configuration.Field `json:"configuration"`
	OutputChannels []core.OutputChannel  `json:"output_channels"`
	ExampleOutput  map[string]any        `json:"example_output"`
}

type ComponentDefinition struct {
	Name       string
	Directory  string
	Entrypoint string
	Manifest   ComponentManifest
}

func DiscoverComponentsFromDir(baseDir string) ([]ComponentDefinition, error) {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		return []ComponentDefinition{}, nil
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", ComponentsDirEnv, err)
	}

	definitions := make([]ComponentDefinition, 0, len(entries))

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		componentDir := filepath.Join(baseDir, entry.Name())
		definition, err := DiscoverComponentFromDirectory(componentDir, entry.Name())
		if err != nil {
			return nil, fmt.Errorf("component %s: %w", entry.Name(), err)
		}
		definitions = append(definitions, definition)
	}

	sort.Slice(definitions, func(i, j int) bool {
		return definitions[i].Name < definitions[j].Name
	})

	return definitions, nil
}

func DiscoverComponentsFromEnv() ([]ComponentDefinition, error) {
	return DiscoverComponentsFromDir(os.Getenv(ComponentsDirEnv))
}

func DiscoverComponentFromDirectory(componentDir string, fallbackName string) (ComponentDefinition, error) {
	entrypoint := filepath.Join(componentDir, "index.ts")
	manifestPath := filepath.Join(componentDir, "manifest.json")

	if _, err := os.Stat(entrypoint); err != nil {
		return ComponentDefinition{}, fmt.Errorf("missing index.ts")
	}

	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return ComponentDefinition{}, fmt.Errorf("missing or unreadable manifest.json: %w", err)
	}

	var manifest ComponentManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return ComponentDefinition{}, fmt.Errorf("invalid manifest.json: %w", err)
	}

	name := strings.TrimSpace(manifest.Name)
	if name == "" {
		name = fallbackName
	}

	manifest.Name = name
	if manifest.Label == "" {
		manifest.Label = name
	}
	if manifest.Icon == "" {
		manifest.Icon = "file-code-2"
	}
	if manifest.Color == "" {
		manifest.Color = "orange"
	}
	if manifest.OutputChannels == nil {
		manifest.OutputChannels = []core.OutputChannel{core.DefaultOutputChannel}
	}
	if manifest.Configuration == nil {
		manifest.Configuration = []configuration.Field{}
	}
	if manifest.ExampleOutput == nil {
		manifest.ExampleOutput = map[string]any{}
	}

	return ComponentDefinition{
		Name:       name,
		Directory:  componentDir,
		Entrypoint: entrypoint,
		Manifest:   manifest,
	}, nil
}
