package claude

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/configuration/structuredoutput"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	CreateBatchMessagePayloadType = "claude.createBatchMessage.result"

	// maxBatchRequests caps the number of requests configurable per batch node.
	// Anthropic's own limit is 100,000 requests (or 256MB); this is a much
	// lower, sane default for a manually-authored workflow configuration.
	maxBatchRequests = 1000

	batchStatusEnded = "ended"
)

type CreateBatchMessage struct{}

// BatchMessageItemSpec is a single prompt configured as part of the batch.
type BatchMessageItemSpec struct {
	CustomID string `json:"customId" mapstructure:"customId"`
	Prompt   string `json:"prompt" mapstructure:"prompt"`
}

// BatchMessageSpec is the workflow node configuration for claude.createBatchMessage.
type BatchMessageSpec struct {
	Model         string                 `json:"model" mapstructure:"model"`
	SystemMessage string                 `json:"systemMessage" mapstructure:"systemMessage"`
	MaxTokens     int                    `json:"maxTokens" mapstructure:"maxTokens"`
	Temperature   *float64               `json:"temperature" mapstructure:"temperature"`
	OutputSchema  string                 `json:"outputSchema" mapstructure:"outputSchema"`
	Requests      []BatchMessageItemSpec `json:"requests" mapstructure:"requests"`
}

// BatchMessageNodeMetadata is node-level metadata surfaced in the UI, mirroring
// TextPromptNodeMetadata so the shared frontend mapper can display it.
type BatchMessageNodeMetadata struct {
	Model            string `json:"model" mapstructure:"model"`
	MaxTokens        int    `json:"maxTokens" mapstructure:"maxTokens"`
	StructuredOutput bool   `json:"structuredOutput" mapstructure:"structuredOutput"`
}

// BatchExecutionMetadata is persisted for the run so poll() can find the batch again.
type BatchExecutionMetadata struct {
	BatchID string `json:"batchId,omitempty" mapstructure:"batchId,omitempty"`
	Status  string `json:"status,omitempty" mapstructure:"status,omitempty"`
}

// BatchItemResult is the per-request outcome surfaced in the emitted payload.
type BatchItemResult struct {
	CustomID     string        `json:"customId"`
	Type         string        `json:"type"` // succeeded | errored | canceled | expired
	Text         string        `json:"text,omitempty"`
	Parsed       any           `json:"parsed,omitempty"`
	StopReason   string        `json:"stopReason,omitempty"`
	Usage        *MessageUsage `json:"usage,omitempty"`
	ErrorType    string        `json:"errorType,omitempty"`
	ErrorMessage string        `json:"errorMessage,omitempty"`
}

// BatchOutput is the payload emitted when the batch reaches a terminal state.
type BatchOutput struct {
	Status        string                    `json:"status"` // ended | timeout | error
	BatchID       string                    `json:"batchId"`
	RequestCounts MessageBatchRequestCounts `json:"requestCounts"`
	Results       []BatchItemResult         `json:"results,omitempty"`
}

func (c *CreateBatchMessage) Name() string {
	return "claude.createBatchMessage"
}

func (c *CreateBatchMessage) Label() string {
	return "Create Batch Message"
}

func (c *CreateBatchMessage) Description() string {
	return "Sends many prompts to Claude in a single Message Batches request, at lower cost than individual calls"
}

func (c *CreateBatchMessage) Documentation() string {
	return `The Create Batch Message component uses [Anthropic's Message Batches API](https://platform.claude.com/docs/en/build-with-claude/batch-processing) to process many prompts asynchronously in one request, at a lower cost than issuing them individually.

## Use Cases

- **Bulk classification or extraction**: Run the same prompt template over many inputs.
- **Large-scale content generation**: Draft many summaries, descriptions, or messages at once.
- **Cost-sensitive workloads**: Batches are billed at a discount versus the equivalent individual requests.

## How It Works

1. Submits all configured requests as a single batch (` + "`POST /v1/messages/batches`" + `).
2. Polls the batch status until it reaches a terminal state.
3. Downloads the results and emits one entry per request, matched by **Custom ID**.

Batches typically complete within an hour, but can take up to 24 hours. This component polls with increasing backoff and keeps the execution open (without emitting) until the batch ends.

## Configuration

- **Model**: The Claude model used for every request in the batch.
- **System Message**: (Optional) Context applied to every request in the batch.
- **Max Tokens**: (Optional) Limit the length of each generated response.
- **Temperature**: (Optional) Control randomness (0.0 to 1.0), applied to every request.
- **Structured Output**: (Optional) A JSON Schema every response must conform to.
- **Requests**: The list of prompts to send. Each has a **Custom ID** (unique within the batch, used to match it to its result) and a **Prompt**.

## Output

Emits a single payload once the batch ends, containing:
- **status**: ` + "`ended`" + `, ` + "`timeout`" + `, or ` + "`error`" + ` (the latter two only if polling could not confirm completion).
- **batchId**: The Anthropic batch ID.
- **requestCounts**: How many requests succeeded, errored, were canceled, or expired.
- **results**: One entry per request (by **Custom ID**), with the generated text, stop reason, token usage, and any error.

## Notes

- Requires a valid Claude API key configured in the integration.
- Custom IDs must be unique within the batch (max 64 characters).
- Cancelling the workflow execution requests cancellation of the batch on Anthropic's side; requests already completed are unaffected.`
}

func (c *CreateBatchMessage) Icon() string {
	return "layers"
}

func (c *CreateBatchMessage) Color() string {
	return "#D97757"
}

func (c *CreateBatchMessage) ExampleOutput() map[string]any {
	return map[string]any{
		"data": BatchOutput{
			Status:  batchStatusEnded,
			BatchID: "msgbatch_01HkcTjaV5uDC8jWR4ZsqFqz",
			RequestCounts: MessageBatchRequestCounts{
				Processing: 0,
				Succeeded:  2,
				Errored:    0,
				Canceled:   0,
				Expired:    0,
			},
			Results: []BatchItemResult{
				{
					CustomID:   "request-1",
					Type:       "succeeded",
					Text:       "Paris is the capital of France.",
					StopReason: "end_turn",
					Usage:      &MessageUsage{InputTokens: 12, OutputTokens: 9},
				},
				{
					CustomID:   "request-2",
					Type:       "succeeded",
					Text:       "Berlin is the capital of Germany.",
					StopReason: "end_turn",
					Usage:      &MessageUsage{InputTokens: 12, OutputTokens: 9},
				},
			},
		},
		"timestamp": "2026-07-07T12:00:00Z",
		"type":      CreateBatchMessagePayloadType,
	}
}

func (c *CreateBatchMessage) OutputChannels(config any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateBatchMessage) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "model",
			Label:       "Model",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Default:     "claude-opus-4-6",
			Placeholder: "Select a Claude model",
			Description: "Model used for every request in the batch",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "model",
				},
			},
		},
		{
			Name:        "systemMessage",
			Label:       "System Message",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Placeholder: "e.g. You are a concise DevOps assistant",
			Description: "Optional context applied to every request in the batch",
		},
		{
			Name:        "maxTokens",
			Label:       "Max Tokens",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     "4096",
			Description: "Maximum number of tokens to generate per request. Defaults to 4096.",
		},
		{
			Name:        "temperature",
			Label:       "Temperature",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     "1.0",
			Description: "Amount of randomness injected into each response (0.0 to 1.0)",
		},
		structuredoutput.ConfigField(
			"outputSchema",
			"Structured Output",
			"A JSON Schema describing every response. Claude is constrained to return JSON matching it (available on each result's `parsed` field). Every object gets `additionalProperties: false`.",
		),
		{
			Name:        "requests",
			Label:       "Requests",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Description: fmt.Sprintf("Prompts to send as one batch (up to %d). Each needs a unique Custom ID used to match it to its result.", maxBatchRequests),
			Default: []map[string]any{
				{"customId": "request-1", "prompt": ""},
			},
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel:   "Request",
					Reorderable: true,
					MaxItems:    intPtr(maxBatchRequests),
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "customId",
								Label:       "Custom ID",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Placeholder: "request-1",
								Description: "Unique identifier for this request within the batch (max 64 characters). Used to match it to its result.",
							},
							{
								Name:        "prompt",
								Label:       "Prompt",
								Type:        configuration.FieldTypeText,
								Required:    true,
								Placeholder: "Enter the user prompt",
								Description: "The user message for this request",
							},
						},
					},
				},
			},
		},
	}
}

func intPtr(v int) *int { return &v }

func (c *CreateBatchMessage) Setup(ctx core.SetupContext) error {
	spec, err := decodeBatchMessageSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateBatchMessageSpec(spec); err != nil {
		return err
	}

	if ctx.Metadata != nil {
		maxTokens := spec.MaxTokens
		if maxTokens == 0 {
			maxTokens = 4096
		}
		hasSchema := strings.TrimSpace(spec.OutputSchema) != ""
		_ = ctx.Metadata.Set(BatchMessageNodeMetadata{
			Model:            spec.Model,
			MaxTokens:        maxTokens,
			StructuredOutput: hasSchema,
		})
	}

	return nil
}

func (c *CreateBatchMessage) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateBatchMessage) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeBatchMessageSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateBatchMessageSpec(spec); err != nil {
		return err
	}

	if spec.MaxTokens == 0 {
		spec.MaxTokens = 4096
	}

	schema, err := structuredoutput.Parse(spec.OutputSchema)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	items := buildBatchRequestItems(spec, schema)

	batch, err := client.CreateMessageBatch(items)
	if err != nil {
		return fmt.Errorf("failed to create message batch: %w", err)
	}

	ctx.Logger.Infof("Created Claude message batch %s with %d requests", batch.ID, len(items))

	metadata := BatchExecutionMetadata{BatchID: batch.ID, Status: batch.ProcessingStatus}
	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to set execution metadata: %w", err)
	}

	if batch.ProcessingStatus == batchStatusEnded {
		return c.finishBatch(client, ctx.ExecutionState, ctx.Metadata, batch, len(schema) > 0)
	}

	ctx.Logger.Infof("Waiting for batch %s to complete (polling)...", batch.ID)
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{"attempt": 1, "errors": 0}, batchInitialPoll)
}

func (c *CreateBatchMessage) Cancel(ctx core.ExecutionContext) error {
	metadata := BatchExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return nil
	}
	if metadata.BatchID == "" {
		return nil
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil
	}

	if err := client.CancelMessageBatch(metadata.BatchID); err != nil {
		ctx.Logger.Warnf("Failed to cancel message batch %s: %v", metadata.BatchID, err)
	} else {
		ctx.Logger.Infof("Requested cancellation of message batch %s", metadata.BatchID)
	}
	return nil
}

func (c *CreateBatchMessage) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateBatchMessage) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

// finishBatch fetches results for an already-ended batch and emits the payload.
func (c *CreateBatchMessage) finishBatch(client *Client, state core.ExecutionStateContext, metadataWriter core.MetadataWriter, batch *MessageBatch, hasSchema bool) error {
	results, err := client.GetMessageBatchResults(batch.ID)
	if err != nil {
		return fmt.Errorf("failed to fetch batch results: %w", err)
	}

	out := buildBatchOutput(batchStatusEnded, batch, results, hasSchema)
	if err := state.Emit(core.DefaultOutputChannel.Name, CreateBatchMessagePayloadType, []any{out}); err != nil {
		return err
	}

	_ = metadataWriter.Set(BatchExecutionMetadata{BatchID: batch.ID, Status: batch.ProcessingStatus})
	return nil
}

func decodeBatchMessageSpec(config any) (BatchMessageSpec, error) {
	var spec BatchMessageSpec
	if err := mapstructure.Decode(config, &spec); err != nil {
		return spec, fmt.Errorf("failed to decode configuration: %w", err)
	}
	return spec, nil
}

func validateBatchMessageSpec(spec BatchMessageSpec) error {
	if strings.TrimSpace(spec.Model) == "" {
		return fmt.Errorf("model is required")
	}

	if spec.MaxTokens < 0 {
		return fmt.Errorf("maxTokens must be at least 1")
	}

	if len(spec.Requests) == 0 {
		return fmt.Errorf("at least one request is required")
	}
	if len(spec.Requests) > maxBatchRequests {
		return fmt.Errorf("a batch cannot contain more than %d requests", maxBatchRequests)
	}

	seen := make(map[string]bool, len(spec.Requests))
	for i, r := range spec.Requests {
		id := strings.TrimSpace(r.CustomID)
		if id == "" {
			return fmt.Errorf("requests[%d].customId is required", i)
		}
		if len(id) > 64 {
			return fmt.Errorf("requests[%d].customId must be at most 64 characters", i)
		}
		if seen[id] {
			return fmt.Errorf("requests[%d].customId %q is duplicated; custom IDs must be unique within a batch", i, id)
		}
		seen[id] = true

		if strings.TrimSpace(r.Prompt) == "" {
			return fmt.Errorf("requests[%d].prompt is required", i)
		}
	}

	hasSchema := strings.TrimSpace(spec.OutputSchema) != ""
	if hasSchema && !strings.Contains(spec.OutputSchema, "{{") {
		if _, err := structuredoutput.Parse(spec.OutputSchema); err != nil {
			return err
		}
	}

	return nil
}

func buildBatchRequestItems(spec BatchMessageSpec, schema map[string]any) []CreateMessageBatchRequestItem {
	items := make([]CreateMessageBatchRequestItem, 0, len(spec.Requests))
	for _, r := range spec.Requests {
		params := BatchRequestParams{
			Model:       spec.Model,
			MaxTokens:   spec.MaxTokens,
			Temperature: spec.Temperature,
			Messages: []Message{
				{Role: "user", Content: r.Prompt},
			},
		}
		if spec.SystemMessage != "" {
			params.System = spec.SystemMessage
		}
		if schema != nil {
			params.OutputConfig = &OutputConfig{
				Format: &OutputFormat{Type: "json_schema", Schema: structuredoutput.Prepare(schema, false)},
			}
		}
		items = append(items, CreateMessageBatchRequestItem{
			CustomID: strings.TrimSpace(r.CustomID),
			Params:   params,
		})
	}
	return items
}

// buildBatchOutput assembles the emitted payload from a batch and its (possibly
// nil, for timeout/error statuses) results.
func buildBatchOutput(status string, batch *MessageBatch, results []MessageBatchResult, hasSchema bool) BatchOutput {
	out := BatchOutput{Status: status}
	if batch != nil {
		out.BatchID = batch.ID
		out.RequestCounts = batch.RequestCounts
	}

	if len(results) == 0 {
		return out
	}

	out.Results = make([]BatchItemResult, 0, len(results))
	for _, r := range results {
		item := BatchItemResult{CustomID: r.CustomID, Type: r.Result.Type}

		switch r.Result.Type {
		case "succeeded":
			if r.Result.Message != nil {
				item.Text = extractMessageText(r.Result.Message)
				item.StopReason = r.Result.Message.StopReason
				item.Usage = &r.Result.Message.Usage

				if hasSchema && item.StopReason == "end_turn" && item.Text != "" {
					var parsed any
					if err := json.Unmarshal([]byte(item.Text), &parsed); err == nil {
						item.Parsed = parsed
					}
				}
			}
		case "errored":
			if r.Result.Error != nil {
				item.ErrorType = r.Result.Error.Type
				item.ErrorMessage = r.Result.Error.Message
			}
		}

		out.Results = append(out.Results, item)
	}

	return out
}

func (c *CreateBatchMessage) Hooks() []core.Hook {
	return []core.Hook{{
		Name: "poll",
		Type: core.HookTypeInternal,
	}}
}
