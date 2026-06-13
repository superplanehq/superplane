package canvases

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func structFromAnyMap(t *testing.T, value map[string]any) *structpb.Struct {
	t.Helper()

	result, err := structpb.NewStruct(value)
	require.NoError(t, err)

	return result
}

func findRegisteredDraftBranch(t *testing.T, canvasID uuid.UUID, branchName string) *models.CanvasVersion {
	t.Helper()

	var version models.CanvasVersion
	err := database.Conn().
		Where("workflow_id = ?", canvasID).
		Where("branch_name = ?", branchName).
		Where("state = ?", models.CanvasVersionStateDraft).
		First(&version).
		Error
	require.NoError(t, err)

	return &version
}

func findRegisteredDraftBranchErr(canvasID uuid.UUID, branchName string) error {
	var version models.CanvasVersion
	return database.Conn().
		Where("workflow_id = ?", canvasID).
		Where("branch_name = ?", branchName).
		Where("state = ?", models.CanvasVersionStateDraft).
		First(&version).
		Error
}

func TestMapCanvasNameUniqueConstraintError(t *testing.T) {
	t.Run("maps workflow name unique violation to already exists", func(t *testing.T) {
		err := mapCanvasNameUniqueConstraintError(&pgconn.PgError{
			Code:           "23505",
			ConstraintName: "workflows_organization_id_name_key",
		})

		assert.Equal(t, codes.AlreadyExists, status.Code(err))
		assert.Equal(t, canvasNameAlreadyExistsMessage, status.Convert(err).Message())
	})

	t.Run("maps model duplicate name error to already exists", func(t *testing.T) {
		err := mapCanvasNameUniqueConstraintError(models.ErrCanvasNameAlreadyExists)

		assert.Equal(t, codes.AlreadyExists, status.Code(err))
		assert.Equal(t, canvasNameAlreadyExistsMessage, status.Convert(err).Message())
	})

	t.Run("preserves unrelated errors", func(t *testing.T) {
		original := errors.New("other error")

		err := mapCanvasNameUniqueConstraintError(original)

		assert.ErrorIs(t, err, original)
	})
}
