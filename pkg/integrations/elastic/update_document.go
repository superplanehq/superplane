package elastic

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type UpdateDocument struct{}

type UpdateDocumentConfiguration struct {
	Index    string         `json:"index" mapstructure:"index"`
	Document string         `json:"document" mapstructure:"document"`
	Fields   map[string]any `json:"fields" mapstructure:"fields"`
}

type UpdateDocumentSetupMetadata struct {
	Index    string `json:"index" mapstructure:"index"`
	Document string `json:"document" mapstructure:"document"`
}

func (c *UpdateDocument) Name() string  { return "elastic.updateDocument" }
func (c *UpdateDocument) Label() string { return "Update Document" }
func (c *UpdateDocument) Description() string {
	return "Partially update an existing document in an Elasticsearch index"
}
func (c *UpdateDocument) Icon() string  { return "elastic" }
func (c *UpdateDocument) Color() string { return "gray" }

func (c *UpdateDocument) Documentation() string {
	return `The Update Document component applies a partial update to an existing document in an Elasticsearch index.

## Configuration

- **Index**: The Elasticsearch index containing the document
- **Document**: The document to update
- **Fields**: The fields to merge into the existing document (partial update). The editor starts with an ` + "`@timestamp`" + ` template for convenience.

## Outputs

The component emits an event containing:
- ` + "`id`" + `: The document ID
- ` + "`index`" + `: The index the document belongs to
- ` + "`result`" + `: Operation result (` + "`updated`" + `)
- ` + "`version`" + `: The new document version number`
}

func (c *UpdateDocument) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *UpdateDocument) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "index",
			Label:       "Index",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Elasticsearch index containing the document to update.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeIndex,
				},
			},
		},
		{
			Name:        "document",
			Label:       "Document",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The document to update from the selected index.",
			Placeholder: "Select a document",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeDocument,
					Parameters: []configuration.ParameterRef{
						{
							Name:      "index",
							ValueFrom: &configuration.ParameterValueFrom{Field: "index"},
						},
					},
				},
			},
		},
		{
			Name:     "fields",
			Label:    "Fields",
			Type:     configuration.FieldTypeObject,
			Required: true,
			Default: map[string]any{
				onDocumentIndexedTimeField: defaultDocumentTimestampTemplate,
			},
			Description: "The fields to merge into the existing document (partial update). Defaults to include an @timestamp field template.",
		},
	}
}

func (c *UpdateDocument) Setup(ctx core.SetupContext) error {
	var config UpdateDocumentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if strings.TrimSpace(config.Index) == "" {
		return fmt.Errorf("index is required")
	}

	config.Document = strings.TrimSpace(config.Document)
	if config.Document == "" {
		return fmt.Errorf("document is required")
	}

	if config.Fields == nil {
		return fmt.Errorf("fields is required and must be a JSON object")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Elastic client: %w", err)
	}
	if err := ensureIndexExists(client, config.Index); err != nil {
		return err
	}
	if err := ensureDocumentExists(client, config.Index, config.Document); err != nil {
		return err
	}

	if ctx.Metadata != nil {
		if err := ctx.Metadata.Set(UpdateDocumentSetupMetadata{
			Index:    config.Index,
			Document: config.Document,
		}); err != nil {
			return fmt.Errorf("failed to save metadata: %w", err)
		}
	}

	return nil
}

func (c *UpdateDocument) Execute(ctx core.ExecutionContext) error {
	var config UpdateDocumentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to decode configuration: %v", err))
	}

	config.Index = strings.TrimSpace(config.Index)
	if config.Index == "" {
		return ctx.ExecutionState.Fail("error", "index is required")
	}

	config.Document = strings.TrimSpace(config.Document)
	if config.Document == "" {
		return ctx.ExecutionState.Fail("error", "document is required")
	}

	if config.Fields == nil {
		return ctx.ExecutionState.Fail("error", "fields is required and must be a JSON object")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create Elastic client: %v", err))
	}

	resp, err := client.UpdateDocument(config.Index, config.Document, config.Fields)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to update document: %v", err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"elastic.document.updated",
		[]any{map[string]any{
			"id":      resp.ID,
			"index":   resp.Index,
			"result":  resp.Result,
			"version": resp.Version,
		}},
	)
}

func (c *UpdateDocument) Actions() []core.Action                  { return nil }
func (c *UpdateDocument) HandleAction(_ core.ActionContext) error { return nil }
func (c *UpdateDocument) Cancel(_ core.ExecutionContext) error    { return nil }
func (c *UpdateDocument) Cleanup(_ core.SetupContext) error       { return nil }
func (c *UpdateDocument) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *UpdateDocument) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
