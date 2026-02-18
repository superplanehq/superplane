package harness

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type HarnessWebhookHandler struct{}

type WebhookConfiguration struct {
	PipelineIdentifier string   `json:"pipelineIdentifier" mapstructure:"pipelineIdentifier"`
	OrgID              string   `json:"orgId" mapstructure:"orgId"`
	ProjectID          string   `json:"projectId" mapstructure:"projectId"`
	EventTypes         []string `json:"eventTypes" mapstructure:"eventTypes"`
}

type WebhookMetadata struct {
	PipelineIdentifier string `json:"pipelineIdentifier,omitempty" mapstructure:"pipelineIdentifier"`
	OrgID              string `json:"orgId,omitempty" mapstructure:"orgId"`
	ProjectID          string `json:"projectId,omitempty" mapstructure:"projectId"`
	RuleIdentifier     string `json:"ruleIdentifier,omitempty" mapstructure:"ruleIdentifier"`
	URL                string `json:"url,omitempty" mapstructure:"url"`
}

var defaultWebhookEventTypes = []string{"PipelineEnd"}

func decodeWebhookConfiguration(value any) (WebhookConfiguration, error) {
	config := WebhookConfiguration{}
	if err := mapstructure.Decode(value, &config); err != nil {
		return WebhookConfiguration{}, fmt.Errorf("failed to decode webhook configuration: %w", err)
	}

	config.PipelineIdentifier = strings.TrimSpace(config.PipelineIdentifier)
	config.OrgID = strings.TrimSpace(config.OrgID)
	config.ProjectID = strings.TrimSpace(config.ProjectID)
	config.EventTypes = normalizeWebhookEventTypes(config.EventTypes)

	return config, nil
}

func webhookConfigurationsEqual(a, b WebhookConfiguration) bool {
	return a.PipelineIdentifier == b.PipelineIdentifier &&
		a.OrgID == b.OrgID &&
		a.ProjectID == b.ProjectID &&
		slices.Equal(a.EventTypes, b.EventTypes)
}

func (h *HarnessWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}

	config, err := decodeWebhookConfiguration(ctx.Webhook.GetConfiguration())
	if err != nil {
		return nil, err
	}

	webhookURL := strings.TrimSpace(ctx.Webhook.GetURL())
	if webhookURL == "" {
		return nil, fmt.Errorf("webhook url is required")
	}

	if config.PipelineIdentifier == "" {
		return WebhookMetadata{
			OrgID:     config.OrgID,
			ProjectID: config.ProjectID,
			URL:       webhookURL,
		}, nil
	}
	if config.OrgID == "" {
		return nil, fmt.Errorf("orgId is required for webhook provisioning")
	}
	if config.ProjectID == "" {
		return nil, fmt.Errorf("projectId is required for webhook provisioning")
	}

	secretBytes, err := ctx.Webhook.GetSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook secret: %w", err)
	}
	secret := strings.TrimSpace(string(secretBytes))

	hash := sha256.Sum256([]byte(ctx.Webhook.GetID()))
	name := fmt.Sprintf("superplane-harness-%x", hash[:8])
	ruleIdentifier := normalizeHarnessIdentifier(name + "-rule")

	headers := map[string]string{}
	if secret != "" {
		headers["Authorization"] = fmt.Sprintf("Bearer %s", secret)
	}

	scopedClient := client.withScope(config.OrgID, config.ProjectID)
	err = scopedClient.UpsertPipelineNotificationRule(UpsertPipelineNotificationRuleRequest{
		PipelineIdentifier: config.PipelineIdentifier,
		RuleIdentifier:     ruleIdentifier,
		RuleName:           name + "-rule",
		EventTypes:         config.EventTypes,
		WebhookURL:         webhookURL,
		Headers:            headers,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to provision Harness notification resources: %w", err)
	}

	return WebhookMetadata{
		PipelineIdentifier: config.PipelineIdentifier,
		OrgID:              config.OrgID,
		ProjectID:          config.ProjectID,
		RuleIdentifier:     ruleIdentifier,
		URL:                webhookURL,
	}, nil
}

func (h *HarnessWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := WebhookMetadata{}
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode webhook metadata: %w", err)
	}

	config, err := decodeWebhookConfiguration(ctx.Webhook.GetConfiguration())
	if err != nil {
		return err
	}

	pipelineIdentifier := firstNonEmpty(
		strings.TrimSpace(metadata.PipelineIdentifier),
		strings.TrimSpace(config.PipelineIdentifier),
	)
	orgID := firstNonEmpty(
		strings.TrimSpace(metadata.OrgID),
		strings.TrimSpace(config.OrgID),
	)
	projectID := firstNonEmpty(
		strings.TrimSpace(metadata.ProjectID),
		strings.TrimSpace(config.ProjectID),
	)
	ruleIdentifier := strings.TrimSpace(metadata.RuleIdentifier)
	if pipelineIdentifier == "" || ruleIdentifier == "" || orgID == "" || projectID == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	return client.withScope(orgID, projectID).DeletePipelineNotificationRule(pipelineIdentifier, ruleIdentifier)
}

func (h *HarnessWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA, err := decodeWebhookConfiguration(a)
	if err != nil {
		return false, err
	}

	configB, err := decodeWebhookConfiguration(b)
	if err != nil {
		return false, err
	}

	return webhookConfigurationsEqual(configA, configB), nil
}

func (h *HarnessWebhookHandler) Merge(current, requested any) (any, bool, error) {
	currentConfig, err := decodeWebhookConfiguration(current)
	if err != nil {
		return nil, false, err
	}

	requestedConfig, err := decodeWebhookConfiguration(requested)
	if err != nil {
		return nil, false, err
	}

	if webhookConfigurationsEqual(currentConfig, requestedConfig) {
		return currentConfig, false, nil
	}

	return requestedConfig, true, nil
}

func normalizeWebhookEventTypes(eventTypes []string) []string {
	normalized := normalizeNotificationRuleEventTypes(eventTypes)
	slices.Sort(normalized)
	return normalized
}
