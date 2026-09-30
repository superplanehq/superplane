package factories

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__UpdateFactory(t *testing.T) {
	r := support.Setup(t)

	t.Run("empty name -> error", func(t *testing.T) {
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "desc", "")
		require.NoError(t, err)

		emptyName := "   "
		_, err = UpdateFactory(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryRequest{
			Id:   factory.ID.String(),
			Name: &emptyName,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("updates factory metadata", func(t *testing.T) {
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "old desc", "")
		require.NoError(t, err)

		newName := support.RandomName("updated-factory")
		newDescription := "Factory description updated"

		response, err := UpdateFactory(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryRequest{
			Id:          factory.ID.String(),
			Name:        &newName,
			Description: &newDescription,
		})
		require.NoError(t, err)
		require.NotNil(t, response.Factory)
		assert.Equal(t, factory.ID.String(), response.Factory.Id)
		assert.Equal(t, newName, response.Factory.Name)
		assert.Equal(t, newDescription, response.Factory.Description)

		updated, err := models.FindFactory(database.DB(t.Context()), r.Organization.ID, factory.ID)
		require.NoError(t, err)
		assert.Equal(t, newName, updated.Name)
		assert.Equal(t, newDescription, updated.Description)
	})

	t.Run("duplicate name -> error", func(t *testing.T) {
		existing, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		target, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		_, err = UpdateFactory(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryRequest{
			Id:   target.ID.String(),
			Name: &existing.Name,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.AlreadyExists, code)
	})

	t.Run("not found -> error", func(t *testing.T) {
		name := support.RandomName("missing")
		_, err := UpdateFactory(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryRequest{
			Id:   "00000000-0000-0000-0000-000000000001",
			Name: &name,
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
	})

	t.Run("sets hosted spend limit", func(t *testing.T) {
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		budget := int64(2500)
		response, err := UpdateFactory(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryRequest{
			Id:                     factory.ID.String(),
			HostedSpendBudgetCents: &budget,
		})
		require.NoError(t, err)
		require.NotNil(t, response.Factory)
		require.NotNil(t, response.Factory.HostedSpendBudgetCents)
		assert.Equal(t, int64(2500), *response.Factory.HostedSpendBudgetCents)

		cleared, err := UpdateFactory(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryRequest{
			Id:                     factory.ID.String(),
			ClearHostedSpendBudget: true,
		})
		require.NoError(t, err)
		assert.Nil(t, cleared.Factory.HostedSpendBudgetCents)
	})

	t.Run("defaults Planning on with Confidence and without Clarity", func(t *testing.T) {
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		response, err := UpdateFactory(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryRequest{
			Id: factory.ID.String(),
		})
		require.NoError(t, err)
		require.NotNil(t, response.Factory.Planning)
		assert.True(t, response.Factory.Planning.Enabled)
		assert.False(t, response.Factory.Planning.Clarity)
		assert.True(t, response.Factory.Planning.Confidence)
	})

	t.Run("updates Planning and keeps score flags when Planning is off", func(t *testing.T) {
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		response, err := UpdateFactory(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryRequest{
			Id: factory.ID.String(),
			Planning: &pb.FactoryPlanning{
				Enabled:        false,
				Clarity:        true,
				Confidence:     false,
				SetupCompleted: true,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, response.Factory.Planning)
		assert.False(t, response.Factory.Planning.Enabled)
		assert.True(t, response.Factory.Planning.Clarity)
		assert.False(t, response.Factory.Planning.Confidence)
		assert.True(t, response.Factory.Planning.SetupCompleted)

		reloaded, err := models.FindFactory(database.DB(t.Context()), r.Organization.ID, factory.ID)
		require.NoError(t, err)
		assert.Equal(t, models.FactoryPlanning{
			Enabled:        false,
			Clarity:        true,
			Confidence:     false,
			SetupCompleted: true,
		}, reloaded.Planning())
	})

	t.Run("public badge defaults cost off and keeps the token", func(t *testing.T) {
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		assert.False(t, factory.PublicBadgeEnabled)
		assert.False(t, factory.PublicBadgeShowCost)
		assert.Nil(t, factory.PublicBadgeToken)

		enabled := true
		response, err := UpdateFactory(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryRequest{
			Id:                 factory.ID.String(),
			PublicBadgeEnabled: &enabled,
		})
		require.NoError(t, err)
		require.NotEmpty(t, response.Factory.PublicBadgeToken)
		assert.True(t, response.Factory.PublicBadgeEnabled)
		assert.False(t, response.Factory.PublicBadgeShowCost)

		reloaded, err := models.FindFactory(database.DB(t.Context()), r.Organization.ID, factory.ID)
		require.NoError(t, err)
		require.NotNil(t, reloaded.PublicBadgeToken)
		token := *reloaded.PublicBadgeToken
		assert.GreaterOrEqual(t, len(token), 22)

		disabled := false
		response, err = UpdateFactory(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryRequest{
			Id:                 factory.ID.String(),
			PublicBadgeEnabled: &disabled,
		})
		require.NoError(t, err)
		assert.False(t, response.Factory.PublicBadgeEnabled)
		assert.Equal(t, token, response.Factory.PublicBadgeToken)
		assert.False(t, response.Factory.PublicBadgeShowCost)

		showCost := true
		response, err = UpdateFactory(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryRequest{
			Id:                  factory.ID.String(),
			PublicBadgeShowCost: &showCost,
		})
		require.NoError(t, err)
		assert.False(t, response.Factory.PublicBadgeEnabled)
		assert.True(t, response.Factory.PublicBadgeShowCost)
		assert.Equal(t, token, response.Factory.PublicBadgeToken)

		reloaded, err = models.FindFactory(database.DB(t.Context()), r.Organization.ID, factory.ID)
		require.NoError(t, err)
		assert.False(t, reloaded.PublicBadgeEnabled)
		assert.True(t, reloaded.PublicBadgeShowCost)
		require.NotNil(t, reloaded.PublicBadgeToken)
		assert.Equal(t, token, *reloaded.PublicBadgeToken)
	})

	t.Run("concurrent first enables keep one token", func(t *testing.T) {
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		const attempts = 8
		enabled := true
		tokens := make([]string, attempts)
		errs := make([]error, attempts)
		var wg sync.WaitGroup
		wg.Add(attempts)
		for i := range attempts {
			go func() {
				defer wg.Done()
				response, err := UpdateFactory(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryRequest{
					Id:                 factory.ID.String(),
					PublicBadgeEnabled: &enabled,
				})
				errs[i] = err
				if err == nil && response.Factory != nil {
					tokens[i] = response.Factory.PublicBadgeToken
				}
			}()
		}
		wg.Wait()

		for _, err := range errs {
			require.NoError(t, err)
		}
		require.NotEmpty(t, tokens[0])
		for _, token := range tokens[1:] {
			assert.Equal(t, tokens[0], token)
		}
		reloaded, err := models.FindFactory(database.DB(t.Context()), r.Organization.ID, factory.ID)
		require.NoError(t, err)
		require.NotNil(t, reloaded.PublicBadgeToken)
		assert.Equal(t, tokens[0], *reloaded.PublicBadgeToken)
	})

	t.Run("stores and clears the auto-start line", func(t *testing.T) {
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		line, err := factory.CreateLine(database.DB(t.Context()), "implement", nil)
		require.NoError(t, err)

		response, err := UpdateFactory(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryRequest{
			Id: factory.ID.String(),
			Planning: &pb.FactoryPlanning{
				Enabled:         true,
				Confidence:      true,
				SetupCompleted:  true,
				AutoStartLineId: line.ID.String(),
			},
		})
		require.NoError(t, err)
		assert.Equal(t, line.ID.String(), response.Factory.Planning.GetAutoStartLineId())

		cleared, err := UpdateFactory(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryRequest{
			Id: factory.ID.String(),
			Planning: &pb.FactoryPlanning{
				Enabled:        true,
				Confidence:     true,
				SetupCompleted: true,
			},
		})
		require.NoError(t, err)
		assert.Empty(t, cleared.Factory.Planning.GetAutoStartLineId())

		reloaded, err := models.FindFactory(database.DB(t.Context()), r.Organization.ID, factory.ID)
		require.NoError(t, err)
		assert.Nil(t, reloaded.PlanningAutoStartLineID)
	})

	t.Run("rejects an auto-start line outside the workspace", func(t *testing.T) {
		factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		other, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("other"), "", "")
		require.NoError(t, err)
		foreignLine, err := other.CreateLine(database.DB(t.Context()), "implement", nil)
		require.NoError(t, err)

		_, err = UpdateFactory(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryRequest{
			Id: factory.ID.String(),
			Planning: &pb.FactoryPlanning{
				Enabled:         true,
				Confidence:      true,
				AutoStartLineId: "not-a-uuid",
			},
		})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))

		_, err = UpdateFactory(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryRequest{
			Id: factory.ID.String(),
			Planning: &pb.FactoryPlanning{
				Enabled:         true,
				Confidence:      true,
				AutoStartLineId: foreignLine.ID.String(),
			},
		})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	})
}
