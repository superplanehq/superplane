package canvasfolders

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"google.golang.org/grpc/codes"
)

func Test__CanvasFolderErrorToStatus__UsesOperationSpecificInternalMessage(t *testing.T) {
	err := canvasFolderErrorToStatus(errors.New("unexpected failure"), "failed to delete canvas folder")

	code, message, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	require.Equal(t, codes.Internal, code)
	require.Equal(t, "failed to delete canvas folder", message)
}
