package claude

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
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

	// modeSingle applies one prompt template to every element of Items: a
	// 1 x N matrix (one prompt, N requests).
	modeSingle = "single"
	// modeMultiple applies several prompt templates to every element of
	// Items: an M x N matrix (M prompts, N items, M*N requests).
	modeMultiple = "multiple"
)

type CreateBatchMessage struct{}

// BatchMessageItemSpec is a single resolved request in the batch.
type BatchMessageItemSpec struct {
	CustomID string `json:"customId" mapstructure:"customId"`
	Prompt   string `json:"prompt" mapstructure:"prompt"`
}

// BatchMessagePromptSpec is one prompt template in "multiple" mode. It's
// combined with every element of Items to produce one request per (prompt,
// item) pair.
type BatchMessagePromptSpec struct {
	ID             string `json:"id" mapstructure:"id"`
	PromptTemplate string `json:"promptTemplate" mapstructure:"promptTemplate"`
}

// BatchMessageSpec is the workflow node configuration for claude.createBatchMessage.
type BatchMessageSpec struct {
	Model         string   `json:"model" mapstructure:"model"`
	SystemMessage string   `json:"systemMessage" mapstructure:"systemMessage"`
	MaxTokens     int      `json:"maxTokens" mapstructure:"maxTokens"`
	Temperature   *float64 `json:"temperature" mapstructure:"temperature"`
	OutputSchema  string   `json:"outputSchema" mapstructure:"outputSchema"`

	// Mode selects how the batch's requests are built. See modeSingle / modeMultiple.
	Mode string `json:"mode" mapstructure:"mode"`

	// Items is a bare expression evaluating to an array; both modes build
	// requests by evaluating a prompt template per element, with `item` (the
	// element) and `index` (its position) bound as extra variables.
	Items string `json:"items" mapstructure:"items"`

	// Single mode: one prompt template, applied to every element of Items.
	PromptTemplate     string `json:"promptTemplate" mapstructure:"promptTemplate"`
	CustomIDExpression string `json:"customIdExpression" mapstructure:"customIdExpression"`

	// Multiple mode: several prompt templates, each applied to every element
	// of Items. Each request's Custom ID defaults to "{prompt.id}-{index+1}".
	Prompts []BatchMessagePromptSpec `json:"prompts" mapstructure:"prompts"`
}

// BatchMessageNodeMetadata is node-level metadata surfaced in the UI, mirroring
// TextPromptNodeMetadata so the shared frontend mapper can display it.
type BatchMessageNodeMetadata struct {
	Model            string `json:"model" mapstructure:"model"`
	MaxTokens        int    `json:"maxTokens" mapstructure:"maxTokens"`
	StructuredOutput bool   `json:"structuredOutput" mapstructure:"structuredOutput"`
}

// BatchExecutionMetadata is persisted for the run so poll() can find the batch
// again, and to surface live progress (RequestCounts) while it's still processing.
type BatchExecutionMetadata struct {
	BatchID       string                     `json:"batchId,omitempty" mapstructure:"batchId,omitempty"`
	Status        string                     `json:"status,omitempty" mapstructure:"status,omitempty"`
	RequestCounts *MessageBatchRequestCounts `json:"requestCounts,omitempty" mapstructure:"requestCounts,omitempty"`
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
	return `The Create Batch Message component uses [Anthropic's Message Batches API](https://platform.claude.com/docs/en/build-with-claude/batch-processing) to run one or more prompts over an array of inputs in a single batch, at a lower cost than issuing them individually.

Every batch is built from a matrix: an array of **Items** (the N data points, e.g. a list of pull requests) crossed with one or more prompt templates. **Mode** controls whether that's a single template (1 x N) or several (M x N).

## Use Cases

- **Update the same kind of resource, one at a time or many at once**: pass a one-element array to update a single pull request's title and description, or the full list to update every open pull request, with the same node.
- **Multiple derived outputs per item**: e.g. generate a title suggestion, a description suggestion, and a risk assessment for every pull request in one batch (Multiple Prompts mode).
- **Bulk classification or extraction**: Run the same prompt template over many inputs.
- **Cost-sensitive workloads**: Batches are billed at a discount versus the equivalent individual requests.

## How It Works

1. Evaluates **Items** to an array, then builds the batch's requests according to **Mode** (see Configuration below): one request per item (Single Prompt), or one request per (prompt, item) pair (Multiple Prompts).
2. Submits them as a single batch (` + "`POST /v1/messages/batches`" + `).
3. Polls the batch status until it reaches a terminal state.
4. Downloads the results and emits one entry per request, matched by **Custom ID**.

Batches typically complete within an hour, but can take up to 24 hours. This component polls with increasing backoff and keeps the execution open (without emitting) until the batch ends.

## Configuration

- **Items**: An expression that evaluates to the array of data points to run over, e.g. ` + "`$['List Open Pull Requests'].body`" + ` for many, or a one-element array (e.g. ` + "`[$['Get Pull Request'].body]`" + `) to run over just one.
- **Mode**: How many prompt templates are applied to each item.
  - **Single Prompt**: one prompt template, applied to every element of Items (1 x N requests).
  - **Multiple Prompts**: several prompt templates, each applied to every element of Items (M x N requests).
- **Model**: The Claude model used for every request in the batch.
- **System Message**: (Optional) Context applied to every request in the batch.
- **Max Tokens**: (Optional) Limit the length of each generated response.
- **Temperature**: (Optional) Control randomness (0.0 to 1.0), applied to every request.
- **Structured Output**: (Optional) A JSON Schema every response must conform to.

**Single Prompt mode:**
- **Prompt Template**: An expression evaluated once per element of Items to build that element's prompt, with ` + "`item`" + ` (the element) and ` + "`index`" + ` (its zero-based position) available, e.g. ` + "`\"Update the title and description for PR #\" + string(item.number) + \": \" + item.title`" + `. This is a bare expression, not a ` + "`{{ }}`" + ` template — wrap string literals in quotes and use ` + "`+`" + ` to build up the text, since it needs to run once per item.
- **Custom ID Expression**: (Optional, advanced) Same idea as Prompt Template — an expression with ` + "`item`" + `/` + "`index`" + ` available, evaluated per element to compute its Custom ID. Defaults to auto-numbered IDs (` + "`request-1`" + `, ` + "`request-2`" + `, ...).

**Multiple Prompts mode:**
- **Prompts**: A short, manually-authored list of prompt templates (each with an **ID** and a **Prompt Template** expression, using ` + "`item`" + `/` + "`index`" + ` the same way as Single Prompt mode above). Each one is evaluated once per element of Items. A request's Custom ID defaults to ` + "`{id}-{index+1}`" + `, e.g. ` + "`title-suggestion-1`" + `.

## Output

Emits a single payload once the batch ends, containing:
- **status**: ` + "`ended`" + `, ` + "`timeout`" + `, or ` + "`error`" + ` (the latter two only if polling could not confirm completion).
- **batchId**: The Anthropic batch ID.
- **requestCounts**: How many requests succeeded, errored, were canceled, or expired.
- **results**: One entry per request (by **Custom ID**), with the generated text, stop reason, token usage, and any error.

## Notes

- Requires a valid Claude API key configured in the integration.
- Custom IDs must be unique within the batch (max 64 characters).
- A batch can contain up to ` + fmt.Sprintf("%d", maxBatchRequests) + ` requests (prompts x items, in Multiple Prompts mode).
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
	singleVisible := []configuration.VisibilityCondition{{Field: "mode", Values: []string{modeSingle}}}
	singleRequired := []configuration.RequiredCondition{{Field: "mode", Values: []string{modeSingle}}}
	multipleVisible := []configuration.VisibilityCondition{{Field: "mode", Values: []string{modeMultiple}}}
	multipleRequired := []configuration.RequiredCondition{{Field: "mode", Values: []string{modeMultiple}}}

	return []configuration.Field{
		{
			Name:        "items",
			Label:       "Items",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Placeholder: `$['List Open Pull Requests'].body`,
			Description: "Expression that evaluates to the array of data points to run the prompt(s) over. Use a one-element array to run over just a single item.",
		},
		{
			Name:        "mode",
			Label:       "Mode",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     modeSingle,
			Description: "How many prompt templates are applied to each element of Items.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Single Prompt", Value: modeSingle, Description: "One prompt template, applied to every item (1 x N requests)"},
						{Label: "Multiple Prompts", Value: modeMultiple, Description: "Several prompt templates, each applied to every item (M x N requests)"},
					},
				},
			},
		},
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
			Name:                 "promptTemplate",
			Label:                "Prompt Template",
			Type:                 configuration.FieldTypeExpression,
			Placeholder:          `"Update the title and description for PR #" + string(item.number) + ": " + item.title + "\n\n" + item.body`,
			Description:          "Expression evaluated once per element of Items to build that request's prompt, with `item` (the element) and `index` (its zero-based position) available. This is a bare expression, not a `{{ }}` template: wrap string literals in quotes and use `+` to build up the text.",
			VisibilityConditions: singleVisible,
			RequiredConditions:   singleRequired,
		},
		{
			Name:                 "customIdExpression",
			Label:                "Custom ID Expression",
			Type:                 configuration.FieldTypeExpression,
			Togglable:            true,
			Placeholder:          `string(item.number)`,
			Description:          fmt.Sprintf("Optional: expression evaluated once per element of Items (with the same `item`/`index` variables as Prompt Template) to compute its Custom ID. Defaults to auto-numbered IDs (request-1, request-2, ...). A batch can contain up to %d requests.", maxBatchRequests),
			VisibilityConditions: singleVisible,
		},
		{
			Name:                 "prompts",
			Label:                "Prompts",
			Type:                 configuration.FieldTypeList,
			Description:          "Each prompt template is evaluated once per element of Items, so this list produces (prompts x items) requests.",
			VisibilityConditions: multipleVisible,
			RequiredConditions:   multipleRequired,
			Default: []map[string]any{
				{"id": "prompt-1", "promptTemplate": ""},
			},
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel:   "Prompt",
					Reorderable: true,
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "id",
								Label:       "ID",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Placeholder: "title-suggestion",
								Description: "Short identifier for this prompt. Used to build each of its requests' Custom ID, e.g. \"title-suggestion-1\".",
							},
							{
								Name:        "promptTemplate",
								Label:       "Prompt Template",
								Type:        configuration.FieldTypeExpression,
								Required:    true,
								Placeholder: `"Suggest a title for PR #" + string(item.number) + ":\n\n" + item.body`,
								Description: "Expression evaluated once per element of Items to build this prompt's text for that element, with `item`/`index` available. A bare expression, not a `{{ }}` template.",
							},
						},
					},
				},
			},
		},
	}
}

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

	// If a batch was already submitted for this execution (Execute is being
	// retried after the batch was created but the run didn't finish, e.g. the
	// process crashed before scheduling/handling the first poll), resume
	// polling it instead of submitting a duplicate batch.
	existing := BatchExecutionMetadata{}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &existing); err == nil && existing.BatchID != "" {
		ctx.Logger.Infof("Message batch %s was already created for this execution; resuming instead of creating a new one", existing.BatchID)
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{"attempt": 1, "errors": 0}, batchInitialPoll)
	}

	items, err := resolveBatchRequests(ctx.Expressions, spec)
	if err != nil {
		return err
	}
	if err := validateRequestItems(items); err != nil {
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

	requestItems := buildBatchRequestItems(items, spec, schema)

	batch, err := client.CreateMessageBatch(requestItems)
	if err != nil {
		return fmt.Errorf("failed to create message batch: %w", err)
	}

	ctx.Logger.Infof("Created Claude message batch %s with %d requests", batch.ID, len(requestItems))

	metadata := BatchExecutionMetadata{BatchID: batch.ID, Status: batch.ProcessingStatus, RequestCounts: &batch.RequestCounts}
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

	_ = metadataWriter.Set(BatchExecutionMetadata{BatchID: batch.ID, Status: batch.ProcessingStatus, RequestCounts: &batch.RequestCounts})
	return nil
}

func decodeBatchMessageSpec(config any) (BatchMessageSpec, error) {
	var spec BatchMessageSpec
	if err := mapstructure.Decode(config, &spec); err != nil {
		return spec, fmt.Errorf("failed to decode configuration: %w", err)
	}
	return spec, nil
}

// validateBatchMessageSpec validates everything that's known at design time.
// The expression fields themselves (Items/PromptTemplate/CustomIDExpression)
// are only fully validated once evaluated, in resolveBatchRequests /
// validateRequestItems.
func validateBatchMessageSpec(spec BatchMessageSpec) error {
	if strings.TrimSpace(spec.Model) == "" {
		return fmt.Errorf("model is required")
	}

	if spec.MaxTokens < 0 {
		return fmt.Errorf("maxTokens must be at least 1")
	}

	if strings.TrimSpace(spec.Items) == "" {
		return fmt.Errorf("items is required")
	}

	switch spec.Mode {
	case modeMultiple:
		if len(spec.Prompts) == 0 {
			return fmt.Errorf("at least one prompt is required in \"Multiple Prompts\" mode")
		}
		for i, p := range spec.Prompts {
			if strings.TrimSpace(p.ID) == "" {
				return fmt.Errorf("prompts[%d].id is required", i)
			}
			if strings.TrimSpace(p.PromptTemplate) == "" {
				return fmt.Errorf("prompts[%d].promptTemplate is required", i)
			}
		}
	default:
		if strings.TrimSpace(spec.PromptTemplate) == "" {
			return fmt.Errorf("promptTemplate is required in \"Single Prompt\" mode")
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

// validateRequestItems validates the resolved list of batch requests, whether
// it came from single mode (one item) or multiple mode (one per array element).
func validateRequestItems(items []BatchMessageItemSpec) error {
	if len(items) == 0 {
		return fmt.Errorf("at least one request is required")
	}
	if len(items) > maxBatchRequests {
		return fmt.Errorf("a batch cannot contain more than %d requests", maxBatchRequests)
	}

	seen := make(map[string]bool, len(items))
	for i, r := range items {
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

	return nil
}

// resolveBatchRequests evaluates Items to an array, then builds the batch's
// requests according to spec.Mode: one prompt template applied to every
// element (single, 1 x N), or several applied to every element (multiple,
// M x N).
func resolveBatchRequests(expressions core.ExpressionContext, spec BatchMessageSpec) ([]BatchMessageItemSpec, error) {
	elements, err := resolveItems(expressions, spec.Items)
	if err != nil {
		return nil, err
	}

	if spec.Mode == modeMultiple {
		return resolveMultiplePrompts(expressions, spec.Prompts, elements)
	}
	return resolveSinglePrompt(expressions, spec, elements)
}

// resolveItems evaluates Items to an array of elements.
func resolveItems(expressions core.ExpressionContext, itemsExpr string) ([]any, error) {
	raw, err := expressions.Run(itemsExpr)
	if err != nil {
		return nil, fmt.Errorf("items: %w", err)
	}
	elements, err := toAnySlice(raw)
	if err != nil {
		return nil, fmt.Errorf("items must evaluate to an array: %w", err)
	}
	return elements, nil
}

// resolveSinglePrompt builds one request per element, by evaluating
// PromptTemplate (and, optionally, CustomIDExpression) with `item`/`index`
// bound as extra variables: a 1 x N matrix.
func resolveSinglePrompt(expressions core.ExpressionContext, spec BatchMessageSpec, elements []any) ([]BatchMessageItemSpec, error) {
	if len(elements) > maxBatchRequests {
		return nil, fmt.Errorf("items has %d elements; a batch cannot contain more than %d requests", len(elements), maxBatchRequests)
	}

	items := make([]BatchMessageItemSpec, 0, len(elements))
	for i, element := range elements {
		vars := map[string]any{"item": element, "index": i}

		prompt, err := evalPromptTemplate(expressions, spec.PromptTemplate, vars, i)
		if err != nil {
			return nil, err
		}

		customID := fmt.Sprintf("request-%d", i+1)
		if strings.TrimSpace(spec.CustomIDExpression) != "" {
			idResult, err := expressions.RunWithExtraVariables(spec.CustomIDExpression, vars)
			if err != nil {
				return nil, fmt.Errorf("customIdExpression (item %d): %w", i, err)
			}
			if idResult != nil {
				if s := strings.TrimSpace(fmt.Sprintf("%v", idResult)); s != "" {
					customID = s
				}
			}
		}

		items = append(items, BatchMessageItemSpec{CustomID: customID, Prompt: prompt})
	}

	return items, nil
}

// resolveMultiplePrompts builds one request per (prompt, item) pair: an
// M x N matrix. Each request's Custom ID defaults to "{prompt.id}-{index+1}".
func resolveMultiplePrompts(expressions core.ExpressionContext, prompts []BatchMessagePromptSpec, elements []any) ([]BatchMessageItemSpec, error) {
	total := len(prompts) * len(elements)
	if total > maxBatchRequests {
		return nil, fmt.Errorf("prompts (%d) x items (%d) = %d requests; a batch cannot contain more than %d requests", len(prompts), len(elements), total, maxBatchRequests)
	}

	items := make([]BatchMessageItemSpec, 0, total)
	for _, p := range prompts {
		id := strings.TrimSpace(p.ID)
		if id == "" {
			return nil, fmt.Errorf("prompts[].id is required")
		}

		for i, element := range elements {
			vars := map[string]any{"item": element, "index": i}

			prompt, err := evalPromptTemplate(expressions, p.PromptTemplate, vars, i)
			if err != nil {
				return nil, fmt.Errorf("prompts[%s]: %w", id, err)
			}

			items = append(items, BatchMessageItemSpec{
				CustomID: fmt.Sprintf("%s-%d", id, i+1),
				Prompt:   prompt,
			})
		}
	}

	return items, nil
}

// evalPromptTemplate evaluates a prompt template expression for one element,
// with `item`/`index` bound as extra variables.
func evalPromptTemplate(expressions core.ExpressionContext, template string, vars map[string]any, index int) (string, error) {
	result, err := expressions.RunWithExtraVariables(template, vars)
	if err != nil {
		return "", fmt.Errorf("promptTemplate (item %d): %w", index, err)
	}
	prompt, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("promptTemplate must evaluate to a string, got %T (item %d)", result, index)
	}
	return prompt, nil
}

// toAnySlice normalizes an expression result into a []any, the same way
// pkg/components/foreach does for its arrayExpression field.
func toAnySlice(v any) ([]any, error) {
	if v == nil {
		return nil, fmt.Errorf("got nil")
	}
	if s, ok := v.([]any); ok {
		return s, nil
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice {
		return nil, fmt.Errorf("got %T", v)
	}
	out := make([]any, rv.Len())
	for i := range out {
		out[i] = rv.Index(i).Interface()
	}
	return out, nil
}

func buildBatchRequestItems(items []BatchMessageItemSpec, spec BatchMessageSpec, schema map[string]any) []CreateMessageBatchRequestItem {
	out := make([]CreateMessageBatchRequestItem, 0, len(items))
	for _, r := range items {
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
		out = append(out, CreateMessageBatchRequestItem{
			CustomID: strings.TrimSpace(r.CustomID),
			Params:   params,
		})
	}
	return out
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
