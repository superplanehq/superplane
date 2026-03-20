package elastic

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type IndexDocument struct{}

const defaultDocumentTimestampTemplate = `{{ now().UTC().Format("2006-01-02T15:04:05Z") }}`

type IndexDocumentConfiguration struct {
	Index      string         `json:"index" mapstructure:"index"`
	Document   map[string]any `json:"document" mapstructure:"document"`
	DocumentID string         `json:"documentId" mapstructure:"documentId"`
}

type IndexDocumentSetupMetadata struct {
	Index string `json:"index" mapstructure:"index"`
}

func (c *IndexDocument) Name() string  { return "elastic.indexDocument" }
func (c *IndexDocument) Label() string { return "Index Document" }
func (c *IndexDocument) Description() string {
	return "Write a JSON document to an Elasticsearch index"
}
func (c *IndexDocument) Icon() string  { return "elastic" }
func (c *IndexDocument) Color() string { return "gray" }

func (c *IndexDocument) Documentation() string {
	return `The Index Document component writes a JSON document to an Elasticsearch index.

## Use Cases

- **Audit logging**: Record workflow actions in Elasticsearch for centralized search and dashboards
- **Incident records**: Index structured incident data for analysis and alerting
- **Workflow output**: Store results from any workflow step for downstream querying

## Configuration

- **Index**: The Elasticsearch index name to write to (e.g. ` + "`workflow-audit`" + `)
- **Document**: The JSON object to index. The editor starts with an ` + "`@timestamp`" + ` template so documents are compatible with On Document Indexed by default.
- **Document ID** *(optional)*: A stable ID for idempotent writes. If omitted, Elasticsearch generates one automatically. Providing an ID means re-runs update the existing document rather than creating a duplicate.

## Outputs

The component emits an event containing:
- ` + "`id`" + `: The document ID assigned by Elasticsearch
- ` + "`index`" + `: The index the document was written to
- ` + "`result`" + `: Operation result (` + "`created`" + ` or ` + "`updated`" + `)
- ` + "`version`" + `: The document version number`
}

func (c *IndexDocument) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *IndexDocument) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "index",
			Label:       "Index",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Elasticsearch index to write the document to.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeIndex,
				},
			},
		},
		{
			Name:     "document",
			Label:    "Document",
			Type:     configuration.FieldTypeObject,
			Required: true,
			Default: map[string]any{
				onDocumentIndexedTimeField: defaultDocumentTimestampTemplate,
			},
			Description: "The JSON document to index. Defaults to include an @timestamp field template.",
		},
		{
			Name:        "documentId",
			Label:       "Document ID",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Optional document ID. Providing an ID enables idempotent writes (re-runs update rather than duplicate).",
		},
	}
}

func (c *IndexDocument) Setup(ctx core.SetupContext) error {
	var config IndexDocumentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.Index = strings.TrimSpace(config.Index)
	if config.Index == "" {
		return fmt.Errorf("index is required")
	}

	if config.Document == nil {
		return fmt.Errorf("document is required and must be a JSON object")
	}

	resolvedIndex := config.Index
	if !isTemplateExpression(config.Index) {
		client, err := NewClient(ctx.HTTP, ctx.Integration)
		if err != nil {
			return fmt.Errorf("failed to create Elastic client: %w", err)
		}

		indices, err := client.ListIndices()
		if err != nil {
			return fmt.Errorf("failed to list Elasticsearch indices: %w", err)
		}

		match := slices.IndexFunc(indices, func(index IndexInfo) bool {
			return strings.EqualFold(index.Index, config.Index)
		})
		if match == -1 {
			return fmt.Errorf("selected index %q was not found in Elasticsearch", config.Index)
		}

		resolvedIndex = indices[match].Index
	}

	if ctx.Metadata != nil {
		if err := ctx.Metadata.Set(IndexDocumentSetupMetadata{Index: resolvedIndex}); err != nil {
			return fmt.Errorf("failed to save metadata: %w", err)
		}
	}

	return nil
}

func (c *IndexDocument) Execute(ctx core.ExecutionContext) error {
	var config IndexDocumentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	config.Index = strings.TrimSpace(config.Index)
	if config.Index == "" {
		return ctx.ExecutionState.Fail("error", "index is required")
	}

	if config.Document == nil {
		return ctx.ExecutionState.Fail("error", "document is required and must be a JSON object")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create Elastic client: %v", err))
	}

	resp, err := client.IndexDocument(config.Index, strings.TrimSpace(config.DocumentID), config.Document)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to index document: %v", err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"elastic.document.indexed",
		[]any{map[string]any{
			"id":      resp.ID,
			"index":   resp.Index,
			"result":  resp.Result,
			"version": resp.Version,
		}},
	)
}

func (c *IndexDocument) Actions() []core.Action                  { return nil }
func (c *IndexDocument) HandleAction(_ core.ActionContext) error { return nil }
func (c *IndexDocument) Cancel(_ core.ExecutionContext) error    { return nil }
func (c *IndexDocument) Cleanup(_ core.SetupContext) error       { return nil }
func (c *IndexDocument) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *IndexDocument) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
