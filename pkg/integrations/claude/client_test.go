package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
)

// --- Mocks ---

// mockHTTPContextForClient implements core.HTTPContext
type mockHTTPContextForClient struct {
	// RoundTripFunc allows us to define the response for a specific request
	RoundTripFunc func(req *http.Request) *http.Response
}

func (m *mockHTTPContextForClient) Do(req *http.Request) (*http.Response, error) {
	if m.RoundTripFunc != nil {
		return m.RoundTripFunc(req), nil
	}
	// Default fallback
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
	}, nil
}

// mockIntegrationContextForClient implements core.IntegrationContext
type mockIntegrationContextForClient struct {
	config map[string][]byte
}

func newMockIntegrationContextForClient() *mockIntegrationContextForClient {
	return &mockIntegrationContextForClient{
		config: make(map[string][]byte),
	}
}

func (m *mockIntegrationContextForClient) GetConfig(name string) ([]byte, error) {
	val, ok := m.config[name]
	if !ok {
		return nil, fmt.Errorf("config not found: %s", name)
	}
	return val, nil
}

// Stubs to satisfy the core.IntegrationContext interface
func (m *mockIntegrationContextForClient) ID() uuid.UUID               { return uuid.New() }
func (m *mockIntegrationContextForClient) GetMetadata() any            { return nil }
func (m *mockIntegrationContextForClient) SetMetadata(any)             {}
func (m *mockIntegrationContextForClient) Ready()                      {}
func (m *mockIntegrationContextForClient) Error(message string)        {}
func (m *mockIntegrationContextForClient) NewBrowserAction(core.BrowserAction) {}
func (m *mockIntegrationContextForClient) RemoveBrowserAction()        {}
func (m *mockIntegrationContextForClient) SetSecret(string, []byte) error { return nil }
func (m *mockIntegrationContextForClient) GetSecrets() ([]core.IntegrationSecret, error) {
	return nil, nil
}
func (m *mockIntegrationContextForClient) RequestWebhook(any) error { return nil }
func (m *mockIntegrationContextForClient) Subscribe(any) (*uuid.UUID, error) {
	return nil, nil
}
func (m *mockIntegrationContextForClient) ScheduleResync(time.Duration) error { return nil }
func (m *mockIntegrationContextForClient) ScheduleActionCall(string, any, time.Duration) error {
	return nil
}
func (m *mockIntegrationContextForClient) ListSubscriptions() ([]core.IntegrationSubscriptionContext, error) {
	return nil, nil
}

// --- Tests ---

func TestNewClient(t *testing.T) {
	mockHTTP := &mockHTTPContextForClient{}

	tests := []struct {
		name        string
		setupMock   func(*mockIntegrationContextForClient)
		ctx         core.IntegrationContext
		expectError bool
	}{
		{
			name: "Success",
			setupMock: func(m *mockIntegrationContextForClient) {
				m.config["apiKey"] = []byte("sk-123")
				m.config["anthropicVersion"] = []byte("2023-06-01")
			},
			expectError: false,
		},
		{
			name:        "Nil Context",
			ctx:         nil,
			expectError: true,
		},
		{
			name: "Missing API Key",
			setupMock: func(m *mockIntegrationContextForClient) {
				// No API Key
				m.config["anthropicVersion"] = []byte("2023-06-01")
			},
			expectError: true,
		},
		{
			name: "Missing Version",
			setupMock: func(m *mockIntegrationContextForClient) {
				m.config["apiKey"] = []byte("sk-123")
				// No Version
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var integrationCtx core.IntegrationContext

			if tt.ctx != nil {
				// Use explicitly provided context (e.g. nil)
				integrationCtx = tt.ctx
			} else {
				// Use the mock
				mockInt := newMockIntegrationContextForClient()
				if tt.setupMock != nil {
					tt.setupMock(mockInt)
				}
				integrationCtx = mockInt
			}

			client, err := NewClient(mockHTTP, integrationCtx)

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
				if client.AnthropicVersion != "2023-06-01" {
					t.Errorf("expected Version '2023-06-01', got %s", client.AnthropicVersion)
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
			mockHTTP := &mockHTTPContextForClient{
				RoundTripFunc: func(req *http.Request) *http.Response {
					if req.Method != http.MethodGet {
						t.Errorf("expected method GET, got %s", req.Method)
					}
					if req.URL.String() != "https://api.anthropic.com/v1/models" {
						t.Errorf("expected URL .../models, got %s", req.URL.String())
					}
					return &http.Response{
						StatusCode: tt.responseStatus,
						Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
					}
				},
			}

			client := &Client{
				APIKey:           "test-key",
				AnthropicVersion: "test-ver",
				BaseURL:          defaultBaseURL,
				http:             mockHTTP,
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
	mockHTTP := &mockHTTPContextForClient{
		RoundTripFunc: func(req *http.Request) *http.Response {
			jsonResp := `{
				"data": [
					{"id": "claude-3-opus"},
					{"id": "claude-3-sonnet"}
				]
			}`
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
			}
		},
	}

	client := &Client{http: mockHTTP, BaseURL: defaultBaseURL}

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
	mockHTTP := &mockHTTPContextForClient{
		RoundTripFunc: func(req *http.Request) *http.Response {
			// Verify Headers
			if req.Header.Get("x-api-key") != "my-secret-key" {
				t.Errorf("missing or wrong x-api-key header")
			}
			if req.Header.Get("anthropic-version") != "2023-06-01" {
				t.Errorf("missing or wrong anthropic-version header")
			}
			if req.Header.Get("Content-Type") != "application/json" {
				t.Errorf("missing or wrong Content-Type")
			}

			// Verify Body
			bodyBytes, _ := io.ReadAll(req.Body)
			var sentReq CreateMessageRequest
			if err := json.Unmarshal(bodyBytes, &sentReq); err != nil {
				t.Errorf("failed to unmarshal sent body: %v", err)
			}
			if sentReq.Model != "claude-3-opus" {
				t.Errorf("sent wrong model: %s", sentReq.Model)
			}

			// Return Success
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
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
			}
		},
	}

	client := &Client{
		APIKey:           "my-secret-key",
		AnthropicVersion: "2023-06-01",
		BaseURL:          defaultBaseURL,
		http:             mockHTTP,
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
			mockHTTP := &mockHTTPContextForClient{
				RoundTripFunc: func(req *http.Request) *http.Response {
					return &http.Response{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(bytes.NewBufferString(tt.responseBody)),
					}
				},
			}

			client := &Client{http: mockHTTP, BaseURL: defaultBaseURL}

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