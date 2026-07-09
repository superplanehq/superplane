package claude

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/configuration/structuredoutput"
	"github.com/superplanehq/superplane/pkg/core"
)

// promptIDPattern mirrors Anthropic's custom_id constraint (the ID is embedded
// in the batch's internal custom ID, see customIDItemPrefix/customIDPromptSep).
var promptIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

const (
	CreateBatchMessagePayloadType = "claude.createBatchMessage.result"

	// maxBatchRequests caps the number of requests configurable per batch node.
	// Anthropic's own limit is 100,000 requests (or 256MB); this is a much
	// lower, sane default for a manually-authored workflow configuration.
	maxBatchRequests = 1000

	// defaultMaxTokens is used for every request; it isn't user-configurable.
	defaultMaxTokens = 4096

	batchStatusEnded = "ended"

	// modeSingle applies one prompt template to every element of Items: a
	// 1 x N matrix (one prompt, N requests).
	modeSingle = "single"
	// modeMultiple applies several prompt templates to every element of
	// Items: an M x N matrix (M prompts, N items, M*N requests).
	modeMultiple = "multiple"

	// customIDItemPrefix/customIDPromptSep encode (item index, prompt id) into
	// the batch's internal, non-user-facing custom IDs, so results can be
	// regrouped by item (and, in multiple mode, by prompt) once the batch ends.
	// Anthropic requires custom_id to match ^[a-zA-Z0-9_-]{1,64}$, so the
	// separator (and prompt IDs) must stick to that character set; a plain
	// hyphen works since the item index is always numeric, so the first
	// hyphen after it unambiguously marks the start of the prompt ID.
	customIDItemPrefix = "item-"
	customIDPromptSep  = "-"
)

type CreateBatchMessage struct{}

// BatchMessageItemSpec is a single resolved request in the batch.
type BatchMessageItemSpec struct {
	CustomID string
	Prompt   string
}

// BatchMessagePromptSpec is one prompt in "multiple" mode. It's combined with
// every element of Items to produce one request per (prompt, item) pair.
type BatchMessagePromptSpec struct {
	ID     string `json:"id" mapstructure:"id"`
	Prompt string `json:"prompt" mapstructure:"prompt"`
}

// BatchMessageSpec is the workflow node configuration for claude.createBatchMessage.
type BatchMessageSpec struct {
	Items         string `json:"items" mapstructure:"items"`
	Model         string `json:"model" mapstructure:"model"`
	SystemMessage string `json:"systemMessage" mapstructure:"systemMessage"`

	// Mode selects how the batch's requests are built. See modeSingle / modeMultiple.
	Mode string `json:"mode" mapstructure:"mode"`

	OutputSchema string `json:"outputSchema" mapstructure:"outputSchema"`

	// Single mode: one prompt, applied to every element of Items.
	Prompt string `json:"prompt" mapstructure:"prompt"`

	// Multiple mode: several prompts, each applied to every element of Items.
	Prompts []BatchMessagePromptSpec `json:"prompts" mapstructure:"prompts"`
}

// BatchMessageNodeMetadata is node-level metadata surfaced in the UI.
type BatchMessageNodeMetadata struct {
	Model            string `json:"model" mapstructure:"model"`
	StructuredOutput bool   `json:"structuredOutput" mapstructure:"structuredOutput"`
}

// BatchExecutionMetadata is persisted for the run so poll() can find the batch
// again, and to surface live progress (RequestCounts) while it's still processing.
type BatchExecutionMetadata struct {
	BatchID       string                     `json:"batchId,omitempty" mapstructure:"batchId,omitempty"`
	Status        string                     `json:"status,omitempty" mapstructure:"status,omitempty"`
	RequestCounts *MessageBatchRequestCounts `json:"requestCounts,omitempty" mapstructure:"requestCounts,omitempty"`
}

// BatchResultOutcome is one request's outcome: either a whole item's result
// (Single Prompt mode) or one prompt's result for an item (Multiple Prompts
// mode, nested under BatchItemResult.Prompts).
type BatchResultOutcome struct {
	Type         string        `json:"type"` // succeeded | errored | canceled | expired
	Text         string        `json:"text,omitempty"`
	Parsed       any           `json:"parsed,omitempty"`
	StopReason   string        `json:"stopReason,omitempty"`
	Usage        *MessageUsage `json:"usage,omitempty"`
	ErrorType    string        `json:"errorType,omitempty"`
	ErrorMessage string        `json:"errorMessage,omitempty"`
}

// BatchItemResult is one element of Items' result. In Single Prompt mode its
// outcome fields are set directly; in Multiple Prompts mode, Prompts holds
// one entry per configured prompt, keyed by its ID.
type BatchItemResult struct {
	Index int `json:"index"`
	BatchResultOutcome
	Prompts map[string]BatchResultOutcome `json:"prompts,omitempty"`
}

// BatchOutput is the payload emitted when the batch reaches a terminal state.
// Results always has one entry per element of Items, in the same order.
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

Every batch is built from a matrix: an array of **Items** (the N data points, e.g. a list of pull requests) crossed with one or more prompts. **Mode** controls whether that's a single prompt (1 x N) or several (M x N).

## Use Cases

- **Update the same kind of resource, one at a time or many at once**: pass a one-element array to update a single pull request, or the full list to update every open one, with the same node.
- **Multiple derived outputs per item**: generate a title suggestion, a description suggestion, and a risk assessment for every pull request in one batch (Multiple Prompts mode).
- **Bulk classification or extraction**: run the same prompt over many inputs at once, at a lower cost than individual requests.

## How It Works

1. Evaluates **Items** to an array, then builds one request per item (Single Prompt) or one request per (prompt, item) pair (Multiple Prompts).
2. Submits them as a single batch and polls it until it ends.
3. Emits **Results**: one entry per element of Items, in the same order.

Batches typically complete within an hour, but can take up to 24 hours. This component polls with increasing backoff and keeps the execution open (without emitting) until the batch ends.

## Configuration

- **Items**: An expression evaluating to the array to run over, e.g. ` + "`$['List Open Pull Requests'].body`" + `. Use a one-element array to run over a single item.
- **Model**: The Claude model used for every request.
- **System Message**: (Optional) Context applied to every request.
- **Mode**: Whether one prompt or several are applied to each item.
  - **Single Prompt**: one prompt, applied to every element of Items.
  - **Multiple Prompts**: several prompts, each applied to every element of Items.
- **Prompt** (Single Prompt mode): An expression evaluated per item to build its prompt, with ` + "`item`" + ` and ` + "`index`" + ` available, e.g. ` + "`\"Suggest a title for PR #\" + string(item.number) + \": \" + item.body`" + `.
- **Prompts** (Multiple Prompts mode): A short list of prompts (each with an **ID** and its own expression, using ` + "`item`" + `/` + "`index`" + ` the same way). Each one is evaluated per element of Items.
- **Structured Output**: (Optional) A JSON Schema every response must conform to.

## Output

Emits a single payload once the batch ends, containing:
- **status**: ` + "`ended`" + `, ` + "`timeout`" + `, or ` + "`error`" + `.
- **batchId**: The Anthropic batch ID.
- **requestCounts**: How many requests succeeded, errored, were canceled, or expired.
- **results**: One entry per element of Items, in order (` + "`results[i]`" + ` corresponds to the i-th item). In Single Prompt mode each entry has its own ` + "`text`" + `/` + "`parsed`" + `/etc. directly; in Multiple Prompts mode each entry has a ` + "`prompts`" + ` object keyed by prompt ID, e.g. ` + "`results[i].prompts.title.text`" + `.

## Notes

- Requires a valid Claude API key configured in the integration.
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
					Index: 0,
					BatchResultOutcome: BatchResultOutcome{
						Type:       "succeeded",
						Text:       "Paris is the capital of France.",
						StopReason: "end_turn",
						Usage:      &MessageUsage{InputTokens: 12, OutputTokens: 9},
					},
				},
				{
					Index: 1,
					BatchResultOutcome: BatchResultOutcome{
						Type:       "succeeded",
						Text:       "Berlin is the capital of Germany.",
						StopReason: "end_turn",
						Usage:      &MessageUsage{InputTokens: 12, OutputTokens: 9},
					},
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
			Description: "Expression evaluating to the array to run the prompt(s) over. Use a one-element array to run over a single item.",
		},
		{
			Name:        "model",
			Label:       "Model",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Default:     "claude-opus-4-6",
			Placeholder: "Select a Claude model",
			Description: "Model used for every request in the batch.",
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
			Description: "Optional context applied to every request in the batch.",
		},
		{
			Name:        "mode",
			Label:       "Mode",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     modeSingle,
			Description: "Whether one prompt or several are applied to each item.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Single Prompt", Value: modeSingle, Description: "One prompt, applied to every item"},
						{Label: "Multiple Prompts", Value: modeMultiple, Description: "Several prompts, each applied to every item"},
					},
				},
			},
		},
		{
			Name:                 "prompt",
			Label:                "Prompt",
			Type:                 configuration.FieldTypeText,
			Placeholder:          `"Suggest a title for PR #" + string(item.number) + ": " + item.body`,
			Description:          "Expression evaluated per item to build its prompt, with `item` and `index` available.",
			VisibilityConditions: singleVisible,
			RequiredConditions:   singleRequired,
		},
		{
			Name:                 "prompts",
			Label:                "Prompts",
			Type:                 configuration.FieldTypeList,
			Description:          "A prompt per row, each evaluated per item. Produces (prompts x items) requests, grouped back into one result per item.",
			VisibilityConditions: multipleVisible,
			RequiredConditions:   multipleRequired,
			Default: []map[string]any{
				{"id": "prompt-1", "prompt": ""},
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
								Placeholder: "title",
								Description: "Short name for this prompt (letters, digits, hyphens, underscores). Keys its result on each item, e.g. `results[i].prompts.title`.",
							},
							{
								Name:        "prompt",
								Label:       "Prompt",
								Type:        configuration.FieldTypeText,
								Required:    true,
								Placeholder: `"Suggest a title for PR #" + string(item.number) + ": " + item.body`,
								Description: "Expression evaluated per item to build this prompt's text, with `item`/`index` available.",
							},
						},
					},
				},
			},
		},
		structuredoutput.ConfigField(
			"outputSchema",
			"Structured Output",
			"A JSON Schema every response must match, available on each result's `parsed` field.",
		),
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
		hasSchema := strings.TrimSpace(spec.OutputSchema) != ""
		_ = ctx.Metadata.Set(BatchMessageNodeMetadata{
			Model:            spec.Model,
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
// The expression fields themselves (Items/Prompt/Prompts[].Prompt)
// are only fully validated once evaluated, in resolveBatchRequests /
// validateRequestItems.
func validateBatchMessageSpec(spec BatchMessageSpec) error {
	if strings.TrimSpace(spec.Model) == "" {
		return fmt.Errorf("model is required")
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
			if strings.TrimSpace(p.Prompt) == "" {
				return fmt.Errorf("prompts[%d].prompt is required", i)
			}
		}
	default:
		if strings.TrimSpace(spec.Prompt) == "" {
			return fmt.Errorf("prompt is required in \"Single Prompt\" mode")
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
// it came from single mode (one per item) or multiple mode (one per prompt,
// per item).
func validateRequestItems(items []BatchMessageItemSpec) error {
	if len(items) == 0 {
		return fmt.Errorf("at least one request is required")
	}
	if len(items) > maxBatchRequests {
		return fmt.Errorf("a batch cannot contain more than %d requests", maxBatchRequests)
	}

	for i, r := range items {
		if strings.TrimSpace(r.Prompt) == "" {
			return fmt.Errorf("prompt for item %d is empty", i)
		}
	}

	return nil
}

// resolveBatchRequests evaluates Items to an array, then builds the batch's
// requests according to spec.Mode: one prompt applied to every element
// (single, 1 x N), or several applied to every element (multiple, M x N).
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

// resolveSinglePrompt builds one request per element, by evaluating Prompt
// with `item`/`index` bound as extra variables: a 1 x N matrix. Each
// request's (internal) Custom ID encodes its item index.
func resolveSinglePrompt(expressions core.ExpressionContext, spec BatchMessageSpec, elements []any) ([]BatchMessageItemSpec, error) {
	if len(elements) > maxBatchRequests {
		return nil, fmt.Errorf("items has %d elements; a batch cannot contain more than %d requests", len(elements), maxBatchRequests)
	}

	items := make([]BatchMessageItemSpec, 0, len(elements))
	for i, element := range elements {
		vars := map[string]any{"item": element, "index": i}

		prompt, err := evalPromptTemplate(expressions, spec.Prompt, vars, i)
		if err != nil {
			return nil, err
		}

		items = append(items, BatchMessageItemSpec{CustomID: itemCustomID(i, ""), Prompt: prompt})
	}

	return items, nil
}

// resolveMultiplePrompts builds one request per (prompt, item) pair: an
// M x N matrix. Each request's (internal) Custom ID encodes both its item
// index and its prompt ID, so results can be regrouped by item afterwards.
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
		if !promptIDPattern.MatchString(id) {
			return nil, fmt.Errorf("prompts[%s].id must contain only letters, digits, hyphens, and underscores", id)
		}

		for i, element := range elements {
			vars := map[string]any{"item": element, "index": i}

			prompt, err := evalPromptTemplate(expressions, p.Prompt, vars, i)
			if err != nil {
				return nil, fmt.Errorf("prompts[%s]: %w", id, err)
			}

			customID := itemCustomID(i, id)
			if len(customID) > 64 {
				return nil, fmt.Errorf("prompts[%s]: prompt id makes the internal request id too long; use a shorter id", id)
			}

			items = append(items, BatchMessageItemSpec{CustomID: customID, Prompt: prompt})
		}
	}

	return items, nil
}

// evalPromptTemplate evaluates a prompt expression for one element, with
// `item`/`index` bound as extra variables.
func evalPromptTemplate(expressions core.ExpressionContext, template string, vars map[string]any, index int) (string, error) {
	result, err := expressions.RunWithExtraVariables(template, vars)
	if err != nil {
		return "", fmt.Errorf("prompt (item %d): %w", index, err)
	}
	prompt, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("prompt must evaluate to a string, got %T (item %d)", result, index)
	}
	return prompt, nil
}

// itemCustomID builds the batch's internal (never user-facing) custom ID for
// an item's request, optionally scoped to one prompt. It's parsed back by
// parseItemCustomID to regroup flat batch results by item/prompt.
func itemCustomID(index int, promptID string) string {
	if promptID == "" {
		return fmt.Sprintf("%s%d", customIDItemPrefix, index+1)
	}
	return fmt.Sprintf("%s%d%s%s", customIDItemPrefix, index+1, customIDPromptSep, promptID)
}

// parseItemCustomID reverses itemCustomID, returning the zero-based item
// index and (if present) the prompt ID.
func parseItemCustomID(customID string) (index int, promptID string, ok bool) {
	rest, found := strings.CutPrefix(customID, customIDItemPrefix)
	if !found {
		return 0, "", false
	}

	indexPart := rest
	if i := strings.Index(rest, customIDPromptSep); i >= 0 {
		indexPart = rest[:i]
		promptID = rest[i+len(customIDPromptSep):]
	}

	n, err := strconv.Atoi(indexPart)
	if err != nil || n < 1 {
		return 0, "", false
	}
	return n - 1, promptID, true
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
			Model:     spec.Model,
			MaxTokens: defaultMaxTokens,
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

// buildBatchOutput assembles the emitted payload from a batch and its
// (possibly nil, for timeout/error statuses) results. Results are regrouped
// from the flat, customId-keyed API response into one entry per item, using
// each result's internal Custom ID (see itemCustomID/parseItemCustomID) to
// find its item index and (in multiple mode) its prompt ID.
func buildBatchOutput(status string, batch *MessageBatch, results []MessageBatchResult, hasSchema bool) BatchOutput {
	out := BatchOutput{Status: status}
	if batch != nil {
		out.BatchID = batch.ID
		out.RequestCounts = batch.RequestCounts
	}

	if len(results) == 0 {
		return out
	}

	itemsByIndex := map[int]*BatchItemResult{}
	maxIndex := -1
	for _, r := range results {
		index, promptID, ok := parseItemCustomID(r.CustomID)
		if !ok {
			continue
		}
		if index > maxIndex {
			maxIndex = index
		}

		item, exists := itemsByIndex[index]
		if !exists {
			item = &BatchItemResult{Index: index}
			itemsByIndex[index] = item
		}

		outcome := buildResultOutcome(r, hasSchema)
		if promptID == "" {
			item.BatchResultOutcome = outcome
		} else {
			if item.Prompts == nil {
				item.Prompts = map[string]BatchResultOutcome{}
			}
			item.Prompts[promptID] = outcome
		}
	}

	out.Results = make([]BatchItemResult, maxIndex+1)
	for i := 0; i <= maxIndex; i++ {
		if item, ok := itemsByIndex[i]; ok {
			out.Results[i] = *item
		} else {
			out.Results[i] = BatchItemResult{Index: i}
		}
	}

	return out
}

// buildResultOutcome extracts one request's outcome (succeeded text/usage or
// error details), parsing it against the configured schema if requested.
func buildResultOutcome(r MessageBatchResult, hasSchema bool) BatchResultOutcome {
	out := BatchResultOutcome{Type: r.Result.Type}

	switch r.Result.Type {
	case "succeeded":
		if r.Result.Message != nil {
			out.Text = extractMessageText(r.Result.Message)
			out.StopReason = r.Result.Message.StopReason
			out.Usage = &r.Result.Message.Usage

			if hasSchema && out.StopReason == "end_turn" && out.Text != "" {
				var parsed any
				if err := json.Unmarshal([]byte(out.Text), &parsed); err == nil {
					out.Parsed = parsed
				}
			}
		}
	case "errored":
		if r.Result.Error != nil {
			out.ErrorType = r.Result.Error.Type
			out.ErrorMessage = r.Result.Error.Message
		}
	}

	return out
}

func (c *CreateBatchMessage) Hooks() []core.Hook {
	return []core.Hook{{
		Name: "poll",
		Type: core.HookTypeInternal,
	}}
}
