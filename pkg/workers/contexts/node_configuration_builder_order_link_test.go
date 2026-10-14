package contexts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

const taskAssigneeCoauthorsExpression = `{{ task() == nil ? "" : join(map(filter(task().assignees, {#.email != ""}), "Co-authored-by: " + #.name + " <" + #.email + ">"), "\n") }}`

func Test_NodeConfigurationBuilder_TaskFromPullRequestLink(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	for _, activity := range []string{"Fixing failed checks", "Address review"} {
		t.Run(activity, func(t *testing.T) {
			linked := newPullRequestLinkedRun(t, r, r.Organization.ID, nil, activity, []uuid.UUID{r.User})
			builder := builderForRun(linked.canvas, linked.nodeExecution)

			assignees := resolveAssignees(t, builder)
			require.Len(t, assignees, 1)
			assert.Equal(t, r.User.String(), assignees[0]["id"])
			assert.Equal(t, r.UserModel.Name, assignees[0]["name"])
			assert.Equal(t, r.UserModel.GetEmail(), assignees[0]["email"])
			assert.Equal(t, coauthorTrailer(r.UserModel), resolveCoauthors(t, builder))
		})
	}

	t.Run("unassigned task renders an empty co-author trailer", func(t *testing.T) {
		linked := newPullRequestLinkedRun(t, r, r.Organization.ID, nil, "Fixing failed checks", nil)
		builder := builderForRun(linked.canvas, linked.nodeExecution)

		assert.Empty(t, resolveAssignees(t, builder))
		assert.Equal(t, "", resolveCoauthors(t, builder))
	})

	t.Run("assignee without email renders an empty co-author trailer", func(t *testing.T) {
		assignee := support.CreateUser(t, r, r.Organization.ID)
		require.NoError(t, database.Conn().Model(assignee).Update("email", nil).Error)
		assignee.Email = nil

		linked := newPullRequestLinkedRun(t, r, r.Organization.ID, nil, "Address review", []uuid.UUID{assignee.ID})
		builder := builderForRun(linked.canvas, linked.nodeExecution)

		assignees := resolveAssignees(t, builder)
		require.Len(t, assignees, 1)
		assert.Equal(t, "", assignees[0]["email"])
		assert.Equal(t, "", resolveCoauthors(t, builder))
	})

	t.Run("unrelated run returns nil", func(t *testing.T) {
		factoryModel := createTestFactory(t, r.Organization.ID)
		canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factoryModel.ID)
		builder := builderForRun(canvas, nodeExecution)

		result, err := builder.ResolveExpression(`task()`)
		require.NoError(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "", resolveCoauthors(t, builder))
	})

	t.Run("pull request in another organization returns nil", func(t *testing.T) {
		otherOrg, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)
		linked := newPullRequestLinkedRun(t, r, otherOrg.ID, nil, "Fixing failed checks", []uuid.UUID{r.User})
		builder := builderForRun(linked.canvas, linked.nodeExecution)

		result, err := builder.ResolveExpression(`task()`)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("pull request in another factory returns nil", func(t *testing.T) {
		otherFactory := createTestFactory(t, r.Organization.ID)
		linked := newPullRequestLinkedRun(t, r, r.Organization.ID, &otherFactory.ID, "Address review", []uuid.UUID{r.User})
		builder := builderForRun(linked.canvas, linked.nodeExecution)

		result, err := builder.ResolveExpression(`task()`)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("link lookup database error is returned", func(t *testing.T) {
		linked := newPullRequestLinkedRun(t, r, r.Organization.ID, nil, "Fixing failed checks", []uuid.UUID{r.User})
		lookupErr := errors.New("link lookup failed")
		builder := NewNodeConfigurationBuilder(dbFailingPullRequestLinkLookup(t, lookupErr), linked.canvas.ID).
			WithRootEvent(&linked.nodeExecution.RootEventID)

		result, err := builder.ResolveExpression(`task().assignees`)
		require.ErrorIs(t, err, lookupErr)
		assert.ErrorContains(t, err, "order() could not")
		_, isAssigneeList := result.([]any)
		assert.False(t, isAssigneeList)
	})
}

type pullRequestLinkedRun struct {
	canvas        *models.Canvas
	nodeExecution *models.CanvasNodeExecution
}

func newPullRequestLinkedRun(
	t *testing.T,
	r *support.ResourceRegistry,
	canvasOrgID uuid.UUID,
	canvasFactoryID *uuid.UUID,
	activity string,
	assigneeIDs []uuid.UUID,
) pullRequestLinkedRun {
	t.Helper()

	orderFactory := createTestFactory(t, r.Organization.ID)
	order, err := orderFactory.CreateWorkOrder(database.Conn(), "Ship feature", "Implement and open PR", &r.User, nil, nil)
	require.NoError(t, err)
	if len(assigneeIDs) > 0 {
		require.NoError(t, order.UpdateAssignees(database.Conn(), assigneeIDs, r.User))
	}

	canvasFactory := orderFactory.ID
	if canvasFactoryID != nil {
		canvasFactory = *canvasFactoryID
	}
	canvas, nodeExecution, run := executionForFactory(t, r, canvasOrgID, canvasFactory)
	pullRequest, err := order.CreatePullRequest(database.Conn(), models.FactoryPullRequestParams{
		URL: fmt.Sprintf("https://github.com/acme/app-%s/pull/1", uuid.NewString()[:8]),
	})
	require.NoError(t, err)
	require.NoError(t, pullRequest.LinkRun(database.Conn(), run.ID, activity))

	return pullRequestLinkedRun{canvas: canvas, nodeExecution: nodeExecution}
}

func createTestFactory(t *testing.T, organizationID uuid.UUID) *models.Factory {
	t.Helper()
	factoryModel, err := models.CreateFactory(database.Conn(), organizationID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	return factoryModel
}

func executionForFactory(
	t *testing.T,
	r *support.ResourceRegistry,
	organizationID uuid.UUID,
	factoryID uuid.UUID,
) (*models.Canvas, *models.CanvasNodeExecution, *models.CanvasRun) {
	t.Helper()
	if organizationID == r.Organization.ID {
		return setupFactoryAppExecution(t, r, factoryID)
	}

	const nodeID = "repair"
	canvas, _ := support.CreateCanvas(
		t,
		organizationID,
		r.User,
		[]models.CanvasNode{{NodeID: nodeID, Type: models.NodeTypeComponent}},
		nil,
	)
	require.NoError(t, database.Conn().Model(canvas).Update("factory_id", factoryID).Error)
	canvas.FactoryID = &factoryID

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, nodeID, "default", nil)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(database.Conn(), rootEvent)
	require.NoError(t, err)
	nodeExecution := support.CreateCanvasNodeExecution(t, canvas.ID, nodeID, rootEvent.ID, rootEvent.ID)
	nodeExecution.RunID = run.ID
	require.NoError(t, database.Conn().Save(nodeExecution).Error)
	return canvas, nodeExecution, run
}

func builderForRun(canvas *models.Canvas, nodeExecution *models.CanvasNodeExecution) *NodeConfigurationBuilder {
	return NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithRootEvent(&nodeExecution.RootEventID)
}

func resolveAssignees(t *testing.T, builder *NodeConfigurationBuilder) []map[string]any {
	t.Helper()
	result, err := builder.ResolveExpression(`task().assignees`)
	require.NoError(t, err)
	items, ok := result.([]any)
	require.True(t, ok)
	assignees := make([]map[string]any, 0, len(items))
	for _, item := range items {
		assignee, ok := item.(map[string]any)
		require.True(t, ok)
		assignees = append(assignees, assignee)
	}
	return assignees
}

func resolveCoauthors(t *testing.T, builder *NodeConfigurationBuilder) string {
	t.Helper()
	built, err := builder.Build(map[string]any{"COAUTHORS": taskAssigneeCoauthorsExpression})
	require.NoError(t, err)
	trailer, ok := built["COAUTHORS"].(string)
	require.True(t, ok)
	return trailer
}

func coauthorTrailer(user *models.User) string {
	return "Co-authored-by: " + user.Name + " <" + user.GetEmail() + ">"
}

type failingQueryPool struct {
	gorm.ConnPool
	needle string
	fail   error
}

func (p failingQueryPool) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if strings.Contains(query, p.needle) {
		return nil, p.fail
	}
	return p.ConnPool.QueryContext(ctx, query, args...)
}

func (p failingQueryPool) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if strings.Contains(query, p.needle) {
		return nil, p.fail
	}
	return p.ConnPool.ExecContext(ctx, query, args...)
}

func (p failingQueryPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	beginner, ok := p.ConnPool.(interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	})
	if !ok {
		return nil, errors.New("connection pool cannot begin a transaction")
	}
	return beginner.BeginTx(ctx, opts)
}

func dbFailingPullRequestLinkLookup(t *testing.T, fail error) *gorm.DB {
	t.Helper()
	base := database.Conn()
	require.NotNil(t, base.Statement)
	original := base.Statement.ConnPool
	base.Statement.ConnPool = failingQueryPool{
		ConnPool: original,
		needle:   "factory_pull_request_runs",
		fail:     fail,
	}
	t.Cleanup(func() {
		base.Statement.ConnPool = original
	})
	return base
}
