package flyio

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

func Test__FlyIO__Components(t *testing.T) {
	f := &FlyIO{}
	components := f.Components()

	require.Len(t, components, 1)
	assert.Equal(t, "flyio.listApps", components[0].Name())
}

func Test__FlyIO__Triggers(t *testing.T) {
	f := &FlyIO{}
	triggers := f.Triggers()

	require.Len(t, triggers, 1)
	assert.Equal(t, "flyio.onAppStateChange", triggers[0].Name())
}

func Test__FlyIO__Configuration(t *testing.T) {
	f := &FlyIO{}
	config := f.Configuration()

	require.Len(t, config, 2)

	// Check for apiToken field
	var apiTokenField, orgSlugField bool
	for _, field := range config {
		if field.Name == "apiToken" {
			apiTokenField = true
			assert.True(t, field.Required)
			assert.True(t, field.Sensitive)
		}
		if field.Name == "orgSlug" {
			orgSlugField = true
			assert.False(t, field.Required)
		}
	}

	assert.True(t, apiTokenField, "apiToken field should exist")
	assert.True(t, orgSlugField, "orgSlug field should exist")
}

func Test__FlyIO__Actions(t *testing.T) {
	f := &FlyIO{}
	actions := f.Actions()
	assert.Empty(t, actions)
}

func Test__FlyIO__Cleanup(t *testing.T) {
	f := &FlyIO{}
	err := f.Cleanup(core.IntegrationCleanupContext{})
	assert.NoError(t, err)
}

// Test Sync() - Successful sync with valid token
func Test__FlyIO__Sync__Success(t *testing.T) {
	f := &FlyIO{}

	// Mock HTTP client that returns a successful response
	mockHTTP := &mockHTTPContext{
		doFunc: func(req *http.Request) (*http.Response, error) {
			// Verify authorization header
			assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
			assert.Contains(t, req.URL.String(), "org_slug=personal")

			return &http.Response{
				StatusCode: 200,
				Body: newMockReadCloser(`{
					"total_apps": 2,
					"apps": [
						{"id": "app1", "name": "my-app-1", "status": "deployed"},
						{"id": "app2", "name": "my-app-2", "status": "deployed"}
					]
				}`),
			}, nil
		},
	}

	mockIntegration := &mockIntegrationContext{
		config: map[string][]byte{
			"apiToken": []byte("test-token"),
			"orgSlug":  []byte("personal"),
		},
	}

	ctx := core.SyncContext{
		HTTP:        mockHTTP,
		Integration: mockIntegration,
		Configuration: map[string]interface{}{
			"orgSlug": "personal",
		},
	}

	err := f.Sync(ctx)
	require.NoError(t, err)

	// Verify metadata was set
	metadata := mockIntegration.metadata.(Metadata)
	assert.Len(t, metadata.Apps, 2)
	assert.Equal(t, "my-app-1", metadata.Apps[0].Name)
	assert.Equal(t, "my-app-2", metadata.Apps[1].Name)
}

// Test Sync() - Invalid API token
func Test__FlyIO__Sync__InvalidToken(t *testing.T) {
	f := &FlyIO{}

	mockHTTP := &mockHTTPContext{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 401,
				Body:       newMockReadCloser(`{"error": "unauthorized"}`),
			}, nil
		},
	}

	mockIntegration := &mockIntegrationContext{
		config: map[string][]byte{
			"apiToken": []byte("invalid-token"),
		},
	}

	ctx := core.SyncContext{
		HTTP:        mockHTTP,
		Integration: mockIntegration,
		Configuration: map[string]interface{}{
			"orgSlug": "personal",
		},
	}

	err := f.Sync(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error listing apps")
}

// Test Sync() - Network error
func Test__FlyIO__Sync__NetworkError(t *testing.T) {
	f := &FlyIO{}

	mockHTTP := &mockHTTPContext{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("network timeout")
		},
	}

	mockIntegration := &mockIntegrationContext{
		config: map[string][]byte{
			"apiToken": []byte("test-token"),
		},
	}

	ctx := core.SyncContext{
		HTTP:        mockHTTP,
		Integration: mockIntegration,
		Configuration: map[string]interface{}{},
	}

	err := f.Sync(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error listing apps")
}

// Test Sync() - Default org slug when not provided
func Test__FlyIO__Sync__DefaultOrgSlug(t *testing.T) {
	f := &FlyIO{}

	mockHTTP := &mockHTTPContext{
		doFunc: func(req *http.Request) (*http.Response, error) {
			// Should default to "personal"
			assert.Contains(t, req.URL.String(), "org_slug=personal")

			return &http.Response{
				StatusCode: 200,
				Body:       newMockReadCloser(`{"total_apps": 0, "apps": []}`),
			}, nil
		},
	}

	mockIntegration := &mockIntegrationContext{
		config: map[string][]byte{
			"apiToken": []byte("test-token"),
		},
	}

	ctx := core.SyncContext{
		HTTP:        mockHTTP,
		Integration: mockIntegration,
		Configuration: map[string]interface{}{
			// No orgSlug provided
		},
	}

	err := f.Sync(ctx)
	require.NoError(t, err)
}

// Mock implementations for testing

type mockHTTPContext struct {
	doFunc func(*http.Request) (*http.Response, error)
}

func (m *mockHTTPContext) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

type mockIntegrationContext struct {
	config   map[string][]byte
	metadata interface{}
	ready    bool
}

func (m *mockIntegrationContext) ID() uuid.UUID {
	return uuid.MustParse("00000000-0000-0000-0000-000000000000")
}

func (m *mockIntegrationContext) GetConfig(key string) ([]byte, error) {
	if val, ok := m.config[key]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("config key not found: %s", key)
}

func (m *mockIntegrationContext) SetMetadata(data interface{}) {
	m.metadata = data
}

func (m *mockIntegrationContext) Ready() {
	m.ready = true
}

func (m *mockIntegrationContext) GetMetadata() interface{} {
	return m.metadata
}

func (m *mockIntegrationContext) Error(msg string) {
	// no-op for testing
}

func (m *mockIntegrationContext) NewBrowserAction(action core.BrowserAction) {
	// no-op for testing
}

func (m *mockIntegrationContext) RemoveBrowserAction() {
	// no-op for testing
}

func (m *mockIntegrationContext) SetSecret(name string, value []byte) error {
	return nil
}

func (m *mockIntegrationContext) GetSecrets() ([]core.IntegrationSecret, error) {
	return nil, nil
}

func (m *mockIntegrationContext) RequestWebhook(configuration any) error {
	return nil
}

func (m *mockIntegrationContext) Subscribe(any) (*uuid.UUID, error) {
	id := uuid.New()
	return &id, nil
}

func (m *mockIntegrationContext) ScheduleResync(interval time.Duration) error {
	return nil
}

func (m *mockIntegrationContext) ScheduleActionCall(actionName string, parameters any, interval time.Duration) error {
	return nil
}

func (m *mockIntegrationContext) ListSubscriptions() ([]core.IntegrationSubscriptionContext, error) {
	return nil, nil
}

type mockReadCloser struct {
	*stringReader
}

func newMockReadCloser(s string) *mockReadCloser {
	return &mockReadCloser{&stringReader{s: s}}
}

func (m *mockReadCloser) Close() error {
	return nil
}

type stringReader struct {
	s   string
	pos int
}

func (r *stringReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.s) {
		return 0, io.EOF
	}
	n = copy(p, r.s[r.pos:])
	r.pos += n
	if r.pos >= len(r.s) {
		return n, io.EOF
	}
	return n, nil
}

