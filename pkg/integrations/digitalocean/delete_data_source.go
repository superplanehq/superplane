package digitalocean

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const deleteDSPollInterval = 30 * time.Second

type DeleteDataSource struct{}

type DeleteDataSourceSpec struct {
	KnowledgeBase string `json:"knowledgeBase" mapstructure:"knowledgeBase"`
	DataSource    string `json:"dataSource" mapstructure:"dataSource"`
}

func (d *DeleteDataSource) Name() string {
	return "digitalocean.deleteDataSource"
}

func (d *DeleteDataSource) Label() string {
	return "Delete Data Source"
}

func (d *DeleteDataSource) Description() string {
	return "Remove a data source from a DigitalOcean Gradient AI knowledge base"
}

func (d *DeleteDataSource) Documentation() string {
	return `The Delete Data Source component removes a data source from an existing knowledge base on the DigitalOcean Gradient AI Platform.

## How it works

Deletes a single data source from a knowledge base. DigitalOcean automatically triggers a re-indexing job after every deletion to clean up stale embeddings from the OpenSearch database. The component waits for that job to complete before emitting the output.

## Output

Returns the deleted data source details:
- **dataSourceUUID**: UUID of the deleted data source
- **knowledgeBaseUUID**: UUID of the knowledge base
- **knowledgeBaseName**: Name of the knowledge base

`
}

func (d *DeleteDataSource) Icon() string {
	return "brain"
}

func (d *DeleteDataSource) Color() string {
	return "blue"
}

func (d *DeleteDataSource) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (d *DeleteDataSource) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "knowledgeBase",
			Label:       "Knowledge Base",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select a knowledge base",
			Description: "The knowledge base to remove the data source from. When using an expression, provide the knowledge base UUID.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "knowledge_base",
				},
			},
		},
		{
			Name:        "dataSource",
			Label:       "Data Source",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select a data source to delete",
			Description: "The data source to remove. Only data sources belonging to the selected knowledge base are shown. When using an expression, provide the data source UUID.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "knowledge_base_data_source",
					Parameters: []configuration.ParameterRef{
						{
							Name:      "knowledgeBase",
							ValueFrom: &configuration.ParameterValueFrom{Field: "knowledgeBase"},
						},
					},
				},
			},
		},
	}
}

func (d *DeleteDataSource) Setup(ctx core.SetupContext) error {
	spec := DeleteDataSourceSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.KnowledgeBase == "" {
		return errors.New("knowledgeBase is required")
	}

	if spec.DataSource == "" {
		return errors.New("dataSource is required")
	}

	if err := resolveDeleteDSMetadata(ctx, spec.KnowledgeBase, spec.DataSource); err != nil {
		return fmt.Errorf("error resolving metadata: %v", err)
	}

	return nil
}

// deleteDSMetadata is stored between poll ticks
type deleteDSMetadata struct {
	KBUUID         string `json:"kbUUID" mapstructure:"kbUUID"`
	KBName         string `json:"kbName" mapstructure:"kbName"`
	DataSourceUUID string `json:"dataSourceUUID" mapstructure:"dataSourceUUID"`
	// PrevIndexingJobID is the knowledge base's last indexing job UUID observed
	// immediately before the delete request. DigitalOcean auto-triggers a new
	// re-indexing job after deletion; we wait until last_indexing_job changes to
	// avoid mistakenly treating an older completed job as the re-indexing result.
	PrevIndexingJobID string         `json:"prevIndexingJobId" mapstructure:"prevIndexingJobId"`
	Output            map[string]any `json:"output" mapstructure:"output"`
}

func (d *DeleteDataSource) Execute(ctx core.ExecutionContext) error {
	spec := DeleteDataSourceSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// Fetch KB name for the output
	var kbName string
	var prevIndexingJobID string
	if kb, err := client.GetKnowledgeBase(spec.KnowledgeBase); err == nil {
		kbName = kb.Name
		if kb.LastIndexingJob != nil {
			prevIndexingJobID = kb.LastIndexingJob.UUID
		}
	}

	if err := client.DeleteKBDataSource(spec.KnowledgeBase, spec.DataSource); err != nil {
		return fmt.Errorf("failed to delete data source: %v", err)
	}

	output := map[string]any{
		"dataSourceUUID":    spec.DataSource,
		"knowledgeBaseUUID": spec.KnowledgeBase,
		"knowledgeBaseName": kbName,
	}

	// DO auto-triggers re-indexing after delete — poll until it completes
	if err := ctx.Metadata.Set(deleteDSMetadata{
		KBUUID:            spec.KnowledgeBase,
		KBName:            kbName,
		DataSourceUUID:    spec.DataSource,
		PrevIndexingJobID: prevIndexingJobID,
		Output:            output,
	}); err != nil {
		return fmt.Errorf("failed to store metadata: %v", err)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, deleteDSPollInterval)
}

func (d *DeleteDataSource) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (d *DeleteDataSource) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (d *DeleteDataSource) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (d *DeleteDataSource) HandleAction(ctx core.ActionContext) error {
	if ctx.Name != "poll" {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var meta deleteDSMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &meta); err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	kb, err := client.GetKnowledgeBase(meta.KBUUID)
	if err != nil {
		return fmt.Errorf("failed to get knowledge base: %v", err)
	}

	if kb.LastIndexingJob == nil {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, deleteDSPollInterval)
	}

	job := kb.LastIndexingJob
	if strings.TrimSpace(job.UUID) == "" {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, deleteDSPollInterval)
	}

	// If the KB already had a completed last_indexing_job from before the delete,
	// the auto-triggered re-indexing may not have started yet. In that window,
	// last_indexing_job will still point to the old job; keep polling until it
	// changes.
	if strings.TrimSpace(meta.PrevIndexingJobID) != "" && job.UUID == meta.PrevIndexingJobID {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, deleteDSPollInterval)
	}

	switch indexingJobState(job.Status) {
	case "completed", "successful", "no_changes":
		meta.Output["indexingJob"] = map[string]any{
			"status":               job.Status,
			"totalTokens":          job.TotalTokens,
			"completedDataSources": job.CompletedDataSources,
			"totalDataSources":     job.TotalDataSources,
			"startedAt":            job.StartedAt,
			"finishedAt":           job.FinishedAt,
		}
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"digitalocean.data_source.deleted",
			[]any{meta.Output},
		)
	case "running", "pending":
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, deleteDSPollInterval)
	case "failed", "cancelled", "partial":
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("indexing job %s for knowledge base %s: %s", job.UUID, meta.KBUUID, job.Status))
	default:
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, deleteDSPollInterval)
	}
}

func (d *DeleteDataSource) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (d *DeleteDataSource) Cleanup(ctx core.SetupContext) error {
	return nil
}

// DeleteDSNodeMetadata stores metadata about the knowledge base and data source for display in the UI
type DeleteDSNodeMetadata struct {
	KnowledgeBaseID   string `json:"knowledgeBaseId" mapstructure:"knowledgeBaseId"`
	KnowledgeBaseName string `json:"knowledgeBaseName" mapstructure:"knowledgeBaseName"`
	DataSourceID      string `json:"dataSourceId" mapstructure:"dataSourceId"`
	DataSourceName    string `json:"dataSourceName" mapstructure:"dataSourceName"`
}

func resolveDeleteDSMetadata(ctx core.SetupContext, kbID, dsID string) error {
	if strings.Contains(kbID, "{{") || strings.Contains(dsID, "{{") {
		return ctx.Metadata.Set(DeleteDSNodeMetadata{
			KnowledgeBaseID:   kbID,
			KnowledgeBaseName: kbID,
			DataSourceID:      dsID,
			DataSourceName:    dsID,
		})
	}

	var existing DeleteDSNodeMetadata
	err := mapstructure.Decode(ctx.Metadata.Get(), &existing)
	if err == nil && existing.KnowledgeBaseID == kbID && existing.DataSourceID == dsID &&
		existing.KnowledgeBaseName != "" && existing.DataSourceName != "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	meta := DeleteDSNodeMetadata{
		KnowledgeBaseID: kbID,
		DataSourceID:    dsID,
		DataSourceName:  dsID,
	}

	if kb, err := client.GetKnowledgeBase(kbID); err == nil {
		meta.KnowledgeBaseName = kb.Name
	} else {
		meta.KnowledgeBaseName = kbID
	}

	if dataSources, err := client.ListKBDataSources(kbID); err == nil {
		for _, ds := range dataSources {
			if ds.UUID == dsID {
				if ds.BucketName != "" {
					meta.DataSourceName = fmt.Sprintf("%s (%s)", ds.BucketName, ds.Region)
				} else if ds.WebCrawlerDataSource != nil && ds.WebCrawlerDataSource.BaseURL != "" {
					meta.DataSourceName = ds.WebCrawlerDataSource.BaseURL
				}
				break
			}
		}
	}

	return ctx.Metadata.Set(meta)
}
