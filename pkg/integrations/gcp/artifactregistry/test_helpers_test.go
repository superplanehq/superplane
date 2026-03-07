package artifactregistry

import (
	"context"
	"errors"
	"testing"

	"github.com/superplanehq/superplane/pkg/core"
)

type mockClient struct {
	projectID string
	getURL    func(ctx context.Context, fullURL string) ([]byte, error)
	postURL   func(ctx context.Context, fullURL string, body any) ([]byte, error)
}

func (m *mockClient) GetURL(ctx context.Context, fullURL string) ([]byte, error) {
	if m.getURL != nil {
		return m.getURL(ctx, fullURL)
	}
	return nil, errors.New("not implemented")
}

func (m *mockClient) PostURL(ctx context.Context, fullURL string, body any) ([]byte, error) {
	if m.postURL != nil {
		return m.postURL(ctx, fullURL, body)
	}
	return nil, errors.New("not implemented")
}

func (m *mockClient) ProjectID() string {
	return m.projectID
}

func setTestClientFactory(
	t *testing.T,
	fn func(httpCtx core.HTTPContext, integration core.IntegrationContext) (Client, error),
) {
	t.Helper()

	clientFactoryMu.RLock()
	previous := clientFactory
	clientFactoryMu.RUnlock()

	SetClientFactory(fn)
	t.Cleanup(func() {
		SetClientFactory(previous)
	})
}
