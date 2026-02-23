package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

var validNameRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]*(\.[a-z0-9][a-z0-9\-]*)*$`)

type PluginManifest struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Main    string `json:"main"`
	Engines struct {
		SuperPlane string `json:"superplane"`
	} `json:"engines"`
	SuperPlane SuperPlaneManifest `json:"superplane"`
}

type SuperPlaneManifest struct {
	ActivationEvents []string            `json:"activationEvents"`
	Contributes      ContributionPoints  `json:"contributes"`
	Integration      IntegrationManifest `json:"integration"`
}

type IntegrationManifest struct {
	Name              string                `json:"name"`
	Label             string                `json:"label"`
	Icon              string                `json:"icon"`
	Description       string                `json:"description"`
	Configuration     []configuration.Field `json:"configuration"`
	HasWebhookHandler bool                  `json:"hasWebhookHandler"`
}

type ContributionPoints struct {
	Components []ComponentContribution `json:"components"`
	Triggers   []TriggerContribution   `json:"triggers"`
}

type ComponentContribution struct {
	Name           string                  `json:"name"`
	Label          string                  `json:"label"`
	Description    string                  `json:"description"`
	Documentation  string                  `json:"documentation"`
	Icon           string                  `json:"icon"`
	Color          string                  `json:"color"`
	Configuration  []configuration.Field   `json:"configuration"`
	OutputChannels []OutputChannelManifest `json:"outputChannels"`
	ExampleOutput  map[string]any          `json:"exampleOutput"`
}

type TriggerContribution struct {
	Name          string                `json:"name"`
	Label         string                `json:"label"`
	Description   string                `json:"description"`
	Documentation string                `json:"documentation"`
	Icon          string                `json:"icon"`
	Color         string                `json:"color"`
	Configuration []configuration.Field `json:"configuration"`
	ExampleData   map[string]any        `json:"exampleData"`
}

type OutputChannelManifest struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

func (o OutputChannelManifest) ToCoreOutputChannel() core.OutputChannel {
	return core.OutputChannel{
		Name:        o.Name,
		Label:       o.Label,
		Description: o.Description,
	}
}

type PluginsJSON struct {
	Plugins []PluginRecord `json:"plugins"`
}

type PluginRecord struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	InstalledAt time.Time `json:"installedAt"`
}

func ParseManifest(pluginDir string) (*PluginManifest, error) {
	data, err := os.ReadFile(filepath.Join(pluginDir, "package.json"))
	if err != nil {
		return nil, fmt.Errorf("reading package.json: %w", err)
	}

	var manifest PluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing package.json: %w", err)
	}

	return &manifest, nil
}

func ValidateManifest(m *PluginManifest) error {
	if m.Name == "" {
		return fmt.Errorf("manifest missing required field: name")
	}
	if m.Version == "" {
		return fmt.Errorf("manifest missing required field: version")
	}
	if m.Engines.SuperPlane == "" {
		return fmt.Errorf("manifest missing required field: engines.superplane")
	}
	if len(m.SuperPlane.ActivationEvents) == 0 {
		return fmt.Errorf("manifest missing required field: superplane.activationEvents")
	}

	for _, c := range m.SuperPlane.Contributes.Components {
		if err := validateContributedName(c.Name); err != nil {
			return fmt.Errorf("component %q: %w", c.Name, err)
		}
		if c.Label == "" {
			return fmt.Errorf("component %q missing required field: label", c.Name)
		}
	}

	for _, t := range m.SuperPlane.Contributes.Triggers {
		if err := validateContributedName(t.Name); err != nil {
			return fmt.Errorf("trigger %q: %w", t.Name, err)
		}
		if t.Label == "" {
			return fmt.Errorf("trigger %q missing required field: label", t.Name)
		}
	}

	return nil
}

func validateContributedName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if !validNameRegexp.MatchString(name) {
		return fmt.Errorf("name %q must be lowercase alphanumeric with hyphens and dots", name)
	}
	return nil
}

func ReadPluginsJSON(pluginsDir string) (*PluginsJSON, error) {
	data, err := os.ReadFile(filepath.Join(pluginsDir, "plugins.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return &PluginsJSON{}, nil
		}
		return nil, fmt.Errorf("reading plugins.json: %w", err)
	}

	var pj PluginsJSON
	if err := json.Unmarshal(data, &pj); err != nil {
		return nil, fmt.Errorf("parsing plugins.json: %w", err)
	}

	return &pj, nil
}

func WritePluginsJSON(pluginsDir string, pj *PluginsJSON) error {
	data, err := json.MarshalIndent(pj, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling plugins.json: %w", err)
	}

	return os.WriteFile(filepath.Join(pluginsDir, "plugins.json"), data, 0644)
}
