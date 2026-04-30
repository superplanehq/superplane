package organizations

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__UpdateIntegrationSecret(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	registerDevelopmentSetupFlowIntegration(t, r, impl.NewDummyIntegrationSetupProvider(impl.DummyIntegrationSetupProviderOptions{
		CapabilityGroups: []core.CapabilityGroup{{Capabilities: []core.Capability{{Name: "feat"}}}},
		FirstStep: func(_ core.SetupStepContext) core.SetupStep {
			return core.SetupStep{Type: core.SetupStepTypeInputs, Name: "step_one"}
		},
		OnStepSubmit: func(ctx core.SetupStepContext) (*core.SetupStep, error) {
			switch ctx.Step {
			case "step_one":
				next := core.SetupStep{Type: core.SetupStepTypeInputs, Name: "step_two"}
				return &next, nil
			case "step_two":
				return nil, nil
			default:
				return nil, nil
			}
		},
	}))

	t.Run("invalid organization ID -> invalid argument", func(t *testing.T) {
		_, err := UpdateIntegrationSecret(ctx, r.Registry, "not-a-uuid", uuid.NewString(), "token", "x")
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("secret not found -> not found", func(t *testing.T) {
		idStr := createSetupFlowIntegration(ctx, t, r, support.RandomName("installation"))
		_, err := UpdateIntegrationSecret(ctx, r.Registry, r.Organization.ID.String(), idStr, "missing_secret", "v")
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("existing secret -> success", func(t *testing.T) {
		idStr := createSetupFlowIntegration(ctx, t, r, support.RandomName("installation"))
		integrationID := uuid.MustParse(idStr)
		seedIntegrationSecret(t, r, integrationID, "api_token", "secret-value")

		resp, err := UpdateIntegrationSecret(ctx, r.Registry, r.Organization.ID.String(), idStr, "api_token", "new-secret")
		require.NoError(t, err)
		require.NotNil(t, resp.Integration)
	})
}
