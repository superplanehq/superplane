package claude

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Client__CreateMessageBatch(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"id":"msgbatch_1","type":"message_batch","processing_status":"in_progress","request_counts":{"processing":2}}`)),
		}},
	}
	client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: httpCtx}

	batch, err := client.CreateMessageBatch([]CreateMessageBatchRequestItem{
		{CustomID: "r1", Params: BatchRequestParams{Model: "claude-opus-4-6", Messages: []Message{{Role: "user", Content: "hi"}}}},
	})
	require.NoError(t, err)
	assert.Equal(t, "msgbatch_1", batch.ID)
	assert.Equal(t, "in_progress", batch.ProcessingStatus)
	assert.Equal(t, 2, batch.RequestCounts.Processing)

	require.Len(t, httpCtx.Requests, 1)
	req := httpCtx.Requests[0]
	assert.Equal(t, http.MethodPost, req.Method)
	assert.True(t, strings.HasSuffix(req.URL.Path, "/messages/batches"))
	assert.Equal(t, "k", req.Header.Get("x-api-key"))
	assert.Empty(t, req.Header.Get("anthropic-beta"))
}

func Test__Client__CreateMessageBatch__empty(t *testing.T) {
	client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: &contexts.HTTPContext{}}
	_, err := client.CreateMessageBatch(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one request")
}

func Test__Client__GetMessageBatch(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"id":"msgbatch_1","processing_status":"ended"}`)),
		}},
	}
	client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: httpCtx}

	batch, err := client.GetMessageBatch("msgbatch_1")
	require.NoError(t, err)
	assert.Equal(t, "ended", batch.ProcessingStatus)
	assert.True(t, strings.HasSuffix(httpCtx.Requests[0].URL.Path, "/messages/batches/msgbatch_1"))
}

func Test__Client__CancelMessageBatch(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"id":"msgbatch_1","processing_status":"canceling"}`)),
		}},
	}
	client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: httpCtx}

	err := client.CancelMessageBatch("msgbatch_1")
	require.NoError(t, err)
	req := httpCtx.Requests[0]
	assert.Equal(t, http.MethodPost, req.Method)
	assert.True(t, strings.HasSuffix(req.URL.Path, "/messages/batches/msgbatch_1/cancel"))
}

func Test__Client__GetMessageBatchResults(t *testing.T) {
	jsonl := `{"custom_id":"r1","result":{"type":"succeeded","message":{"id":"msg_1","content":[{"type":"text","text":"Hello"}],"stop_reason":"end_turn","usage":{"input_tokens":3,"output_tokens":2}}}}
{"custom_id":"r2","result":{"type":"errored","error":{"type":"invalid_request_error","message":"bad"}}}
`
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(jsonl)),
		}},
	}
	client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: httpCtx}

	results, err := client.GetMessageBatchResults("msgbatch_1")
	require.NoError(t, err)
	require.Len(t, results, 2)

	assert.Equal(t, "r1", results[0].CustomID)
	assert.Equal(t, "succeeded", results[0].Result.Type)
	require.NotNil(t, results[0].Result.Message)
	assert.Equal(t, "Hello", extractMessageText(results[0].Result.Message))

	assert.Equal(t, "r2", results[1].CustomID)
	assert.Equal(t, "errored", results[1].Result.Type)
	require.NotNil(t, results[1].Result.Error)
	assert.Equal(t, "bad", results[1].Result.Error.Message)

	assert.True(t, strings.HasSuffix(httpCtx.Requests[0].URL.Path, "/messages/batches/msgbatch_1/results"))
}

func Test__Client__GetMessageBatchResults__emptyID(t *testing.T) {
	client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: &contexts.HTTPContext{}}
	_, err := client.GetMessageBatchResults("")
	require.Error(t, err)
}
