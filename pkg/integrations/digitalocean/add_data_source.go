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

When **Index after adding** is enabled, the component starts an indexing job scoped **only to the newly added data source** and polls every 30 seconds until the job completes. Other existing data sources in the knowledge base are not re-indexed. Disable it if you want to add multiple data sources first and index them all at once using the Index Knowledge Base component.

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

	dsFields := dataSourceItemSchema()
	for i, field := range dsFields {
		switch field.Name {
		case "maxChunkSize":
			dsFields[i].Description = "Tokens per chunk. Range: 100–256 (All MiniLM), 100–512 (Multi QA), 100–8192 (GTE Large / Qwen3)."
		case "parentChunkSize":
			dsFields[i].Description = "Context chunk tokens. Range: 100–256 (All MiniLM), 100–512 (Multi QA), 100–8192 (GTE Large / Qwen3). Must be larger than child chunk size."
		case "childChunkSize":
			dsFields[i].Description = "Retrieval chunk tokens. Range: 100–256 (All MiniLM), 100–512 (Multi QA), 100–8192 (GTE Large / Qwen3). Must be smaller than parent chunk size."
		}
	}

	fields = append(fields, dsFields...)
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

	if !strings.Contains(spec.KnowledgeBase, "{{") && hasAnyChunkSize(spec.DataSourceSpec) {
		if err := validateChunkSizesAgainstModel(ctx, spec.KnowledgeBase, spec.DataSourceSpec); err != nil {
			return err
		}
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
		"dataSourceName":    dataSourceName(added),
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

	// Start indexing for the newly added data source only
	job, err := client.StartIndexingJob(spec.KnowledgeBase, added.UUID)
	if err != nil {
		return fmt.Errorf("failed to start indexing job: %v", err)
	}

	_ = job // job UUID tracked via KB's last_indexing_job
	if err := ctx.Metadata.Set(addDSMetadata{
		KBUUID:         spec.KnowledgeBase,
		KBName:         kbName,
		DataSourceUUID: added.UUID,
		Output:         output,
	}); err != nil {
		return fmt.Errorf("failed to store metadata: %v", err)
	}

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
	case "completed", "successful", "no_changes", "partial":
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
	case "failed", "cancelled":
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

// dataSourceName returns a human-readable label for a data source:
// "bucket (region)" for Spaces sources, the base URL for web sources,
// or the UUID as a fallback.
func dataSourceName(ds *KBDataSourceInfo) string {
	if ds.BucketName != "" {
		if ds.Region != "" {
			return fmt.Sprintf("%s (%s)", ds.BucketName, ds.Region)
		}
		return ds.BucketName
	}
	if ds.WebCrawlerDataSource != nil && ds.WebCrawlerDataSource.BaseURL != "" {
		return ds.WebCrawlerDataSource.BaseURL
	}
	return ds.UUID
}

// hasAnyChunkSize reports whether any chunk size field is set in the spec.
func hasAnyChunkSize(ds DataSourceSpec) bool {
	return ds.MaxChunkSize > 0 || ds.ParentChunkSize > 0 || ds.ChildChunkSize > 0
}

// validateChunkSizesAgainstModel fetches the KB's embedding model and validates
// any configured chunk sizes against the model's min/max limits.
// Validation is skipped silently if the API is unreachable.
func validateChunkSizesAgainstModel(ctx core.SetupContext, kbID string, ds DataSourceSpec) error {
	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil
	}

	kb, err := client.GetKnowledgeBase(kbID)
	if err != nil {
		return nil
	}

	models, err := client.ListEmbeddingModels()
	if err != nil {
		return nil
	}

	for _, m := range models {
		if m.UUID == kb.EmbeddingModelUUID {
			return validateChunkSizeRange(ds, m.KBMinChunkSize, m.KBMaxChunkSize)
		}
	}

	return nil
}

// validateChunkSizeRange checks that each set chunk size falls within [min, max].
func validateChunkSizeRange(ds DataSourceSpec, min, max int) error {
	if ds.MaxChunkSize > 0 && (ds.MaxChunkSize < min || ds.MaxChunkSize > max) {
		return fmt.Errorf("maxChunkSize %d is out of range for the selected model (valid: %d–%d)", ds.MaxChunkSize, min, max)
	}
	if ds.ParentChunkSize > 0 && (ds.ParentChunkSize < min || ds.ParentChunkSize > max) {
		return fmt.Errorf("parentChunkSize %d is out of range for the selected model (valid: %d–%d)", ds.ParentChunkSize, min, max)
	}
	if ds.ChildChunkSize > 0 && (ds.ChildChunkSize < min || ds.ChildChunkSize > max) {
		return fmt.Errorf("childChunkSize %d is out of range for the selected model (valid: %d–%d)", ds.ChildChunkSize, min, max)
	}
	return nil
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
