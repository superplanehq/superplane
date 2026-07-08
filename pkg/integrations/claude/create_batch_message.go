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

	// modeSingle sends exactly one request, built from Prompt/CustomID.
	modeSingle = "single"
	// modeMultiple sends one request per element of Items, built from
	// PromptTemplate/CustomIDExpression.
	modeMultiple = "multiple"
)

type CreateBatchMessage struct{}

// BatchMessageItemSpec is a single resolved request in the batch: one entry
// per element when Mode is "multiple", or the sole entry when Mode is "single".
type BatchMessageItemSpec struct {
	CustomID string `json:"customId" mapstructure:"customId"`
	Prompt   string `json:"prompt" mapstructure:"prompt"`
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

	// Single mode: one request, built directly from these two fields.
	Prompt   string `json:"prompt" mapstructure:"prompt"`
	CustomID string `json:"customId" mapstructure:"customId"`

	// Multiple mode: Items is a bare expression evaluating to an array; for
	// each element, PromptTemplate (and optionally CustomIDExpression) is
	// evaluated with `item` (the element) and `index` (its position) bound
	// as extra variables, one request per element.
	Items              string `json:"items" mapstructure:"items"`
	PromptTemplate     string `json:"promptTemplate" mapstructure:"promptTemplate"`
	CustomIDExpression string `json:"customIdExpression" mapstructure:"customIdExpression"`
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
	return `The Create Batch Message component uses [Anthropic's Message Batches API](https://platform.claude.com/docs/en/build-with-claude/batch-processing) to process one or many prompts asynchronously in one request, at a lower cost than issuing them individually.

## Use Cases

- **Update the same kind of resource, one at a time or many at once**: e.g. rewrite a single pull request's title and description with one node, and reuse the same node for every open pull request by switching its Mode.
- **Bulk classification or extraction**: Run the same prompt template over many inputs.
- **Cost-sensitive workloads**: Batches are billed at a discount versus the equivalent individual requests, even for a single request.

## How It Works

1. Builds the batch's requests according to **Mode** (see Configuration below).
2. Submits them as a single batch (` + "`POST /v1/messages/batches`" + `).
3. Polls the batch status until it reaches a terminal state.
4. Downloads the results and emits one entry per request, matched by **Custom ID**.

Batches typically complete within an hour, but can take up to 24 hours. This component polls with increasing backoff and keeps the execution open (without emitting) until the batch ends.

## Configuration

- **Mode**: How the batch's requests are built.
  - **Single Prompt**: sends exactly one request, built from **Prompt** and **Custom ID**.
  - **One Prompt Per Item**: sends one request per element of **Items**, built from **Prompt Template** and (optionally) **Custom ID Expression**.
- **Model**: The Claude model used for every request in the batch.
- **System Message**: (Optional) Context applied to every request in the batch.
- **Max Tokens**: (Optional) Limit the length of each generated response.
- **Temperature**: (Optional) Control randomness (0.0 to 1.0), applied to every request.
- **Structured Output**: (Optional) A JSON Schema every response must conform to.

**Single Prompt mode:**
- **Prompt**: The user message for the request. A regular text field, like any other — write plain text and drop in ` + "`{{ }}`" + ` placeholders to reference upstream data, e.g. ` + "`Update the title and description for PR #{{ $['Get Pull Request'].body.number }}: {{ $['Get Pull Request'].body.title }}`" + `.
- **Custom ID**: (Optional, advanced) Defaults to ` + "`request-1`" + `.

**One Prompt Per Item mode:**
- **Items**: An expression that evaluates to the array to build one request per element from, e.g. ` + "`$['List Open Pull Requests'].body`" + `.
- **Prompt Template**: An expression evaluated once per element of Items to build that element's prompt, with ` + "`item`" + ` (the element) and ` + "`index`" + ` (its zero-based position) available, e.g. ` + "`\"Update the title and description for PR #\" + string(item.number) + \": \" + item.title`" + `. Unlike Prompt above, this is a bare expression rather than a ` + "`{{ }}`" + ` template — wrap string literals in quotes and use ` + "`+`" + ` to build up the text, since it needs to run once per item rather than once for the whole field.
- **Custom ID Expression**: (Optional, advanced) Same idea as Prompt Template — an expression with ` + "`item`" + `/` + "`index`" + ` available, evaluated per element to compute its Custom ID. Defaults to auto-numbered IDs (` + "`request-1`" + `, ` + "`request-2`" + `, ...).

## Output

Emits a single payload once the batch ends, containing:
- **status**: ` + "`ended`" + `, ` + "`timeout`" + `, or ` + "`error`" + ` (the latter two only if polling could not confirm completion).
- **batchId**: The Anthropic batch ID.
- **requestCounts**: How many requests succeeded, errored, were canceled, or expired.
- **results**: One entry per request (by **Custom ID**), with the generated text, stop reason, token usage, and any error.

## Notes

- Requires a valid Claude API key configured in the integration.
- Custom IDs must be unique within the batch (max 64 characters).
- A batch can contain up to ` + fmt.Sprintf("%d", maxBatchRequests) + ` requests.
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
			Name:        "mode",
			Label:       "Mode",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Default:     modeSingle,
			Description: "Single Prompt sends one request. One Prompt Per Item sends one request per element of an array, built from a shared prompt template.",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Single Prompt", Value: modeSingle, Description: "Send exactly one request"},
						{Label: "One Prompt Per Item", Value: modeMultiple, Description: "Send one request per element of an array"},
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
			Name:                 "prompt",
			Label:                "Prompt",
			Type:                 configuration.FieldTypeText,
			Placeholder:          `Update the title and description for PR #{{ $['Get Pull Request'].body.number }}: {{ $['Get Pull Request'].body.title }}`,
			Description:          "The user message for the single request in this batch. A regular text field: write plain text and drop in `{{ }}` expressions to reference upstream data.",
			VisibilityConditions: singleVisible,
			RequiredConditions:   singleRequired,
		},
		{
			Name:                 "customId",
			Label:                "Custom ID",
			Type:                 configuration.FieldTypeString,
			Togglable:            true,
			Default:              "request-1",
			Placeholder:          "request-1",
			Description:          "Unique identifier for this request (max 64 characters). Used to match it to its result. Defaults to \"request-1\".",
			VisibilityConditions: singleVisible,
		},
		{
			Name:                 "items",
			Label:                "Items",
			Type:                 configuration.FieldTypeExpression,
			Placeholder:          `$['List Open Pull Requests'].body`,
			Description:          "Expression that evaluates to the array to build one request per element from.",
			VisibilityConditions: multipleVisible,
			RequiredConditions:   multipleRequired,
		},
		{
			Name:                 "promptTemplate",
			Label:                "Prompt Template",
			Type:                 configuration.FieldTypeExpression,
			Placeholder:          `"Update the title and description for PR #" + string(item.number) + ": " + item.title + "\n\n" + item.body`,
			Description:          "Expression evaluated once per element of Items to build that request's prompt, with `item` (the element) and `index` (its zero-based position) available. This is a bare expression, not a `{{ }}` template: wrap string literals in quotes and use `+` to build up the text.",
			VisibilityConditions: multipleVisible,
			RequiredConditions:   multipleRequired,
		},
		{
			Name:                 "customIdExpression",
			Label:                "Custom ID Expression",
			Type:                 configuration.FieldTypeExpression,
			Togglable:            true,
			Placeholder:          `string(item.number)`,
			Description:          fmt.Sprintf("Optional: expression evaluated once per element of Items (with the same `item`/`index` variables as Prompt Template) to compute its Custom ID. Defaults to auto-numbered IDs (request-1, request-2, ...). A batch can contain up to %d requests.", maxBatchRequests),
			VisibilityConditions: multipleVisible,
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
// The mode-specific expression fields (Items/PromptTemplate/CustomIDExpression
// in multiple mode) are only fully validated once evaluated, in
// resolveBatchRequests / validateRequestItems.
func validateBatchMessageSpec(spec BatchMessageSpec) error {
	if strings.TrimSpace(spec.Model) == "" {
		return fmt.Errorf("model is required")
	}

	if spec.MaxTokens < 0 {
		return fmt.Errorf("maxTokens must be at least 1")
	}

	switch spec.Mode {
	case modeMultiple:
		if strings.TrimSpace(spec.Items) == "" {
			return fmt.Errorf("items is required in \"One Prompt Per Item\" mode")
		}
		if strings.TrimSpace(spec.PromptTemplate) == "" {
			return fmt.Errorf("promptTemplate is required in \"One Prompt Per Item\" mode")
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

// resolveBatchRequests builds the batch's requests according to spec.Mode.
func resolveBatchRequests(expressions core.ExpressionContext, spec BatchMessageSpec) ([]BatchMessageItemSpec, error) {
	if spec.Mode == modeMultiple {
		return resolveMultipleRequests(expressions, spec)
	}
	return resolveSingleRequest(spec)
}

// resolveSingleRequest builds the sole request from Prompt/CustomID. Prompt is
// a FieldTypeText field, so by the time Execute runs, any `{{ }}` expressions
// in it have already been resolved to their final values — no further
// evaluation is needed here.
func resolveSingleRequest(spec BatchMessageSpec) ([]BatchMessageItemSpec, error) {
	if strings.TrimSpace(spec.Prompt) == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	customID := strings.TrimSpace(spec.CustomID)
	if customID == "" {
		customID = "request-1"
	}

	return []BatchMessageItemSpec{{CustomID: customID, Prompt: spec.Prompt}}, nil
}

// resolveMultipleRequests evaluates Items to an array, then builds one
// request per element by evaluating PromptTemplate (and, optionally,
// CustomIDExpression) with `item`/`index` bound as extra variables.
func resolveMultipleRequests(expressions core.ExpressionContext, spec BatchMessageSpec) ([]BatchMessageItemSpec, error) {
	raw, err := expressions.Run(spec.Items)
	if err != nil {
		return nil, fmt.Errorf("items: %w", err)
	}

	elements, err := toAnySlice(raw)
	if err != nil {
		return nil, fmt.Errorf("items must evaluate to an array: %w", err)
	}
	if len(elements) > maxBatchRequests {
		return nil, fmt.Errorf("items has %d elements; a batch cannot contain more than %d requests", len(elements), maxBatchRequests)
	}

	items := make([]BatchMessageItemSpec, 0, len(elements))
	for i, element := range elements {
		vars := map[string]any{"item": element, "index": i}

		promptResult, err := expressions.RunWithExtraVariables(spec.PromptTemplate, vars)
		if err != nil {
			return nil, fmt.Errorf("promptTemplate (item %d): %w", i, err)
		}
		prompt, ok := promptResult.(string)
		if !ok {
			return nil, fmt.Errorf("promptTemplate must evaluate to a string, got %T (item %d)", promptResult, i)
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
