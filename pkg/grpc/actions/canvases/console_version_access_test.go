package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// These tests pin down the access rules for the console / version
// endpoints that gate the change-request review flow. Two regression
// guards are encoded here:
//
//  1. Snapshots attached to a change request must be visible to ANY
//     authenticated user in the organization. The CR itself is described
//     org-wide (see DescribeCanvasChangeRequest) and reviewers cannot
//     evaluate the proposed console without this access — otherwise the
//     UI surfaces a "version is not visible in current flow" 403 when a
//     reviewer opens a CR opened by someone else (for example a draft
//     produced by `superplane console set`).
//  2. Drafts remain user-private: only their owner can read them.

func TestEnsureConsoleVersionReadable_SnapshotVisibleToAnyOrgMember(t *testing.T) {
	r := support.Setup(t)
	authorCtx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvasID := createCanvasWithNoopNode(authorCtx, t, r, "snapshot-access-cr")
	draftVersionID := createDraftVersion(authorCtx, t, r, canvasID, "Draft One")

	createCRResponse, err := CreateCanvasChangeRequest(authorCtx, r.Organization.ID.String(), canvasID, draftVersionID)
	require.NoError(t, err)
	snapshotVersionID := createCRResponse.ChangeRequest.Metadata.VersionId
	require.NotEmpty(t, snapshotVersionID)
	require.NotEqual(t, draftVersionID, snapshotVersionID, "CR must snapshot the draft into a new version")

	otherUser := support.CreateUser(t, r, r.Organization.ID)
	reviewerCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())

	t.Run("GetConsole returns the snapshot console to a reviewer", func(t *testing.T) {
		resp, err := GetConsole(reviewerCtx, r.Organization.ID.String(), canvasID, snapshotVersionID)
		require.NoError(t, err, "reviewer should see the CR snapshot console")
		require.NotNil(t, resp.GetConsole())
		assert.Equal(t, snapshotVersionID, resp.GetConsole().GetVersionId())
	})

	t.Run("DescribeCanvasVersion returns the snapshot to a reviewer", func(t *testing.T) {
		resp, err := DescribeCanvasVersion(reviewerCtx, r.Organization.ID.String(), canvasID, snapshotVersionID)
		require.NoError(t, err, "reviewer should describe the CR snapshot version")
		require.NotNil(t, resp.GetVersion())
		assert.Equal(t, snapshotVersionID, resp.GetVersion().GetMetadata().GetId())
	})

	t.Run("GetConsole returns the snapshot console to its owner", func(t *testing.T) {
		resp, err := GetConsole(authorCtx, r.Organization.ID.String(), canvasID, snapshotVersionID)
		require.NoError(t, err)
		assert.Equal(t, snapshotVersionID, resp.GetConsole().GetVersionId())
	})
}

func TestEnsureConsoleVersionReadable_DraftRemainsOwnerPrivate(t *testing.T) {
	r := support.Setup(t)
	authorCtx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvasID := createCanvasWithNoopNode(authorCtx, t, r, "draft-access-private")
	draftVersionID := createDraftVersion(authorCtx, t, r, canvasID, "Author Draft")

	otherUser := support.CreateUser(t, r, r.Organization.ID)
	reviewerCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())

	t.Run("GetConsole denies non-owners on a draft", func(t *testing.T) {
		_, err := GetConsole(reviewerCtx, r.Organization.ID.String(), canvasID, draftVersionID)
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, s.Code())
	})

	t.Run("DescribeCanvasVersion denies non-owners on a draft", func(t *testing.T) {
		_, err := DescribeCanvasVersion(reviewerCtx, r.Organization.ID.String(), canvasID, draftVersionID)
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, s.Code())
	})

	t.Run("Owner can still read their draft", func(t *testing.T) {
		resp, err := GetConsole(authorCtx, r.Organization.ID.String(), canvasID, draftVersionID)
		require.NoError(t, err)
		assert.Equal(t, draftVersionID, resp.GetConsole().GetVersionId())
	})
}

func TestEnsureConsoleVersionReadable_NoCRStillDenies(t *testing.T) {
	r := support.Setup(t)
	authorCtx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvasID := createCanvasWithNoopNode(authorCtx, t, r, "snapshot-without-cr")
	canvasUUID := uuid.MustParse(canvasID)

	// Hand-craft a snapshot version that is NOT attached to any change
	// request to exercise the fall-through deny path. This shouldn't
	// happen in production (snapshots are only minted by CRs), but the
	// access check must still hold its ground.
	authorUUID := r.User
	orphan := &models.CanvasVersion{
		ID:         uuid.New(),
		WorkflowID: canvasUUID,
		OwnerID:    &authorUUID,
		State:      models.CanvasVersionStateSnapshot,
	}
	require.NoError(t, database.Conn().Create(orphan).Error)

	otherUser := support.CreateUser(t, r, r.Organization.ID)
	reviewerCtx := authentication.SetUserIdInMetadata(context.Background(), otherUser.ID.String())

	_, err := GetConsole(reviewerCtx, r.Organization.ID.String(), canvasID, orphan.ID.String())
	require.Error(t, err)
	s, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, s.Code())
}
