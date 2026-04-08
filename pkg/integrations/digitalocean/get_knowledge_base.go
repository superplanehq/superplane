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

type GetKnowledgeBase struct{}

type GetKnowledgeBaseSpec struct {
	KnowledgeBase string `json:"knowledgeBase" mapstructure:"knowledgeBase"`
}

func (g *GetKnowledgeBase) Name() string {
	return "digitalocean.getKnowledgeBase"
}

func (g *GetKnowledgeBase) Label() string {
	return "Get Knowledge Base"
}

func (g *GetKnowledgeBase) Description() string {
	return "Retrieve detailed information about a DigitalOcean Gradient AI knowledge base"
}

func (g *GetKnowledgeBase) Documentation() string {
	return `The Get Knowledge Base component retrieves comprehensive information about an existing knowledge base on the DigitalOcean Gradient AI Platform.

## How it works

Fetches the knowledge base details including its OpenSearch database, all attached data sources, and the latest indexing job status.

## Use Cases

- **Pre-flight checks**: Inspect a knowledge base before attaching it to an agent or running evaluations
- **Health monitoring**: Check indexing status and data source count on a schedule
- **Teardown workflows**: Fetch the database ID before deleting the knowledge base
- **Auditing**: Verify region, embedding model, and data source configuration

## Output

Returns the full knowledge base object including:
- **uuid**, **name**, **status**, **region**, **tags** — core properties
- **embeddingModelUUID**, **embeddingModelName** — embedding model details
- **projectId**, **projectName** — associated project
- **database** — OpenSearch database object with id, name, and status
- **dataSources** — array of all attached data sources with type and source details
- **lastIndexingJob** — full indexing job details: status, phase, totalTokens, data source progress, timing, and report availability
- **createdAt**, **updatedAt** — timestamps`
}

func (g *GetKnowledgeBase) Icon() string {
	return "brain"
}

func (g *GetKnowledgeBase) Color() string {
	return "blue"
}

func (g *GetKnowledgeBase) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetKnowledgeBase) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "knowledgeBase",
			Label:       "Knowledge Base",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select a knowledge base",
			Description: "The knowledge base to retrieve. When using an expression, provide the knowledge base UUID.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "knowledge_base",
				},
			},
		},
	}
}

func (g *GetKnowledgeBase) Setup(ctx core.SetupContext) error {
	spec := GetKnowledgeBaseSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.KnowledgeBase == "" {
		return errors.New("knowledgeBase is required")
	}

	if err := resolveGetKBMetadata(ctx, spec.KnowledgeBase); err != nil {
		return fmt.Errorf("error resolving metadata: %v", err)
	}

	return nil
}

func (g *GetKnowledgeBase) Execute(ctx core.ExecutionContext) error {
	spec := GetKnowledgeBaseSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	kb, err := client.GetKnowledgeBase(spec.KnowledgeBase)
	if err != nil {
		return fmt.Errorf("failed to get knowledge base: %v", err)
	}

	output := buildGetKBOutput(kb)

	resolveGetKBDisplayNames(client, kb, output)
	resolveGetKBDataSources(client, kb.UUID, output)

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"digitalocean.knowledge_base.fetched",
		[]any{output},
	)
}

func buildGetKBOutput(kb *KnowledgeBase) map[string]any {
	output := map[string]any{
		"uuid":               kb.UUID,
		"name":               kb.Name,
		"databaseStatus":     kb.DatabaseStatus,
		"region":             kb.Region,
		"embeddingModelUUID": kb.EmbeddingModelUUID,
		"projectId":          kb.ProjectID,
		"tags":               kb.Tags,
		"createdAt":          kb.CreatedAt,
		"updatedAt":          kb.UpdatedAt,
	}

	// Database — just the ID from the KB response; name and status are resolved separately
	if kb.DatabaseID != "" {
		output["database"] = map[string]any{
			"id": kb.DatabaseID,
		}
	}

	// Last indexing job with full details
	if kb.LastIndexingJob != nil {
		output["lastIndexingJob"] = map[string]any{
			"uuid":                 kb.LastIndexingJob.UUID,
			"status":               kb.LastIndexingJob.Status,
			"phase":                kb.LastIndexingJob.Phase,
			"totalTokens":          kb.LastIndexingJob.TotalTokens,
			"completedDataSources": kb.LastIndexingJob.CompletedDataSources,
			"totalDataSources":     kb.LastIndexingJob.TotalDataSources,
			"startedAt":            kb.LastIndexingJob.StartedAt,
			"finishedAt":           kb.LastIndexingJob.FinishedAt,
			"createdAt":            kb.LastIndexingJob.CreatedAt,
			"isReportAvailable":    kb.LastIndexingJob.IsReportAvailable,
		}
	}

	return output
}

// resolveGetKBDisplayNames enriches the output with human-readable names for
// embedding model, project, and database. Failures are ignored so a lookup
// error never blocks the result.
func resolveGetKBDisplayNames(client *Client, kb *KnowledgeBase, output map[string]any) {
	if models, err := client.ListEmbeddingModels(); err == nil {
		for _, m := range models {
			if m.UUID == kb.EmbeddingModelUUID {
				output["embeddingModelName"] = m.Name
				break
			}
		}
	}

	if projects, err := client.ListProjects(); err == nil {
		for _, p := range projects {
			if p.ID == kb.ProjectID {
				output["projectName"] = p.Name
				break
			}
		}
	}

	if kb.DatabaseID != "" {
		if databases, err := client.ListDatabasesByEngine("opensearch"); err == nil {
			for _, db := range databases {
				if db.ID == kb.DatabaseID {
					dbMap, ok := output["database"].(map[string]any)
					if ok {
						dbMap["name"] = db.Name
						dbMap["status"] = db.Status
					}
					break
				}
			}
		}
	}
}

// resolveGetKBDataSources fetches data sources and adds them to the output.
// Failures are ignored so a lookup error never blocks the result.
func resolveGetKBDataSources(client *Client, kbUUID string, output map[string]any) {
	dataSources, err := client.ListKBDataSources(kbUUID)
	if err != nil {
		return
	}

	dsOutput := make([]map[string]any, 0, len(dataSources))
	for _, ds := range dataSources {
		item := map[string]any{
			"uuid": ds.UUID,
		}

		if ds.BucketName != "" {
			item["type"] = "spaces"
			item["spacesBucket"] = ds.Region + "/" + ds.BucketName
		} else if ds.WebCrawlerDataSource != nil {
			item["type"] = "web"
			item["webURL"] = ds.WebCrawlerDataSource.BaseURL
			if ds.WebCrawlerDataSource.CrawlingOption != "" {
				item["crawlingOption"] = ds.WebCrawlerDataSource.CrawlingOption
			}
		}

		if ds.ChunkingAlgorithm != "" {
			item["chunkingAlgorithm"] = ds.ChunkingAlgorithm
		}
		if ds.CreatedAt != "" {
			item["createdAt"] = ds.CreatedAt
		}
		if ds.UpdatedAt != "" {
			item["updatedAt"] = ds.UpdatedAt
		}

		dsOutput = append(dsOutput, item)
	}

	output["dataSources"] = dsOutput
}

// GetKBNodeMetadata stores metadata about a knowledge base for display in the UI
type GetKBNodeMetadata struct {
	KnowledgeBaseID   string `json:"knowledgeBaseId" mapstructure:"knowledgeBaseId"`
	KnowledgeBaseName string `json:"knowledgeBaseName" mapstructure:"knowledgeBaseName"`
}

// resolveGetKBMetadata fetches the knowledge base name from the API and stores it in metadata
func resolveGetKBMetadata(ctx core.SetupContext, kbID string) error {
	if strings.Contains(kbID, "{{") {
		return ctx.Metadata.Set(GetKBNodeMetadata{
			KnowledgeBaseID:   kbID,
			KnowledgeBaseName: kbID,
		})
	}

	var existing GetKBNodeMetadata
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

	return ctx.Metadata.Set(GetKBNodeMetadata{
		KnowledgeBaseID:   kbID,
		KnowledgeBaseName: kb.Name,
	})
}

func (g *GetKnowledgeBase) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetKnowledgeBase) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetKnowledgeBase) Actions() []core.Action {
	return []core.Action{}
}

func (g *GetKnowledgeBase) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (g *GetKnowledgeBase) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (g *GetKnowledgeBase) Cleanup(ctx core.SetupContext) error {
	return nil
}
