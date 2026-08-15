package models_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__CreateMultiInputNodeReplay_RefusesTriggerNode(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNodeID := "trigger-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNodeID,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
		},
		[]models.Edge{},
	)

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, triggerNodeID)
	require.NoError(t, err)

	var run *models.CanvasRun
	var queueItems []*models.CanvasNodeQueueItem
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var txErr error
		run, queueItems, txErr = models.CreateMultiInputNodeReplay(tx, node, nil, []models.ReplayInput{{Payload: map[string]any{"a": 1}}})
		return txErr
	})

	require.ErrorIs(t, err, models.ErrReplayTriggerNode)
	assert.Nil(t, run)
	assert.Nil(t, queueItems)

	runs, err := models.ListCanvasRuns(canvas.ID, 100, nil, models.CanvasRunFilters{})
	require.NoError(t, err)
	assert.Empty(t, runs, "no run should have been created for a refused trigger replay")

	executions, err := models.ListNodeExecutions(database.Conn(), canvas.ID, triggerNodeID, nil, nil, 100, nil)
	require.NoError(t, err)
	assert.Empty(t, executions, "no execution should have been created for a refused trigger replay")
}

func Test__CreateMultiInputNodeReplay_RequiresExplicitSourceNodeWithMultipleIncomingEdges(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	sourceA := "source-a"
	sourceB := "source-b"
	targetNodeID := "target-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: sourceA, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
			{NodeID: sourceB, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
			{NodeID: targetNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
		},
		[]models.Edge{
			{SourceID: sourceA, TargetID: targetNodeID, Channel: "default"},
			{SourceID: sourceB, TargetID: targetNodeID, Channel: "default"},
		},
	)

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, targetNodeID)
	require.NoError(t, err)

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		_, _, txErr := models.CreateMultiInputNodeReplay(tx, node, nil, []models.ReplayInput{{Payload: map[string]any{"a": 1}}})
		return txErr
	})

	require.ErrorIs(t, err, models.ErrReplaySourceNodeRequired)
}

func Test__CreateMultiInputNodeReplay_ResolvesSingleIncomingEdgeAutomatically(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	sourceNodeID := "source-1"
	targetNodeID := "target-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: sourceNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
			{NodeID: targetNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
		},
		[]models.Edge{
			{SourceID: sourceNodeID, TargetID: targetNodeID, Channel: "default"},
		},
	)

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, targetNodeID)
	require.NoError(t, err)

	var queueItems []*models.CanvasNodeQueueItem
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var txErr error
		_, queueItems, txErr = models.CreateMultiInputNodeReplay(tx, node, nil, []models.ReplayInput{{Payload: map[string]any{"a": 1}}})
		return txErr
	})
	require.NoError(t, err)

	event, err := models.FindCanvasEvent(queueItems[0].EventID)
	require.NoError(t, err)
	assert.Equal(t, sourceNodeID, event.NodeID)
}

func Test__CreateMultiInputNodeReplay_AttributesToSelfWithNoIncomingEdges(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	targetNodeID := "target-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: targetNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
		},
		[]models.Edge{},
	)

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, targetNodeID)
	require.NoError(t, err)

	var queueItems []*models.CanvasNodeQueueItem
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var txErr error
		_, queueItems, txErr = models.CreateMultiInputNodeReplay(tx, node, nil, []models.ReplayInput{{Payload: map[string]any{"a": 1}}})
		return txErr
	})
	require.NoError(t, err)

	event, err := models.FindCanvasEvent(queueItems[0].EventID)
	require.NoError(t, err)
	assert.Equal(t, targetNodeID, event.NodeID)
}

// CreateMultiInputNodeReplay must set RunID explicitly on the synthetic event, or
// CanvasEvent.BeforeCreate would manufacture a stray extra run.
func Test__CreateMultiInputNodeReplay_DoesNotCreateStrayRun(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	targetNodeID := "target-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: targetNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
		},
		[]models.Edge{},
	)

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, targetNodeID)
	require.NoError(t, err)

	var run *models.CanvasRun
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var txErr error
		run, _, txErr = models.CreateMultiInputNodeReplay(tx, node, nil, []models.ReplayInput{{Payload: map[string]any{"a": 1}}})
		return txErr
	})
	require.NoError(t, err)

	runs, err := models.ListCanvasRuns(canvas.ID, 100, nil, models.CanvasRunFilters{})
	require.NoError(t, err)
	require.Len(t, runs, 1, "exactly one run should exist: the replay run itself, no stray run from event creation")
	assert.Equal(t, run.ID, runs[0].ID)
	assert.True(t, runs[0].IsReplay)
}

func Test__ReplayInputsFromRun(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	laneA, laneB := "lane-a", "lane-b"
	joinNodeID := "join"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: laneA, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
			{NodeID: laneB, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
			{NodeID: joinNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "merge"}})},
		},
		[]models.Edge{
			{SourceID: laneA, TargetID: joinNodeID, Channel: "default"},
			{SourceID: laneB, TargetID: joinNodeID, Channel: "default"},
		},
	)

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, joinNodeID)
	require.NoError(t, err)

	sourceExecutionID := uuid.New()

	t.Run("a bare payload replays as one input attributed through the source execution", func(t *testing.T) {
		run := &models.CanvasRun{ReplayPayload: models.NewJSONValue(map[string]any{"v": "bare"})}

		inputs, err := models.ReplayInputsFromRun(database.Conn(), node, run, sourceExecutionID)
		require.NoError(t, err)
		require.Len(t, inputs, 1)
		assert.Equal(t, map[string]any{"v": "bare"}, inputs[0].Payload)
		require.NotNil(t, inputs[0].SourceExecutionID)
		assert.Equal(t, sourceExecutionID, *inputs[0].SourceExecutionID)
		assert.Nil(t, inputs[0].SourceNodeID)
	})

	t.Run("attributed pairs expand into one input per pair, keeping their own source", func(t *testing.T) {
		run := &models.CanvasRun{ReplayPayload: models.NewJSONValue([]any{
			map[string]any{"payload": map[string]any{"v": "from-a"}, "sourceNodeId": laneA},
			map[string]any{"payload": map[string]any{"v": "from-b"}, "sourceNodeId": laneB},
		})}

		inputs, err := models.ReplayInputsFromRun(database.Conn(), node, run, sourceExecutionID)
		require.NoError(t, err)
		require.Len(t, inputs, 2)

		payloadBySource := map[string]any{}
		for _, input := range inputs {
			require.NotNil(t, input.SourceNodeID)
			assert.Nil(t, input.SourceExecutionID, "a per-input source execution would attribute every input to one source")
			payloadBySource[*input.SourceNodeID] = input.Payload
		}

		assert.Equal(t, map[string]any{"v": "from-a"}, payloadBySource[laneA])
		assert.Equal(t, map[string]any{"v": "from-b"}, payloadBySource[laneB])
	})

	attributedPair := map[string]any{"payload": map[string]any{"v": "from-a"}, "sourceNodeId": laneA}

	unattributed := []struct {
		name    string
		payload any
	}{
		{"an empty list", []any{}},
		{"a bare list of payloads", []any{map[string]any{"v": "from-a"}, map[string]any{"v": "from-b"}}},
		{"a list holding something that is not an object", []any{"from-a"}},
		{"a pair without a sourceNodeId", []any{map[string]any{"payload": map[string]any{"v": "from-a"}}}},
		{"a pair whose sourceNodeId is empty", []any{map[string]any{"payload": map[string]any{"v": "from-a"}, "sourceNodeId": ""}}},
		{"a pair without a payload", []any{map[string]any{"sourceNodeId": laneA}}},
		{"a list mixing an attributed pair with a bare payload", []any{attributedPair, map[string]any{"v": "from-b"}}},
	}

	for _, tc := range unattributed {
		t.Run(tc.name+" is refused naming every live source", func(t *testing.T) {
			run := &models.CanvasRun{ReplayPayload: models.NewJSONValue(tc.payload)}

			inputs, err := models.ReplayInputsFromRun(database.Conn(), node, run, sourceExecutionID)
			require.ErrorIs(t, err, models.ErrReplayMissingInputs)
			assert.Nil(t, inputs)
			assert.Contains(t, err.Error(), laneA)
			assert.Contains(t, err.Error(), laneB)
		})
	}
}

func Test__ReplayInputsFromRun_UnattributedPayloadOnANodeWithNoIncomingSources(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	targetNodeID := "target-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: targetNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
		},
		[]models.Edge{},
	)

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, targetNodeID)
	require.NoError(t, err)

	run := &models.CanvasRun{ReplayPayload: models.NewJSONValue([]any{})}

	inputs, err := models.ReplayInputsFromRun(database.Conn(), node, run, uuid.New())
	require.ErrorIs(t, err, models.ErrReplayMissingInputs)
	assert.Nil(t, inputs)
	assert.Contains(t, err.Error(), "no source attribution recorded", "with no live source to name, the refusal must still say why")
}

func Test__CreateMultiInputNodeReplay_ReplayPayloadHoldsAttributedPairs(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	laneA, laneB := "lane-a", "lane-b"
	joinNodeID := "join"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: laneA, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
			{NodeID: laneB, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}})},
			{NodeID: joinNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "merge"}})},
		},
		[]models.Edge{
			{SourceID: laneA, TargetID: joinNodeID, Channel: "default"},
			{SourceID: laneB, TargetID: joinNodeID, Channel: "default"},
		},
	)

	node, err := models.FindCanvasNode(database.Conn(), canvas.ID, joinNodeID)
	require.NoError(t, err)

	inputs := []models.ReplayInput{
		{Payload: map[string]any{"v": "from-a"}, SourceNodeID: &laneA},
		{Payload: map[string]any{"v": "from-b"}, SourceNodeID: &laneB},
	}

	var run *models.CanvasRun
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		var txErr error
		run, _, txErr = models.CreateMultiInputNodeReplay(tx, node, nil, inputs)
		return txErr
	})
	require.NoError(t, err)

	// Re-read through the DB so ReplayPayload is decoded fresh from JSON,
	// not just the in-memory value this call happened to build.
	reread, err := models.FindCanvasRunInTransaction(database.Conn(), canvas.ID, run.ID)
	require.NoError(t, err)

	list, ok := reread.ReplayPayload.Data().([]any)
	require.True(t, ok, "ReplayPayload must be a list for a multi-input replay")
	require.Len(t, list, 2)

	bySource := make(map[string]any, 2)
	for _, raw := range list {
		entry, ok := raw.(map[string]any)
		require.True(t, ok, "each entry must be a {payload, sourceNodeId} pair, not a bare payload")
		sourceNodeID, ok := entry["sourceNodeId"].(string)
		require.True(t, ok, "each entry must carry its sourceNodeId")
		payload, ok := entry["payload"]
		require.True(t, ok, "each entry must carry its payload")
		bySource[sourceNodeID] = payload
	}

	assert.Equal(t, map[string]any{"v": "from-a"}, bySource[laneA])
	assert.Equal(t, map[string]any{"v": "from-b"}, bySource[laneB])
}
