package canvases

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__CanvasGroups__CreateListUpdateAndDelete(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	createResponse, err := CreateCanvasGroup(ctx, r.Organization.ID.String(), &pb.CanvasGroup{
		Spec: &pb.CanvasGroup_Spec{
			Title:           "  Production  ",
			BackgroundColor: models.CanvasGroupColorGreen800,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, createResponse.Group)
	require.NotNil(t, createResponse.Group.Metadata)
	require.NotNil(t, createResponse.Group.Spec)
	assert.Equal(t, "Production", createResponse.Group.Spec.Title)
	assert.Equal(t, models.CanvasGroupColorGreen800, createResponse.Group.Spec.BackgroundColor)

	listResponse, err := ListCanvasGroups(ctx, r.Organization.ID.String())
	require.NoError(t, err)
	require.Len(t, listResponse.Groups, 1)
	assert.Equal(t, createResponse.Group.Metadata.Id, listResponse.Groups[0].Metadata.Id)

	updateResponse, err := UpdateCanvasGroup(ctx, r.Organization.ID.String(), createResponse.Group.Metadata.Id, &pb.CanvasGroup{
		Spec: &pb.CanvasGroup_Spec{
			Title:           "Production Ops",
			BackgroundColor: models.CanvasGroupColorViolet800,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, updateResponse.Group)
	require.NotNil(t, updateResponse.Group.Spec)
	assert.Equal(t, "Production Ops", updateResponse.Group.Spec.Title)
	assert.Equal(t, models.CanvasGroupColorViolet800, updateResponse.Group.Spec.BackgroundColor)

	_, err = DeleteCanvasGroup(ctx, r.Organization.ID.String(), createResponse.Group.Metadata.Id)
	require.NoError(t, err)

	listResponse, err = ListCanvasGroups(ctx, r.Organization.ID.String())
	require.NoError(t, err)
	assert.Empty(t, listResponse.Groups)
}

func Test__CanvasGroups__Validation(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	_, err := CreateCanvasGroup(ctx, r.Organization.ID.String(), &pb.CanvasGroup{
		Spec: &pb.CanvasGroup_Spec{Title: "   "},
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = CreateCanvasGroup(ctx, r.Organization.ID.String(), &pb.CanvasGroup{
		Spec: &pb.CanvasGroup_Spec{
			Title:           "Invalid color",
			BackgroundColor: "red-800",
		},
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = CreateCanvasGroup(ctx, r.Organization.ID.String(), &pb.CanvasGroup{
		Spec: &pb.CanvasGroup_Spec{
			Title: strings.Repeat("a", 129),
		},
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func Test__CanvasGroups__RejectsDuplicateTitles(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	group := &pb.CanvasGroup{
		Spec: &pb.CanvasGroup_Spec{Title: "Deployments"},
	}

	_, err := CreateCanvasGroup(ctx, r.Organization.ID.String(), group)
	require.NoError(t, err)

	_, err = CreateCanvasGroup(ctx, r.Organization.ID.String(), group)
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}

func Test__CanvasGroups__RejectsDuplicateTitleOnUpdate(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	firstGroup, err := CreateCanvasGroup(ctx, r.Organization.ID.String(), &pb.CanvasGroup{
		Spec: &pb.CanvasGroup_Spec{Title: "Deployments"},
	})
	require.NoError(t, err)

	secondGroup, err := CreateCanvasGroup(ctx, r.Organization.ID.String(), &pb.CanvasGroup{
		Spec: &pb.CanvasGroup_Spec{Title: "Operations"},
	})
	require.NoError(t, err)

	_, err = UpdateCanvasGroup(ctx, r.Organization.ID.String(), secondGroup.Group.Metadata.Id, &pb.CanvasGroup{
		Spec: &pb.CanvasGroup_Spec{
			Title:           firstGroup.Group.Spec.Title,
			BackgroundColor: models.CanvasGroupColorBlue800,
		},
	})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
}

func Test__CanvasGroups__AreOrganizationScoped(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()
	otherOrganization := support.CreateOrganization(t, r, r.User)

	_, err := CreateCanvasGroup(ctx, otherOrganization.ID.String(), &pb.CanvasGroup{
		Spec: &pb.CanvasGroup_Spec{Title: "Other org"},
	})
	require.NoError(t, err)

	listResponse, err := ListCanvasGroups(ctx, r.Organization.ID.String())
	require.NoError(t, err)
	assert.Empty(t, listResponse.Groups)
}

func Test__CanvasGroups__MembershipCanBeAssignedAndRemoved(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	groupResponse, err := CreateCanvasGroup(ctx, r.Organization.ID.String(), &pb.CanvasGroup{
		Spec: &pb.CanvasGroup_Spec{Title: "Team A", BackgroundColor: models.CanvasGroupColorBlue800},
	})
	require.NoError(t, err)
	groupID := groupResponse.Group.Metadata.Id

	assignResponse, err := UpdateCanvasGroupMembership(ctx, r.Organization.ID.String(), canvas.ID.String(), groupID)
	require.NoError(t, err)
	require.NotNil(t, assignResponse.Canvas)
	require.NotNil(t, assignResponse.Canvas.Metadata)
	assert.Equal(t, groupID, assignResponse.Canvas.Metadata.CanvasGroupId)

	persistedCanvas, err := models.FindCanvas(r.Organization.ID, canvas.ID)
	require.NoError(t, err)
	require.NotNil(t, persistedCanvas.CanvasGroupID)
	assert.Equal(t, groupID, persistedCanvas.CanvasGroupID.String())

	listResponse, err := ListCanvases(ctx, r.Registry, r.Organization.ID.String(), false)
	require.NoError(t, err)
	require.Len(t, listResponse.Canvases, 1)
	assert.Equal(t, groupID, listResponse.Canvases[0].Metadata.CanvasGroupId)

	removeResponse, err := UpdateCanvasGroupMembership(ctx, r.Organization.ID.String(), canvas.ID.String(), "")
	require.NoError(t, err)
	require.NotNil(t, removeResponse.Canvas)
	require.NotNil(t, removeResponse.Canvas.Metadata)
	assert.Empty(t, removeResponse.Canvas.Metadata.CanvasGroupId)

	persistedCanvas, err = models.FindCanvas(r.Organization.ID, canvas.ID)
	require.NoError(t, err)
	assert.Nil(t, persistedCanvas.CanvasGroupID)
}

func Test__CanvasGroups__DeletingGroupFreesCanvases(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
	groupResponse, err := CreateCanvasGroup(ctx, r.Organization.ID.String(), &pb.CanvasGroup{
		Spec: &pb.CanvasGroup_Spec{Title: "Temporary"},
	})
	require.NoError(t, err)
	groupID := groupResponse.Group.Metadata.Id

	_, err = UpdateCanvasGroupMembership(ctx, r.Organization.ID.String(), canvas.ID.String(), groupID)
	require.NoError(t, err)

	_, err = DeleteCanvasGroup(ctx, r.Organization.ID.String(), groupID)
	require.NoError(t, err)

	persistedCanvas, err := models.FindCanvas(r.Organization.ID, canvas.ID)
	require.NoError(t, err)
	assert.Nil(t, persistedCanvas.CanvasGroupID)
}

func Test__CanvasGroups__ListUsesManualOrderWithNewestFirstByDefault(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	firstGroup, err := CreateCanvasGroup(ctx, r.Organization.ID.String(), &pb.CanvasGroup{
		Spec: &pb.CanvasGroup_Spec{Title: "First"},
	})
	require.NoError(t, err)

	secondGroup, err := CreateCanvasGroup(ctx, r.Organization.ID.String(), &pb.CanvasGroup{
		Spec: &pb.CanvasGroup_Spec{Title: "Second"},
	})
	require.NoError(t, err)

	listResponse, err := ListCanvasGroups(ctx, r.Organization.ID.String())
	require.NoError(t, err)
	require.Len(t, listResponse.Groups, 2)
	assert.Equal(t, secondGroup.Group.Metadata.Id, listResponse.Groups[0].Metadata.Id)
	assert.Equal(t, firstGroup.Group.Metadata.Id, listResponse.Groups[1].Metadata.Id)
}

func Test__CanvasGroups__MoveUpAndDown(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	firstGroup, err := CreateCanvasGroup(ctx, r.Organization.ID.String(), &pb.CanvasGroup{
		Spec: &pb.CanvasGroup_Spec{Title: "First"},
	})
	require.NoError(t, err)

	secondGroup, err := CreateCanvasGroup(ctx, r.Organization.ID.String(), &pb.CanvasGroup{
		Spec: &pb.CanvasGroup_Spec{Title: "Second"},
	})
	require.NoError(t, err)

	thirdGroup, err := CreateCanvasGroup(ctx, r.Organization.ID.String(), &pb.CanvasGroup{
		Spec: &pb.CanvasGroup_Spec{Title: "Third"},
	})
	require.NoError(t, err)

	moveUpResponse, err := UpdateCanvasGroupPosition(ctx, r.Organization.ID.String(), secondGroup.Group.Metadata.Id, pb.UpdateCanvasGroupPositionRequest_DIRECTION_UP)
	require.NoError(t, err)
	require.Len(t, moveUpResponse.Groups, 3)
	assert.Equal(t, []string{
		secondGroup.Group.Metadata.Id,
		thirdGroup.Group.Metadata.Id,
		firstGroup.Group.Metadata.Id,
	}, []string{
		moveUpResponse.Groups[0].Metadata.Id,
		moveUpResponse.Groups[1].Metadata.Id,
		moveUpResponse.Groups[2].Metadata.Id,
	})

	moveDownResponse, err := UpdateCanvasGroupPosition(ctx, r.Organization.ID.String(), secondGroup.Group.Metadata.Id, pb.UpdateCanvasGroupPositionRequest_DIRECTION_DOWN)
	require.NoError(t, err)
	require.Len(t, moveDownResponse.Groups, 3)
	assert.Equal(t, []string{
		thirdGroup.Group.Metadata.Id,
		secondGroup.Group.Metadata.Id,
		firstGroup.Group.Metadata.Id,
	}, []string{
		moveDownResponse.Groups[0].Metadata.Id,
		moveDownResponse.Groups[1].Metadata.Id,
		moveDownResponse.Groups[2].Metadata.Id,
	})
}
