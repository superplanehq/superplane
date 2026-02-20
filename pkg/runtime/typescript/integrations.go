package typescript

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
)

const IntegrationsDirEnv = "TYPESCRIPT_INTEGRATIONS_DIR"

type IntegrationManifest struct {
	Name          string                `json:"name"`
	Label         string                `json:"label"`
	Icon          string                `json:"icon"`
	Description   string                `json:"description"`
	Instructions  string                `json:"instructions"`
	Configuration []configuration.Field  `json:"configuration"`
	Components    []IntegrationComponentRef `json:"components"`
	Triggers      []IntegrationTriggerRef `json:"triggers"`
}

type TriggerManifest struct {
	Name          string                `json:"name"`
	Label         string                `json:"label"`
	Description   string                `json:"description"`
	Documentation string                `json:"documentation"`
	Icon          string                `json:"icon"`
	Color         string                `json:"color"`
	Configuration []configuration.Field `json:"configuration"`
	ExampleData   map[string]any        `json:"example_data"`
}

type IntegrationComponentRef struct {
	Name      string `json:"name"`
	Directory string `json:"directory"`
}

type IntegrationTriggerRef struct {
	Name      string `json:"name"`
	Directory string `json:"directory"`
}

type IntegrationComponentDefinition struct {
	Name       string
	Entrypoint string
	Manifest   ComponentManifest
}

type IntegrationTriggerDefinition struct {
	Name       string
	Entrypoint string
	Manifest   TriggerManifest
}

type IntegrationDefinition struct {
	Name       string
	Directory  string
	Entrypoint string
	Manifest   IntegrationManifest
	Components []IntegrationComponentDefinition
	Triggers   []IntegrationTriggerDefinition
}

func DiscoverIntegrationsFromDir(baseDir string) ([]IntegrationDefinition, error) {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		return []IntegrationDefinition{}, nil
	}

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", IntegrationsDirEnv, err)
	}

	definitions := make([]IntegrationDefinition, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		integrationDir := filepath.Join(baseDir, entry.Name())
		entrypoint := filepath.Join(integrationDir, "index.ts")
		manifestPath := filepath.Join(integrationDir, "manifest.json")

		if _, err := os.Stat(entrypoint); err != nil {
			return nil, fmt.Errorf("integration %s missing index.ts", entry.Name())
		}

		manifestData, err := os.ReadFile(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("integration %s missing or unreadable manifest.json: %w", entry.Name(), err)
		}

		var manifest IntegrationManifest
		if err := json.Unmarshal(manifestData, &manifest); err != nil {
			return nil, fmt.Errorf("integration %s has invalid manifest.json: %w", entry.Name(), err)
		}

		name := strings.TrimSpace(manifest.Name)
		if name == "" {
			name = entry.Name()
		}

		manifest.Name = name
		if manifest.Label == "" {
			manifest.Label = name
		}
		if manifest.Icon == "" {
			manifest.Icon = "plug"
		}
		if manifest.Configuration == nil {
			manifest.Configuration = []configuration.Field{}
		}

		definition := IntegrationDefinition{
			Name:       name,
			Directory:  integrationDir,
			Entrypoint: entrypoint,
			Manifest:   manifest,
			Components: []IntegrationComponentDefinition{},
			Triggers:   []IntegrationTriggerDefinition{},
		}

		for _, componentRef := range manifest.Components {
			componentDir, err := resolveNestedDirectory(integrationDir, componentRef.Directory)
			if err != nil {
				return nil, fmt.Errorf("integration %s component %s: %w", name, componentRef.Name, err)
			}

			componentDef, err := DiscoverComponentFromDirectory(componentDir, componentRef.Name)
			if err != nil {
				return nil, fmt.Errorf("integration %s component directory %s: %w", name, componentRef.Directory, err)
			}

			componentName := normalizeNestedName(name, firstNonEmpty(componentRef.Name, componentDef.Manifest.Name))
			if componentName == "" {
				return nil, fmt.Errorf("integration %s has component with empty name", name)
			}

			componentDef.Manifest.Name = componentName
			if componentDef.Manifest.Label == "" {
				componentDef.Manifest.Label = componentName
			}
			if componentDef.Manifest.Icon == "" {
				componentDef.Manifest.Icon = manifest.Icon
			}
			if componentDef.Manifest.Color == "" {
				componentDef.Manifest.Color = "orange"
			}

			definition.Components = append(definition.Components, IntegrationComponentDefinition{
				Name:       componentName,
				Entrypoint: componentDef.Entrypoint,
				Manifest:   componentDef.Manifest,
			})
		}

		for _, triggerRef := range manifest.Triggers {
			triggerDir, err := resolveNestedDirectory(integrationDir, triggerRef.Directory)
			if err != nil {
				return nil, fmt.Errorf("integration %s trigger %s: %w", name, triggerRef.Name, err)
			}

			triggerManifest, triggerEntrypoint, err := discoverTriggerFromDirectory(triggerDir, triggerRef.Name)
			if err != nil {
				return nil, fmt.Errorf("integration %s trigger directory %s: %w", name, triggerRef.Directory, err)
			}

			triggerName := normalizeNestedName(name, firstNonEmpty(triggerRef.Name, triggerManifest.Name))
			if triggerName == "" {
				return nil, fmt.Errorf("integration %s has trigger with empty name", name)
			}

			triggerManifest.Name = triggerName
			if triggerManifest.Label == "" {
				triggerManifest.Label = triggerName
			}
			if triggerManifest.Icon == "" {
				triggerManifest.Icon = manifest.Icon
			}
			if triggerManifest.Color == "" {
				triggerManifest.Color = "orange"
			}
			if triggerManifest.Configuration == nil {
				triggerManifest.Configuration = []configuration.Field{}
			}
			if triggerManifest.ExampleData == nil {
				triggerManifest.ExampleData = map[string]any{}
			}

			definition.Triggers = append(definition.Triggers, IntegrationTriggerDefinition{
				Name:       triggerName,
				Entrypoint: triggerEntrypoint,
				Manifest:   triggerManifest,
			})
		}

		definitions = append(definitions, definition)
	}

	sort.Slice(definitions, func(i, j int) bool {
		return definitions[i].Name < definitions[j].Name
	})

	return definitions, nil
}

func DiscoverIntegrationsFromEnv() ([]IntegrationDefinition, error) {
	return DiscoverIntegrationsFromDir(os.Getenv(IntegrationsDirEnv))
}

func normalizeNestedName(integrationName, nestedName string) string {
	nestedName = strings.TrimSpace(nestedName)
	if nestedName == "" {
		return ""
	}
	if strings.HasPrefix(nestedName, integrationName+".") {
		return nestedName
	}
	if strings.Contains(nestedName, ".") {
		return nestedName
	}
	return integrationName + "." + nestedName
}

func resolveNestedDirectory(integrationDir string, relativeDir string) (string, error) {
	dir := strings.TrimSpace(relativeDir)
	if dir == "" {
		return "", fmt.Errorf("missing directory")
	}

	if filepath.IsAbs(dir) {
		return dir, nil
	}

	return filepath.Join(integrationDir, dir), nil
}

func discoverTriggerFromDirectory(dir string, fallbackName string) (TriggerManifest, string, error) {
	entrypoint := filepath.Join(dir, "index.ts")
	manifestPath := filepath.Join(dir, "manifest.json")

	if _, err := os.Stat(entrypoint); err != nil {
		return TriggerManifest{}, "", fmt.Errorf("missing index.ts")
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return TriggerManifest{}, "", fmt.Errorf("missing or unreadable manifest.json: %w", err)
	}

	var manifest TriggerManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return TriggerManifest{}, "", fmt.Errorf("invalid manifest.json: %w", err)
	}

	if strings.TrimSpace(manifest.Name) == "" {
		manifest.Name = strings.TrimSpace(fallbackName)
	}
	if manifest.Configuration == nil {
		manifest.Configuration = []configuration.Field{}
	}
	if manifest.ExampleData == nil {
		manifest.ExampleData = map[string]any{}
	}

	return manifest, entrypoint, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}

	return ""
}
