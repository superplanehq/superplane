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

const addDSPollInterval = 30 * time.Second

type AddDataSource struct{}

type AddDataSourceSpec struct {
	KnowledgeBase    string `json:"knowledgeBase" mapstructure:"knowledgeBase"`
	IndexAfterAdding bool   `json:"indexAfterAdding" mapstructure:"indexAfterAdding"`

	// Data source fields (same as DataSourceSpec but flat, not in a list)
	DataSourceSpec `mapstructure:",squash"`
}

func (a *AddDataSource) Name() string {
	return "digitalocean.addDataSource"
}

func (a *AddDataSource) Label() string {
	return "Add Data Source"
}

func (a *AddDataSource) Description() string {
	return "Add a Spaces bucket or web URL data source to an existing DigitalOcean Gradient AI knowledge base"
}

func (a *AddDataSource) Documentation() string {
	return `The Add Data Source component adds a new data source to an existing knowledge base on the DigitalOcean Gradient AI Platform.

## How it works

Adds a single data source — either a Spaces bucket or a web/sitemap URL — to a knowledge base. When **Index after adding** is enabled (the default), the component also starts an indexing job and waits for it to complete before emitting the output.

## Data Source Types

- **Spaces Bucket or Folder** — indexes all supported files in a DigitalOcean Spaces bucket or folder
- **Web or Sitemap URL** — crawls a public website (seed URL) or a list of URLs from a sitemap

## Chunking Strategies

Each data source has its own independent chunking configuration:

- **Section-based** (default) — splits on structural elements like headings and paragraphs; fast and low-cost
- **Semantic** — groups sentences by meaning; slower but context-aware
- **Hierarchical** — creates parent (context) and child (retrieval) chunk pairs
- **Fixed-length** — splits strictly by token count; best for logs and unstructured text

## Indexing

When **Index after adding** is enabled, the component starts an indexing job after adding the data source and polls every 30 seconds until the job completes. Disable it if you want to add multiple data sources first and index them all at once using the Index Knowledge Base component.

## Output

Returns the added data source details:
- **dataSourceUUID**: UUID of the newly added data source
- **knowledgeBaseUUID**: UUID of the knowledge base
- **knowledgeBaseName**: Name of the knowledge base

When indexing is enabled, the output also includes:
- **indexingJob**: Full indexing job details (status, totalTokens, completedDataSources, totalDataSources, startedAt, finishedAt)`
}

func (a *AddDataSource) Icon() string {
	return "brain"
}

func (a *AddDataSource) Color() string {
	return "blue"
}

func (a *AddDataSource) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (a *AddDataSource) Configuration() []configuration.Field {
	fields := []configuration.Field{
		{
			Name:        "knowledgeBase",
			Label:       "Knowledge Base",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select a knowledge base",
			Description: "The knowledge base to add the data source to. When using an expression, provide the knowledge base UUID.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "knowledge_base",
				},
			},
		},
		{
			Name:        "indexAfterAdding",
			Label:       "Index after adding",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     true,
			Description: "Start an indexing job after adding the data source and wait for it to complete. Disable if you plan to add multiple data sources first and index them separately.",
		},
	}

	fields = append(fields, dataSourceItemSchema()...)
	return fields
}

func (a *AddDataSource) Setup(ctx core.SetupContext) error {
	spec := AddDataSourceSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.KnowledgeBase == "" {
		return errors.New("knowledgeBase is required")
	}

	if err := validateDataSource(1, spec.DataSourceSpec); err != nil {
		return err
	}

	if err := resolveAddDSMetadata(ctx, spec.KnowledgeBase); err != nil {
		return fmt.Errorf("error resolving metadata: %v", err)
	}

	return nil
}

// addDSMetadata is stored between poll ticks
type addDSMetadata struct {
	KBUUID         string         `json:"kbUUID" mapstructure:"kbUUID"`
	KBName         string         `json:"kbName" mapstructure:"kbName"`
	DataSourceUUID string         `json:"dataSourceUUID" mapstructure:"dataSourceUUID"`
	Output         map[string]any `json:"output" mapstructure:"output"`
}

func (a *AddDataSource) Execute(ctx core.ExecutionContext) error {
	spec := AddDataSourceSpec{}
	if err := mapstructure.WeakDecode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	// Fetch KB name for the output
	var kbName string
	if kb, err := client.GetKnowledgeBase(spec.KnowledgeBase); err == nil {
		kbName = kb.Name
	}

	ds := buildKBDataSource(spec.DataSourceSpec)
	added, err := client.AddKBDataSource(spec.KnowledgeBase, ds)
	if err != nil {
		return fmt.Errorf("failed to add data source: %v", err)
	}

	output := map[string]any{
		"dataSourceUUID":    added.UUID,
		"knowledgeBaseUUID": spec.KnowledgeBase,
		"knowledgeBaseName": kbName,
	}

	if !spec.IndexAfterAdding {
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"digitalocean.data_source.added",
			[]any{output},
		)
	}

	// Start indexing and poll
	job, err := client.StartIndexingJob(spec.KnowledgeBase)
	if err != nil {
		return fmt.Errorf("failed to start indexing job: %v", err)
	}

	if err := ctx.Metadata.Set(addDSMetadata{
		KBUUID:         spec.KnowledgeBase,
		KBName:         kbName,
		DataSourceUUID: added.UUID,
		Output:         output,
	}); err != nil {
		return fmt.Errorf("failed to store metadata: %v", err)
	}

	_ = job // job UUID tracked via KB's last_indexing_job
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, addDSPollInterval)
}

func (a *AddDataSource) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (a *AddDataSource) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (a *AddDataSource) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (a *AddDataSource) HandleAction(ctx core.ActionContext) error {
	if ctx.Name != "poll" {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var meta addDSMetadata
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
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, addDSPollInterval)
	}

	job := kb.LastIndexingJob
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
			"digitalocean.data_source.added",
			[]any{meta.Output},
		)
	case "running", "pending":
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, addDSPollInterval)
	case "failed", "cancelled", "partial":
		return ctx.ExecutionState.Fail("error", fmt.Sprintf("indexing job %s for knowledge base %s: %s", job.UUID, meta.KBUUID, job.Status))
	default:
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, addDSPollInterval)
	}
}

func (a *AddDataSource) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (a *AddDataSource) Cleanup(ctx core.SetupContext) error {
	return nil
}

// AddDSNodeMetadata stores metadata about the knowledge base for display in the UI
type AddDSNodeMetadata struct {
	KnowledgeBaseID   string `json:"knowledgeBaseId" mapstructure:"knowledgeBaseId"`
	KnowledgeBaseName string `json:"knowledgeBaseName" mapstructure:"knowledgeBaseName"`
}

func resolveAddDSMetadata(ctx core.SetupContext, kbID string) error {
	if strings.Contains(kbID, "{{") {
		return ctx.Metadata.Set(AddDSNodeMetadata{
			KnowledgeBaseID:   kbID,
			KnowledgeBaseName: kbID,
		})
	}

	var existing AddDSNodeMetadata
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

	return ctx.Metadata.Set(AddDSNodeMetadata{
		KnowledgeBaseID:   kbID,
		KnowledgeBaseName: kb.Name,
	})
}
