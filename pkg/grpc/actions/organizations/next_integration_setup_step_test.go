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
	"google.golang.org/protobuf/types/known/structpb"
)

func Test__NextIntegrationSetupStep(t *testing.T) {
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

	t.Run("invalid organization ID -> invalid argument", func(t *testing.T) {
		_, err := NextIntegrationSetupStep(ctx, r.Registry, baseURL, baseURL, "not-a-uuid", uuid.NewString(), nil)
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("integration not found -> not found", func(t *testing.T) {
		_, err := NextIntegrationSetupStep(ctx, r.Registry, baseURL, baseURL, r.Organization.ID.String(), uuid.NewString(), nil)
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("submit advances step and completes flow", func(t *testing.T) {
		idStr := createSetupFlowIntegration(t, ctx, r, support.RandomName("installation"))

		afterOne, err := NextIntegrationSetupStep(ctx, r.Registry, baseURL, baseURL, r.Organization.ID.String(), idStr, nil)
		require.NoError(t, err)
		require.NotNil(t, afterOne.Integration)
		require.NotNil(t, afterOne.Integration.Status.SetupState)
		require.NotNil(t, afterOne.Integration.Status.SetupState.CurrentStep)
		assert.Equal(t, "step_two", afterOne.Integration.Status.SetupState.CurrentStep.Name)

		afterTwo, err := NextIntegrationSetupStep(ctx, r.Registry, baseURL, baseURL, r.Organization.ID.String(), idStr, nil)
		require.NoError(t, err)
		require.NotNil(t, afterTwo.Integration)

		stored, err := models.FindIntegration(r.Organization.ID, uuid.MustParse(idStr))
		require.NoError(t, err)
		assert.Nil(t, stored.SetupState)
	})

	t.Run("current step not set -> invalid argument", func(t *testing.T) {
		r2 := support.Setup(t)
		ctx2 := authentication.SetUserIdInMetadata(context.Background(), r2.User.String())
		registerDevelopmentSetupFlowIntegration(t, r2, impl.NewDummyIntegrationSetupProvider(impl.DummyIntegrationSetupProviderOptions{
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
		appConfig, err := structpb.NewStruct(map[string]any{})
		require.NoError(t, err)
		r2.Registry.AppEnv = ""
		resp, err := CreateIntegration(ctx2, r2.Registry, nil, baseURL, baseURL, r2.Organization.ID.String(), "github", support.RandomName("legacy"), appConfig)
		require.NoError(t, err)
		legacyID := resp.Integration.Metadata.Id

		_, err = NextIntegrationSetupStep(ctx2, r2.Registry, baseURL, baseURL, r2.Organization.ID.String(), legacyID, nil)
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "current step is not set")
	})

	t.Run("done step clears setup and marks ready", func(t *testing.T) {
		r3 := support.Setup(t)
		ctx3 := authentication.SetUserIdInMetadata(context.Background(), r3.User.String())
		registerDevelopmentSetupFlowIntegration(t, r3, impl.NewDummyIntegrationSetupProvider(impl.DummyIntegrationSetupProviderOptions{
			FirstStep: func(_ core.SetupStepContext) core.SetupStep {
				return core.SetupStep{Type: core.SetupStepTypeDone, Name: "finish"}
			},
		}))
		idStr := createSetupFlowIntegration(t, ctx3, r3, support.RandomName("installation"))

		resp, err := NextIntegrationSetupStep(ctx3, r3.Registry, baseURL, baseURL, r3.Organization.ID.String(), idStr, nil)
		require.NoError(t, err)
		require.NotNil(t, resp.Integration)

		stored, err := models.FindIntegration(r3.Organization.ID, uuid.MustParse(idStr))
		require.NoError(t, err)
		assert.Nil(t, stored.SetupState)
		assert.Equal(t, models.IntegrationStateReady, stored.State)
	})
}
