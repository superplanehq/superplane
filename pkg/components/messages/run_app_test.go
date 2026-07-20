package messages

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestRunAppFailureMessage(t *testing.T) {
	timedOutAt := "2026-07-20T12:00:00Z"
	assert.Equal(t, "timed out after 30s", runAppFailureMessage(runAppExecutionMetadata{
		TimedOutAt: &timedOutAt,
	}, map[string]any{"timeout": 30}, nil))
	assert.Equal(t, "timed out after 3600s", runAppFailureMessage(runAppExecutionMetadata{
		TimedOutAt: &timedOutAt,
	}, map[string]any{}, nil))

	runError := "child failed"
	assert.Equal(t, "child failed", runAppFailureMessage(runAppExecutionMetadata{}, nil, &runError))
	assert.Equal(t, "", runAppFailureMessage(runAppExecutionMetadata{}, nil, nil))
}

func TestRunAppTimeoutSeconds(t *testing.T) {
	defaultTimeout := runAppTimeoutSeconds(nil)
	assert.Equal(t, defaultRunAppTimeoutSeconds, defaultTimeout)

	zero := 0
	assert.Equal(t, defaultRunAppTimeoutSeconds, runAppTimeoutSeconds(&zero))

	custom := 120
	assert.Equal(t, 120, runAppTimeoutSeconds(&custom))
}

func TestRunAppExecuteSchedulesTimeoutWhenConfigured(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	parentCanvas, parentNodes := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{
				NodeID: "runApp",
				Name:   "Run App",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runApp"}}),
			},
		},
		nil,
	)

	childCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{{NodeID: "onRun", Type: models.NodeTypeTrigger}},
		nil,
	)

	parentRun := createRunAppRunRecord(t, parentCanvas.ID, "trigger", nil)
	execution := createRunAppExecutionRecord(t, parentCanvas.ID, parentRun.ID, parentNodes[1].NodeID)

	requests := &scheduledRunAppRequestContext{}
	timeout := 45

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		parentNodes[1].Metadata = datatypes.NewJSONType(map[string]any{
			"app": map[string]any{
				"id":   childCanvas.ID.String(),
				"name": "Child App",
			},
			"node": map[string]any{
				"id":   "onRun",
				"name": "On Run",
			},
		})

		return (&RunApp{}).Execute(core.ExecutionContext{
			WorkflowID: parentCanvas.ID.String(),
			Configuration: map[string]any{
				"app":        childCanvas.ID.String(),
				"node":       "onRun",
				"parameters": map[string]any{},
				"timeout":    timeout,
			},
			Metadata:     contexts.NewExecutionMetadataContext(tx, execution),
			NodeMetadata: contexts.NewNodeMetadataContext(tx, &parentNodes[1]),
			Requests:     requests,
			Runs:         contexts.NewRunExecutionContext(tx, parentCanvas, &parentNodes[1], execution),
		})
	})
	require.NoError(t, err)
	require.True(t, requests.called)
	assert.Equal(t, ActionRunTimeout, requests.actionName)
	assert.Equal(t, 45*time.Second, requests.interval)

	var metadata runAppExecutionMetadata
	require.NoError(t, mapstructureDecodeExecutionMetadata(t, execution, &metadata))
	require.NotNil(t, metadata.Run)
	assert.NotEmpty(t, metadata.Run.ID)
	assert.Nil(t, metadata.TimedOutAt)
}

func TestRunAppExecuteSchedulesDefaultTimeout(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	parentCanvas, parentNodes := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{
				NodeID: "runApp",
				Name:   "Run App",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runApp"}}),
			},
		},
		nil,
	)

	childCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{{NodeID: "onRun", Type: models.NodeTypeTrigger}},
		nil,
	)

	parentRun := createRunAppRunRecord(t, parentCanvas.ID, "trigger", nil)
	execution := createRunAppExecutionRecord(t, parentCanvas.ID, parentRun.ID, parentNodes[1].NodeID)

	requests := &scheduledRunAppRequestContext{}

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		parentNodes[1].Metadata = datatypes.NewJSONType(map[string]any{
			"app": map[string]any{
				"id":   childCanvas.ID.String(),
				"name": "Child App",
			},
			"node": map[string]any{
				"id":   "onRun",
				"name": "On Run",
			},
		})

		return (&RunApp{}).Execute(core.ExecutionContext{
			WorkflowID: parentCanvas.ID.String(),
			Configuration: map[string]any{
				"app":        childCanvas.ID.String(),
				"node":       "onRun",
				"parameters": map[string]any{},
			},
			Metadata:     contexts.NewExecutionMetadataContext(tx, execution),
			NodeMetadata: contexts.NewNodeMetadataContext(tx, &parentNodes[1]),
			Requests:     requests,
			Runs:         contexts.NewRunExecutionContext(tx, parentCanvas, &parentNodes[1], execution),
		})
	})
	require.NoError(t, err)
	require.True(t, requests.called)
	assert.Equal(t, ActionRunTimeout, requests.actionName)
	assert.Equal(t, time.Duration(defaultRunAppTimeoutSeconds)*time.Second, requests.interval)
}

func TestRunAppHandleRunTimeoutCancelsChildRun(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	parentCanvas, parentNodes := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{
				NodeID: "runApp",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runApp"}}),
			},
		},
		nil,
	)
	childCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{{NodeID: "onRun", Type: models.NodeTypeTrigger}},
		nil,
	)

	parentRun := createRunAppRunRecord(t, parentCanvas.ID, "trigger", nil)
	execution := createRunAppExecutionRecord(t, parentCanvas.ID, parentRun.ID, parentNodes[1].NodeID)
	childRun := createRunAppChildRunRecord(t, childCanvas.ID, "onRun", parentRun.ID, parentCanvas.ID, execution.ID)

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		metadataCtx := contexts.NewExecutionMetadataContext(tx, execution)
		require.NoError(t, metadataCtx.Set(runAppExecutionMetadata{
			Run: &RunMetadata{
				ID: childRun.ID.String(),
			},
		}))

		return (&RunApp{}).handleRunTimeout(core.ActionHookContext{
			Metadata:       metadataCtx,
			ExecutionState: contexts.NewExecutionStateContext(tx, execution, nil),
			Runs:           contexts.NewRunExecutionContext(tx, parentCanvas, &parentNodes[1], execution),
		})
	})
	require.NoError(t, err)

	updatedChild, err := models.FindCanvasRunInTransaction(database.Conn(), childCanvas.ID, childRun.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateCancelling, updatedChild.State)

	var metadata runAppExecutionMetadata
	require.NoError(t, mapstructureDecodeExecutionMetadata(t, execution, &metadata))
	require.NotNil(t, metadata.TimedOutAt)
	assert.NotEmpty(t, *metadata.TimedOutAt)
}

func TestRunAppHandleRunTimeoutNoOpWhenFinished(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	parentCanvas, parentNodes := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{
				NodeID: "runApp",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runApp"}}),
			},
		},
		nil,
	)
	childCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{{NodeID: "onRun", Type: models.NodeTypeTrigger}},
		nil,
	)

	parentRun := createRunAppRunRecord(t, parentCanvas.ID, "trigger", nil)
	execution := createRunAppExecutionRecord(t, parentCanvas.ID, parentRun.ID, parentNodes[1].NodeID)
	require.NoError(t, database.Conn().Model(execution).Updates(map[string]any{
		"state":  models.CanvasNodeExecutionStateFinished,
		"result": models.CanvasNodeExecutionResultPassed,
	}).Error)

	childRun := createRunAppChildRunRecord(t, childCanvas.ID, "onRun", parentRun.ID, parentCanvas.ID, execution.ID)

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		metadataCtx := contexts.NewExecutionMetadataContext(tx, execution)
		require.NoError(t, metadataCtx.Set(runAppExecutionMetadata{
			Run: &RunMetadata{ID: childRun.ID.String()},
		}))

		return (&RunApp{}).handleRunTimeout(core.ActionHookContext{
			Metadata:       metadataCtx,
			ExecutionState: contexts.NewExecutionStateContext(tx, execution, nil),
			Runs:           contexts.NewRunExecutionContext(tx, parentCanvas, &parentNodes[1], execution),
		})
	})
	require.NoError(t, err)

	updatedChild, err := models.FindCanvasRunInTransaction(database.Conn(), childCanvas.ID, childRun.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStatePending, updatedChild.State)
}

func TestRunAppHandleRunFinishedUsesTimeoutMessage(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	parentCanvas, parentNodes := support.CreateCanvas(t, r.Organization.ID, r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{
				NodeID: "runApp",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runApp"}}),
			},
		},
		nil,
	)

	parentRun := createRunAppRunRecord(t, parentCanvas.ID, "trigger", nil)
	execution := createRunAppExecutionRecord(t, parentCanvas.ID, parentRun.ID, parentNodes[1].NodeID)

	var emittedChannel string
	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		metadataCtx := contexts.NewExecutionMetadataContext(tx, execution)
		timedOutAt := time.Now().UTC().Format(time.RFC3339)
		require.NoError(t, metadataCtx.Set(runAppExecutionMetadata{
			Run: &RunMetadata{
				ID: uuid.New().String(),
			},
			TimedOutAt: &timedOutAt,
		}))

		execState := contexts.NewExecutionStateContext(tx, execution, func(events []models.CanvasEvent) {
			require.Len(t, events, 1)
			emittedChannel = events[0].Channel
		})

		params, err := core.NewRunFinishedCallback(core.NewRun(
			uuid.New(),
			uuid.New(),
			core.RunResultCancelled,
			nil,
		)).ToParameters()
		require.NoError(t, err)

		return (&RunApp{}).handleRunFinished(core.ActionHookContext{
			Metadata:       metadataCtx,
			ExecutionState: execState,
			Parameters:     params,
			Configuration: map[string]any{
				"timeout": 5,
			},
		})
	})
	require.NoError(t, err)
	assert.Equal(t, FailedOutputChannel, emittedChannel)

	updatedExecution, err := models.FindNodeExecution(parentCanvas.ID, execution.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasNodeExecutionStateFinished, updatedExecution.State)

	var metadata runAppExecutionMetadata
	require.NoError(t, mapstructureDecodeExecutionMetadata(t, updatedExecution, &metadata))
	require.NotNil(t, metadata.Run)
	require.NotNil(t, metadata.Run.Error)
	assert.Equal(t, "timed out after 5s", *metadata.Run.Error)
}

type scheduledRunAppRequestContext struct {
	called     bool
	actionName string
	interval   time.Duration
}

func (s *scheduledRunAppRequestContext) ScheduleActionCall(actionName string, parameters map[string]any, interval time.Duration) error {
	s.called = true
	s.actionName = actionName
	s.interval = interval
	return nil
}

func createRunAppRunRecord(t *testing.T, workflowID uuid.UUID, nodeID string, parentRunID *uuid.UUID) *models.CanvasRun {
	t.Helper()

	now := time.Now()
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), workflowID)
	require.NoError(t, err)

	run := models.CanvasRun{
		ID:          uuid.New(),
		WorkflowID:  workflowID,
		NodeID:      nodeID,
		VersionID:   liveVersion.ID,
		ParentRunID: parentRunID,
		State:       models.CanvasRunStateStarted,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}
	require.NoError(t, database.Conn().Create(&run).Error)
	return &run
}

func createRunAppChildRunRecord(
	t *testing.T,
	workflowID uuid.UUID,
	nodeID string,
	parentRunID uuid.UUID,
	parentWorkflowID uuid.UUID,
	parentExecutionID uuid.UUID,
) *models.CanvasRun {
	t.Helper()

	now := time.Now()
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(database.Conn(), workflowID)
	require.NoError(t, err)

	run := models.CanvasRun{
		ID:                uuid.New(),
		WorkflowID:        workflowID,
		NodeID:            nodeID,
		VersionID:         liveVersion.ID,
		ParentRunID:       &parentRunID,
		ParentWorkflowID:  &parentWorkflowID,
		ParentExecutionID: &parentExecutionID,
		State:             models.CanvasRunStatePending,
		CreatedAt:         &now,
		UpdatedAt:         &now,
	}
	require.NoError(t, database.Conn().Create(&run).Error)
	return &run
}

func createRunAppExecutionRecord(t *testing.T, workflowID, runID uuid.UUID, nodeID string) *models.CanvasNodeExecution {
	t.Helper()

	rootEvent := support.EmitCanvasEventForNode(t, workflowID, nodeID, "default", nil)
	execution := support.CreateCanvasNodeExecution(t, workflowID, nodeID, rootEvent.ID, rootEvent.ID)
	execution.RunID = runID
	require.NoError(t, database.Conn().Save(execution).Error)
	return execution
}

func mapstructureDecodeExecutionMetadata(t *testing.T, execution *models.CanvasNodeExecution, target any) error {
	t.Helper()

	updated, err := models.FindNodeExecution(execution.WorkflowID, execution.ID)
	require.NoError(t, err)

	return mapstructureDecode(updated.Metadata.Data(), target)
}

func mapstructureDecode(input any, target any) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:  target,
		TagName: "mapstructure",
	})
	if err != nil {
		return err
	}

	return decoder.Decode(input)
}
