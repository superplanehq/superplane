package claude

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

// --- Mocks ---

// mockHTTPContext implements core.HTTPContext for testing
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

// mockIntegrationContext implements core.IntegrationContext for testing
type mockIntegrationContext struct {
	config map[string][]byte
	ready  bool
	errMsg string
}

func newMockIntegrationContext() *mockIntegrationContext {
	return &mockIntegrationContext{
		config: make(map[string][]byte),
	}
}

func (m *mockIntegrationContext) GetConfig(name string) ([]byte, error) {
	val, ok := m.config[name]
	if !ok {
		return nil, fmt.Errorf("config not found: %s", name)
	}
	return val, nil
}

func (m *mockIntegrationContext) Ready() {
	m.ready = true
}

func (m *mockIntegrationContext) Error(message string) {
	m.errMsg = message
}

// Stubs for other interface methods
func (m *mockIntegrationContext) ID() uuid.UUID                       { return uuid.New() }
func (m *mockIntegrationContext) GetMetadata() any                    { return nil }
func (m *mockIntegrationContext) SetMetadata(any)                     {}
func (m *mockIntegrationContext) NewBrowserAction(core.BrowserAction) {}
func (m *mockIntegrationContext) RemoveBrowserAction()                {}
func (m *mockIntegrationContext) SetSecret(string, []byte) error      { return nil }
func (m *mockIntegrationContext) GetSecrets() ([]core.IntegrationSecret, error) {
	return nil, nil
}
func (m *mockIntegrationContext) RequestWebhook(any) error { return nil }
func (m *mockIntegrationContext) Subscribe(any) (*uuid.UUID, error) {
	u := uuid.New()
	return &u, nil
}
func (m *mockIntegrationContext) ScheduleResync(time.Duration) error { return nil }
func (m *mockIntegrationContext) ScheduleActionCall(string, any, time.Duration) error {
	return nil
}
func (m *mockIntegrationContext) ListSubscriptions() ([]core.IntegrationSubscriptionContext, error) {
	return nil, nil
}

// --- Tests ---

func TestClaude_Configuration(t *testing.T) {
	i := &Claude{}
	configs := i.Configuration()

	// Use string here instead of configuration.FieldType to avoid type errors
	expectedFields := map[string]struct {
		Type     string
		Required bool
	}{
		"apiKey": {string(configuration.FieldTypeString), true},
	}

	if len(configs) != len(expectedFields) {
		t.Errorf("expected %d config fields, got %d", len(expectedFields), len(configs))
	}

	for _, field := range configs {
		expected, ok := expectedFields[field.Name]
		if !ok {
			t.Errorf("unexpected field: %s", field.Name)
			continue
		}
		// Cast field.Type to string for comparison
		if string(field.Type) != expected.Type {
			t.Errorf("field %s: expected type %s, got %s", field.Name, expected.Type, field.Type)
		}
		if field.Required != expected.Required {
			t.Errorf("field %s: expected required %v, got %v", field.Name, expected.Required, field.Required)
		}
	}
}

func TestClaude_Sync(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	tests := []struct {
		name          string
		config        map[string]interface{}
		mockResponses func(*http.Request) *http.Response
		expectError   bool
		expectReady   bool
	}{
		{
			name: "Success",
			config: map[string]interface{}{
				"apiKey": "sk-ant-test",
			},
			mockResponses: func(req *http.Request) *http.Response {
				if req.URL.Path == "/v1/models" {
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewBufferString(`{"data": [{"id": "claude-2"}]}`)),
					}
				}
				return &http.Response{StatusCode: 404, Body: io.NopCloser(bytes.NewBufferString(""))}
			},
			expectError: false,
			expectReady: true,
		},
		{
			name: "Missing API Key",
			config: map[string]interface{}{
				"apiKey": "",
			},
			expectError: true,
		},
		{
			name: "Verify Fails (401)",
			config: map[string]interface{}{
				"apiKey": "invalid",
			},
			mockResponses: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: 401,
					Body:       io.NopCloser(bytes.NewBufferString(`{"error": {"type": "authentication_error", "message": "invalid api key"}}`)),
				}
			},
			expectError: true,
			expectReady: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Claude{}
			mockInt := newMockIntegrationContext()
			mockHTTP := &mockHTTPContext{RoundTripFunc: tt.mockResponses}

			// Populate mock integration config (used by NewClient)
			if v, ok := tt.config["apiKey"].(string); ok {
				mockInt.config["apiKey"] = []byte(v)
			}

			ctx := core.SyncContext{
				Logger:        logger,
				Configuration: tt.config, // Used by mapstructure decode
				HTTP:          mockHTTP,
				Integration:   mockInt,
			}

			err := i.Sync(ctx)

			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.expectReady && !mockInt.ready {
				t.Error("expected integration to be marked ready")
			}
		})
	}
}

func TestClaude_ListResources(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	tests := []struct {
		name          string
		resourceType  string
		config        map[string][]byte // Config is pulled from integration context
		mockResponses func(*http.Request) *http.Response
		expectedIDs   []string
		expectError   bool
	}{
		{
			name:         "List Models Success",
			resourceType: "model",
			config: map[string][]byte{
				"apiKey": []byte("test"),
			},
			mockResponses: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(bytes.NewBufferString(`{
						"data": [
							{"id": "claude-2.1"},
							{"id": "claude-instant-1.2"}
						]
					}`)),
				}
			},
			expectedIDs: []string{"claude-2.1", "claude-instant-1.2"},
		},
		{
			name:         "Invalid Resource Type",
			resourceType: "deployment",
			expectedIDs:  []string{}, // Returns empty list, no error
		},
		{
			name:         "API Error",
			resourceType: "model",
			config: map[string][]byte{
				"apiKey": []byte("test"),
			},
			mockResponses: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: 500,
					Body:       io.NopCloser(bytes.NewBufferString("server error")),
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Claude{}
			mockInt := newMockIntegrationContext()
			mockInt.config = tt.config
			mockHTTP := &mockHTTPContext{RoundTripFunc: tt.mockResponses}

			ctx := core.ListResourcesContext{
				Logger:      logger,
				HTTP:        mockHTTP,
				Integration: mockInt,
			}

			resources, err := i.ListResources(tt.resourceType, ctx)

			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(resources) != len(tt.expectedIDs) {
				t.Errorf("expected %d resources, got %d", len(tt.expectedIDs), len(resources))
			}

			for idx, id := range tt.expectedIDs {
				if resources[idx].ID != id {
					t.Errorf("resource %d: expected ID %s, got %s", idx, id, resources[idx].ID)
				}
			}
		})
	}
}
