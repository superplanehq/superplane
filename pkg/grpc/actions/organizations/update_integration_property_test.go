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
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__UpdateIntegrationProperty(t *testing.T) {
	r := support.Setup(t)

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	//
	// Register dummy integration and setup provider
	//
	r.Registry.AppEnv = "development"
	r.Registry.Integrations["dummy"] = impl.NewDummyIntegration(impl.DummyIntegrationOptions{})
	r.Registry.SetupProviders["dummy"] = impl.NewDummyIntegrationSetupProvider(impl.DummyIntegrationSetupProviderOptions{
		CapabilityGroups: []core.CapabilityGroup{{Capabilities: []core.Capability{{Name: "feat"}}}},
		FirstStep: func(ctx core.SetupStepContext) core.SetupStep {
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
		OnPropertyUpdate: func(ctx core.PropertyUpdateContext) (*core.SetupStep, error) {
			err := ctx.Properties.Delete(ctx.PropertyName)
			if err != nil {
				return nil, err
			}

			return nil, ctx.Properties.Create(core.IntegrationPropertyDefinition{
				Type:     core.IntegrationPropertyTypeString,
				Name:     ctx.PropertyName,
				Label:    "Repo",
				Value:    ctx.Value,
				Editable: true,
			})
		},
	})

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

		integrationID := resp.Integration.Metadata.Id
		_, err = UpdateIntegrationProperty(ctx, r.Registry, r.Organization.ID.String(), integrationID, "missing_prop", "v")
		require.Error(t, err)

		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Contains(t, s.Message(), "property missing_prop not found")
	})

	t.Run("property not editable -> invalid argument", func(t *testing.T) {
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

		integration, err := models.FindIntegration(r.Organization.ID, uuid.MustParse(resp.Integration.Metadata.Id))
		require.NoError(t, err)
		integration.Properties = append(integration.Properties, core.IntegrationPropertyDefinition{
			Type:     core.IntegrationPropertyTypeString,
			Name:     "not_editable",
			Label:    "Not editable",
			Value:    "old",
			Editable: false,
		})

		require.NoError(t, database.Conn().Save(&integration).Error)

		_, err = UpdateIntegrationProperty(ctx, r.Registry, r.Organization.ID.String(), integration.ID.String(), "not_editable", "v")
		require.Error(t, err)

		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "not editable")
	})

	t.Run("existing property -> success", func(t *testing.T) {
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
		// Seed an integration property
		//
		integration, err := models.FindIntegration(r.Organization.ID, uuid.MustParse(resp.Integration.Metadata.Id))
		require.NoError(t, err)
		integration.Properties = append(integration.Properties, core.IntegrationPropertyDefinition{
			Type:     core.IntegrationPropertyTypeString,
			Name:     "repo",
			Label:    "Repo",
			Value:    "old",
			Editable: true,
		})

		require.NoError(t, database.Conn().Save(&integration).Error)

		updateResponse, err := UpdateIntegrationProperty(ctx, r.Registry, r.Organization.ID.String(), integration.ID.String(), "repo", "new-value")
		require.NoError(t, err)
		require.NotNil(t, updateResponse.Integration)
		require.Len(t, updateResponse.Integration.Status.Properties, 1)
		assert.Equal(t, "new-value", updateResponse.Integration.Status.Properties[0].Value)
	})
}
