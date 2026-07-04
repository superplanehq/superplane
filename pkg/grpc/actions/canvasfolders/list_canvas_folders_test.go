package canvasfolders

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pb "github.com/superplanehq/superplane/pkg/protos/canvas_folders"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListCanvasFolders__AreOrganizationScoped(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	otherOrganization := support.CreateOrganization(t, r, r.User)

	_, err := CreateCanvasFolder(ctx, otherOrganization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{Title: "Other org"},
	})
	require.NoError(t, err)

	listResponse, err := ListCanvasFolders(ctx, r.Organization.ID.String())
	require.NoError(t, err)
	assert.Empty(t, listResponse.Folders)
}

func Test__ListCanvasFolders__UsesManualOrderWithNewestFirstByDefault(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	firstFolder, err := CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{Title: "First"},
	})
	require.NoError(t, err)

	secondFolder, err := CreateCanvasFolder(ctx, r.Organization.ID.String(), &pb.CanvasFolder{
		Spec: &pb.CanvasFolder_Spec{Title: "Second"},
	})
	require.NoError(t, err)

	listResponse, err := ListCanvasFolders(ctx, r.Organization.ID.String())
	require.NoError(t, err)
	require.Len(t, listResponse.Folders, 2)
	assert.Equal(t, secondFolder.Folder.Metadata.Id, listResponse.Folders[0].Metadata.Id)
	assert.Equal(t, firstFolder.Folder.Metadata.Id, listResponse.Folders[1].Metadata.Id)
}
