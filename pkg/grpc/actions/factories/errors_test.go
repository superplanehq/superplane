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

	t.Run("classifies an unknown error as a client-safe internal failure", func(t *testing.T) {
		err := factoryErrorToStatus(errors.New("db down"), "failed to create factory intake")

		assert.Equal(t, codes.Internal, grpcerrors.Code(err))
		message, ok := grpcerrors.HandlerMessage(err)
		require.True(t, ok)
		assert.Equal(t, "failed to create factory intake", message)
	})
}
