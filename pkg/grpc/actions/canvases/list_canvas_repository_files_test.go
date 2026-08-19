package canvases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__ListCanvasRepositoryFiles(t *testing.T) {
	r := support.Setup(t)

	t.Run("repository missing -> returns virtual spec files", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		response, err := ListCanvasRepositoryFiles(context.Background(), r.GitProvider, canvas)
		require.NoError(t, err)
		require.Len(t, response.Files, 2)
		assert.Equal(t, CanvasYAMLRepositoryPath, response.Files[0].Path)
		assert.Equal(t, ConsoleYAMLRepositoryPath, response.Files[1].Path)
	})

	t.Run("list files fails -> error", func(t *testing.T) {
		canvas, _ := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, false)

		_, err := ListCanvasRepositoryFiles(context.Background(), r.GitProvider, canvas)
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, code)
	})

	t.Run("returns repository files", func(t *testing.T) {
		canvas, _ := support.CreateCanvasWithRepository(t, r, models.RepositoryStatusReady, true)

		response, err := ListCanvasRepositoryFiles(context.Background(), r.GitProvider, canvas)
		require.NoError(t, err)
		require.Len(t, response.Files, 3)
		assert.Equal(t, "README.md", response.Files[0].Path)
		assert.Equal(t, CanvasYAMLRepositoryPath, response.Files[1].Path)
		assert.Equal(t, ConsoleYAMLRepositoryPath, response.Files[2].Path)
	})
}
