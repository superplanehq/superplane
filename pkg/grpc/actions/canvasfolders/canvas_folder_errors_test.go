package canvasfolders

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__CanvasFolderErrorToStatus__UsesOperationSpecificInternalMessage(t *testing.T) {
	err := canvasFolderErrorToStatus(errors.New("unexpected failure"), "failed to delete canvas folder")

	require.Equal(t, codes.Internal, status.Code(err))
	require.Equal(t, "failed to delete canvas folder", status.Convert(err).Message())
}
