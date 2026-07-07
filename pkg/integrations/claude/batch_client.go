package claude

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// BatchRequestParams mirrors CreateMessageRequest but is used specifically for
// the per-item "params" object of a Message Batches request.
type BatchRequestParams struct {
	Model        string        `json:"model"`
	Messages     []Message     `json:"messages"`
	System       string        `json:"system,omitempty"`
	MaxTokens    int           `json:"max_tokens,omitempty"`
	Temperature  *float64      `json:"temperature,omitempty"`
	OutputConfig *OutputConfig `json:"output_config,omitempty"`
}

// CreateMessageBatchRequestItem is a single entry in a POST /v1/messages/batches request.
type CreateMessageBatchRequestItem struct {
	CustomID string             `json:"custom_id"`
	Params   BatchRequestParams `json:"params"`
}

// createMessageBatchBody is the JSON body for POST /v1/messages/batches.
type createMessageBatchBody struct {
	Requests []CreateMessageBatchRequestItem `json:"requests"`
}

// MessageBatchRequestCounts summarizes the processing status of every request in a batch.
type MessageBatchRequestCounts struct {
	Processing int `json:"processing"`
	Succeeded  int `json:"succeeded"`
	Errored    int `json:"errored"`
	Canceled   int `json:"canceled"`
	Expired    int `json:"expired"`
}

// MessageBatch is the resource returned by the Message Batches API.
type MessageBatch struct {
	ID               string                    `json:"id"`
	Type             string                    `json:"type"`
	ProcessingStatus string                    `json:"processing_status"`
	RequestCounts    MessageBatchRequestCounts `json:"request_counts"`
	CreatedAt        string                    `json:"created_at,omitempty"`
	EndedAt          string                    `json:"ended_at,omitempty"`
	ExpiresAt        string                    `json:"expires_at,omitempty"`
	ResultsURL       string                    `json:"results_url,omitempty"`
}

// MessageBatchResultError is the error object of an "errored" batch result.
type MessageBatchResultError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// messageBatchResultBody is the "result" object nested in each batch results line.
type messageBatchResultBody struct {
	Type    string                   `json:"type"` // succeeded | errored | canceled | expired
	Message *CreateMessageResponse   `json:"message,omitempty"`
	Error   *MessageBatchResultError `json:"error,omitempty"`
}

// MessageBatchResult is a single line of the batch results JSONL stream.
type MessageBatchResult struct {
	CustomID string                 `json:"custom_id"`
	Result   messageBatchResultBody `json:"result"`
}

const messageBatchesPath = "/messages/batches"

// CreateMessageBatch submits a batch of Messages API requests for asynchronous processing.
func (c *Client) CreateMessageBatch(items []CreateMessageBatchRequestItem) (*MessageBatch, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("at least one request is required")
	}

	body, err := json.Marshal(createMessageBatchBody{Requests: items})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal batch request: %w", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, c.BaseURL+messageBatchesPath, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	var batch MessageBatch
	if err := json.Unmarshal(responseBody, &batch); err != nil {
		return nil, fmt.Errorf("failed to unmarshal batch response: %w", err)
	}
	return &batch, nil
}

// GetMessageBatch retrieves the current status of a batch (GET /v1/messages/batches/{id}).
func (c *Client) GetMessageBatch(batchID string) (*MessageBatch, error) {
	if batchID == "" {
		return nil, fmt.Errorf("batch id is required")
	}

	responseBody, err := c.execRequest(http.MethodGet, c.BaseURL+messageBatchesPath+"/"+url.PathEscape(batchID), nil)
	if err != nil {
		return nil, err
	}

	var batch MessageBatch
	if err := json.Unmarshal(responseBody, &batch); err != nil {
		return nil, fmt.Errorf("failed to unmarshal batch: %w", err)
	}
	return &batch, nil
}

// CancelMessageBatch requests cancellation of a still-processing batch
// (POST /v1/messages/batches/{id}/cancel). Already-completed requests are unaffected.
func (c *Client) CancelMessageBatch(batchID string) error {
	if batchID == "" {
		return fmt.Errorf("batch id is required")
	}
	_, err := c.execRequest(http.MethodPost, c.BaseURL+messageBatchesPath+"/"+url.PathEscape(batchID)+"/cancel", nil)
	return err
}

// GetMessageBatchResults streams and parses the batch's results file
// (GET /v1/messages/batches/{id}/results), which is a JSONL document — one JSON
// object per line, in no particular order. Only callable once processing_status
// is "ended".
func (c *Client) GetMessageBatchResults(batchID string) ([]MessageBatchResult, error) {
	if batchID == "" {
		return nil, fmt.Errorf("batch id is required")
	}

	responseBody, err := c.execRequest(http.MethodGet, c.BaseURL+messageBatchesPath+"/"+url.PathEscape(batchID)+"/results", nil)
	if err != nil {
		return nil, err
	}

	var results []MessageBatchResult
	scanner := bufio.NewScanner(bytes.NewReader(responseBody))
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var result MessageBatchResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal batch result line: %w", err)
		}
		results = append(results, result)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read batch results: %w", err)
	}
	return results, nil
}
