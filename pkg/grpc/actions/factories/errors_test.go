package factories

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"google.golang.org/grpc/codes"
)

func Test__factoryErrorToStatus(t *testing.T) {
	t.Run("keeps a handler error and its client message", func(t *testing.T) {
		original := grpcerrors.Unauthenticated(errors.New("missing user"), "user not authenticated")

		err := factoryErrorToStatus(original, "failed to create factory intake")

		assert.Equal(t, original, err)
		assert.Equal(t, codes.Unauthenticated, grpcerrors.Code(err))
	})

	t.Run("keeps a resource-exhausted handler error", func(t *testing.T) {
		original := grpcerrors.ResourceExhausted(errors.New("limit"), "organization canvas limit exceeded")

		err := factoryErrorToStatus(original, "failed to create factory intake")

		assert.Equal(t, original, err)
		assert.Equal(t, codes.ResourceExhausted, grpcerrors.Code(err))
		message, ok := grpcerrors.HandlerMessage(err)
		require.True(t, ok)
		assert.Equal(t, "organization canvas limit exceeded", message)
	})
}
