package models

import (
	"testing"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestFactoryWorkOrder_IssueCommentDoesNotAssignGitHubOwner(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org, creatorID, factoryModel := setupFactoryWithUser(t, "github-comment-owner")
	member := createOrgUser(t, org.ID, "ada")
	linkGitHubLogin(t, member, "AdaLovelace")

	runID := createTypedSourceEvent(t, org.ID, creatorID, "github.issueComment", map[string]any{
		"action": "created",
		"issue": map[string]any{
			"html_url":  "https://github.com/acme/payments/issues/12",
			"assignees": []any{map[string]any{"login": "AdaLovelace"}},
		},
		"comment":    map[string]any{"body": "please look"},
		"repository": map[string]any{"full_name": "acme/payments"},
	})
	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Comment task", "", nil, nil, &runID)
	require.NoError(t, err)

	hook := logtest.NewGlobal()
	defer hook.Reset()

	_, err = order.UpdateStatus(database.Conn(), FactoryWorkOrderStatusUpdate{
		ToState: FactoryWorkOrderStateOpen,
	})
	require.NoError(t, err)

	loaded := reloadWorkOrder(t, factoryModel, order.ID)
	assert.Empty(t, loaded.Assignees)
	assert.Equal(t, FactoryWorkOrderStateOpen, loaded.State)
	for _, entry := range hook.AllEntries() {
		assert.NotContains(t, entry.Message, "GitHub issue owner")
	}
}

func TestFactoryWorkOrder_GitHubOwnerLookupLogsReadFailures(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, _, factoryModel := setupFactoryWithUser(t, "github-owner-log")
	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Lookup failure", "", nil, nil, nil)
	require.NoError(t, err)
	runID := uuid.New()
	order.SourceRunID = &runID

	hook := logtest.NewGlobal()
	defer hook.Reset()

	tx := database.Conn().Begin()
	require.NoError(t, tx.Rollback().Error)

	ownerID, ok := order.githubIssueOwner(tx)
	assert.False(t, ok)
	assert.Equal(t, uuid.Nil, ownerID)

	require.NotEmpty(t, hook.AllEntries())
	entry := hook.LastEntry()
	assert.Equal(t, log.WarnLevel, entry.Level)
	assert.Contains(t, entry.Message, "failed to read the GitHub issue owner")
	assert.Equal(t, order.ID, entry.Data["work_order_id"])
	assert.Equal(t, runID.String(), entry.Data["source_run_id"])
	assert.Error(t, entry.Data["error"].(error))
}
