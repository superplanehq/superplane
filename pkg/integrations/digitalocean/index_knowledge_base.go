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

const indexPollInterval = 30 * time.Second

type IndexKnowledgeBase struct{}

type IndexKnowledgeBaseSpec struct {
	KnowledgeBase string `json:"knowledgeBase" mapstructure:"knowledgeBase"`
}

func (i *IndexKnowledgeBase) Name() string {
	return "digitalocean.indexKnowledgeBase"
}

func (i *IndexKnowledgeBase) Label() string {
	return "Index Knowledge Base"
}

func (i *IndexKnowledgeBase) Description() string {
	return "Start an indexing job for a DigitalOcean Gradient AI knowledge base and wait for it to complete"
}

func (i *IndexKnowledgeBase) Documentation() string {
	return `The Index Knowledge Base component triggers a new indexing job on an existing knowledge base and polls until it completes.

## How it works

Starts an indexing job that re-processes all data sources attached to the knowledge base. The component polls the job status every 30 seconds until the indexing job finishes successfully or fails.

## Use Cases

- **Content refresh**: Re-index a knowledge base after updating files in a Spaces bucket or changing a website
- **Scheduled re-indexing**: Combine with a Schedule trigger to re-index on a regular cadence (e.g. nightly)
- **Pipeline orchestration**: Re-index after an upstream component adds or modifies data sources

## Output

Returns the completed indexing job details:
- **knowledgeBaseUUID**: UUID of the knowledge base
- **knowledgeBaseName**: Name of the knowledge base
- **jobUUID**: UUID of the indexing job
- **status**: Final job status (e.g. INDEX_JOB_STATUS_COMPLETED)
- **phase**: Final job phase (e.g. BATCH_JOB_PHASE_SUCCEEDED)
- **totalTokens**: Total tokens consumed by the indexing job
- **completedDataSources**: Number of data sources that finished indexing
- **totalDataSources**: Total number of data sources
- **startedAt**, **finishedAt**: Timing information
- **isReportAvailable**: Whether a detailed indexing report is available`
}

func (i *IndexKnowledgeBase) Icon() string {
	return "brain"
}

func (i *IndexKnowledgeBase) Color() string {
	return "blue"
}

func (i *IndexKnowledgeBase) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (i *IndexKnowledgeBase) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "knowledgeBase",
			Label:       "Knowledge Base",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "Select a knowledge base",
			Description: "The knowledge base to index. When using an expression, provide the knowledge base UUID.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "knowledge_base",
				},
			},
		},
	}
}

func (i *IndexKnowledgeBase) Setup(ctx core.SetupContext) error {
	spec := IndexKnowledgeBaseSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.KnowledgeBase == "" {
		return errors.New("knowledgeBase is required")
	}

	if err := resolveIndexKBMetadata(ctx, spec.KnowledgeBase); err != nil {
		return fmt.Errorf("error resolving metadata: %v", err)
	}

	return nil
}

// indexKBMetadata is stored between poll ticks
type indexKBMetadata struct {
	KBUUID string `json:"kbUUID" mapstructure:"kbUUID"`
	KBName string `json:"kbName" mapstructure:"kbName"`
	JobID  string `json:"jobId" mapstructure:"jobId"`
}

func (i *IndexKnowledgeBase) Execute(ctx core.ExecutionContext) error {
	spec := IndexKnowledgeBaseSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
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

	job, err := client.StartIndexingJob(spec.KnowledgeBase)
	if err != nil {
		return fmt.Errorf("failed to start indexing job: %v", err)
	}

	if err := ctx.Metadata.Set(indexKBMetadata{
		KBUUID: spec.KnowledgeBase,
		KBName: kbName,
		JobID:  job.UUID,
	}); err != nil {
		return fmt.Errorf("failed to store metadata: %v", err)
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, indexPollInterval)
}

func (i *IndexKnowledgeBase) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (i *IndexKnowledgeBase) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (i *IndexKnowledgeBase) Actions() []core.Action {
	return []core.Action{
		{
			Name:           "poll",
			UserAccessible: false,
		},
	}
}

func (i *IndexKnowledgeBase) HandleAction(ctx core.ActionContext) error {
	if ctx.Name != "poll" {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}

	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var meta indexKBMetadata
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
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, indexPollInterval)
	}

	job := kb.LastIndexingJob
	state := indexingJobState(job.Status)
	switch state {
	case "completed", "successful", "no_changes":
		output := buildIndexJobOutput(meta, job)
		return ctx.ExecutionState.Emit(
			core.DefaultOutputChannel.Name,
			"digitalocean.knowledge_base.indexed",
			[]any{output},
		)
	case "running", "pending":
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, indexPollInterval)
	case "failed", "cancelled", "partial":
		return fmt.Errorf("indexing job %s for knowledge base %s: %s", job.UUID, meta.KBUUID, job.Status)
	default:
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, indexPollInterval)
	}
}

// indexingJobState normalises a DO indexing job status to a simple lowercase keyword.
// Extends the base indexJobState with "no_change" for re-indexing scenarios
// where data sources haven't changed.
func indexingJobState(status string) string {
	lower := strings.ToLower(status)
	for _, state := range []string{"completed", "successful", "no_changes", "partial", "running", "pending", "failed", "cancelled"} {
		if strings.HasSuffix(lower, state) {
			return state
		}
	}
	return lower
}

func buildIndexJobOutput(meta indexKBMetadata, job *IndexJob) map[string]any {
	return map[string]any{
		"knowledgeBaseUUID":    meta.KBUUID,
		"knowledgeBaseName":    meta.KBName,
		"jobUUID":              job.UUID,
		"status":               job.Status,
		"phase":                job.Phase,
		"totalTokens":          job.TotalTokens,
		"completedDataSources": job.CompletedDataSources,
		"totalDataSources":     job.TotalDataSources,
		"startedAt":            job.StartedAt,
		"finishedAt":           job.FinishedAt,
		"isReportAvailable":    job.IsReportAvailable,
	}
}

func (i *IndexKnowledgeBase) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (i *IndexKnowledgeBase) Cleanup(ctx core.SetupContext) error {
	return nil
}

// IndexKBNodeMetadata stores metadata about a knowledge base for display in the UI
type IndexKBNodeMetadata struct {
	KnowledgeBaseID   string `json:"knowledgeBaseId" mapstructure:"knowledgeBaseId"`
	KnowledgeBaseName string `json:"knowledgeBaseName" mapstructure:"knowledgeBaseName"`
}

// resolveIndexKBMetadata fetches the knowledge base name from the API and stores it in metadata
func resolveIndexKBMetadata(ctx core.SetupContext, kbID string) error {
	if strings.Contains(kbID, "{{") {
		return ctx.Metadata.Set(IndexKBNodeMetadata{
			KnowledgeBaseID:   kbID,
			KnowledgeBaseName: kbID,
		})
	}

	var existing IndexKBNodeMetadata
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

	return ctx.Metadata.Set(IndexKBNodeMetadata{
		KnowledgeBaseID:   kbID,
		KnowledgeBaseName: kb.Name,
	})
}
