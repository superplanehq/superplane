package organizations

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__PreviousIntegrationSetupStep(t *testing.T) {
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
	baseURL := "http://localhost"

	t.Run("no previous steps -> invalid argument", func(t *testing.T) {
		idStr := createSetupFlowIntegration(ctx, t, r, support.RandomName("installation"))
		_, err := PreviousIntegrationSetupStep(ctx, r.Registry, r.Organization.ID.String(), idStr)
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "no previous steps")
	})

	t.Run("after advancing once -> previous restores first step", func(t *testing.T) {
		idStr := createSetupFlowIntegration(ctx, t, r, support.RandomName("installation"))

		_, err := NextIntegrationSetupStep(ctx, r.Registry, baseURL, baseURL, r.Organization.ID.String(), idStr, nil)
		require.NoError(t, err)

		back, err := PreviousIntegrationSetupStep(ctx, r.Registry, r.Organization.ID.String(), idStr)
		require.NoError(t, err)
		require.NotNil(t, back.Integration)
		require.NotNil(t, back.Integration.Status.SetupState)
		require.NotNil(t, back.Integration.Status.SetupState.CurrentStep)
		assert.Equal(t, "step_one", back.Integration.Status.SetupState.CurrentStep.Name)

		stored, err := models.FindIntegration(r.Organization.ID, uuid.MustParse(idStr))
		require.NoError(t, err)
		require.NotNil(t, stored.SetupState)
		assert.Equal(t, "step_one", stored.SetupState.Data().CurrentStep.Name)
		assert.Empty(t, stored.SetupState.Data().PreviousSteps)
	})
}
