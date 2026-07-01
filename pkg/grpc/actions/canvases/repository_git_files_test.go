package canvases_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestResolveRepositoryGitRef(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvasID := uuid.New()
	var mainVersionID uuid.UUID
	var featureVersionID uuid.UUID

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		canvas := models.Canvas{
			ID:             canvasID,
			OrganizationID: r.Organization.ID,
			Name:           "git-ref-test",
			CreatedBy:      &r.User,
			CreatedAt:      &now,
			UpdatedAt:      &now,
		}
		if err := tx.Create(&canvas).Error; err != nil {
			return err
		}

		mainVersion, _, err := models.CreateInitialCommitInTransaction(
			tx,
			canvasID,
			r.User,
			models.CanvasGitBranchMain,
			"Initial commit",
			nil,
			nil,
		)
		if err != nil {
			return err
		}
		mainVersion.CommitSHA = "abc123deadbeef"
		if err := tx.Save(mainVersion).Error; err != nil {
			return err
		}
		mainVersionID = mainVersion.ID

		featureBranch, err := models.CreateWorkflowBranch(tx, canvasID, "docs/readme-update", &mainVersion.ID)
		if err != nil {
			return err
		}

		featureVersion, err := models.CreateCommitOnBranch(tx, models.CreateCommitInput{
			WorkflowID:    canvasID,
			BranchName:    featureBranch.Name,
			OwnerID:       r.User,
			CommitMessage: "Update README",
			CommitSHA:     "7b3afcf554c57de6494b953dda990bf3f48d3d7f",
		})
		if err != nil {
			return err
		}
		featureVersionID = featureVersion.ID
		return nil
	})
	require.NoError(t, err)

	ref, err := canvases.ResolveRepositoryGitRef(
		ctx,
		r.Organization.ID.String(),
		canvasID.String(),
		featureVersionID.String(),
	)
	require.NoError(t, err)
	assert.Equal(t, "7b3afcf554c57de6494b953dda990bf3f48d3d7f", ref)

	ref, err = canvases.ResolveRepositoryGitRef(
		ctx,
		r.Organization.ID.String(),
		canvasID.String(),
		mainVersionID.String(),
	)
	require.NoError(t, err)
	assert.Equal(t, "abc123deadbeef", ref)

	ref, err = canvases.ResolveRepositoryGitRef(ctx, r.Organization.ID.String(), canvasID.String(), "")
	require.NoError(t, err)
	assert.Empty(t, ref)
}
