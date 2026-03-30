package elastic

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

// SigningHeaderName is the HTTP header Kibana will include on every webhook
// delivery to prove the request originates from the configured connector.
const SigningHeaderName = "X-Superplane-Secret"

// KibanaConnectorName is the display name used for all Kibana webhook connectors
// created by SuperPlane. Used by triggers to locate the shared connector.
const KibanaConnectorName = "SuperPlane Alert"

// ElasticWebhookHandler automatically creates and tears down Kibana Webhook
// connectors when the OnAlertFires trigger is set up or removed.
type ElasticWebhookHandler struct{}

type webhookConfig struct {
	KibanaURL string `json:"kibanaUrl" mapstructure:"kibanaUrl"`
	RuleID    string `json:"ruleId" mapstructure:"ruleId"`
}

type webhookMetadata struct {
	ConnectorID string `json:"connectorId" mapstructure:"connectorId"`
	RuleID      string `json:"ruleId" mapstructure:"ruleId"`
}

func (h *ElasticWebhookHandler) CompareConfig(a, b any) (bool, error) {
	var ca, cb webhookConfig
	if err := mapstructure.Decode(a, &ca); err != nil {
		return false, fmt.Errorf("decode webhook config a: %w", err)
	}
	if err := mapstructure.Decode(b, &cb); err != nil {
		return false, fmt.Errorf("decode webhook config b: %w", err)
	}
	return ca.KibanaURL == cb.KibanaURL && ca.RuleID == cb.RuleID, nil
}

func (h *ElasticWebhookHandler) Merge(current, requested any) (any, bool, error) {
	var cur, req webhookConfig
	if err := mapstructure.Decode(current, &cur); err != nil {
		return nil, false, fmt.Errorf("decode current webhook config: %w", err)
	}
	if err := mapstructure.Decode(requested, &req); err != nil {
		return nil, false, fmt.Errorf("decode requested webhook config: %w", err)
	}

	if cur.KibanaURL == req.KibanaURL && cur.RuleID == req.RuleID {
		return current, false, nil
	}

	// The selected rule or Kibana instance changed. Re-queue provisioning so
	// Setup() recreates and re-attaches the connector for the new target.
	return req, true, nil
}

func (h *ElasticWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	var config webhookConfig
	if err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config); err != nil {
		return nil, fmt.Errorf("error decoding webhook config: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	connector, err := client.FindKibanaWebhookConnector(ctx.Webhook.GetURL())
	if err != nil {
		return nil, fmt.Errorf("failed to find existing Kibana connector: %w", err)
	}

	if connector == nil {
		// Generate a random signing secret and store it on the webhook so
		// HandleWebhook can retrieve it for validation.
		secret, err := crypto.Base64String(32)
		if err != nil {
			return nil, fmt.Errorf("failed to generate signing secret: %w", err)
		}

		if err := ctx.Webhook.SetSecret([]byte(secret)); err != nil {
			return nil, fmt.Errorf("failed to store webhook secret: %w", err)
		}

		connector, err = client.CreateKibanaConnector(
			"SuperPlane Alert",
			ctx.Webhook.GetURL(),
			secret,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create Kibana connector: %w", err)
		}
	} else if connector.Config.Headers != nil && connector.Config.Headers[SigningHeaderName] != "" {
		if err := ctx.Webhook.SetSecret([]byte(connector.Config.Headers[SigningHeaderName])); err != nil {
			return nil, fmt.Errorf("failed to sync webhook secret from connector: %w", err)
		}
	}

	var previousMeta webhookMetadata
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &previousMeta); err != nil {
		return nil, fmt.Errorf("error decoding webhook metadata: %v", err)
	}

	if config.RuleID != "" {
		if err := client.EnsureKibanaRuleHasConnector(config.RuleID, connector.ID); err != nil {
			return nil, fmt.Errorf("failed to attach connector %s to rule %s: %w", connector.ID, config.RuleID, err)
		}
	}

	if previousMeta.RuleID != "" && previousMeta.RuleID != config.RuleID {
		if err := client.RemoveKibanaRuleConnector(previousMeta.RuleID, connector.ID); err != nil {
			return nil, fmt.Errorf("failed to detach connector %s from previous rule %s: %w", connector.ID, previousMeta.RuleID, err)
		}
	}

	return webhookMetadata{ConnectorID: connector.ID, RuleID: config.RuleID}, nil
}

func (h *ElasticWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	var meta webhookMetadata
	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &meta); err != nil {
		return fmt.Errorf("error decoding webhook metadata: %v", err)
	}
	if meta.ConnectorID == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	if meta.RuleID != "" {
		if err := client.RemoveKibanaRuleConnector(meta.RuleID, meta.ConnectorID); err != nil {
			return fmt.Errorf("failed to detach Kibana connector %s from rule %s: %w", meta.ConnectorID, meta.RuleID, err)
		}
	}

	if err := client.DeleteKibanaConnector(meta.ConnectorID); err != nil {
		return fmt.Errorf("failed to delete Kibana connector %s: %w", meta.ConnectorID, err)
	}

	return nil
}
