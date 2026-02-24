package organizations

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func Test__InvokeIntegrationAction(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	baseURL := "http://localhost"

	t.Run("invoke action with valid parameters -> success", func(t *testing.T) {
		called := false
		r.Registry.Integrations["dummy"] = support.NewDummyIntegration(support.DummyIntegrationOptions{
			Actions: []core.Action{
				{
					Name: "completeSetup",
					Parameters: []configuration.Field{
						{Name: "code", Type: configuration.FieldTypeString, Required: true},
					},
				},
			},
			OnSync: func(ctx core.SyncContext) error {
				ctx.Integration.Ready()
				return nil
			},
			HandleAction: func(ctx core.IntegrationActionContext) error {
				called = true
				assert.Equal(t, "completeSetup", ctx.Name)
				assert.Equal(t, "abc123", ctx.Parameters.(map[string]any)["code"])
				ctx.Integration.Ready()
				return nil
			},
		})

		name := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{})
		require.NoError(t, err)

		createResponse, err := CreateIntegration(
			ctx,
			r.Registry,
			nil,
			baseURL,
			baseURL,
			r.Organization.ID.String(),
			"dummy",
			name,
			appConfig,
		)
		require.NoError(t, err)
		require.NotNil(t, createResponse)

		_, err = InvokeIntegrationAction(
			ctx,
			r.Registry,
			baseURL,
			r.Organization.ID.String(),
			createResponse.Integration.Metadata.Id,
			"completeSetup",
			map[string]any{"code": "abc123"},
		)
		require.NoError(t, err)
		assert.True(t, called)

		integration, err := models.FindIntegrationByName(r.Organization.ID, name)
		require.NoError(t, err)
		assert.Equal(t, models.IntegrationStateReady, integration.State)
	})

	t.Run("action execution failure -> invalid argument", func(t *testing.T) {
		r.Registry.Integrations["dummy"] = support.NewDummyIntegration(support.DummyIntegrationOptions{
			Actions: []core.Action{{Name: "completeSetup"}},
			OnSync: func(ctx core.SyncContext) error {
				ctx.Integration.Ready()
				return nil
			},
			HandleAction: func(ctx core.IntegrationActionContext) error {
				return errors.New("failed action")
			},
		})

		name := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{})
		require.NoError(t, err)

		createResponse, err := CreateIntegration(
			ctx,
			r.Registry,
			nil,
			baseURL,
			baseURL,
			r.Organization.ID.String(),
			"dummy",
			name,
			appConfig,
		)
		require.NoError(t, err)

		_, err = InvokeIntegrationAction(
			ctx,
			r.Registry,
			baseURL,
			r.Organization.ID.String(),
			createResponse.Integration.Metadata.Id,
			"completeSetup",
			map[string]any{},
		)
		require.Error(t, err)

		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "action execution failed")
	})

	t.Run("missing required action parameter -> invalid argument", func(t *testing.T) {
		r.Registry.Integrations["dummy"] = support.NewDummyIntegration(support.DummyIntegrationOptions{
			Actions: []core.Action{
				{
					Name: "completeSetup",
					Parameters: []configuration.Field{
						{Name: "code", Type: configuration.FieldTypeString, Required: true},
					},
				},
			},
			OnSync: func(ctx core.SyncContext) error {
				ctx.Integration.Ready()
				return nil
			},
		})

		name := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{})
		require.NoError(t, err)

		createResponse, err := CreateIntegration(
			ctx,
			r.Registry,
			nil,
			baseURL,
			baseURL,
			r.Organization.ID.String(),
			"dummy",
			name,
			appConfig,
		)
		require.NoError(t, err)

		_, err = InvokeIntegrationAction(
			ctx,
			r.Registry,
			baseURL,
			r.Organization.ID.String(),
			createResponse.Integration.Metadata.Id,
			"completeSetup",
			map[string]any{},
		)
		require.Error(t, err)

		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "action parameter validation failed")
	})

	t.Run("non-existent action -> not found", func(t *testing.T) {
		r.Registry.Integrations["dummy"] = support.NewDummyIntegration(support.DummyIntegrationOptions{
			Actions: []core.Action{},
			OnSync: func(ctx core.SyncContext) error {
				ctx.Integration.Ready()
				return nil
			},
		})

		name := support.RandomName("integration")
		appConfig, err := structpb.NewStruct(map[string]any{})
		require.NoError(t, err)

		createResponse, err := CreateIntegration(
			ctx,
			r.Registry,
			nil,
			baseURL,
			baseURL,
			r.Organization.ID.String(),
			"dummy",
			name,
			appConfig,
		)
		require.NoError(t, err)

		_, err = InvokeIntegrationAction(
			ctx,
			r.Registry,
			baseURL,
			r.Organization.ID.String(),
			createResponse.Integration.Metadata.Id,
			"missingAction",
			map[string]any{},
		)
		require.Error(t, err)

		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Contains(t, s.Message(), "action 'missingAction' not found")
	})

	t.Run("invalid integration ID -> invalid argument", func(t *testing.T) {
		_, err := InvokeIntegrationAction(
			ctx,
			r.Registry,
			baseURL,
			r.Organization.ID.String(),
			"invalid-uuid",
			"completeSetup",
			map[string]any{},
		)
		require.Error(t, err)

		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "invalid integration ID")
	})

	t.Run("integration not found -> not found", func(t *testing.T) {
		_, err := InvokeIntegrationAction(
			ctx,
			r.Registry,
			baseURL,
			r.Organization.ID.String(),
			uuid.NewString(),
			"completeSetup",
			map[string]any{},
		)
		require.Error(t, err)

		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Contains(t, s.Message(), "integration not found")
	})
}
