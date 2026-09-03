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

	previous := newClient
	newClient = fn
	t.Cleanup(func() {
		newClient = previous
	})
}
