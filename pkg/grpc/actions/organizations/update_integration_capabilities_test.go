package organizations

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"
)

func Test__UpdateIntegrationCapabilities(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	//
	// Register dummy integration and setup provider
	//
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

	t.Run("invalid organization ID -> invalid argument", func(t *testing.T) {
		_, err := UpdateIntegrationCapabilities(ctx, r.Registry, "not-a-uuid", uuid.NewString(), []*pb.Integration_CapabilityState{
			{Name: "feat", State: pb.Integration_CapabilityState_STATE_ENABLED},
		})
		require.Error(t, err)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
		assert.Contains(t, msg, "invalid organization ID")
	})

	t.Run("invalid integration ID -> invalid argument", func(t *testing.T) {
		_, err := UpdateIntegrationCapabilities(ctx, r.Registry, r.Organization.ID.String(), "bad-id", []*pb.Integration_CapabilityState{
			{Name: "feat", State: pb.Integration_CapabilityState_STATE_ENABLED},
		})
		require.Error(t, err)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
		assert.Contains(t, msg, "invalid integration ID")
	})

	t.Run("integration not found -> record not found", func(t *testing.T) {
		_, err := UpdateIntegrationCapabilities(ctx, r.Registry, r.Organization.ID.String(), uuid.NewString(), []*pb.Integration_CapabilityState{
			{Name: "feat", State: pb.Integration_CapabilityState_STATE_ENABLED},
		})
		require.Error(t, err)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
		assert.Contains(t, msg, "integration not found")
	})

	t.Run("no capability changes -> invalid argument", func(t *testing.T) {
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

		//
		// Capability must have been exposed by the integration setup provider.
		// Here, we mock that, by setting the capability state to unavailable directly.
		//
		stored, err := models.FindIntegration(r.Organization.ID, uuid.MustParse(resp.Integration.Metadata.Id))
		require.NoError(t, err)
		newStates := []models.CapabilityState{{Name: "feat", State: core.IntegrationCapabilityStateUnavailable}}
		stored.Capabilities = datatypes.NewJSONSlice(newStates)
		require.NoError(t, database.Conn().Save(stored).Error)

		pbState := CapabilityStateToProto(core.IntegrationCapabilityStateUnavailable)
		_, err = UpdateIntegrationCapabilities(
			ctx,
			r.Registry,
			r.Organization.ID.String(),
			resp.Integration.Metadata.Id,
			[]*pb.Integration_CapabilityState{
				{Name: "feat", State: pbState},
			},
		)
		require.Error(t, err)

		code, msg, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
		assert.Contains(t, msg, "no changes")
	})

	t.Run("enable unavailable capability -> success", func(t *testing.T) {
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

		//
		// Capability must have been exposed by the integration setup provider.
		// Here, we mock that, by setting the capability state to unavailable directly.
		//
		stored, err := models.FindIntegration(r.Organization.ID, uuid.MustParse(resp.Integration.Metadata.Id))
		require.NoError(t, err)
		newStates := []models.CapabilityState{{Name: "feat", State: core.IntegrationCapabilityStateUnavailable}}
		stored.Capabilities = datatypes.NewJSONSlice(newStates)
		require.NoError(t, database.Conn().Save(stored).Error)

		updateResponse, err := UpdateIntegrationCapabilities(
			ctx,
			r.Registry,
			r.Organization.ID.String(),
			resp.Integration.Metadata.Id,
			[]*pb.Integration_CapabilityState{
				{Name: "feat", State: pb.Integration_CapabilityState_STATE_ENABLED},
			},
		)
		require.NoError(t, err)
		require.NotNil(t, updateResponse.Integration)

		stored, err = models.FindIntegration(r.Organization.ID, uuid.MustParse(resp.Integration.Metadata.Id))
		require.NoError(t, err)
		require.Len(t, stored.Capabilities, 1)
		assert.Equal(t, core.IntegrationCapabilityStateEnabled, stored.Capabilities[0].State)
		assert.Equal(t, "feat", stored.Capabilities[0].Name)
	})
}
