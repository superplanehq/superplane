package factories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__UpdateFactoryLine__ColumnColors(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	newFactoryAndLine := func(t *testing.T) (*models.Factory, *models.FactoryLine) {
		t.Helper()
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		line, err := factory.CreateLine(db, "ship", nil)
		require.NoError(t, err)
		return factory, line
	}

	t.Run("sets a color", func(t *testing.T) {
		factory, line := newFactoryAndLine(t)

		response, err := UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId:    factory.ID.String(),
			LineId:       line.ID.String(),
			ColumnColors: map[string]string{"backlog": "lime"},
		})
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"backlog": "lime"}, response.Line.ColumnColors)

		updated, err := factory.FindLine(db, line.ID)
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"backlog": "lime"}, updated.ColumnColorsValue())
	})

	t.Run("colors-only update does not require name or steps", func(t *testing.T) {
		factory, line := newFactoryAndLine(t)

		_, err := UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId:    factory.ID.String(),
			LineId:       line.ID.String(),
			ColumnColors: map[string]string{"verify": "teal"},
		})
		require.NoError(t, err)
	})

	t.Run("clearing a color replaces the stored map", func(t *testing.T) {
		factory, line := newFactoryAndLine(t)

		_, err := UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId:    factory.ID.String(),
			LineId:       line.ID.String(),
			ColumnColors: map[string]string{"backlog": "lime", "verify": "teal"},
		})
		require.NoError(t, err)

		response, err := UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId:    factory.ID.String(),
			LineId:       line.ID.String(),
			ColumnColors: map[string]string{"verify": "teal"},
		})
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"verify": "teal"}, response.Line.ColumnColors)
	})

	t.Run("empty map clears all colors", func(t *testing.T) {
		factory, line := newFactoryAndLine(t)

		_, err := UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId:    factory.ID.String(),
			LineId:       line.ID.String(),
			ColumnColors: map[string]string{"backlog": "lime"},
		})
		require.NoError(t, err)

		response, err := UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId:    factory.ID.String(),
			LineId:       line.ID.String(),
			ColumnColors: map[string]string{},
		})
		require.NoError(t, err)
		assert.Empty(t, response.Line.ColumnColors)

		updated, err := factory.FindLine(db, line.ID)
		require.NoError(t, err)
		assert.Empty(t, updated.ColumnColorsValue())
	})

	t.Run("unknown color id -> error", func(t *testing.T) {
		factory, line := newFactoryAndLine(t)

		_, err := UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId:    factory.ID.String(),
			LineId:       line.ID.String(),
			ColumnColors: map[string]string{"backlog": "not-a-color"},
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("sets column automations", func(t *testing.T) {
		factory, line := newFactoryAndLine(t)
		app, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "extra", "start-extra")

		response, err := UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId: factory.ID.String(),
			LineId:    line.ID.String(),
			ColumnAutomations: map[string]*pb.FactoryLine_ColumnAutomations{
				"phase-0": {CanvasIds: []string{app.ID.String()}},
			},
		})
		require.NoError(t, err)
		require.Contains(t, response.Line.ColumnAutomations, "phase-0")
		assert.Equal(t, []string{app.ID.String()}, response.Line.ColumnAutomations["phase-0"].CanvasIds)

		updated, err := factory.FindLine(db, line.ID)
		require.NoError(t, err)
		assert.Equal(t, map[string][]uuid.UUID{"phase-0": {app.ID}}, updated.ColumnAutomationsValue())
	})

	t.Run("automations-only update does not require name or steps", func(t *testing.T) {
		factory, line := newFactoryAndLine(t)
		app, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "extra", "start-extra")

		_, err := UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId: factory.ID.String(),
			LineId:    line.ID.String(),
			ColumnAutomations: map[string]*pb.FactoryLine_ColumnAutomations{
				"backlog": {CanvasIds: []string{app.ID.String()}},
			},
		})
		require.NoError(t, err)
	})

	t.Run("empty map clears all column automations", func(t *testing.T) {
		factory, line := newFactoryAndLine(t)
		app, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "extra", "start-extra")

		_, err := UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId: factory.ID.String(),
			LineId:    line.ID.String(),
			ColumnAutomations: map[string]*pb.FactoryLine_ColumnAutomations{
				"backlog": {CanvasIds: []string{app.ID.String()}},
			},
		})
		require.NoError(t, err)

		response, err := UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId:         factory.ID.String(),
			LineId:            line.ID.String(),
			ColumnAutomations: map[string]*pb.FactoryLine_ColumnAutomations{},
		})
		require.NoError(t, err)
		assert.Empty(t, response.Line.ColumnAutomations)

		updated, err := factory.FindLine(db, line.ID)
		require.NoError(t, err)
		assert.Empty(t, updated.ColumnAutomationsValue())
	})

	t.Run("unknown column key -> error", func(t *testing.T) {
		factory, line := newFactoryAndLine(t)
		app, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "extra", "start-extra")

		_, err := UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId: factory.ID.String(),
			LineId:    line.ID.String(),
			ColumnAutomations: map[string]*pb.FactoryLine_ColumnAutomations{
				"inbox": {CanvasIds: []string{app.ID.String()}},
			},
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("canvas not owned by factory -> error", func(t *testing.T) {
		factory, line := newFactoryAndLine(t)
		otherFactory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		app, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, otherFactory.ID, "foreign", "start-foreign")

		_, err = UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId: factory.ID.String(),
			LineId:    line.ID.String(),
			ColumnAutomations: map[string]*pb.FactoryLine_ColumnAutomations{
				"phase-0": {CanvasIds: []string{app.ID.String()}},
			},
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("duplicate canvas id -> error", func(t *testing.T) {
		factory, line := newFactoryAndLine(t)
		app, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "extra", "start-extra")

		_, err := UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId: factory.ID.String(),
			LineId:    line.ID.String(),
			ColumnAutomations: map[string]*pb.FactoryLine_ColumnAutomations{
				"phase-0": {CanvasIds: []string{app.ID.String(), app.ID.String()}},
			},
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("no fields provided -> error", func(t *testing.T) {
		factory, line := newFactoryAndLine(t)

		_, err := UpdateFactoryLine(context.Background(), r.Organization.ID.String(), &pb.UpdateFactoryLineRequest{
			FactoryId: factory.ID.String(),
			LineId:    line.ID.String(),
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})
}
