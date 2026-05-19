package jira

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// NodeMetadata stores metadata on action component nodes.
type NodeMetadata struct {
	Project        *Project        `json:"project,omitempty"`
	IssueType      string          `json:"issueType,omitempty"`
	Status         string          `json:"status,omitempty"`
	WorkflowName   string          `json:"workflowName,omitempty"`
	WorkflowScheme *WorkflowScheme `json:"workflowScheme,omitempty"`
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

// applyStatus moves an issue to the requested status. It looks up available
// transitions from the issue's current state and executes the one whose target
// status name matches. Returns an error if no such transition exists.
func applyStatus(client *Client, issueKey, status string) error {
	return applyStatusWithOptions(client, issueKey, status, DoTransitionOptions{})
}

func applyStatusWithOptions(client *Client, issueKey, status string, opts DoTransitionOptions) error {
	transitions, err := client.GetIssueTransitions(issueKey)
	if err != nil {
		return fmt.Errorf("failed to fetch transitions: %v", err)
	}

	for _, t := range transitions {
		if strings.EqualFold(t.To.Name, status) {
			return client.DoTransitionWithOptions(issueKey, t.ID, opts)
		}
	}

	available := make([]string, 0, len(transitions))
	for _, t := range transitions {
		available = append(available, t.To.Name)
	}
	return fmt.Errorf("no transition available to status %q (available: %v)", status, available)
}
