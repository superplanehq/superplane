package common

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestSyncGitHubIssueEdit(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	factory, err := models.CreateFactory(database.DB(ctx), r.Organization.ID, "test-factory", "", "")
	require.NoError(t, err)

	// Create a work order with a GitHub issue origin
	origin := models.WorkOrderOrigin{
		URL:   "https://github.com/owner/repo/issues/42",
		Label: "#42",
	}
	workOrder, err := factory.CreateWorkOrderWithOrigin(database.DB(ctx), "Original Title", "Original Description", nil, nil, nil, origin)
	require.NoError(t, err)

	// Sync an edit
	updated, err := SyncGitHubIssueEdit(ctx, "owner/repo", 42, "Updated Title", "Updated Description")
	require.NoError(t, err)
	assert.True(t, updated, "expected work order to be updated")

	// Verify the work order was updated
	var updatedOrder models.FactoryWorkOrder
	err = database.DB(ctx).Where("id = ?", workOrder.ID).First(&updatedOrder).Error
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", updatedOrder.Title)
	assert.Equal(t, "Updated Description", updatedOrder.Description)
}

func TestSyncGitHubIssueEdit_NoWorkOrder(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	// Try to sync an edit for an issue without a work order
	updated, err := SyncGitHubIssueEdit(ctx, "owner/repo", 99, "Title", "Description")
	require.NoError(t, err)
	assert.False(t, updated, "expected no work order to be updated")
}

func TestSyncGitHubIssueComment(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	factory, err := models.CreateFactory(database.DB(ctx), r.Organization.ID, "test-factory", "", "")
	require.NoError(t, err)

	// Create a work order with a GitHub issue origin
	origin := models.WorkOrderOrigin{
		URL:   "https://github.com/owner/repo/issues/42",
		Label: "#42",
	}
	workOrder, err := factory.CreateWorkOrderWithOrigin(database.DB(ctx), "Test Task", "Test Description", nil, nil, nil, origin)
	require.NoError(t, err)

	// Add a comment by syncing
	added, err := SyncGitHubIssueComment(ctx, "owner/repo", 42, "This is a comment", "commenter")
	require.NoError(t, err)
	assert.True(t, added, "expected comment to be added")

	// Verify the comment was added
	comments, err := workOrder.ListComments(database.DB(ctx))
	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "This is a comment", comments[0].Body)
	assert.Equal(t, "automation", comments[0].AuthorKind)
}

func TestSyncGitHubIssueComment_NoWorkOrder(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	// Try to add a comment to an issue without a work order
	added, err := SyncGitHubIssueComment(ctx, "owner/repo", 99, "Comment", "commenter")
	require.NoError(t, err)
	assert.False(t, added, "expected no comment to be added")
}
