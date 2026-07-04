package prometheus

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

// mockClient implements the Client interface for tests.
type mockClient struct {
	projectID string
	getFunc   func(ctx context.Context, url string) ([]byte, error)
}

func (m *mockClient) GetURL(ctx context.Context, url string) ([]byte, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, url)
	}
	return nil, fmt.Errorf("unexpected GetURL(%s)", url)
}

func (m *mockClient) ProjectID() string {
	return m.projectID
}

// withFactory installs a factory that returns the given mock client.
func withFactory(mc *mockClient) {
	SetClientFactory(func(httpCtx core.HTTPContext, integration core.IntegrationContext) (Client, error) {
		return mc, nil
	})
}

// firstPayload returns the single emitted payload's data map.
func firstPayload(t *testing.T, state *contexts.ExecutionStateContext) map[string]any {
	t.Helper()
	require.Len(t, state.Payloads, 1)
	wrapped, ok := state.Payloads[0].(map[string]any)
	require.True(t, ok)
	data, ok := wrapped["data"].(map[string]any)
	require.True(t, ok)
	return data
}
