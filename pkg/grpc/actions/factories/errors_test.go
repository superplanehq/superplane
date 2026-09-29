package factories

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/integrations/datadog"
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

	t.Run("Datadog 403 becomes a failed precondition the UI can show", func(t *testing.T) {
		err := factoryErrorToStatus(datadog.ErrErrorTrackingForbidden, "failed to search factory intake items")

		code, message, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Equal(t, datadog.ErrorTrackingForbiddenMessage, message)
	})

	t.Run("Datadog API errors become failed preconditions with the response body", func(t *testing.T) {
		apiErr := &datadog.APIError{
			StatusCode: http.StatusBadRequest,
			Body:       `{"errors":["Bad Request"]}`,
		}

		err := factoryErrorToStatus(apiErr, "failed to search factory intake items")

		code, message, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Equal(t, `{"errors":["Bad Request"]}`, message)
	})

	t.Run("intake connection errors surface the stored description", func(t *testing.T) {
		original := errors.Join(errIntakeConnectionBroken, errors.New("invalid credentials: Forbidden"))

		err := factoryErrorToStatus(original, "failed to search factory intake items")

		code, message, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Equal(t, "invalid credentials: Forbidden", message)
	})
}
