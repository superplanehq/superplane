package railway

import (
	"fmt"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type RailwayWebhookHandler struct{}

type WebhookConfiguration struct {
	ProjectID  string   `json:"projectId" mapstructure:"projectId"`
	EventTypes []string `json:"eventTypes" mapstructure:"eventTypes"`
}

type WebhookMetadata struct {
	RuleID      string `json:"ruleId" mapstructure:"ruleId"`
	WorkspaceID string `json:"workspaceId" mapstructure:"workspaceId"`
}

func decodeWebhookConfiguration(value any) (WebhookConfiguration, error) {
	config := WebhookConfiguration{}
	if err := mapstructure.Decode(value, &config); err != nil {
		return WebhookConfiguration{}, err
	}
	return config, nil
}

func decodeWebhookMetadata(value any) (WebhookMetadata, error) {
	metadata := WebhookMetadata{}
	if err := mapstructure.Decode(value, &metadata); err != nil {
		return WebhookMetadata{}, err
	}
	return metadata, nil
}

func (h *RailwayWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA, err := decodeWebhookConfiguration(a)
	if err != nil {
		return false, fmt.Errorf("failed to decode webhook configuration A: %w", err)
	}

	configB, err := decodeWebhookConfiguration(b)
	if err != nil {
		return false, fmt.Errorf("failed to decode webhook configuration B: %w", err)
	}

	return configA.ProjectID == configB.ProjectID, nil
}

func (h *RailwayWebhookHandler) Merge(current, requested any) (any, bool, error) {
	currentConfig, err := decodeWebhookConfiguration(current)
	if err != nil {
		return nil, false, fmt.Errorf("failed to decode current webhook configuration: %w", err)
	}

	requestedConfig, err := decodeWebhookConfiguration(requested)
	if err != nil {
		return nil, false, fmt.Errorf("failed to decode requested webhook configuration: %w", err)
	}

	if currentConfig.ProjectID != requestedConfig.ProjectID {
		return currentConfig, false, nil
	}

	mergedEventTypes := currentConfig.EventTypes
	for _, eventType := range requestedConfig.EventTypes {
		if !slices.Contains(mergedEventTypes, eventType) {
			mergedEventTypes = append(mergedEventTypes, eventType)
		}
	}

	changed := len(mergedEventTypes) > len(currentConfig.EventTypes)
	mergedConfig := currentConfig
	mergedConfig.EventTypes = mergedEventTypes

	return mergedConfig, changed, nil
}

func (h *RailwayWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	config, err := decodeWebhookConfiguration(ctx.Webhook.GetConfiguration())
	if err != nil {
		return nil, err
	}

	project, err := client.GetProjectDetails(config.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to load Railway project %q: %w", config.ProjectID, err)
	}

	workspaceID := strings.TrimSpace(project.WorkspaceID)
	if workspaceID == "" {
		return nil, fmt.Errorf("project %q has no workspaceId", config.ProjectID)
	}

	webhookURL := ctx.Webhook.GetURL()
	if webhookURL == "" {
		return nil, fmt.Errorf("webhook URL is required")
	}

	rule, err := client.CreateNotificationRule(workspaceID, config.ProjectID, config.EventTypes, webhookURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create Railway notification rule: %w", err)
	}

	return WebhookMetadata{
		RuleID:      rule.ID,
		WorkspaceID: workspaceID,
	}, nil
}

func (h *RailwayWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata, err := decodeWebhookMetadata(ctx.Webhook.GetMetadata())
	if err != nil {
		return fmt.Errorf("failed to decode webhook metadata: %w", err)
	}

	if metadata.RuleID == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	err = client.DeleteNotificationRule(metadata.RuleID)
	if err != nil {
		// Log warning and ignore "Not Authorized" or typical token permissions errors
		// to prevent blocking node/integration deletion in SuperPlane.
		ctx.Logger.Warnf("Failed to delete Railway notification rule %q: %v", metadata.RuleID, err)
		if strings.Contains(strings.ToLower(err.Error()), "not authorized") || strings.Contains(strings.ToLower(err.Error()), "unauthorized") {
			return nil
		}
		return err
	}

	return nil
}
