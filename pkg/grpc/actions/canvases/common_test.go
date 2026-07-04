package canvases

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
)

func setupLiveCanvasStaging(t *testing.T) (*support.ResourceRegistry, context.Context, string, string) {
	t.Helper()

	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
	liveVersion, err := models.FindLiveCanvasVersion(canvas.ID)
	require.NoError(t, err)

	return r, ctx, canvas.ID.String(), liveVersion.ID.String()
}

func structFromAnyMap(t *testing.T, value map[string]any) *structpb.Struct {
	t.Helper()

	result, err := structpb.NewStruct(value)
	require.NoError(t, err)

	return result
}

func TestMapCanvasNameUniqueConstraintError(t *testing.T) {
	t.Run("maps workflow name unique violation to already exists", func(t *testing.T) {
		err := mapCanvasNameUniqueConstraintError(&pgconn.PgError{
			Code:           "23505",
			ConstraintName: "workflows_organization_id_name_key",
		})

		assert.Equal(t, codes.AlreadyExists, grpcerrors.Code(err))
		assert.Equal(t, canvasNameAlreadyExistsMessage, func() string {
			_, msg, ok := grpcerrors.HandlerStatus(err)
			if ok {
				return msg
			}
			return err.Error()
		}())
	})

	t.Run("maps model duplicate name error to already exists", func(t *testing.T) {
		err := mapCanvasNameUniqueConstraintError(models.ErrCanvasNameAlreadyExists)

		assert.Equal(t, codes.AlreadyExists, grpcerrors.Code(err))
		assert.Equal(t, canvasNameAlreadyExistsMessage, func() string {
			_, msg, ok := grpcerrors.HandlerStatus(err)
			if ok {
				return msg
			}
			return err.Error()
		}())
	})

	t.Run("preserves unrelated errors", func(t *testing.T) {
		original := errors.New("other error")

		err := mapCanvasNameUniqueConstraintError(original)

		assert.ErrorIs(t, err, original)
	})
}
