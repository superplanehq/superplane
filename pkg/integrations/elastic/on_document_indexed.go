package elastic

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	checkConnectorAction        = "checkConnectorAvailability"
	checkConnectorRetryInterval = 10 * time.Second
	onDocumentIndexedPageSize   = 100
	onDocumentIndexedTimeField  = "@timestamp"
)

type OnDocumentIndexed struct{}

type OnDocumentIndexedConfiguration struct {
	Index string `json:"index" mapstructure:"index"`
}

type OnDocumentIndexedMetadata struct {
	LastTimestamp string `json:"lastTimestamp,omitempty" mapstructure:"lastTimestamp"`
	RouteKey      string `json:"routeKey,omitempty" mapstructure:"routeKey"`
	RuleID        string `json:"ruleId,omitempty" mapstructure:"ruleId"`
	Index         string `json:"index,omitempty" mapstructure:"index"`
}

func (t *OnDocumentIndexed) Name() string  { return "elastic.onDocumentIndexed" }
func (t *OnDocumentIndexed) Label() string { return "On Document Indexed" }
func (t *OnDocumentIndexed) Description() string {
	return "React when new documents are indexed in Elasticsearch"
}
func (t *OnDocumentIndexed) Icon() string  { return "database" }
func (t *OnDocumentIndexed) Color() string { return "gray" }

func (t *OnDocumentIndexed) Documentation() string {
	return `The On Document Indexed trigger starts a workflow execution when a new document is indexed into an Elasticsearch index.

## Shared Connector

SuperPlane creates **one Kibana Webhook connector per integration**, shared across all triggers that use the same Kibana instance. Each incoming request is routed to the correct trigger instance using two fields in the request body:

- ` + "`eventType`" + `: must be ` + "`\"document_indexed\"`" + ` — requests with any other value are silently ignored, allowing the shared connector to serve both this trigger and others (e.g. When Alert Fires).
- ` + "`routeKey`" + `: a unique ID assigned per trigger node — allows multiple On Document Indexed nodes on the same canvas to each react only to their own Kibana rule.

## How it works

1. When the trigger is saved, SuperPlane creates or reuses the shared Kibana Webhook connector and provisions a Kibana Elasticsearch query rule for the configured index.
2. Every minute, the rule checks for documents with an ` + "`@timestamp`" + ` value within the current window. When matches are found, Kibana fires the connector.
3. SuperPlane receives the webhook, queries Elasticsearch for all documents newer than its stored checkpoint, and emits one event per document.

## Configuration

- **Index**: The Elasticsearch index to monitor for new documents.

> **Note**: This trigger requires an ` + "`@timestamp`" + ` field mapped as ` + "`date`" + ` on indexed documents. Documents without that field will be missed. To ensure all documents are captured, configure an ingest pipeline on the index to auto-populate the field if absent:
> ` + "```" + `json
> { "set": { "field": "@timestamp", "value": "{{{_ingest.timestamp}}}", "override": false } }
> ` + "```" + `

## Webhook Verification

SuperPlane generates a random signing secret and configures the Kibana connector to include it on every request. Requests without the correct secret are rejected automatically.

## Event Data

The webhook acts as a signal. When it fires, SuperPlane queries Elasticsearch for documents newer than the stored checkpoint and emits one event per document containing its ID, index, and full source.`
}

func (t *OnDocumentIndexed) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "index",
			Label:       "Index",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Elasticsearch index to monitor for new documents.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeIndex,
				},
			},
		},
	}
}

func (t *OnDocumentIndexed) Setup(ctx core.TriggerContext) error {
	var config OnDocumentIndexedConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Index = strings.TrimSpace(config.Index)
	if config.Index == "" {
		return fmt.Errorf("index is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Elastic client: %w", err)
	}
	if err := ensureIndexExists(client, config.Index); err != nil {
		return err
	}

	if ctx.Metadata != nil {
		meta := loadDocumentIndexedMetadata(ctx.Metadata)
		changed := false

		// Backfill legacy metadata that predates the Index field without forcing
		// reprovisioning an already-created rule.
		if meta.Index == "" {
			meta.Index = config.Index
			changed = true
		} else if meta.Index != config.Index {
			// Index changes require a new rule and a fresh timestamp checkpoint.
			meta.Index = config.Index
			meta.RuleID = ""
			meta.LastTimestamp = time.Now().UTC().Format(time.RFC3339Nano)
			changed = true
		}
		if meta.LastTimestamp == "" {
			meta.LastTimestamp = time.Now().UTC().Format(time.RFC3339Nano)
			changed = true
		}
		if meta.RouteKey == "" {
			meta.RouteKey = uuid.NewString()
			changed = true
		}
		if changed {
			if err := ctx.Metadata.Set(meta); err != nil {
				return fmt.Errorf("failed to save metadata: %w", err)
			}
		}

		// If the rule is already provisioned, nothing else to do.
		if meta.RuleID != "" {
			return nil
		}
	}

	kibanaURL, err := ctx.Integration.GetConfig("kibanaUrl")
	if err != nil {
		return fmt.Errorf("failed to get Kibana URL: %w", err)
	}

	if err := ctx.Integration.RequestWebhook(map[string]any{"kibanaUrl": string(kibanaURL)}); err != nil {
		return fmt.Errorf("failed to request webhook: %w", err)
	}

	return ctx.Requests.ScheduleActionCall(checkConnectorAction, map[string]any{}, checkConnectorRetryInterval)
}

func (t *OnDocumentIndexed) Actions() []core.Action {
	return []core.Action{
		{
			Name:           checkConnectorAction,
			Description:    "Find the Kibana connector and create the Elasticsearch query rule",
			UserAccessible: false,
		},
	}
}

func (t *OnDocumentIndexed) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	if ctx.Name == checkConnectorAction {
		return nil, t.checkConnectorAndCreateRule(ctx)
	}
	return nil, fmt.Errorf("unknown action: %s", ctx.Name)
}

func (t *OnDocumentIndexed) checkConnectorAndCreateRule(ctx core.TriggerActionContext) error {
	var config OnDocumentIndexedConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	// If the rule was already created (e.g. action fired twice), nothing to do.
	meta := loadDocumentIndexedMetadata(ctx.Metadata)
	if meta.RuleID != "" {
		return nil
	}
	if meta.RouteKey == "" {
		meta.RouteKey = uuid.NewString()
		if ctx.Metadata != nil {
			if err := ctx.Metadata.Set(meta); err != nil {
				return fmt.Errorf("failed to save metadata: %w", err)
			}
		}
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("elastic onDocumentIndexed: failed to create client: %v", err)
		}
		return ctx.Requests.ScheduleActionCall(checkConnectorAction, map[string]any{}, checkConnectorRetryInterval)
	}

	connectors, err := client.ListKibanaConnectors()
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("elastic onDocumentIndexed: failed to list connectors: %v", err)
		}
		return ctx.Requests.ScheduleActionCall(checkConnectorAction, map[string]any{}, checkConnectorRetryInterval)
	}

	var connectorID string
	for _, c := range connectors {
		if c.Name == KibanaConnectorName {
			connectorID = c.ID
			break
		}
	}

	if connectorID == "" {
		if ctx.Logger != nil {
			ctx.Logger.Infof("elastic onDocumentIndexed: connector %q not found yet, retrying", KibanaConnectorName)
		}
		return ctx.Requests.ScheduleActionCall(checkConnectorAction, map[string]any{}, checkConnectorRetryInterval)
	}

	rule, err := client.CreateKibanaQueryRule(config.Index, connectorID, meta.RouteKey)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("elastic onDocumentIndexed: failed to create rule: %v", err)
		}
		return ctx.Requests.ScheduleActionCall(checkConnectorAction, map[string]any{}, checkConnectorRetryInterval)
	}

	meta.RuleID = rule.ID
	if ctx.Metadata != nil {
		return ctx.Metadata.Set(meta)
	}

	return nil
}

func (t *OnDocumentIndexed) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("error retrieving webhook secret: %v", err)
	}

	headerVal := ctx.Headers.Get(SigningHeaderName)
	if headerVal == "" {
		return http.StatusForbidden, nil, fmt.Errorf("missing required header %q", SigningHeaderName)
	}
	if len(headerVal) != len(secret) || subtle.ConstantTimeCompare([]byte(headerVal), secret) != 1 {
		return http.StatusForbidden, nil, fmt.Errorf("invalid value for header %q", SigningHeaderName)
	}

	var payload map[string]any
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("invalid JSON payload: %w", err)
	}

	if extractString(payload, "eventType") != "document_indexed" {
		return http.StatusOK, nil, nil
	}

	meta := loadDocumentIndexedMetadata(ctx.Metadata)
	if meta.RouteKey == "" || extractString(payload, "routeKey") != meta.RouteKey {
		return http.StatusOK, nil, nil
	}

	var config OnDocumentIndexedConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	if meta.LastTimestamp == "" {
		meta.LastTimestamp = time.Now().UTC().Format(time.RFC3339Nano)
		if ctx.Metadata != nil {
			if err := ctx.Metadata.Set(meta); err != nil {
				return http.StatusInternalServerError, nil, fmt.Errorf("failed to initialize metadata: %w", err)
			}
		}
		return http.StatusOK, nil, nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return http.StatusInternalServerError, nil, fmt.Errorf("failed to create client: %w", err)
	}

	hits, err := client.SearchDocumentsAfter(config.Index, meta.LastTimestamp, onDocumentIndexedPageSize)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("elastic onDocumentIndexed: failed to search documents after webhook: %v", err)
		}
		return http.StatusOK, nil, nil
	}

	newLastTimestamp := meta.LastTimestamp
	for _, hit := range hits {
		eventPayload := map[string]any{
			"id":     hit.ID,
			"index":  hit.Index,
			"source": hit.Source,
		}
		if err := ctx.Events.Emit("elastic.document.indexed", eventPayload); err != nil {
			return http.StatusInternalServerError, nil, fmt.Errorf("error emitting event: %v", err)
		}

		if timestamp := hit.TimestampValue(); timestamp != "" && timestamp > newLastTimestamp {
			newLastTimestamp = timestamp
		}
	}

	if newLastTimestamp != meta.LastTimestamp && ctx.Metadata != nil {
		meta.LastTimestamp = newLastTimestamp
		if err := ctx.Metadata.Set(meta); err != nil {
			return http.StatusInternalServerError, nil, fmt.Errorf("failed to update metadata: %w", err)
		}
	}

	return http.StatusOK, nil, nil
}

func (t *OnDocumentIndexed) Cleanup(ctx core.TriggerContext) error {
	var meta OnDocumentIndexedMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &meta); err != nil || meta.RuleID == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	return client.DeleteKibanaRule(meta.RuleID)
}

func loadDocumentIndexedMetadata(metadata core.MetadataContext) OnDocumentIndexedMetadata {
	var meta OnDocumentIndexedMetadata
	if metadata == nil {
		return meta
	}

	_ = mapstructure.Decode(metadata.Get(), &meta)
	return meta
}
