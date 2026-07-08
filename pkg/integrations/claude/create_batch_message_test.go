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
		"model":  "claude-opus-4-6",
		"mode":   modeSingle,
		"prompt": "Capital of France?",
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

	t.Run("single mode missing prompt", func(t *testing.T) {
		cfg := validBatchConfig()
		delete(cfg, "prompt")
		ctx := core.SetupContext{Configuration: cfg}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "prompt is required")
	})

	t.Run("multiple mode missing items", func(t *testing.T) {
		cfg := map[string]any{
			"model":          "claude-opus-4-6",
			"mode":           modeMultiple,
			"promptTemplate": `item`,
		}
		ctx := core.SetupContext{Configuration: cfg}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "items is required")
	})

	t.Run("multiple mode missing promptTemplate", func(t *testing.T) {
		cfg := map[string]any{
			"model": "claude-opus-4-6",
			"mode":  modeMultiple,
			"items": `$['Fetch'].body`,
		}
		ctx := core.SetupContext{Configuration: cfg}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "promptTemplate is required")
	})

	t.Run("multiple mode does not require prompt/customId", func(t *testing.T) {
		cfg := map[string]any{
			"model":          "claude-opus-4-6",
			"mode":           modeMultiple,
			"items":          `$['Fetch'].body`,
			"promptTemplate": `item`,
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
			"model":  "claude-opus-4-6",
			"mode":   modeSingle,
			"prompt": "Capital of France?",
			// customId intentionally omitted
		},
		HTTP:           httpContext,
		Integration:    integrationCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Requests:       &contexts.RequestContext{},
		Logger:         logrus.NewEntry(logrus.New()),
	}

	err := c.Execute(execCtx)
	require.NoError(t, err)

	require.Len(t, httpContext.Requests, 2)
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

func Test__CreateBatchMessage__Setup__multipleMode_skipsSingleModeValidation(t *testing.T) {
	c := &CreateBatchMessage{}
	ctx := core.SetupContext{
		Configuration: map[string]any{
			"model":          "claude-opus-4-6",
			"mode":           modeMultiple,
			"items":          `$['Fetch Issues'].json`,
			"promptTemplate": `"Triage: " + item.title`,
		},
		Metadata: &contexts.MetadataContext{},
	}
	require.NoError(t, c.Setup(ctx))
}

func Test__CreateBatchMessage__Execute__multipleMode_buildsOneRequestPerItem(t *testing.T) {
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
			"mode":               modeMultiple,
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
			WithVariablesOutputFn: func(expression string, variables map[string]any) (any, error) {
				item := variables["item"].(map[string]any)
				switch expression {
				case "dummy":
					return "pr-" + strconv.Itoa(int(item["number"].(float64))), nil
				}
				return nil, nil
			},
			Output: items,
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

func Test__CreateBatchMessage__Execute__multipleMode_notAnArray(t *testing.T) {
	c := &CreateBatchMessage{}
	execCtx := core.ExecutionContext{
		Configuration: map[string]any{
			"model":          "claude-opus-4-6",
			"mode":           modeMultiple,
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

func Test__CreateBatchMessage__Execute__multipleMode_promptTemplateNotString(t *testing.T) {
	c := &CreateBatchMessage{}
	execCtx := core.ExecutionContext{
		Configuration: map[string]any{
			"model":          "claude-opus-4-6",
			"mode":           modeMultiple,
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
