package elastic

import (
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	onDocumentIndexedPollAction   = "poll"
	onDocumentIndexedPollInterval = 1 * time.Minute
	onDocumentIndexedPageSize     = 100
)

type OnDocumentIndexed struct{}

type OnDocumentIndexedConfiguration struct {
	Index          string `json:"index" mapstructure:"index"`
	TimestampField string `json:"timestampField" mapstructure:"timestampField"`
}

type OnDocumentIndexedMetadata struct {
	LastTimestamp string `json:"lastTimestamp,omitempty" mapstructure:"lastTimestamp"`
}

func (t *OnDocumentIndexed) Name() string  { return "elastic.onDocumentIndexed" }
func (t *OnDocumentIndexed) Label() string { return "When Document Is Indexed" }
func (t *OnDocumentIndexed) Description() string {
	return "React when new documents are indexed in Elasticsearch"
}
func (t *OnDocumentIndexed) Icon() string  { return "database" }
func (t *OnDocumentIndexed) Color() string { return "gray" }

func (t *OnDocumentIndexed) Documentation() string {
	return `The When Document Is Indexed trigger starts a workflow execution when a new document is indexed into an Elasticsearch index.

## How it works

SuperPlane polls the configured index every minute, querying for documents with a timestamp field value newer than the last seen document. Each new document triggers a workflow execution.

## Configuration

- **Index**: The Elasticsearch index to monitor
- **Timestamp Field** *(optional)*: The document field used to identify new documents. Defaults to ` + "`@timestamp`" + `.

## Event Data

The trigger emits the full document source as the event data, along with the document ID, index, and version.`
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
		{
			Name:        "timestampField",
			Label:       "Timestamp Field",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "@timestamp",
			Description: "The document field used to detect new documents. Must be a date field. Defaults to @timestamp.",
		},
	}
}

func (t *OnDocumentIndexed) Setup(ctx core.TriggerContext) error {
	var config OnDocumentIndexedConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.Index) == "" {
		return fmt.Errorf("index is required")
	}

	if ctx.Metadata != nil {
		var meta OnDocumentIndexedMetadata
		if err := mapstructure.Decode(ctx.Metadata.Get(), &meta); err != nil || meta.LastTimestamp == "" {
			meta.LastTimestamp = time.Now().UTC().Format(time.RFC3339Nano)
		}
		if err := ctx.Metadata.Set(meta); err != nil {
			return fmt.Errorf("failed to save metadata: %w", err)
		}
	}

	if ctx.Requests != nil {
		if err := ctx.Requests.ScheduleActionCall(onDocumentIndexedPollAction, map[string]any{}, onDocumentIndexedPollInterval); err != nil {
			return fmt.Errorf("failed to schedule poll: %w", err)
		}
	}

	return nil
}

func (t *OnDocumentIndexed) Actions() []core.Action {
	return []core.Action{
		{
			Name:           onDocumentIndexedPollAction,
			Description:    "Poll Elasticsearch index for new documents",
			UserAccessible: false,
		},
	}
}

func (t *OnDocumentIndexed) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	if ctx.Name == onDocumentIndexedPollAction {
		return nil, t.poll(ctx)
	}
	return nil, nil
}

func (t *OnDocumentIndexed) poll(ctx core.TriggerActionContext) error {
	var config OnDocumentIndexedConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.TimestampField) == "" {
		config.TimestampField = "@timestamp"
	}

	var meta OnDocumentIndexedMetadata
	if ctx.Metadata != nil {
		if err := mapstructure.Decode(ctx.Metadata.Get(), &meta); err != nil || meta.LastTimestamp == "" {
			meta.LastTimestamp = time.Now().UTC().Format(time.RFC3339Nano)
		}
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("elastic onDocumentIndexed: failed to create client: %v", err)
		}
		return ctx.Requests.ScheduleActionCall(onDocumentIndexedPollAction, map[string]any{}, onDocumentIndexedPollInterval)
	}

	hits, err := client.SearchDocumentsAfter(config.Index, config.TimestampField, meta.LastTimestamp, onDocumentIndexedPageSize)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.Warnf("elastic onDocumentIndexed: failed to search documents: %v", err)
		}
		return ctx.Requests.ScheduleActionCall(onDocumentIndexedPollAction, map[string]any{}, onDocumentIndexedPollInterval)
	}

	for _, hit := range hits {
		payload := map[string]any{
			"id":     hit.ID,
			"index":  hit.Index,
			"source": hit.Source,
		}
		if err := ctx.Events.Emit("elastic.document.indexed", payload); err != nil {
			return fmt.Errorf("failed to emit event: %w", err)
		}

		ts := hit.TimestampValue(config.TimestampField)
		if ts != "" && ts > meta.LastTimestamp {
			meta.LastTimestamp = ts
			if ctx.Metadata != nil {
				if err := ctx.Metadata.Set(meta); err != nil {
					return fmt.Errorf("failed to update metadata: %w", err)
				}
			}
		}
	}

	return ctx.Requests.ScheduleActionCall(onDocumentIndexedPollAction, map[string]any{}, onDocumentIndexedPollInterval)
}

func (t *OnDocumentIndexed) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (t *OnDocumentIndexed) Cleanup(_ core.TriggerContext) error {
	return nil
}
