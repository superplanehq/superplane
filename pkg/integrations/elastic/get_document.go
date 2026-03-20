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

type GetDocument struct{}

type GetDocumentConfiguration struct {
	Index    string `json:"index" mapstructure:"index"`
	Document string `json:"document" mapstructure:"document"`
}

type GetDocumentSetupMetadata struct {
	Index    string `json:"index" mapstructure:"index"`
	Document string `json:"document" mapstructure:"document"`
}

func (c *GetDocument) Name() string  { return "elastic.getDocument" }
func (c *GetDocument) Label() string { return "Get Document" }
func (c *GetDocument) Description() string {
	return "Retrieve a document from Elasticsearch by index and document ID"
}
func (c *GetDocument) Icon() string  { return "elastic" }
func (c *GetDocument) Color() string { return "gray" }

func (c *GetDocument) Documentation() string {
	return `The Get Document component retrieves a JSON document from an Elasticsearch index by its ID.

## Configuration

- **Index**: The Elasticsearch index to read from
- **Document**: The document to retrieve

## Outputs

The component emits an event containing:
- ` + "`id`" + `: The document ID
- ` + "`index`" + `: The index the document was read from
- ` + "`version`" + `: The document version number
- ` + "`source`" + `: The document fields`
}

func (c *GetDocument) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetDocument) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "index",
			Label:       "Index",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The Elasticsearch index to retrieve the document from.",
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
			Description: "The document to retrieve from the selected index.",
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
	}
}

func (c *GetDocument) Setup(ctx core.SetupContext) error {
	var config GetDocumentConfiguration
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
		if err := ctx.Metadata.Set(GetDocumentSetupMetadata{
			Index:    config.Index,
			Document: config.Document,
		}); err != nil {
			return fmt.Errorf("failed to save metadata: %w", err)
		}
	}

	return nil
}

func (c *GetDocument) Execute(ctx core.ExecutionContext) error {
	var config GetDocumentConfiguration
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

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to create Elastic client: %v", err))
	}

	resp, err := client.GetDocument(config.Index, config.Document)
	if err != nil {
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("failed to get document: %v", err))
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"elastic.document.retrieved",
		[]any{map[string]any{
			"id":      resp.ID,
			"index":   resp.Index,
			"version": resp.Version,
			"source":  resp.Source,
		}},
	)
}

func (c *GetDocument) Actions() []core.Action                  { return nil }
func (c *GetDocument) HandleAction(_ core.ActionContext) error { return nil }
func (c *GetDocument) Cancel(_ core.ExecutionContext) error    { return nil }
func (c *GetDocument) Cleanup(_ core.SetupContext) error       { return nil }
func (c *GetDocument) HandleWebhook(_ core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
func (c *GetDocument) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}
