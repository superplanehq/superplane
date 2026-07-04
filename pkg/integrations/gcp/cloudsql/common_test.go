package cloudsql

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

// mockClient is a configurable cloudsql.Client used by the component tests.
type mockClient struct {
	projectID  string
	getFunc    func(ctx context.Context, url string) ([]byte, error)
	postFunc   func(ctx context.Context, url string, body any) ([]byte, error)
	deleteFunc func(ctx context.Context, url string) ([]byte, error)
}

func (m *mockClient) GetURL(ctx context.Context, url string) ([]byte, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, url)
	}
	return nil, fmt.Errorf("unexpected GetURL(%s)", url)
}

func (m *mockClient) PostURL(ctx context.Context, url string, body any) ([]byte, error) {
	if m.postFunc != nil {
		return m.postFunc(ctx, url, body)
	}
	return nil, fmt.Errorf("unexpected PostURL(%s)", url)
}

func (m *mockClient) DeleteURL(ctx context.Context, url string) ([]byte, error) {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, url)
	}
	return nil, fmt.Errorf("unexpected DeleteURL(%s)", url)
}

func (m *mockClient) ProjectID() string { return m.projectID }

// withFactory installs a mock client for the duration of a component test.
func withFactory(mc *mockClient) {
	SetClientFactory(func(httpCtx core.HTTPContext, integration core.IntegrationContext) (Client, error) {
		return mc, nil
	})
}

// firstData returns the data map of the first emitted payload.
func firstData(t *testing.T, state *contexts.ExecutionStateContext) map[string]any {
	t.Helper()
	require.NotEmpty(t, state.Payloads)
	return state.Payloads[0].(map[string]any)["data"].(map[string]any)
}

// doneOperation is an operation response that is already finished, so the
// component's waitForOperation returns without polling/sleeping.
const doneOperation = `{"name":"op-1","status":"DONE"}`
