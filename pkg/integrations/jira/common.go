package jira

import (
	"encoding/json"
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// NodeMetadata stores metadata on action component nodes.
type NodeMetadata struct {
	Project   *Project `json:"project,omitempty"`
	IssueType string   `json:"issueType,omitempty"`
	Status    string   `json:"status,omitempty"`
}

func requireProject(httpCtx core.HTTPContext, integration core.IntegrationContext, projectKey string) (*Project, error) {
	if httpCtx != nil {
		client, err := NewClient(httpCtx, integration)
		if err == nil {
			projects, err := client.ListProjects()
			if err == nil {
				return findProject(projects, projectKey)
			}
		}
	}

	return requireProjectFromMetadata(integration, projectKey)
}

func requireProjectFromMetadata(integration core.IntegrationContext, projectKey string) (*Project, error) {
	metadata := Metadata{}
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	return findProject(metadata.Projects, projectKey)
}

func findProject(projects []Project, projectKey string) (*Project, error) {
	for _, project := range projects {
		if project.Key == projectKey {
			p := project
			return &p, nil
		}
	}

	return nil, fmt.Errorf("project %s not found", projectKey)
}

// CreateIncidentNodeMetadata is stored on create-incident nodes at setup for canvas labels and field mapping.
type CreateIncidentNodeMetadata struct {
	ServiceDeskName string `json:"serviceDeskName,omitempty"`
	RequestTypeName string `json:"requestTypeName,omitempty"`
	ImpactFieldID   string `json:"impactFieldId,omitempty"`
	UrgencyFieldID  string `json:"urgencyFieldId,omitempty"`
}

// OpsAlertPickerMetadata summarizes the Ops alert referenced on picker-driven components.
type OpsAlertPickerMetadata struct {
	AlertLabel string `json:"alertLabel,omitempty"`
}

// UpdateAlertNodeMetadata summarizes configured update operations for workflow cards.
type UpdateAlertNodeMetadata struct {
	AlertLabel      string   `json:"alertLabel,omitempty"`
	UpdateSummaries []string `json:"updateSummaries,omitempty"`
}

func cloudIDFromIntegration(integration core.IntegrationContext) (string, error) {
	meta := Metadata{}
	if err := mapstructure.Decode(integration.GetMetadata(), &meta); err != nil {
		return "", fmt.Errorf("decode integration metadata: %w", err)
	}
	if meta.CloudID == "" {
		return "", fmt.Errorf("integration is missing cloud id; re-sync the Jira integration after upgrading SuperPlane")
	}
	return meta.CloudID, nil
}

// ConfigurationAsSliceMap returns slice-style configuration as map[string]any if possible.
func ConfigurationAsSliceMap(cfg any) map[string]any {
	if cfg == nil {
		return map[string]any{}
	}
	if m, ok := cfg.(map[string]any); ok {
		return m
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return map[string]any{}
	}
	if out == nil {
		return map[string]any{}
	}
	return out
}
