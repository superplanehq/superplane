package claude

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func validBatchConfig() map[string]any {
	return map[string]any{
		"model":          "claude-opus-4-6",
		"mode":           modeSingle,
		"items":          `$['Fetch'].body`,
		"promptTemplate": `"Capital of " + item.country + "?"`,
	}
}

// singleItemExpressions builds an Expression context resolving Items to a
// single element and PromptTemplate to a fixed prompt, regardless of the
// bound item/index variables.
func singleItemExpressions(prompt string) *contexts.ExpressionContext {
	return &contexts.ExpressionContext{
		Output: []any{map[string]any{"country": "France"}},
		WithVariablesOutputFn: func(expression string, variables map[string]any) (any, error) {
			return prompt, nil
		},
	}
}

func Test__CreateBatchMessage__Setup(t *testing.T) {
	c := &CreateBatchMessage{}

	t.Run("valid config", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: validBatchConfig(),
			Metadata:      &contexts.MetadataContext{},
		}
		require.NoError(t, c.Setup(ctx))
	})

	t.Run("missing model", func(t *testing.T) {
		cfg := validBatchConfig()
		delete(cfg, "model")
		ctx := core.SetupContext{Configuration: cfg}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "model")
	})

	t.Run("missing items", func(t *testing.T) {
		cfg := validBatchConfig()
		delete(cfg, "items")
		ctx := core.SetupContext{Configuration: cfg}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "items is required")
	})

	t.Run("single mode missing promptTemplate", func(t *testing.T) {
		cfg := validBatchConfig()
		delete(cfg, "promptTemplate")
		ctx := core.SetupContext{Configuration: cfg}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "promptTemplate is required")
	})

	t.Run("multiple mode missing prompts", func(t *testing.T) {
		cfg := map[string]any{
			"model": "claude-opus-4-6",
			"mode":  modeMultiple,
			"items": `$['Fetch'].body`,
		}
		ctx := core.SetupContext{Configuration: cfg}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one prompt is required")
	})

	t.Run("multiple mode prompt missing id", func(t *testing.T) {
		cfg := map[string]any{
			"model":   "claude-opus-4-6",
			"mode":    modeMultiple,
			"items":   `$['Fetch'].body`,
			"prompts": []map[string]any{{"promptTemplate": "item"}},
		}
		ctx := core.SetupContext{Configuration: cfg}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "prompts[0].id is required")
	})

	t.Run("multiple mode prompt missing promptTemplate", func(t *testing.T) {
		cfg := map[string]any{
			"model":   "claude-opus-4-6",
			"mode":    modeMultiple,
			"items":   `$['Fetch'].body`,
			"prompts": []map[string]any{{"id": "title"}},
		}
		ctx := core.SetupContext{Configuration: cfg}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "prompts[0].promptTemplate is required")
	})

	t.Run("multiple mode does not require promptTemplate/customIdExpression", func(t *testing.T) {
		cfg := map[string]any{
			"model":   "claude-opus-4-6",
			"mode":    modeMultiple,
			"items":   `$['Fetch'].body`,
			"prompts": []map[string]any{{"id": "title", "promptTemplate": "item"}},
		}
		ctx := core.SetupContext{Configuration: cfg, Metadata: &contexts.MetadataContext{}}
		require.NoError(t, c.Setup(ctx))
	})

	t.Run("sets node metadata", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		ctx := core.SetupContext{
			Configuration: validBatchConfig(),
			Metadata:      metadataCtx,
		}
		require.NoError(t, c.Setup(ctx))
		meta := metadataCtx.Metadata.(BatchMessageNodeMetadata)
		assert.Equal(t, "claude-opus-4-6", meta.Model)
		assert.Equal(t, 4096, meta.MaxTokens)
		assert.False(t, meta.StructuredOutput)
	})
}

func Test__CreateBatchMessage__Execute__endsImmediately(t *testing.T) {
	c := &CreateBatchMessage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"msgbatch_1","type":"message_batch","processing_status":"ended","request_counts":{"succeeded":1}}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
				`{"custom_id":"request-1","result":{"type":"succeeded","message":{"id":"msg_1","content":[{"type":"text","text":"Paris"}],"stop_reason":"end_turn","usage":{"input_tokens":5,"output_tokens":3}}}}`,
			))},
		},
	}
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}}
	metadataCtx := &contexts.MetadataContext{}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	requestsCtx := &contexts.RequestContext{}

	execCtx := core.ExecutionContext{
		ID:             uuid.New(),
		Configuration:  validBatchConfig(),
		HTTP:           httpContext,
		Integration:    integrationCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Requests:       requestsCtx,
		Expressions:    singleItemExpressions("Capital of France?"),
		Logger:         logrus.NewEntry(logrus.New()),
	}

	err := c.Execute(execCtx)
	require.NoError(t, err)
	require.True(t, executionState.Finished)
	assert.Equal(t, CreateBatchMessagePayloadType, executionState.Type)
	assert.Equal(t, "", requestsCtx.Action)

	out := executionState.Payloads[0].(map[string]any)["data"].(BatchOutput)
	assert.Equal(t, batchStatusEnded, out.Status)
	assert.Equal(t, "msgbatch_1", out.BatchID)
	require.Len(t, out.Results, 1)
	assert.Equal(t, "succeeded", out.Results[0].Type)
	assert.Equal(t, "Paris", out.Results[0].Text)

	require.Len(t, httpContext.Requests, 2)
	assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
	assert.Contains(t, httpContext.Requests[0].URL.Path, "/messages/batches")
	assert.Contains(t, httpContext.Requests[1].URL.Path, "/results")
}

func Test__CreateBatchMessage__Execute__singleMode_defaultCustomId(t *testing.T) {
	c := &CreateBatchMessage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"msgbatch_1","processing_status":"ended"}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
				`{"custom_id":"request-1","result":{"type":"succeeded","message":{"id":"msg_1","content":[{"type":"text","text":"Paris"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}}}`,
			))},
		},
	}
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}}
	metadataCtx := &contexts.MetadataContext{}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	execCtx := core.ExecutionContext{
		ID: uuid.New(),
		Configuration: map[string]any{
			"model":          "claude-opus-4-6",
			"mode":           modeSingle,
			"items":          `$['Fetch'].body`,
			"promptTemplate": `"Capital of " + item.country + "?"`,
			// customIdExpression intentionally omitted
		},
		HTTP:           httpContext,
		Integration:    integrationCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Requests:       &contexts.RequestContext{},
		Expressions:    singleItemExpressions("Capital of France?"),
		Logger:         logrus.NewEntry(logrus.New()),
	}

	err := c.Execute(execCtx)
	require.NoError(t, err)

	require.Len(t, httpContext.Requests, 2)
}

func Test__CreateBatchMessage__Execute__singleMode_oneRequestPerItem(t *testing.T) {
	c := &CreateBatchMessage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"msgbatch_1","processing_status":"ended","request_counts":{"succeeded":2}}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
				`{"custom_id":"pr-1","result":{"type":"succeeded","message":{"id":"msg_1","content":[{"type":"text","text":"A"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}}}` + "\n" +
					`{"custom_id":"pr-2","result":{"type":"succeeded","message":{"id":"msg_2","content":[{"type":"text","text":"B"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}}}`,
			))},
		},
	}
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}}
	metadataCtx := &contexts.MetadataContext{}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	requestsCtx := &contexts.RequestContext{}

	items := []any{
		map[string]any{"number": float64(1), "title": "Fix A"},
		map[string]any{"number": float64(2), "title": "Fix B"},
	}

	execCtx := core.ExecutionContext{
		ID: uuid.New(),
		Configuration: map[string]any{
			"model":              "claude-opus-4-6",
			"mode":               modeSingle,
			"items":              `dummy`,
			"promptTemplate":     `dummy`,
			"customIdExpression": `dummy`,
		},
		HTTP:           httpContext,
		Integration:    integrationCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Requests:       requestsCtx,
		Expressions: &contexts.ExpressionContext{
			Output: items,
			WithVariablesOutputFn: func(expression string, variables map[string]any) (any, error) {
				item := variables["item"].(map[string]any)
				return "pr-" + strconv.Itoa(int(item["number"].(float64))), nil
			},
		},
		Logger: logrus.NewEntry(logrus.New()),
	}

	err := c.Execute(execCtx)
	require.NoError(t, err)
	require.True(t, executionState.Finished)

	out := executionState.Payloads[0].(map[string]any)["data"].(BatchOutput)
	require.Len(t, out.Results, 2)
	assert.Equal(t, "pr-1", out.Results[0].CustomID)
	assert.Equal(t, "pr-2", out.Results[1].CustomID)
}

func Test__CreateBatchMessage__Execute__multipleMode_crossProduct(t *testing.T) {
	c := &CreateBatchMessage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"msgbatch_1","processing_status":"ended","request_counts":{"succeeded":4}}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
				`{"custom_id":"title-1","result":{"type":"succeeded","message":{"id":"m1","content":[{"type":"text","text":"T1"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}}}` + "\n" +
					`{"custom_id":"title-2","result":{"type":"succeeded","message":{"id":"m2","content":[{"type":"text","text":"T2"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}}}` + "\n" +
					`{"custom_id":"description-1","result":{"type":"succeeded","message":{"id":"m3","content":[{"type":"text","text":"D1"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}}}` + "\n" +
					`{"custom_id":"description-2","result":{"type":"succeeded","message":{"id":"m4","content":[{"type":"text","text":"D2"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}}}`,
			))},
		},
	}
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}}
	metadataCtx := &contexts.MetadataContext{}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	requestsCtx := &contexts.RequestContext{}

	items := []any{
		map[string]any{"number": float64(1)},
		map[string]any{"number": float64(2)},
	}

	execCtx := core.ExecutionContext{
		ID: uuid.New(),
		Configuration: map[string]any{
			"model": "claude-opus-4-6",
			"mode":  modeMultiple,
			"items": `dummy`,
			"prompts": []map[string]any{
				{"id": "title", "promptTemplate": "dummy"},
				{"id": "description", "promptTemplate": "dummy"},
			},
		},
		HTTP:           httpContext,
		Integration:    integrationCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Requests:       requestsCtx,
		Expressions: &contexts.ExpressionContext{
			Output: items,
			WithVariablesOutputFn: func(expression string, variables map[string]any) (any, error) {
				return "prompt text", nil
			},
		},
		Logger: logrus.NewEntry(logrus.New()),
	}

	err := c.Execute(execCtx)
	require.NoError(t, err)
	require.True(t, executionState.Finished)

	require.Len(t, httpContext.Requests, 2)
	out := executionState.Payloads[0].(map[string]any)["data"].(BatchOutput)
	require.Len(t, out.Results, 4)

	customIDs := make([]string, 0, 4)
	for _, r := range out.Results {
		customIDs = append(customIDs, r.CustomID)
	}
	assert.ElementsMatch(t, []string{"title-1", "title-2", "description-1", "description-2"}, customIDs)
}

func Test__CreateBatchMessage__Execute__multipleMode_tooManyRequests(t *testing.T) {
	c := &CreateBatchMessage{}
	items := make([]any, maxBatchRequests)
	for i := range items {
		items[i] = map[string]any{"number": i}
	}

	execCtx := core.ExecutionContext{
		Configuration: map[string]any{
			"model": "claude-opus-4-6",
			"mode":  modeMultiple,
			"items": `dummy`,
			"prompts": []map[string]any{
				{"id": "title", "promptTemplate": "dummy"},
				{"id": "description", "promptTemplate": "dummy"},
			},
		},
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}},
		Metadata:    &contexts.MetadataContext{},
		Expressions: &contexts.ExpressionContext{Output: items},
		Logger:      logrus.NewEntry(logrus.New()),
	}

	err := c.Execute(execCtx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot contain more than")
}

func Test__CreateBatchMessage__Execute__itemsNotAnArray(t *testing.T) {
	c := &CreateBatchMessage{}
	execCtx := core.ExecutionContext{
		Configuration: map[string]any{
			"model":          "claude-opus-4-6",
			"mode":           modeSingle,
			"items":          `dummy`,
			"promptTemplate": `dummy`,
		},
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}},
		Metadata:    &contexts.MetadataContext{},
		Expressions: &contexts.ExpressionContext{Output: "not-an-array"},
		Logger:      logrus.NewEntry(logrus.New()),
	}

	err := c.Execute(execCtx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must evaluate to an array")
}

func Test__CreateBatchMessage__Execute__promptTemplateNotString(t *testing.T) {
	c := &CreateBatchMessage{}
	execCtx := core.ExecutionContext{
		Configuration: map[string]any{
			"model":          "claude-opus-4-6",
			"mode":           modeSingle,
			"items":          `dummy`,
			"promptTemplate": `dummy`,
		},
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}},
		Metadata:    &contexts.MetadataContext{},
		Expressions: &contexts.ExpressionContext{
			Output: []any{map[string]any{"number": float64(1)}},
			WithVariablesOutputFn: func(expression string, variables map[string]any) (any, error) {
				return 42, nil // not a string
			},
		},
		Logger: logrus.NewEntry(logrus.New()),
	}

	err := c.Execute(execCtx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must evaluate to a string")
}

func Test__CreateBatchMessage__Execute__schedulesPoll(t *testing.T) {
	c := &CreateBatchMessage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"msgbatch_1","processing_status":"in_progress","request_counts":{"processing":1}}`))},
		},
	}
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}}
	metadataCtx := &contexts.MetadataContext{}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	requestsCtx := &contexts.RequestContext{}

	execCtx := core.ExecutionContext{
		ID:             uuid.New(),
		Configuration:  validBatchConfig(),
		HTTP:           httpContext,
		Integration:    integrationCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Requests:       requestsCtx,
		Expressions:    singleItemExpressions("Capital of France?"),
		Logger:         logrus.NewEntry(logrus.New()),
	}

	err := c.Execute(execCtx)
	require.NoError(t, err)
	assert.False(t, executionState.Finished)
	assert.Equal(t, "poll", requestsCtx.Action)
	assert.Equal(t, batchInitialPoll, requestsCtx.Duration)

	meta := metadataCtx.Metadata.(BatchExecutionMetadata)
	assert.Equal(t, "msgbatch_1", meta.BatchID)
	require.NotNil(t, meta.RequestCounts)
	assert.Equal(t, 1, meta.RequestCounts.Processing)
}

// If Execute is retried after a batch was already submitted for this
// execution (e.g. the run didn't finish before the first poll could be
// scheduled/handled), it must resume polling the existing batch rather than
// submitting a duplicate one.
func Test__CreateBatchMessage__Execute__resumesExistingBatchInsteadOfCreatingDuplicate(t *testing.T) {
	c := &CreateBatchMessage{}
	httpContext := &contexts.HTTPContext{}
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}}
	metadataCtx := &contexts.MetadataContext{Metadata: BatchExecutionMetadata{BatchID: "msgbatch_existing", Status: "in_progress"}}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	requestsCtx := &contexts.RequestContext{}

	execCtx := core.ExecutionContext{
		ID:             uuid.New(),
		Configuration:  validBatchConfig(),
		HTTP:           httpContext,
		Integration:    integrationCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Requests:       requestsCtx,
		Expressions:    singleItemExpressions("Capital of France?"),
		Logger:         logrus.NewEntry(logrus.New()),
	}

	err := c.Execute(execCtx)
	require.NoError(t, err)
	assert.Empty(t, httpContext.Requests, "must not submit a new batch when one already exists for this execution")
	assert.Equal(t, "poll", requestsCtx.Action)
	assert.Equal(t, batchInitialPoll, requestsCtx.Duration)
}

func Test__CreateBatchMessage__poll__terminal(t *testing.T) {
	c := &CreateBatchMessage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"msgbatch_1","processing_status":"ended","request_counts":{"succeeded":1}}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
				`{"custom_id":"request-1","result":{"type":"succeeded","message":{"id":"msg_1","content":[{"type":"text","text":"Done"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}}}`,
			))},
		},
	}
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	metadataCtx := &contexts.MetadataContext{Metadata: BatchExecutionMetadata{BatchID: "msgbatch_1", Status: "in_progress"}}

	hookCtx := core.ActionHookContext{
		Name:           "poll",
		Configuration:  validBatchConfig(),
		Parameters:     map[string]any{"attempt": float64(1), "errors": float64(0)},
		HTTP:           httpContext,
		Integration:    integrationCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Logger:         logrus.NewEntry(logrus.New()),
		Requests:       &contexts.RequestContext{},
	}

	err := c.HandleHook(hookCtx)
	require.NoError(t, err)
	require.True(t, executionState.Finished)
	out := executionState.Payloads[0].(map[string]any)["data"].(BatchOutput)
	assert.Equal(t, batchStatusEnded, out.Status)
	require.Len(t, out.Results, 1)
	assert.Equal(t, "Done", out.Results[0].Text)
}

func Test__CreateBatchMessage__poll__stillProcessing(t *testing.T) {
	c := &CreateBatchMessage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"msgbatch_1","processing_status":"in_progress","request_counts":{"processing":18,"succeeded":5}}`))},
		},
	}
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	metadataCtx := &contexts.MetadataContext{Metadata: BatchExecutionMetadata{BatchID: "msgbatch_1", Status: "in_progress"}}
	requestsCtx := &contexts.RequestContext{}

	hookCtx := core.ActionHookContext{
		Name:           "poll",
		Configuration:  validBatchConfig(),
		Parameters:     map[string]any{"attempt": float64(2), "errors": float64(0)},
		HTTP:           httpContext,
		Integration:    integrationCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Logger:         logrus.NewEntry(logrus.New()),
		Requests:       requestsCtx,
	}

	err := c.HandleHook(hookCtx)
	require.NoError(t, err)
	assert.False(t, executionState.Finished)
	assert.Equal(t, "poll", requestsCtx.Action)
	// attempt=2 in -> nextAttempt=3 -> interval = batchInitialPoll * 2^min(3-1,8) = 4x
	assert.Equal(t, 4*batchInitialPoll, requestsCtx.Duration)

	// Live progress (RequestCounts) is refreshed on every poll, not just at the end.
	meta := metadataCtx.Metadata.(BatchExecutionMetadata)
	require.NotNil(t, meta.RequestCounts)
	assert.Equal(t, 18, meta.RequestCounts.Processing)
	assert.Equal(t, 5, meta.RequestCounts.Succeeded)
}

// Once the batch has ended, metadata must be refreshed with its terminal
// status/counts even if downloading results fails and a retry is scheduled,
// so the UI doesn't keep showing stale in-progress state.
func Test__CreateBatchMessage__poll__endedButResultsFetchFails_refreshesMetadataAndRetries(t *testing.T) {
	c := &CreateBatchMessage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"msgbatch_1","processing_status":"ended","request_counts":{"succeeded":2}}`))},
			{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"error":"boom"}`))},
		},
	}
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	metadataCtx := &contexts.MetadataContext{Metadata: BatchExecutionMetadata{BatchID: "msgbatch_1", Status: "in_progress"}}
	requestsCtx := &contexts.RequestContext{}

	hookCtx := core.ActionHookContext{
		Name:           "poll",
		Configuration:  validBatchConfig(),
		Parameters:     map[string]any{"attempt": float64(1), "errors": float64(0)},
		HTTP:           httpContext,
		Integration:    integrationCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Logger:         logrus.NewEntry(logrus.New()),
		Requests:       requestsCtx,
	}

	err := c.HandleHook(hookCtx)
	require.NoError(t, err)
	assert.False(t, executionState.Finished)
	assert.Equal(t, "poll", requestsCtx.Action)

	meta := metadataCtx.Metadata.(BatchExecutionMetadata)
	assert.Equal(t, batchStatusEnded, meta.Status)
	require.NotNil(t, meta.RequestCounts)
	assert.Equal(t, 2, meta.RequestCounts.Succeeded)
}

// After enough consecutive results-fetch failures for an ended batch, the
// run must finish with an "error" outcome (not silently retry until it hits
// the unrelated max-attempts timeout).
func Test__CreateBatchMessage__poll__endedButResultsFetchFailsRepeatedly_emitsError(t *testing.T) {
	c := &CreateBatchMessage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"msgbatch_1","processing_status":"ended","request_counts":{"succeeded":2}}`))},
			{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"error":"boom"}`))},
		},
	}
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	metadataCtx := &contexts.MetadataContext{Metadata: BatchExecutionMetadata{BatchID: "msgbatch_1", Status: "in_progress"}}

	hookCtx := core.ActionHookContext{
		Name:           "poll",
		Configuration:  validBatchConfig(),
		Parameters:     map[string]any{"attempt": float64(5), "errors": float64(batchMaxPollErrors - 1)},
		HTTP:           httpContext,
		Integration:    integrationCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Logger:         logrus.NewEntry(logrus.New()),
		Requests:       &contexts.RequestContext{},
	}

	err := c.HandleHook(hookCtx)
	require.NoError(t, err)
	require.True(t, executionState.Finished)
	out := executionState.Payloads[0].(map[string]any)["data"].(BatchOutput)
	assert.Equal(t, "error", out.Status)
	assert.Equal(t, "msgbatch_1", out.BatchID)
}

func Test__CreateBatchMessage__poll__timeout(t *testing.T) {
	c := &CreateBatchMessage{}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	metadataCtx := &contexts.MetadataContext{Metadata: BatchExecutionMetadata{BatchID: "msgbatch_1", Status: "in_progress"}}

	hookCtx := core.ActionHookContext{
		Name: "poll",
		Parameters: map[string]any{
			"attempt": float64(batchMaxPollAttempts + 1),
			"errors":  float64(0),
		},
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Logger:         logrus.NewEntry(logrus.New()),
	}

	err := c.HandleHook(hookCtx)
	require.NoError(t, err)
	require.True(t, executionState.Finished)
	out := executionState.Payloads[0].(map[string]any)["data"].(BatchOutput)
	assert.Equal(t, "timeout", out.Status)
}

// If the poll hook is ever invoked with no batch id in execution metadata
// (e.g. metadata was cleared or corrupted), it must finish the run with an
// "error" outcome rather than silently no-op and leave the run open forever.
func Test__CreateBatchMessage__poll__noBatchId_emitsError(t *testing.T) {
	c := &CreateBatchMessage{}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	metadataCtx := &contexts.MetadataContext{Metadata: BatchExecutionMetadata{}}

	hookCtx := core.ActionHookContext{
		Name:           "poll",
		Parameters:     map[string]any{"attempt": float64(1), "errors": float64(0)},
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Logger:         logrus.NewEntry(logrus.New()),
	}

	err := c.HandleHook(hookCtx)
	require.NoError(t, err)
	require.True(t, executionState.Finished)
	out := executionState.Payloads[0].(map[string]any)["data"].(BatchOutput)
	assert.Equal(t, "error", out.Status)
}

// If the batch has ended but the node's own configuration fails to decode
// (a permanent error that won't resolve by retrying), the run must finish
// with the documented "error" output shape instead of a raw, unhandled error.
func Test__CreateBatchMessage__poll__endedButConfigurationDecodeFails_emitsError(t *testing.T) {
	c := &CreateBatchMessage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"msgbatch_1","processing_status":"ended","request_counts":{"succeeded":1}}`))},
		},
	}
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	metadataCtx := &contexts.MetadataContext{Metadata: BatchExecutionMetadata{BatchID: "msgbatch_1", Status: "in_progress"}}

	hookCtx := core.ActionHookContext{
		Name: "poll",
		// maxTokens is a number field; a string value fails mapstructure decoding.
		Configuration:  map[string]any{"model": "claude-opus-4-6", "mode": modeSingle, "maxTokens": "not-a-number"},
		Parameters:     map[string]any{"attempt": float64(1), "errors": float64(0)},
		HTTP:           httpContext,
		Integration:    integrationCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Logger:         logrus.NewEntry(logrus.New()),
		Requests:       &contexts.RequestContext{},
	}

	err := c.HandleHook(hookCtx)
	require.NoError(t, err)
	require.True(t, executionState.Finished)
	out := executionState.Payloads[0].(map[string]any)["data"].(BatchOutput)
	assert.Equal(t, "error", out.Status)
	assert.Equal(t, "msgbatch_1", out.BatchID)
}

func Test__CreateBatchMessage__Cancel(t *testing.T) {
	c := &CreateBatchMessage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"msgbatch_1","processing_status":"canceling"}`))},
		},
	}
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}}
	metadataCtx := &contexts.MetadataContext{Metadata: BatchExecutionMetadata{BatchID: "msgbatch_1"}}

	execCtx := core.ExecutionContext{
		HTTP:        httpContext,
		Integration: integrationCtx,
		Metadata:    metadataCtx,
		Logger:      logrus.NewEntry(logrus.New()),
	}

	require.NoError(t, c.Cancel(execCtx))
	require.Len(t, httpContext.Requests, 1)
	assert.Contains(t, httpContext.Requests[0].URL.Path, "/messages/batches/msgbatch_1/cancel")
}
