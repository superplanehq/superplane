package canvases

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
)

func Test__PutCanvasStaging__RejectsInvalidConsoleYAML(t *testing.T) {
	r, ctx, canvas, _ := setupLiveCanvasStaging(t)
	defer r.Close()

	_, err := PutCanvasStaging(ctx, database.DB(t.Context()), r.Registry, canvas, []*pb.CanvasRepositoryFileOperation{
		{Path: ConsoleYAMLRepositoryPath, Content: []byte("just a scalar, not an object")},
	})
	require.Error(t, err)
	code, msg, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, code)
	assert.Contains(t, msg, "invalid console yaml")
}
