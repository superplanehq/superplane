package snyk

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type OnNewIssueDetected struct{}

type OnNewIssueDetectedConfiguration struct {
	OrganizationID string `json:"organizationId" mapstructure:"organizationId"`
	ProjectID      string `json:"projectId" mapstructure:"projectId"` // Optional - if not specified, applies to all projects
	Severity       string `json:"severity" mapstructure:"severity"`   // Optional - filter by severity (low, medium, high, critical)
}

func (t *OnNewIssueDetected) Name() string {
	return "snyk.onNewIssueDetected"
}

func (t *OnNewIssueDetected) Label() string {
	return "On New Issue Detected"
}

func (t *OnNewIssueDetected) Description() string {
	return "Listen to Snyk for new security issues"
}

func (t *OnNewIssueDetected) Documentation() string {
	return `The On New Issue Detected trigger starts a workflow execution when Snyk detects new security issues.

## Use Cases

- **Security alerts**: Get notified immediately when new vulnerabilities are found
- **Ticket creation**: Automatically create tickets for new security issues
- **Compliance workflows**: Trigger compliance processes when issues are detected
- **Team notifications**: Notify security teams of new findings

## Configuration

- **Organization ID**: The Snyk organization to monitor
- **Project ID**: Optional project filter - if specified, only issues from this project will trigger (leave empty to listen to all projects)
- **Severity**: Optional severity filter - if specified, only issues of this severity or higher will trigger (low, medium, high, critical)

## Event Data

Each issue detection event includes:
- **issue**: Issue information including ID, title, severity, and description
- **project**: Project information where the issue was found
- **package**: Package information related to the issue
- **timestamp**: When the issue was detected

## Webhook Setup

This trigger automatically sets up a Snyk webhook when configured. The webhook is managed by SuperPlane and will be cleaned up when the trigger is removed.`
}

func (t *OnNewIssueDetected) Icon() string {
	return "shield"
}

func (t *OnNewIssueDetected) Color() string {
	return "gray"
}

func (t *OnNewIssueDetected) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "organizationId",
			Label:       "Organization ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Snyk organization ID to monitor",
		},
		{
			Name:        "projectId",
			Label:       "Project ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional project ID filter - if specified, only issues from this project will trigger",
		},
		{
			Name:        "severity",
			Label:       "Severity",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Description: "Optional severity filter - only issues of this severity or higher will trigger",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Low", Value: "low"},
						{Label: "Medium", Value: "medium"},
						{Label: "High", Value: "high"},
						{Label: "Critical", Value: "critical"},
					},
				},
			},
		},
	}
}

func (t *OnNewIssueDetected) Setup(ctx core.TriggerContext) error {
	var config OnNewIssueDetectedConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.OrganizationID == "" {
		return fmt.Errorf("organizationId is required")
	}

	webhookConfig := WebhookConfiguration{
		EventType: "issue.detected",
		OrgID:     config.OrganizationID,
		ProjectID: config.ProjectID,
	}

	return ctx.Integration.RequestWebhook(webhookConfig)
}

func (t *OnNewIssueDetected) Actions() []core.Action {
	return []core.Action{}
}

func (t *OnNewIssueDetected) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	return nil, nil
}

func (t *OnNewIssueDetected) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	config := OnNewIssueDetectedConfiguration{}
	err := mapstructure.Decode(ctx.Configuration, &config)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to decode configuration: %w", err)
	}

	eventType := ctx.Headers.Get("X-Snyk-Event")
	if eventType == "" {
		// Alternative header that Snyk might use
		eventType = ctx.Headers.Get("X-Snyk-Event-Type")
		if eventType == "" {
			return http.StatusBadRequest, fmt.Errorf("missing Snyk event header")
		}
	}

	// Snyk sends project_snapshot/v0 events that contain new issues
	if eventType != "project_snapshot/v0" {
		return http.StatusOK, nil
	}

	var payload map[string]any
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("error parsing request body: %v", err)
	}

	newIssues, ok := payload["newIssues"].([]any)
	if !ok || len(newIssues) == 0 {
		return http.StatusOK, nil
	}

	for _, issue := range newIssues {
		issueMap, ok := issue.(map[string]any)
		if !ok {
			continue
		}

		issuePayload := map[string]any{
			"issue":   issueMap,
			"project": payload["project"],
			"org":     payload["org"],
		}

		if t.matchesFilters(issuePayload, config) {
			if err := ctx.Events.Emit("snyk.issue.detected", issuePayload); err != nil {
				return http.StatusInternalServerError, fmt.Errorf("error emitting event: %v", err)
			}
		}
	}

	return http.StatusOK, nil
}

func (t *OnNewIssueDetected) matchesFilters(payload map[string]any, config OnNewIssueDetectedConfiguration) bool {
	if config.ProjectID != "" {
		if projectData, ok := payload["project"]; ok {
			if projectMap, isMap := projectData.(map[string]any); isMap {
				if id, hasID := projectMap["id"]; hasID {
					if idStr, isString := id.(string); isString {
						if idStr != config.ProjectID {
							return false
						}
					}
				}
			}
		}
	}

	if config.Severity != "" {
		if issueData, ok := payload["issue"].(map[string]any); ok {
			if severity, ok := issueData["severity"]; ok {
				if severityStr, isString := severity.(string); isString {
					if !t.isSeverityEqualOrHigher(severityStr, config.Severity) {
						return false
					}
				}
			}
		}
	}

	return true
}

func (t *OnNewIssueDetected) isSeverityEqualOrHigher(actual, threshold string) bool {
	severityOrder := map[string]int{
		"low":      1,
		"medium":   2,
		"high":     3,
		"critical": 4,
	}

	actualLevel, hasActual := severityOrder[actual]
	thresholdLevel, hasThreshold := severityOrder[threshold]

	if !hasActual || !hasThreshold {
		return true
	}

	return actualLevel >= thresholdLevel
}

func (t *OnNewIssueDetected) Cleanup(ctx core.TriggerContext) error {
	return nil
}
