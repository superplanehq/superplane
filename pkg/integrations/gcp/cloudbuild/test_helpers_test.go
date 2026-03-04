package cloudbuild

import (
	"testing"

	"github.com/superplanehq/superplane/pkg/core"
)

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
