package snyk

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type IgnoreIssue struct{}

type IgnoreIssueConfiguration struct {
	ProjectID string `json:"projectId" mapstructure:"projectId"`
	IssueID   string `json:"issueId" mapstructure:"issueId"`
	Reason    string `json:"reason" mapstructure:"reason"`
	ExpiresAt string `json:"expiresAt" mapstructure:"expiresAt"` // Optional
}

type IgnoreIssueMetadata struct {
	ProjectID string `json:"projectId" mapstructure:"projectId"`
	IssueID   string `json:"issueId" mapstructure:"issueId"`
}

func (c *IgnoreIssue) Name() string {
	return "snyk.ignoreIssue"
}

func (c *IgnoreIssue) Label() string {
	return "Ignore Issue"
}

func (c *IgnoreIssue) Description() string {
	return "Ignore a specific Snyk security issue"
}

func (c *IgnoreIssue) Documentation() string {
	return `The Ignore Issue component allows you to programmatically ignore a specific Snyk security issue.

## Use Cases

- **Risk acceptance**: Temporarily accept risks while a fix is being developed
- **False positive handling**: Suppress issues that are determined to be false positives
- **Automated suppression**: Automatically ignore issues based on predefined criteria
- **Workflow integration**: Integrate issue ignoring into broader security workflows

## Configuration

- **Project ID**: The project ID where the issue exists
- **Issue ID**: The specific issue ID to ignore
- **Reason**: The reason for ignoring the issue
- **Expires At**: Optional expiration date for the ignore rule (ISO 8601 format)

## Output

Returns information about the ignored issue including success status and any messages from the API.

## Notes

- The issue will no longer appear in Snyk reports after being ignored
- Ignored issues can be unignored later through the Snyk UI or API`
}

func (c *IgnoreIssue) Icon() string {
	return "shield"
}

func (c *IgnoreIssue) Color() string {
	return "gray"
}

func (c *IgnoreIssue) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *IgnoreIssue) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "projectId",
			Label:       "Project ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Snyk project ID containing the issue",
		},
		{
			Name:        "issueId",
			Label:       "Issue ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "The specific issue ID to ignore",
		},
		{
			Name:        "reason",
			Label:       "Reason",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "Reason for ignoring the issue",
		},
		{
			Name:        "expiresAt",
			Label:       "Expires At",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional expiration date for the ignore rule (ISO 8601 format)",
		},
	}
}

func (c *IgnoreIssue) Setup(ctx core.SetupContext) error {
	var config IgnoreIssueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if config.ProjectID == "" {
		return errors.New("projectId is required")
	}

	if config.IssueID == "" {
		return errors.New("issueId is required")
	}

	if config.Reason == "" {
		return errors.New("reason is required")
	}

	metadata := IgnoreIssueMetadata{
		ProjectID: config.ProjectID,
		IssueID:   config.IssueID,
	}

	return ctx.Metadata.Set(metadata)
}

func (c *IgnoreIssue) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *IgnoreIssue) Execute(ctx core.ExecutionContext) error {
	var config IgnoreIssueConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	orgID, err := ctx.Integration.GetConfig("organizationId")
	if err != nil {
		return fmt.Errorf("error getting organizationId: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Snyk client: %w", err)
	}

	ignoreReq := IgnoreIssueRequest{
		Reason:    config.Reason,
		ExpiresAt: config.ExpiresAt,
	}

	response, err := client.IgnoreIssue(string(orgID), config.ProjectID, config.IssueID, ignoreReq)
	if err != nil {
		return fmt.Errorf("failed to ignore issue: %w", err)
	}

	result := map[string]any{
		"success":   response.Success,
		"message":   response.Message,
		"projectId": config.ProjectID,
		"issueId":   config.IssueID,
		"reason":    config.Reason,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"snyk.issue.ignored",
		[]any{result},
	)
}

func (c *IgnoreIssue) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *IgnoreIssue) Actions() []core.Action {
	return []core.Action{}
}

func (c *IgnoreIssue) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *IgnoreIssue) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *IgnoreIssue) Cleanup(ctx core.SetupContext) error {
	return nil
}
