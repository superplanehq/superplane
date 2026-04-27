package canvases

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMapCanvasNameUniqueConstraintError(t *testing.T) {
	t.Run("maps workflow name unique violation to already exists", func(t *testing.T) {
		err := mapCanvasNameUniqueConstraintError(&pgconn.PgError{
			Code:           "23505",
			ConstraintName: "workflows_organization_id_name_key",
		})

		assert.Equal(t, codes.AlreadyExists, status.Code(err))
		assert.Equal(t, canvasNameAlreadyExistsMessage, status.Convert(err).Message())
	})

	t.Run("preserves unrelated errors", func(t *testing.T) {
		original := errors.New("other error")

		err := mapCanvasNameUniqueConstraintError(original)

		assert.ErrorIs(t, err, original)
	})
}
