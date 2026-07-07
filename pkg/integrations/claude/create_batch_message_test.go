package claude

import (
	"io"
	"net/http"
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
		"model": "claude-opus-4-6",
		"requests": []map[string]any{
			{"customId": "request-1", "prompt": "Capital of France?"},
			{"customId": "request-2", "prompt": "Capital of Germany?"},
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

	t.Run("no requests", func(t *testing.T) {
		cfg := validBatchConfig()
		cfg["requests"] = []map[string]any{}
		ctx := core.SetupContext{Configuration: cfg}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one request")
	})

	t.Run("missing customId", func(t *testing.T) {
		cfg := validBatchConfig()
		cfg["requests"] = []map[string]any{{"customId": "", "prompt": "hi"}}
		ctx := core.SetupContext{Configuration: cfg}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "customId is required")
	})

	t.Run("missing prompt", func(t *testing.T) {
		cfg := validBatchConfig()
		cfg["requests"] = []map[string]any{{"customId": "r1", "prompt": ""}}
		ctx := core.SetupContext{Configuration: cfg}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "prompt is required")
	})

	t.Run("duplicate customId", func(t *testing.T) {
		cfg := validBatchConfig()
		cfg["requests"] = []map[string]any{
			{"customId": "dup", "prompt": "a"},
			{"customId": "dup", "prompt": "b"},
		}
		ctx := core.SetupContext{Configuration: cfg}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicated")
	})

	t.Run("too many requests", func(t *testing.T) {
		requests := make([]map[string]any, maxBatchRequests+1)
		for i := range requests {
			requests[i] = map[string]any{"customId": uuid.New().String(), "prompt": "x"}
		}
		cfg := validBatchConfig()
		cfg["requests"] = requests
		ctx := core.SetupContext{Configuration: cfg}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot contain more than")
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
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"msgbatch_1","type":"message_batch","processing_status":"ended","request_counts":{"succeeded":2}}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
				`{"custom_id":"request-1","result":{"type":"succeeded","message":{"id":"msg_1","content":[{"type":"text","text":"Paris"}],"stop_reason":"end_turn","usage":{"input_tokens":5,"output_tokens":3}}}}` + "\n" +
					`{"custom_id":"request-2","result":{"type":"errored","error":{"type":"invalid_request_error","message":"bad input"}}}`,
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
	require.Len(t, out.Results, 2)
	assert.Equal(t, "succeeded", out.Results[0].Type)
	assert.Equal(t, "Paris", out.Results[0].Text)
	assert.Equal(t, "errored", out.Results[1].Type)
	assert.Equal(t, "invalid_request_error", out.Results[1].ErrorType)

	require.Len(t, httpContext.Requests, 2)
	assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
	assert.Contains(t, httpContext.Requests[0].URL.Path, "/messages/batches")
	assert.Contains(t, httpContext.Requests[1].URL.Path, "/results")
}

func Test__CreateBatchMessage__Execute__schedulesPoll(t *testing.T) {
	c := &CreateBatchMessage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"msgbatch_1","processing_status":"in_progress"}`))},
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
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"msgbatch_1","processing_status":"in_progress"}`))},
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

func Test__CreateBatchMessage__Setup__requestsExpression_skipsListValidation(t *testing.T) {
	c := &CreateBatchMessage{}
	ctx := core.SetupContext{
		Configuration: map[string]any{
			"model":              "claude-opus-4-6",
			"requestsExpression": `$['Fetch Issues'].json`,
		},
		Metadata: &contexts.MetadataContext{},
	}
	require.NoError(t, c.Setup(ctx))
}

func Test__CreateBatchMessage__Execute__requestsExpression_strings(t *testing.T) {
	c := &CreateBatchMessage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"msgbatch_1","processing_status":"ended","request_counts":{"succeeded":2}}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
				`{"custom_id":"request-1","result":{"type":"succeeded","message":{"id":"msg_1","content":[{"type":"text","text":"A"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}}}` + "\n" +
					`{"custom_id":"request-2","result":{"type":"succeeded","message":{"id":"msg_2","content":[{"type":"text","text":"B"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}}}`,
			))},
		},
	}
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}}
	metadataCtx := &contexts.MetadataContext{}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	requestsCtx := &contexts.RequestContext{}

	execCtx := core.ExecutionContext{
		ID: uuid.New(),
		Configuration: map[string]any{
			"model":              "claude-opus-4-6",
			"requestsExpression": `dummy`,
		},
		HTTP:           httpContext,
		Integration:    integrationCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Requests:       requestsCtx,
		Expressions:    &contexts.ExpressionContext{Output: []any{"Prompt A", "Prompt B"}},
		Logger:         logrus.NewEntry(logrus.New()),
	}

	err := c.Execute(execCtx)
	require.NoError(t, err)
	require.True(t, executionState.Finished)

	out := executionState.Payloads[0].(map[string]any)["data"].(BatchOutput)
	require.Len(t, out.Results, 2)
	assert.Equal(t, "A", out.Results[0].Text)
	assert.Equal(t, "B", out.Results[1].Text)
}

func Test__CreateBatchMessage__Execute__requestsExpression_objects(t *testing.T) {
	c := &CreateBatchMessage{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"msgbatch_1","processing_status":"ended"}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
				`{"custom_id":"issue-42","result":{"type":"succeeded","message":{"id":"msg_1","content":[{"type":"text","text":"Triaged"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}}}`,
			))},
		},
	}
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}}
	metadataCtx := &contexts.MetadataContext{}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	execCtx := core.ExecutionContext{
		ID: uuid.New(),
		Configuration: map[string]any{
			"model":              "claude-opus-4-6",
			"requestsExpression": `dummy`,
		},
		HTTP:           httpContext,
		Integration:    integrationCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Requests:       &contexts.RequestContext{},
		Expressions: &contexts.ExpressionContext{Output: []any{
			map[string]any{"customId": "issue-42", "prompt": "Triage issue 42"},
		}},
		Logger: logrus.NewEntry(logrus.New()),
	}

	err := c.Execute(execCtx)
	require.NoError(t, err)
	require.True(t, executionState.Finished)

	out := executionState.Payloads[0].(map[string]any)["data"].(BatchOutput)
	require.Len(t, out.Results, 1)
	assert.Equal(t, "issue-42", out.Results[0].CustomID)
	assert.Equal(t, "Triaged", out.Results[0].Text)
}

func Test__CreateBatchMessage__Execute__requestsExpression_notAnArray(t *testing.T) {
	c := &CreateBatchMessage{}
	execCtx := core.ExecutionContext{
		Configuration: map[string]any{
			"model":              "claude-opus-4-6",
			"requestsExpression": `dummy`,
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

func Test__CreateBatchMessage__Execute__requestsExpression_missingPrompt(t *testing.T) {
	c := &CreateBatchMessage{}
	execCtx := core.ExecutionContext{
		Configuration: map[string]any{
			"model":              "claude-opus-4-6",
			"requestsExpression": `dummy`,
		},
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "sk-test"}},
		Metadata:    &contexts.MetadataContext{},
		Expressions: &contexts.ExpressionContext{Output: []any{
			map[string]any{"customId": "issue-1", "prompt": ""},
		}},
		Logger: logrus.NewEntry(logrus.New()),
	}

	err := c.Execute(execCtx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prompt is required")
}
