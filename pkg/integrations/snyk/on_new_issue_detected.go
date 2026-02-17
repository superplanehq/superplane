package snyk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

type OnNewIssueDetected struct{}

type OnNewIssueDetectedConfiguration struct {
	ProjectID string   `json:"projectId" mapstructure:"projectId"` // Optional - if not specified, applies to all projects
	Severity  []string `json:"severity" mapstructure:"severity"`   // Optional - filter by severity (low, medium, high, critical)
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

- **Project**: Optional project filter - select a project to only trigger on issues from that project
- **Severity**: Optional severity filter - select one or more severities to trigger on (low, medium, high, critical)

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
			Name:     "projectId",
			Label:    "Project",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "project",
				},
			},
			Description: "Optional project filter - if specified, only issues from this project will trigger",
		},
		{
			Name:        "severity",
			Label:       "Severity",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    false,
			Description: "Optional severity filter - only issues with the selected severities will trigger",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{
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

	orgID, err := ctx.Integration.GetConfig("organizationId")
	if err != nil {
		return fmt.Errorf("error getting organizationId: %v", err)
	}

	if string(orgID) == "" {
		return fmt.Errorf("organizationId is required in integration configuration")
	}

	webhookConfig := WebhookConfiguration{
		EventType: "issue.detected",
		OrgID:     string(orgID),
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

	signature := ctx.Headers.Get("X-Hub-Signature")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("missing signature")
	}

	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature format")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error getting secret: %v", err)
	}

	if err := crypto.VerifySignature(secret, ctx.Body, signature); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature: %v", err)
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
		projectData, ok := payload["project"]
		if !ok {
			return false
		}

		projectMap, isMap := projectData.(map[string]any)
		if !isMap {
			return false
		}

		idStr, isString := projectMap["id"].(string)
		if !isString || idStr != config.ProjectID {
			return false
		}
	}

	if len(config.Severity) > 0 {
		issueRaw, ok := payload["issue"].(map[string]any)
		if !ok {
			return false
		}

		// Support both flat (severity at top level) and nested (issueData.severity) formats.
		severityStr, isString := issueRaw["severity"].(string)
		if !isString {
			if issueData, ok := issueRaw["issueData"].(map[string]any); ok {
				severityStr, isString = issueData["severity"].(string)
			}
			if !isString {
				return false
			}
		}

		matched := false
		for _, s := range config.Severity {
			if severityStr == s {
				matched = true
				break
			}
		}

		if !matched {
			return false
		}
	}

	return true
}

func (t *OnNewIssueDetected) Cleanup(ctx core.TriggerContext) error {
	return nil
}
