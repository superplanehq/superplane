package gcp

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test_GCP_Sync_WIF(t *testing.T) {
	g := &GCP{}
	const validProvider = "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/my-pool/providers/superplane"

	t.Run("WIF without service account email fails instead of storing a weak token", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"workloadIdentityProvider":  validProvider,
				"workloadIdentityProjectId": "my-project",
				// workloadIdentityServiceAccountEmail intentionally omitted (legacy config).
			},
		}

		// No HTTP context is supplied: the guard must reject before any token
		// exchange / impersonation network call is attempted.
		err := g.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
		})
		require.ErrorContains(t, err, "Service account email is required")
	})

	t.Run("WIF still validates provider and project before the email check", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"workloadIdentityProvider":  validProvider,
				"workloadIdentityProjectId": "",
			},
		}

		err := g.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
		})
		require.ErrorContains(t, err, "Project ID is required")
	})
}
