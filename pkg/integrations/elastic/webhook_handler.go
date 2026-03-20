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
}

type webhookMetadata struct {
	ConnectorID string `json:"connectorId" mapstructure:"connectorId"`
}

func (h *ElasticWebhookHandler) CompareConfig(a, b any) (bool, error) {
	var ca, cb webhookConfig
	if err := mapstructure.Decode(a, &ca); err != nil {
		return false, fmt.Errorf("decode webhook config a: %w", err)
	}
	if err := mapstructure.Decode(b, &cb); err != nil {
		return false, fmt.Errorf("decode webhook config b: %w", err)
	}
	// Triggers sharing one connector only if they point at the same Kibana.
	// A URL change returns false, triggering connector replacement.
	return ca.KibanaURL == cb.KibanaURL, nil
}

func (h *ElasticWebhookHandler) Merge(current, requested any) (any, bool, error) {
	var cur, req webhookConfig
	if err := mapstructure.Decode(current, &cur); err != nil {
		return nil, false, fmt.Errorf("decode current webhook config: %w", err)
	}
	if err := mapstructure.Decode(requested, &req); err != nil {
		return nil, false, fmt.Errorf("decode requested webhook config: %w", err)
	}

	if cur.KibanaURL == req.KibanaURL {
		return current, false, nil
	}

	// KibanaURL changed: update the stored config and re-queue for provisioning
	// so Setup() recreates the connector on the new Kibana instance.
	return req, true, nil
}

func (h *ElasticWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	// Generate a random signing secret and store it on the webhook so
	// HandleWebhook can retrieve it for validation.
	secret, err := crypto.Base64String(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate signing secret: %w", err)
	}

	if err := ctx.Webhook.SetSecret([]byte(secret)); err != nil {
		return nil, fmt.Errorf("failed to store webhook secret: %w", err)
	}

	connector, err := client.CreateKibanaConnector(
		KibanaConnectorName,
		ctx.Webhook.GetURL(),
		secret,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kibana connector: %w", err)
	}

	return webhookMetadata{ConnectorID: connector.ID}, nil
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

	if err := client.DeleteKibanaConnector(meta.ConnectorID); err != nil {
		return fmt.Errorf("failed to delete Kibana connector %s: %w", meta.ConnectorID, err)
	}

	return nil
}
