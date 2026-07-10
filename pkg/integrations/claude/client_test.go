package claude

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestNewClient(t *testing.T) {
	httpCtx := &contexts.HTTPContext{}

	tests := []struct {
		name           string
		integrationCtx *contexts.IntegrationContext
		expectError    bool
	}{
		{
			name: "Success",
			integrationCtx: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "sk-123"},
			},
			expectError: false,
		},
		{
			name:           "Nil Context",
			integrationCtx: nil,
			expectError:    true,
		},
		{
			name: "Missing API Key",
			integrationCtx: &contexts.IntegrationContext{
				Configuration: map[string]any{},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var integrationCtx core.IntegrationContext
			if tt.integrationCtx != nil {
				integrationCtx = tt.integrationCtx
			}

			client, err := NewClient(httpCtx, integrationCtx)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if client == nil {
					t.Error("expected client, got nil")
				}
				if client.APIKey != "sk-123" {
					t.Errorf("expected API Key 'sk-123', got %s", client.APIKey)
				}
			}
		})
	}
}

func TestClient_Verify(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		expectError    bool
	}{
		{name: "Success 200", responseStatus: 200, expectError: false},
		{name: "Unauthorized 401", responseStatus: 401, expectError: true},
		{name: "Server Error 500", responseStatus: 500, expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpCtx := &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: tt.responseStatus,
						Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
					},
				},
			}

			client := &Client{
				APIKey:  "test-key",
				BaseURL: defaultBaseURL,
				http:    httpCtx,
			}

			err := client.Verify()
			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestClient_ListModels(t *testing.T) {
	jsonResp := `{
		"data": [
			{"id": "claude-3-opus"},
			{"id": "claude-3-sonnet"}
		]
	}`
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
			},
		},
	}

	client := &Client{http: httpCtx, BaseURL: defaultBaseURL}

	models, err := client.ListModels()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(models) != 2 {
		t.Errorf("expected 2 models, got %d", len(models))
	}
	if models[0].ID != "claude-3-opus" {
		t.Errorf("expected first model to be claude-3-opus, got %s", models[0].ID)
	}
}

func TestClient_CreateMessage(t *testing.T) {
	jsonResp := `{
		"id": "msg_123",
		"type": "message",
		"role": "assistant",
		"content": [
			{"type": "text", "text": "Hello there"}
		],
		"model": "claude-3-opus",
		"stop_reason": "end_turn",
		"usage": {"input_tokens": 10, "output_tokens": 5}
	}`
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
			},
		},
	}

	client := &Client{
		APIKey:  "my-secret-key",
		BaseURL: defaultBaseURL,
		http:    httpCtx,
	}

	req := CreateMessageRequest{
		Model: "claude-3-opus",
		Messages: []Message{
			{Role: "user", Content: "Hi"},
		},
		MaxTokens: 1024,
	}

	resp, err := client.CreateMessage(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.ID != "msg_123" {
		t.Errorf("expected ID msg_123, got %s", resp.ID)
	}
	if len(resp.Content) == 0 || resp.Content[0].Text != "Hello there" {
		t.Error("response content mismatch")
	}

	// Verify request that was sent
	if len(httpCtx.Requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(httpCtx.Requests))
	}
	sentReq := httpCtx.Requests[0]
	if sentReq.Header.Get("x-api-key") != "my-secret-key" {
		t.Errorf("missing or wrong x-api-key header")
	}
	if sentReq.Header.Get("anthropic-version") != "2023-06-01" {
		t.Errorf("missing or wrong anthropic-version header")
	}
	if sentReq.Header.Get("Content-Type") != "application/json" {
		t.Errorf("missing or wrong Content-Type")
	}
	bodyBytes, _ := io.ReadAll(sentReq.Body)
	var sentBody CreateMessageRequest
	if err := json.Unmarshal(bodyBytes, &sentBody); err != nil {
		t.Errorf("failed to unmarshal sent body: %v", err)
	}
	if sentBody.Model != "claude-3-opus" {
		t.Errorf("sent wrong model: %s", sentBody.Model)
	}
}

func TestClient_ErrorHandling(t *testing.T) {
	tests := []struct {
		name             string
		responseBody     string
		statusCode       int
		expectedErrorMsg string
	}{
		{
			name: "Structured Anthropic Error",
			responseBody: `{
				"error": {
					"type": "invalid_request_error",
					"message": "max_tokens is too large"
				}
			}`,
			statusCode:       400,
			expectedErrorMsg: "request failed (400): max_tokens is too large",
		},
		{
			name:             "Unstructured Plain Text Error",
			responseBody:     `Bad Gateway`,
			statusCode:       502,
			expectedErrorMsg: "request failed (502): Bad Gateway",
		},
		{
			name: "Auth Error (401)",
			responseBody: `{
				"error": {
					"type": "authentication_error",
					"message": "invalid x-api-key"
				}
			}`,
			statusCode:       401,
			expectedErrorMsg: "Claude credentials are invalid or expired: invalid x-api-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpCtx := &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(bytes.NewBufferString(tt.responseBody)),
					},
				},
			}

			client := &Client{http: httpCtx, BaseURL: defaultBaseURL}

			// We use ListModels as a simple way to trigger execRequest
			_, err := client.ListModels()

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if err.Error() != tt.expectedErrorMsg {
				t.Errorf("expected error message '%s', got '%s'", tt.expectedErrorMsg, err.Error())
			}
		})
	}
}

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
