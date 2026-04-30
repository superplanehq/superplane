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

func Test__UpdateIntegrationProperty(t *testing.T) {
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
		_, err := UpdateIntegrationProperty(ctx, r.Registry, "not-a-uuid", uuid.NewString(), "repo", "x")
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("invalid integration ID -> invalid argument", func(t *testing.T) {
		_, err := UpdateIntegrationProperty(ctx, r.Registry, r.Organization.ID.String(), "bad-id", "repo", "x")
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("property not found -> not found", func(t *testing.T) {
		idStr := createSetupFlowIntegration(t, ctx, r, support.RandomName("installation"))
		_, err := UpdateIntegrationProperty(ctx, r.Registry, r.Organization.ID.String(), idStr, "missing_prop", "v")
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Contains(t, s.Message(), "missing_prop")
	})

	t.Run("existing property -> success", func(t *testing.T) {
		idStr := createSetupFlowIntegration(t, ctx, r, support.RandomName("installation"))
		integrationID := uuid.MustParse(idStr)
		seedIntegrationProperty(t, integrationID, core.IntegrationPropertyDefinition{
			Type:     core.IntegrationPropertyTypeString,
			Name:     "repo",
			Label:    "Repo",
			Value:    "old",
			Editable: true,
		})

		resp, err := UpdateIntegrationProperty(ctx, r.Registry, r.Organization.ID.String(), idStr, "repo", "new-value")
		require.NoError(t, err)
		require.NotNil(t, resp.Integration)
	})
}
