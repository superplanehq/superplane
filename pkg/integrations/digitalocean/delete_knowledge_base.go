package digitalocean

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type DeleteKnowledgeBase struct{}

type DeleteKnowledgeBaseSpec struct {
	KnowledgeBase            string `json:"knowledgeBase" mapstructure:"knowledgeBase"`
	DeleteOpenSearchDatabase bool   `json:"deleteOpenSearchDatabase" mapstructure:"deleteOpenSearchDatabase"`
}

func (d *DeleteKnowledgeBase) Name() string {
	return "digitalocean.deleteKnowledgeBase"
}

func (d *DeleteKnowledgeBase) Label() string {
	return "Delete Knowledge Base"
}

func (d *DeleteKnowledgeBase) Description() string {
	return "Delete a DigitalOcean Gradient AI knowledge base and optionally its OpenSearch database"
}

func (d *DeleteKnowledgeBase) Documentation() string {
	return `The Delete Knowledge Base component removes a knowledge base from the DigitalOcean Gradient AI Platform.

## How it works

Deletes the specified knowledge base. Optionally, you can also delete the associated OpenSearch database that stores the vector embeddings.

## Use Cases

- **Cleanup**: Remove knowledge bases that are no longer needed
- **Resource management**: Free up resources by deleting unused knowledge bases and their databases
- **Rotation**: Delete an old knowledge base after a new one has been verified and attached

## Configuration

- **Knowledge Base**: The knowledge base to delete (required)
- **Delete OpenSearch Database**: Whether to also delete the associated OpenSearch database (optional, defaults to off)

## Output

Returns confirmation of the deletion including:
- **knowledgeBaseUUID**: UUID of the deleted knowledge base
- **databaseDeleted**: Whether the OpenSearch database was also deleted
- **databaseId**: UUID of the deleted database (included when the database was deleted)
- **databaseName**: Name of the deleted database (included when the database was deleted)

## Notes

- If the knowledge base is currently attached to any agents, it will automatically be removed from those agents upon deletion. Consider using the Detach Knowledge Base component first if you need more control over the detachment process.
- Deleting the OpenSearch database is irreversible and will remove all vector embeddings`
}

func (d *DeleteKnowledgeBase) Icon() string {
	return "brain"
}

func (d *DeleteKnowledgeBase) Color() string {
	return "red"
}

func (d *DeleteKnowledgeBase) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteKnowledgeBase) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "knowledgeBase",
			Label:       "Knowledge Base",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select a knowledge base",
			Description: "The knowledge base to delete. When using an expression, provide the knowledge base UUID.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "knowledge_base",
				},
			},
		},
		{
			Name:        "deleteOpenSearchDatabase",
			Label:       "Delete OpenSearch Database",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Also delete the OpenSearch database that stores the vector embeddings for this knowledge base. This is irreversible.",
		},
	}
}

func (d *DeleteKnowledgeBase) Setup(ctx core.SetupContext) error {
	spec := DeleteKnowledgeBaseSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.KnowledgeBase == "" {
		return errors.New("knowledgeBase is required")
	}

	if err := resolveDeleteKBMetadata(ctx, spec.KnowledgeBase); err != nil {
		return fmt.Errorf("error resolving metadata: %v", err)
	}

	return nil
}

func (d *DeleteKnowledgeBase) Execute(ctx core.ExecutionContext) error {
	spec := DeleteKnowledgeBaseSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	output := map[string]any{
		"knowledgeBaseUUID": spec.KnowledgeBase,
		"databaseDeleted":   false,
	}

	// If deleting the database, fetch the KB first to get the database ID
	var databaseID string
	if spec.DeleteOpenSearchDatabase {
		kb, err := client.GetKnowledgeBase(spec.KnowledgeBase)
		if err != nil {
			if doErr, ok := err.(*DOAPIError); ok && doErr.StatusCode == http.StatusNotFound {
				// KB already gone — nothing to do
				return ctx.ExecutionState.Emit(
					core.DefaultOutputChannel.Name,
					"digitalocean.knowledge_base.deleted",
					[]any{output},
				)
			}
			return fmt.Errorf("failed to get knowledge base: %v", err)
		}
		databaseID = kb.DatabaseID
	}

	// Delete the knowledge base
	if err := client.DeleteKnowledgeBase(spec.KnowledgeBase); err != nil {
		if doErr, ok := err.(*DOAPIError); ok && doErr.StatusCode == http.StatusNotFound {
			// Idempotent — already deleted
		} else {
			return fmt.Errorf("failed to delete knowledge base: %v", err)
		}
	}

	// Delete the OpenSearch database if requested and a database ID was found
	if spec.DeleteOpenSearchDatabase && databaseID != "" {
		// Resolve database name before deleting so we can include it in the output
		var databaseName string
		if databases, err := client.ListDatabasesByEngine("opensearch"); err == nil {
			for _, db := range databases {
				if db.ID == databaseID {
					databaseName = db.Name
					break
				}
			}
		}

		if err := client.DeleteDatabase(databaseID); err != nil {
			if doErr, ok := err.(*DOAPIError); ok && doErr.StatusCode == http.StatusNotFound {
				// Database already absent — don't claim we deleted it
			} else {
				return fmt.Errorf("knowledge base deleted, but failed to delete OpenSearch database %s: %v", databaseID, err)
			}
		} else {
			output["databaseDeleted"] = true
			output["databaseId"] = databaseID
			if databaseName != "" {
				output["databaseName"] = databaseName
			}
		}
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.knowledge_base.deleted",
		[]any{output},
	)
}

func (d *DeleteKnowledgeBase) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteKnowledgeBase) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteKnowledgeBase) Actions() []core.Action {
	return []core.Action{}
}

func (d *DeleteKnowledgeBase) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions defined")
}

func (d *DeleteKnowledgeBase) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeleteKnowledgeBase) Cleanup(ctx core.SetupContext) error {
	return nil
}

// DeleteKBNodeMetadata stores metadata about a knowledge base for display in the UI
type DeleteKBNodeMetadata struct {
	KnowledgeBaseID   string `json:"knowledgeBaseId" mapstructure:"knowledgeBaseId"`
	KnowledgeBaseName string `json:"knowledgeBaseName" mapstructure:"knowledgeBaseName"`
}

// resolveDeleteKBMetadata fetches the knowledge base name from the API and stores it in metadata
func resolveDeleteKBMetadata(ctx core.SetupContext, kbID string) error {
	if strings.Contains(kbID, "{{") {
		return ctx.Metadata.Set(DeleteKBNodeMetadata{
			KnowledgeBaseID:   kbID,
			KnowledgeBaseName: kbID,
		})
	}

	var existing DeleteKBNodeMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &existing)
	if err == nil && existing.KnowledgeBaseID == kbID && existing.KnowledgeBaseName != "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	kb, err := client.GetKnowledgeBase(kbID)
	if err != nil {
		return fmt.Errorf("failed to fetch knowledge base %q: %w", kbID, err)
	}

	return ctx.Metadata.Set(DeleteKBNodeMetadata{
		KnowledgeBaseID:   kbID,
		KnowledgeBaseName: kb.Name,
	})
}
