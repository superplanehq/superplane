package openai

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
)

// --- Mocks ---

type mockHTTPContext struct {
	RoundTripFunc func(req *http.Request) *http.Response
}

func (m *mockHTTPContext) Do(req *http.Request) (*http.Response, error) {
	if m.RoundTripFunc != nil {
		return m.RoundTripFunc(req), nil
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
	}, nil
}

type mockIntegrationContext struct {
	config map[string][]byte
}

func (m *mockIntegrationContext) GetConfig(name string) ([]byte, error) {
	val, ok := m.config[name]
	if !ok {
		return nil, fmt.Errorf("config not found: %s", name)
	}
	return val, nil
}

func (m *mockIntegrationContext) ID() uuid.UUID                       { return uuid.New() }
func (m *mockIntegrationContext) GetMetadata() any                    { return nil }
func (m *mockIntegrationContext) SetMetadata(any)                     {}
func (m *mockIntegrationContext) Ready()                              {}
func (m *mockIntegrationContext) Error(string)                        {}
func (m *mockIntegrationContext) NewBrowserAction(core.BrowserAction) {}
func (m *mockIntegrationContext) RemoveBrowserAction()                {}
func (m *mockIntegrationContext) SetSecret(string, []byte) error      { return nil }
func (m *mockIntegrationContext) GetSecrets() ([]core.IntegrationSecret, error) {
	return nil, nil
}
func (m *mockIntegrationContext) RequestWebhook(any) error            { return nil }
func (m *mockIntegrationContext) Subscribe(any) (*uuid.UUID, error)   { return nil, nil }
func (m *mockIntegrationContext) ScheduleResync(time.Duration) error  { return nil }
func (m *mockIntegrationContext) ScheduleActionCall(string, any, time.Duration) error {
	return nil
}
func (m *mockIntegrationContext) ListSubscriptions() ([]core.IntegrationSubscriptionContext, error) {
	return nil, nil
}

// --- Tests ---

func TestNewClient_BaseURL(t *testing.T) {
	mockHTTP := &mockHTTPContext{}

	tests := []struct {
		name      string
		config    map[string][]byte
		expectURL string
	}{
		{
			name: "Default base URL when not configured",
			config: map[string][]byte{
				"apiKey": []byte("sk-123"),
			},
			expectURL: defaultBaseURL,
		},
		{
			name: "Custom base URL",
			config: map[string][]byte{
				"apiKey":  []byte("sk-123"),
				"baseURL": []byte("https://custom.example.com/v1"),
			},
			expectURL: "https://custom.example.com/v1",
		},
		{
			name: "Empty base URL falls back to default",
			config: map[string][]byte{
				"apiKey":  []byte("sk-123"),
				"baseURL": []byte(""),
			},
			expectURL: defaultBaseURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockInt := &mockIntegrationContext{config: tt.config}

			client, err := NewClient(mockHTTP, mockInt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if client.BaseURL != tt.expectURL {
				t.Errorf("expected BaseURL '%s', got '%s'", tt.expectURL, client.BaseURL)
			}
		})
	}
}

func TestClient_CustomBaseURL_UsedInRequests(t *testing.T) {
	customURL := "https://my-provider.example.com/v1"
	var requestedURL string

	mockHTTP := &mockHTTPContext{
		RoundTripFunc: func(req *http.Request) *http.Response {
			requestedURL = req.URL.String()
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"data": []}`)),
			}
		},
	}

	client := &Client{
		APIKey:  "test-key",
		BaseURL: customURL,
		http:    mockHTTP,
	}

	_ = client.Verify()

	if requestedURL != customURL+"/models" {
		t.Errorf("expected request to %s/models, got %s", customURL, requestedURL)
	}
}
