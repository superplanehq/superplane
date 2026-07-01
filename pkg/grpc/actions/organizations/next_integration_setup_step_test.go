package organizations

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"google.golang.org/grpc/codes"
)

func Test__NextIntegrationSetupStep(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	r.Registry.AppEnv = "development"
	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
	r.Registry.SetupProviders["dummy"] = impl.NewDummyIntegrationSetupProvider(impl.DummyIntegrationSetupProviderOptions{
		CapabilityGroups: []core.CapabilityGroup{{Capabilities: []core.Capability{{Name: "feat"}}}},
		FirstStep: func(_ core.SetupStepContext) core.SetupStep {
			return core.SetupStep{Type: core.SetupStepTypeInputs, Name: "step_one"}
		},
		OnStepSubmit: func(ctx core.SetupStepContext) (*core.SetupStep, error) {
			switch ctx.Step.Name {
			case "step_one":
				return &core.SetupStep{Type: core.SetupStepTypeInputs, Name: "step_two"}, nil

			case "step_two":
				return nil, nil

			default:
				return nil, nil
			}
		},
	})

	baseURL := "http://localhost"

	t.Run("invalid organization ID -> invalid argument", func(t *testing.T) {
		_, err := NextIntegrationSetupStep(ctx, r.Registry, baseURL, baseURL, "not-a-uuid", uuid.NewString(), nil, nil)
		require.Error(t, err)
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("integration not found -> not found", func(t *testing.T) {
		_, err := NextIntegrationSetupStep(ctx, r.Registry, baseURL, baseURL, r.Organization.ID.String(), uuid.NewString(), nil, nil)
		require.Error(t, err)
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("submit advances step and completes flow", func(t *testing.T) {
		resp, err := CreateIntegration(
			ctx,
			r.Registry,
			nil,
			"http://localhost",
			"http://localhost",
			r.Organization.ID.String(),
			"dummy",
			support.RandomName("installation"),
			nil,
		)

		require.NoError(t, err)
		require.NotNil(t, resp.Integration)

		afterOne, err := NextIntegrationSetupStep(ctx, r.Registry, baseURL, baseURL, r.Organization.ID.String(), resp.Integration.Metadata.Id, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, afterOne.Integration)
		require.NotNil(t, afterOne.Integration.Status.SetupState)
		require.NotNil(t, afterOne.Integration.Status.SetupState.CurrentStep)
		assert.Equal(t, "step_two", afterOne.Integration.Status.SetupState.CurrentStep.Name)

		afterTwo, err := NextIntegrationSetupStep(ctx, r.Registry, baseURL, baseURL, r.Organization.ID.String(), resp.Integration.Metadata.Id, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, afterTwo.Integration)

		stored, err := models.FindIntegration(r.Organization.ID, uuid.MustParse(resp.Integration.Metadata.Id))
		require.NoError(t, err)
		assert.Nil(t, stored.SetupState)
	})

	t.Run("capability selection with invalid capability -> invalid argument", func(t *testing.T) {
		r4 := support.Setup(t)
		ctx4 := authentication.SetUserIdInMetadata(context.Background(), r4.User.String())
		onStepSubmitCalled := false

		r4.Registry.AppEnv = "development"
		r4.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
		r4.Registry.SetupProviders["dummy"] = impl.NewDummyIntegrationSetupProvider(impl.DummyIntegrationSetupProviderOptions{
			CapabilityGroups: []core.CapabilityGroup{{Capabilities: []core.Capability{{Name: "feat"}}}},
			FirstStep: func(ctx core.SetupStepContext) core.SetupStep {
				return core.SetupStep{Type: core.SetupStepTypeCapabilitySelection, Name: "select_capabilities", Capabilities: []string{"feat"}}
			},
			OnStepSubmit: func(ctx core.SetupStepContext) (*core.SetupStep, error) {
				onStepSubmitCalled = true
				return nil, nil
			},
		})

		resp, err := CreateIntegration(
			ctx4,
			r4.Registry,
			nil,
			"http://localhost",
			"http://localhost",
			r4.Organization.ID.String(),
			"dummy",
			support.RandomName("installation"),
			nil,
		)

		require.NoError(t, err)
		require.NotNil(t, resp.Integration)
		require.NotNil(t, resp.Integration.Status.SetupState)
		require.NotNil(t, resp.Integration.Status.SetupState.CurrentStep)
		assert.Equal(t, "select_capabilities", resp.Integration.Status.SetupState.CurrentStep.Name)
		assert.Equal(t, pb.Integration_SetupStepDefinition_CAPABILITY_SELECTION, resp.Integration.Status.SetupState.CurrentStep.Type)

		_, err = NextIntegrationSetupStep(ctx4, r4.Registry, baseURL, baseURL, r4.Organization.ID.String(), resp.Integration.Metadata.Id, nil, []string{"invalid-capability"})
		require.Error(t, err)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
		assert.Equal(t, "invalid capability: invalid-capability", msg)
		assert.False(t, onStepSubmitCalled)
	})

	t.Run("done step clears setup and marks ready", func(t *testing.T) {
		r3 := support.Setup(t)
		ctx3 := authentication.SetUserIdInMetadata(context.Background(), r3.User.String())

		r3.Registry.AppEnv = "development"
		r3.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
		r3.Registry.SetupProviders["dummy"] = impl.NewDummyIntegrationSetupProvider(impl.DummyIntegrationSetupProviderOptions{
			FirstStep: func(ctx core.SetupStepContext) core.SetupStep {
				return core.SetupStep{Type: core.SetupStepTypeDone, Name: "finish"}
			},
		})

		resp, err := CreateIntegration(
			ctx3,
			r3.Registry,
			nil,
			"http://localhost",
			"http://localhost",
			r3.Organization.ID.String(),
			"dummy",
			support.RandomName("installation"),
			nil,
		)

		require.NoError(t, err)
		require.NotNil(t, resp.Integration)

		nextResponse, err := NextIntegrationSetupStep(ctx3, r3.Registry, baseURL, baseURL, r3.Organization.ID.String(), resp.Integration.Metadata.Id, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, nextResponse.Integration)

		stored, err := models.FindIntegration(r3.Organization.ID, uuid.MustParse(resp.Integration.Metadata.Id))
		require.NoError(t, err)
		assert.Nil(t, stored.SetupState)
		assert.Equal(t, models.IntegrationStateReady, stored.State)
	})
}
