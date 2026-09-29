package factories

import (
	"context"
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
