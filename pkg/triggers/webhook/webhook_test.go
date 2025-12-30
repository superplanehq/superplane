package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestWebhook_Name(t *testing.T) {
	webhook := &Webhook{}
	assert.Equal(t, "webhook", webhook.Name())
}

func TestWebhook_Label(t *testing.T) {
	webhook := &Webhook{}
	assert.Equal(t, "Webhook", webhook.Label())
}

func TestWebhook_Description(t *testing.T) {
	webhook := &Webhook{}
	assert.Equal(t, "Start a new execution chain when a webhook is called", webhook.Description())
}

func TestWebhook_Icon(t *testing.T) {
	webhook := &Webhook{}
	assert.Equal(t, "webhook", webhook.Icon())
}

func TestWebhook_Color(t *testing.T) {
	webhook := &Webhook{}
	assert.Equal(t, "black", webhook.Color())
}

func TestWebhook_Configuration(t *testing.T) {
	webhook := &Webhook{}
	config := webhook.Configuration()

	assert.Len(t, config, 2)

	urlField := config[0]
	assert.Equal(t, "url", urlField.Name)
	assert.Equal(t, "Webhook URL", urlField.Label)
	assert.True(t, urlField.ReadOnly)

	authField := config[1]
	assert.Equal(t, "authentication", authField.Name)
	assert.Equal(t, "Authentication", authField.Label)
	assert.True(t, authField.Required)
	assert.Equal(t, "signature", authField.Default)
	assert.Len(t, authField.TypeOptions.Select.Options, 3)

	// No third field anymore since we removed headerKeyName
}

func TestWebhook_Actions(t *testing.T) {
	webhook := &Webhook{}
	actions := webhook.Actions()

	assert.Len(t, actions, 1)
	assert.Equal(t, "resetAuthentication", actions[0].Name)
	assert.True(t, actions[0].UserAccessible)
}

func TestGetWebhooksBaseURL(t *testing.T) {
	tests := []struct {
		name           string
		baseURL        string
		basePath       string
		expectedResult string
	}{
		{
			name:           "BASE_URL and PUBLIC_API_BASE_PATH are set",
			baseURL:        "https://api.superplane.com",
			basePath:       "/api/v1",
			expectedResult: "https://api.superplane.com/api/v1",
		},
		{
			name:           "BASE_URL set, no basePath",
			baseURL:        "https://api.superplane.com",
			basePath:       "",
			expectedResult: "https://api.superplane.com",
		},
		{
			name:           "both empty",
			baseURL:        "",
			basePath:       "",
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.baseURL != "" {
				os.Setenv("BASE_URL", tt.baseURL)
			} else {
				os.Unsetenv("BASE_URL")
			}

			if tt.basePath != "" {
				os.Setenv("PUBLIC_API_BASE_PATH", tt.basePath)
			} else {
				os.Unsetenv("PUBLIC_API_BASE_PATH")
			}

			defer func() {
				os.Unsetenv("BASE_URL")
				os.Unsetenv("PUBLIC_API_BASE_PATH")
			}()

			result := getWebhooksBaseURL()
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

type mockRequestContext struct {
	workflowID string
	nodeID     string
}

func (m *mockRequestContext) ScheduleActionCall(actionName string, parameters map[string]any, interval time.Duration) error {
	return nil
}

func (m *mockRequestContext) GetWorkflowID() string {
	return m.workflowID
}

func (m *mockRequestContext) GetNodeID() string {
	return m.nodeID
}

type mockNodeMetadataContext struct {
	Node     *models.WorkflowNode
	metadata any
}

func (m *mockNodeMetadataContext) Get() any {
	return m.metadata
}

func (m *mockNodeMetadataContext) Set(metadata any) error {
	m.metadata = metadata
	return nil
}

type mockWebhookContext struct {
	Node *models.WorkflowNode
}

func (m *mockWebhookContext) GetSecret() ([]byte, error) {
	return []byte("test-secret"), nil
}

func (m *mockWebhookContext) ResetSecret() ([]byte, []byte, error) {
	return []byte("test-secret"), []byte("test-secret"), nil
}

func (m *mockWebhookContext) Setup(options *core.WebhookSetupOptions) (*uuid.UUID, error) {

	webhookID := uuid.New()
	m.Node.WebhookID = &webhookID
	return &webhookID, nil
}

func TestWebhook_Setup(t *testing.T) {

	tests := []struct {
		name         string
		config       Configuration
		existingMeta interface{}
		expectedURL  string
		expectError  bool
		baseURL      string
	}{
		{
			name: "setup with no authentication",
			config: Configuration{
				Authentication: "none",
			},
			existingMeta: nil,
			baseURL:      "https://api.superplane.com",
			expectedURL:  "https://api.superplane.com/webhooks/test-workflow-test-node",
		},
		{
			name: "setup with signature authentication",
			config: Configuration{
				Authentication: "signature",
			},
			existingMeta: nil,
			baseURL:      "https://api.superplane.com",
			expectedURL:  "https://api.superplane.com/webhooks/test-workflow-test-node",
		},
		{
			name: "already setup - should not change",
			config: Configuration{
				Authentication: "none",
			},
			existingMeta: Metadata{
				URL:            "https://existing.com/webhook",
				Authentication: "none",
			},
			baseURL:     "https://api.superplane.com",
			expectedURL: "https://existing.com/webhook",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			os.Setenv("BASE_URL", tt.baseURL)
			defer os.Unsetenv("BASE_URL")

			webhook := &Webhook{}

			node := &models.WorkflowNode{}

			metadataCtx := &mockNodeMetadataContext{
				Node:     node,
				metadata: tt.existingMeta,
			}

			webhookCtx := &mockWebhookContext{
				Node: node,
			}

			ctx := core.TriggerContext{
				Configuration:   tt.config,
				MetadataContext: metadataCtx,
				RequestContext: &mockRequestContext{
					workflowID: "test-workflow",
					nodeID:     "test-node",
				},
				WebhookContext: webhookCtx,
			}

			err := webhook.Setup(ctx)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			metadata, ok := metadataCtx.Get().(Metadata)
			require.True(t, ok)

			if tt.existingMeta != nil {
				assert.Equal(t, tt.expectedURL, metadata.URL)
			} else {

				assert.Contains(t, metadata.URL, tt.baseURL+"/webhooks/")

				urlParts := strings.Split(metadata.URL, "/webhooks/")
				require.Len(t, urlParts, 2, "URL should contain /webhooks/ followed by UUID")
				uuidPart := urlParts[1]
				_, err := uuid.Parse(uuidPart)
				assert.NoError(t, err, "URL should end with a valid UUID")
			}

			assert.Equal(t, tt.config.Authentication, metadata.Authentication)

			if tt.config.Authentication == "signature" {
				// Signature key is managed by the webhook context, not stored in metadata
			}
		})
	}
}

func TestWebhook_Setup_AuthenticationMethodChange(t *testing.T) {
	webhook := &Webhook{}

	tests := []struct {
		name                string
		initialAuth         string
		initialSignatureKey string
		newAuth             string
		expectAuthChange    bool
	}{
		{
			name:             "change from none to signature",
			initialAuth:      "none",
			newAuth:          "signature",
			expectAuthChange: true,
		},
		{
			name:                "same authentication method should not change",
			initialAuth:         "signature",
			initialSignatureKey: "existingkey123",
			newAuth:             "signature",
			expectAuthChange:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			existingMetadata := Metadata{
				URL:            "http://localhost:8000/webhooks/existing-webhook-id",
				Authentication: tt.initialAuth,
			}

			metadataCtx := &contexts.MetadataContext{
				Metadata: existingMetadata,
			}

			webhookCtx := &mockWebhookContext{}

			config := Configuration{
				Authentication: tt.newAuth,
			}

			ctx := core.TriggerContext{
				Configuration:   config,
				MetadataContext: metadataCtx,
				WebhookContext:  webhookCtx,
			}

			err := webhook.Setup(ctx)
			require.NoError(t, err)

			metadata, ok := metadataCtx.Get().(Metadata)
			require.True(t, ok)

			if tt.expectAuthChange {

				assert.Equal(t, tt.newAuth, metadata.Authentication)

				assert.Equal(t, existingMetadata.URL, metadata.URL)

				switch tt.newAuth {
				case "signature":
					// For signature auth, we would check signature key if it was in metadata
				case "bearer":
					// For bearer auth, we would check bearer token if it was in metadata
				case "none":
					// No authentication fields to check
				}
			} else {

				assert.Equal(t, tt.initialAuth, metadata.Authentication)
			}
		})
	}
}

func TestWebhook_HandleAction_ResetAuth(t *testing.T) {
	tests := []struct {
		name           string
		existingMeta   Metadata
		expectError    bool
		expectedErrMsg string
		checkField     string
	}{
		{
			name: "reset signature key successfully",
			existingMeta: Metadata{
				URL:            "https://api.superplane.com/webhook",
				Authentication: "signature",
			},
			checkField: "SignatureKey",
		},
		{
			name: "error when no authentication is enabled",
			existingMeta: Metadata{
				URL:            "https://api.superplane.com/webhook",
				Authentication: "none",
			},
			expectError:    true,
			expectedErrMsg: "unsupported authentication method: none",
		},
		{
			name: "error with unsupported authentication method",
			existingMeta: Metadata{
				URL:            "https://api.superplane.com/webhook",
				Authentication: "invalid",
			},
			expectError:    true,
			expectedErrMsg: "unsupported authentication method: invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			webhook := &Webhook{}
			metadataCtx := &contexts.MetadataContext{
				Metadata: tt.existingMeta,
			}

			webhookCtx := &mockWebhookContext{}

			ctx := core.TriggerActionContext{
				Name:            "resetAuthentication",
				MetadataContext: metadataCtx,
				WebhookContext:  webhookCtx,
			}

			_, err := webhook.HandleAction(ctx)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				return
			}

			require.NoError(t, err)

			newMetadata, ok := metadataCtx.Metadata.(Metadata)
			require.True(t, ok)

			switch tt.checkField {
			case "SignatureKey":
				// Since we removed signature key from metadata, we just verify the action completed
				assert.Equal(t, "signature", newMetadata.Authentication)
			}
		})
	}
}

func TestWebhook_HandleAction_UnsupportedAction(t *testing.T) {
	webhook := &Webhook{}
	ctx := core.TriggerActionContext{
		Name: "unsupportedAction",
	}

	_, err := webhook.HandleAction(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "action unsupportedAction not supported")
}

func TestWebhook_HandleWebhook(t *testing.T) {
	tests := []struct {
		name           string
		body           []byte
		headers        map[string]string
		config         Metadata
		expectCode     int
		expectError    bool
		expectedErrMsg string
		expectEmit     bool
	}{
		{
			name: "successful webhook with no authentication",
			body: []byte(`{"message": "hello", "data": {"key": "value"}}`),
			config: Metadata{
				Authentication: "none",
				URL:            "https://api.superplane.com/webhook",
			},
			expectCode: http.StatusOK,
			expectEmit: true,
		},
		{
			name: "payload too large",
			body: make([]byte, MaxEventSize+1),
			config: Metadata{
				Authentication: "none",
			},
			expectCode:     http.StatusRequestEntityTooLarge,
			expectError:    true,
			expectedErrMsg: "payload too large",
		},
		{
			name: "signature authentication - missing header",
			body: []byte(`{"message": "hello"}`),
			config: Metadata{
				Authentication: "signature",
			},
			expectCode:     http.StatusForbidden,
			expectError:    true,
			expectedErrMsg: "missing signature header",
		},
		{
			name: "signature authentication - empty signature",
			body: []byte(`{"message": "hello"}`),
			headers: map[string]string{
				"X-Signature-256": "sha256=",
			},
			config: Metadata{
				Authentication: "signature",
			},
			expectCode:     http.StatusForbidden,
			expectError:    true,
			expectedErrMsg: "invalid signature format",
		},
		{
			name: "signature authentication - invalid signature value",
			body: []byte(`{"message": "hello"}`),
			headers: map[string]string{
				"X-Signature-256": "sha256=invalid-signature",
			},
			config: Metadata{
				Authentication: "signature",
			},
			expectCode:     http.StatusForbidden,
			expectError:    true,
			expectedErrMsg: "invalid signature",
		},
		{
			name: "signature authentication - valid signature",
			body: []byte(`{"message": "hello"}`),
			config: Metadata{
				Authentication: "signature",
			},
			expectCode: http.StatusOK,
			expectEmit: true,
		},
		{
			name: "bearer token authentication - missing header",
			body: []byte(`{"message": "hello"}`),
			config: Metadata{
				Authentication: "bearer",
			},
			expectCode:     http.StatusUnauthorized,
			expectError:    true,
			expectedErrMsg: "missing Authorization header",
		},
		{
			name: "bearer token authentication - invalid token",
			body: []byte(`{"message": "hello"}`),
			headers: map[string]string{
				"Authorization": "Bearer wrong-token",
			},
			config: Metadata{
				Authentication: "bearer",
			},
			expectCode:     http.StatusUnauthorized,
			expectError:    true,
			expectedErrMsg: "invalid Bearer token",
		},
		{
			name: "bearer token authentication - valid token",
			body: []byte(`{"message": "hello"}`),
			headers: map[string]string{
				"Authorization": "Bearer test-secret",
			},
			config: Metadata{
				Authentication: "bearer",
			},
			expectCode: http.StatusOK,
			expectEmit: true,
		},
		{
			name: "invalid JSON body",
			body: []byte(`invalid json`),
			config: Metadata{
				Authentication: "none",
			},
			expectCode:     http.StatusBadRequest,
			expectError:    true,
			expectedErrMsg: "error parsing request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			webhook := &Webhook{}

			headers := http.Header{}
			for k, v := range tt.headers {
				headers.Set(k, v)
			}

			if tt.config.Authentication == "signature" && tt.expectCode == http.StatusOK {
				// Use the mock secret "test-secret"
				h := hmac.New(sha256.New, []byte("test-secret"))
				h.Write(tt.body)
				signature := hex.EncodeToString(h.Sum(nil))
				headers.Set("X-Signature-256", "sha256="+signature)
			}

			eventCtx := &contexts.EventContext{}
			webhookCtx := &mockWebhookContext{}

			ctx := core.WebhookRequestContext{
				Body:           tt.body,
				Headers:        headers,
				Configuration:  tt.config,
				EventContext:   eventCtx,
				WebhookContext: webhookCtx,
			}

			code, err := webhook.HandleWebhook(ctx)

			assert.Equal(t, tt.expectCode, code)

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tt.expectedErrMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			if tt.expectEmit {
				assert.Equal(t, 1, eventCtx.Count())
				assert.Equal(t, "webhook", eventCtx.Payloads[0].Type)

				payload, ok := eventCtx.Payloads[0].Data.(map[string]any)
				require.True(t, ok)

				assert.NotNil(t, payload["body"])
				assert.NotNil(t, payload["headers"])
			} else {
				assert.Equal(t, 0, eventCtx.Count())
			}
		})
	}
}

func TestWebhook_HandleWebhook_SignatureVerification(t *testing.T) {
	webhook := &Webhook{}
	secretKey := "test-secret"
	payload := []byte(`{"test": "data"}`)

	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write(payload)
	validSignature := hex.EncodeToString(h.Sum(nil))

	tests := []struct {
		name           string
		signature      string
		expectCode     int
		expectError    bool
		expectedErrMsg string
	}{
		{
			name:       "valid signature",
			signature:  "sha256=" + validSignature,
			expectCode: http.StatusOK,
		},
		{
			name:           "invalid signature",
			signature:      "sha256=invalidsignature",
			expectCode:     http.StatusForbidden,
			expectError:    true,
			expectedErrMsg: "invalid signature",
		},
		{
			name:           "empty signature after prefix",
			signature:      "sha256=",
			expectCode:     http.StatusForbidden,
			expectError:    true,
			expectedErrMsg: "invalid signature format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := http.Header{}
			headers.Set("X-Signature-256", tt.signature)

			eventCtx := &contexts.EventContext{}
			webhookCtx := &mockWebhookContext{}

			ctx := core.WebhookRequestContext{
				Body:    payload,
				Headers: headers,
				Configuration: Metadata{
					Authentication: "signature",
				},
				EventContext:   eventCtx,
				WebhookContext: webhookCtx,
			}

			code, err := webhook.HandleWebhook(ctx)

			assert.Equal(t, tt.expectCode, code)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, 1, eventCtx.Count())
			}
		})
	}
}

func TestWebhook_URLGeneration_WithTrailingSlash(t *testing.T) {

	os.Setenv("BASE_URL", "https://api.superplane.com/")
	defer os.Unsetenv("BASE_URL")

	webhook := &Webhook{}

	node := &models.WorkflowNode{}

	metadataCtx := &mockNodeMetadataContext{
		Node:     node,
		metadata: nil,
	}

	webhookCtx := &mockWebhookContext{
		Node: node,
	}

	ctx := core.TriggerContext{
		Configuration: Configuration{
			Authentication: "none",
		},
		MetadataContext: metadataCtx,
		RequestContext: &mockRequestContext{
			workflowID: "test-workflow",
			nodeID:     "test-node",
		},
		WebhookContext: webhookCtx,
	}

	err := webhook.Setup(ctx)
	require.NoError(t, err)

	metadata, ok := metadataCtx.Get().(Metadata)
	require.True(t, ok)

	assert.Contains(t, metadata.URL, "https://api.superplane.com/webhooks/")

	urlParts := strings.Split(metadata.URL, "/webhooks/")
	require.Len(t, urlParts, 2, "URL should contain /webhooks/ followed by UUID")
	uuidPart := urlParts[1]
	_, err = uuid.Parse(uuidPart)
	assert.NoError(t, err, "URL should end with a valid UUID")

	pathPart := strings.TrimPrefix(metadata.URL, "https://")
	pathPart = strings.TrimPrefix(pathPart, "http://")
	assert.False(t, strings.Contains(pathPart, "//"), "URL should not contain double slash in path: %s", metadata.URL)
}
