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
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
			switch ctx.Step {
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
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid organization ID")
	})

	t.Run("invalid integration ID -> invalid argument", func(t *testing.T) {
		_, err := UpdateIntegrationCapabilities(ctx, r.Registry, r.Organization.ID.String(), "bad-id", []*pb.Integration_CapabilityState{
			{Name: "feat", State: pb.Integration_CapabilityState_STATE_ENABLED},
		})
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid integration ID")
	})

	t.Run("integration not found -> record not found", func(t *testing.T) {
		_, err := UpdateIntegrationCapabilities(ctx, r.Registry, r.Organization.ID.String(), uuid.NewString(), []*pb.Integration_CapabilityState{
			{Name: "feat", State: pb.Integration_CapabilityState_STATE_ENABLED},
		})
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Contains(t, s.Message(), "integration not found")
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

		stored, err := models.FindIntegration(r.Organization.ID, uuid.MustParse(resp.Integration.Metadata.Id))
		require.NoError(t, err)
		require.Len(t, stored.Capabilities, 1)
		current := stored.Capabilities[0].State

		pbState := CapabilityStateToProto(current)
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

		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "no changes")
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

		stored, err := models.FindIntegration(r.Organization.ID, uuid.MustParse(resp.Integration.Metadata.Id))
		require.NoError(t, err)
		require.Len(t, stored.Capabilities, 1)
		assert.Equal(t, core.IntegrationCapabilityStateEnabled, stored.Capabilities[0].State)
		assert.Equal(t, "feat", stored.Capabilities[0].Name)
	})
}
