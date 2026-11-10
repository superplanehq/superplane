package factories

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/runs"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestHumanReviewRecoveryCreatesSeparateRunsForWorkspaceCandidates(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("review-recovery"), "", "")
	require.NoError(t, err)
	canvas, nodes := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{NodeID: models.HumanReviewTriggerNodeID, Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onRun"}}), Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata(models.FactoryAppTemplateHumanReviewID, 2))},
		{NodeID: models.HumanReviewPrepareNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "prepareHumanReview"}}), Configuration: datatypes.NewJSONType(map[string]any{"enabled": false})},
		{NodeID: "recover", Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "schedule"}})},
		{NodeID: "list", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "listHumanReviewRecoveryCandidates"}})},
		{NodeID: "run", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runApp"}})},
	}, nil)
	require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
	createPR := func(owner *models.Factory, number int64) *models.FactoryPullRequest {
		order, err := owner.CreateWorkOrder(db, "Review", "", &r.User, nil, nil)
		require.NoError(t, err)
		pr, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: fmt.Sprintf("https://github.com/acme/app/pull/%d", number)})
		require.NoError(t, err)
		return pr
	}
	first, second := createPR(factory, 41), createPR(factory, 42)
	closed := createPR(factory, 43)
	require.NoError(t, db.Model(closed).Update("state", models.FactoryPullRequestStateClosed).Error)
	otherFactory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("other-review"), "", "")
	require.NoError(t, err)
	createPR(otherFactory, 44)
	otherCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{NodeID: models.HumanReviewTriggerNodeID, Type: models.NodeTypeTrigger, Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata(models.FactoryAppTemplateHumanReviewID, 2))}}, nil)
	require.NoError(t, db.Model(otherCanvas).Update("factory_id", otherFactory.ID).Error)

	root := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "recover", "default", nil, map[string]any{"type": "schedule.tick", "data": map[string]any{}})
	listExecution := support.CreateNodeExecutionWithConfiguration(t, canvas.ID, "list", root.ID, root.ID, map[string]any{})
	component := &listHumanReviewRecoveryComponent{}
	_, err = component.HandleBackgroundHook(core.BackgroundHookContext{
		WorkflowID: canvas.ID, OrganizationID: r.Organization.ID, NodeID: "list", RunID: root.RunID, ExecutionID: listExecution.ID,
		ActionHookContext: core.ActionHookContext{ExecutionState: contexts.NewExecutionStateContext(db, listExecution, nil)},
	})
	require.NoError(t, err)
	var candidates []models.CanvasEvent
	require.NoError(t, db.Where("execution_id = ?", listExecution.ID).Find(&candidates).Error)
	require.Len(t, candidates, 2)
	candidateIDs := make([]string, 0, len(candidates))
	runner := &runs.RunApp{}
	runConfiguration := map[string]any{"app": canvas.ID.String(), "node": models.HumanReviewTriggerNodeID}
	require.NoError(t, runner.Setup(core.SetupContext{
		Configuration: runConfiguration,
		Apps:          contexts.NewAppContext(db, canvas, &nodes[4]),
		Metadata:      contexts.NewNodeMetadataContext(db, &nodes[4]),
	}))
	for _, candidate := range candidates {
		payload := models.RootEventSourcePayload(candidate.Data.Data()).(map[string]any)
		candidateIDs = append(candidateIDs, payload["pullRequestId"].(string))
		execution := support.CreateNodeExecutionWithConfiguration(t, canvas.ID, "run", root.ID, candidate.ID, map[string]any{})
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			return runner.Execute(core.ExecutionContext{
				WorkflowID: canvas.ID.String(), NodeID: "run", RunID: root.RunID,
				Configuration: map[string]any{"app": canvas.ID.String(), "node": models.HumanReviewTriggerNodeID, "parameters": payload},
				NodeMetadata:  contexts.NewNodeMetadataContext(tx, &nodes[4]),
				Metadata:      contexts.NewExecutionMetadataContext(tx, execution),
				Requests:      contexts.NewExecutionRequestContext(tx, execution),
				Runs:          contexts.NewRunExecutionContext(tx, canvas, &nodes[4], execution),
			})
		}))
	}
	require.ElementsMatch(t, []string{first.ID.String(), second.ID.String()}, candidateIDs)
	var children []models.CanvasRun
	require.NoError(t, db.Where("parent_run_id = ?", root.RunID).Find(&children).Error)
	require.Len(t, children, 2)
	require.NotEqual(t, children[0].ID, children[1].ID)
	recoveredIDs := make([]string, 0, len(children))
	for _, child := range children {
		require.Equal(t, models.HumanReviewTriggerNodeID, child.NodeID)
		require.Equal(t, canvas.ID, child.WorkflowID)
		input := child.Input.Data().(map[string]any)
		recoveredIDs = append(recoveredIDs, input["parameters"].(map[string]any)["pullRequestId"].(string))
		require.NotEqual(t, uuid.Nil, *child.ParentExecutionID)
	}
	require.ElementsMatch(t, candidateIDs, recoveredIDs)
}
