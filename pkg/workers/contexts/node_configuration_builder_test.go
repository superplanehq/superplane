package contexts

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test_NodeConfigurationBuilder_WorkflowLevelNode_Root(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	//
	// Create a simple canvas with a trigger and a component node
	//
	triggerNode := "trigger-1"
	componentNode := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Name:   triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: componentNode,
				Name:   componentNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: componentNode, Channel: "default"},
		},
	)

	//
	// Emit root event
	//
	rootEventData := map[string]any{
		"user":    "john",
		"action":  "login",
		"success": true,
		"count":   42,
	}
	rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNode, "default", nil, rootEventData)

	//
	// Use message chain access to get information from the root event
	//
	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithRootEvent(&rootEvent.ID).
		WithInput(map[string]any{triggerNode: rootEventData})

	configuration := map[string]any{
		"user":    "{{ $[\"" + triggerNode + "\"].user }}",
		"action":  "{{ $[\"" + triggerNode + "\"].action }}",
		"success": "{{ $[\"" + triggerNode + "\"].success }}",
		"count":   "{{ $[\"" + triggerNode + "\"].count }}",
	}

	result, err := builder.Build(configuration)
	require.NoError(t, err)
	assert.Equal(t, "john", result["user"])
	assert.Equal(t, "login", result["action"])
	assert.Equal(t, "true", result["success"])
	assert.Equal(t, "42", result["count"])
}

func Test_NodeConfigurationBuilder_JSONNumberTemplateUsesOriginalToken(t *testing.T) {
	builder := NewNodeConfigurationBuilder(nil, uuid.New()).
		WithInput(map[string]any{
			"trigger": map[string]any{
				"id":    json.Number("14000000"),
				"small": json.Number("0.0000001"),
			},
		})

	result, err := builder.Build(map[string]any{
		"id":    "{{ previous().id }}",
		"small": "{{ previous().small }}",
	})

	require.NoError(t, err)
	assert.Equal(t, "14000000", result["id"])
	assert.Equal(t, "0.0000001", result["small"])
}

func Test_NodeConfigurationBuilder_JSONNumberExpressionsUseNumericTypes(t *testing.T) {
	builder := NewNodeConfigurationBuilder(nil, uuid.New()).
		WithInput(map[string]any{
			"trigger": map[string]any{
				"count": json.Number("42"),
				"price": json.Number("10.5"),
				"items": []any{json.Number("2")},
			},
		})

	result, err := builder.ResolveExpression(`previous().count > 10 && previous().price + previous().items[0] == 12.5`)

	require.NoError(t, err)
	assert.Equal(t, true, result)
}

func Test_NodeConfigurationBuilder_ObjectFieldPreservesWholeTemplateTypes(t *testing.T) {
	builder := NewNodeConfigurationBuilder(nil, uuid.New()).
		WithInput(map[string]any{
			"trigger": map[string]any{
				"enabled": true,
				"poolID":  "pool-a",
				"weight":  json.Number("0.1"),
			},
		}).
		WithConfigurationFields([]configuration.Field{
			{Name: "json", Type: configuration.FieldTypeObject},
			{Name: "name", Type: configuration.FieldTypeString},
		})

	result, err := builder.Build(map[string]any{
		"json": map[string]any{
			"enabled": "{{ previous().enabled }}",
			"label":   "pool-{{ previous().poolID }}",
			"nested": map[string]any{
				"weight": "{{ previous().weight }}",
			},
			"weights": []any{"{{ previous().weight }}"},
		},
		"name": "{{ previous().weight }}",
	})

	require.NoError(t, err)

	payload := result["json"].(map[string]any)
	assert.Equal(t, true, payload["enabled"])
	assert.Equal(t, "pool-pool-a", payload["label"])

	nested := payload["nested"].(map[string]any)
	assert.Equal(t, 0.1, nested["weight"])

	weights := payload["weights"].([]any)
	assert.Equal(t, 0.1, weights[0])

	assert.Equal(t, "0.1", result["name"])
}

func Test_NodeConfigurationBuilder_ObjectFieldResolvesRawJSONTemplateString(t *testing.T) {
	builder := NewNodeConfigurationBuilder(nil, uuid.New()).
		WithInput(map[string]any{
			"trigger": map[string]any{
				"canary": true,
			},
		}).
		WithConfigurationFields([]configuration.Field{
			{Name: "json", Type: configuration.FieldTypeObject},
		})

	result, err := builder.Build(map[string]any{
		"json": `{
			"pool_weights": {
				"pool-a": {{ previous().canary ? 0.1 : 0.9 }},
				"pool-b": {{ previous().canary ? 0.9 : 0.1 }}
			},
			"enabled": {{ previous().canary }}
		}`,
	})

	require.NoError(t, err)

	payload := result["json"].(map[string]any)
	assert.Equal(t, true, payload["enabled"])

	poolWeights := payload["pool_weights"].(map[string]any)
	assert.Equal(t, json.Number("0.1"), poolWeights["pool-a"])
	assert.Equal(t, json.Number("0.9"), poolWeights["pool-b"])
}

func Test_NodeConfigurationBuilder_JSONNumberDivisionUsesFloatSemantics(t *testing.T) {
	builder := NewNodeConfigurationBuilder(nil, uuid.New()).
		WithInput(map[string]any{
			"trigger": map[string]any{
				"count":   json.Number("10"),
				"divisor": json.Number("3"),
			},
		})

	result, err := builder.ResolveExpression(`previous().count / 3`)
	require.NoError(t, err)
	assert.InDelta(t, 10.0/3.0, result, 1e-9)

	resultBoth, err := builder.ResolveExpression(`previous().count / previous().divisor`)
	require.NoError(t, err)
	assert.InDelta(t, 10.0/3.0, resultBoth, 1e-9)
}

func Test_NodeConfigurationBuilder_JSONNumberRootPayloadExpressionsUseNumericTypes(t *testing.T) {
	builder := NewNodeConfigurationBuilder(nil, uuid.New()).
		WithRootPayload(map[string]any{
			"count": json.Number("42"),
			"nested": map[string]any{
				"price": json.Number("10.5"),
			},
		})

	result, err := builder.ResolveExpression(`root().count >= 42 && root().nested.price * 2 == 21`)

	require.NoError(t, err)
	assert.Equal(t, true, result)
}

func Test_NodeConfigurationBuilder_WorkflowLevelNode_RootFunction(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNode := "trigger-1"
	componentNode := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Name:   triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: componentNode,
				Name:   componentNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: componentNode, Channel: "default"},
		},
	)

	rootEventData := map[string]any{
		"user":    "john",
		"action":  "login",
		"success": true,
		"count":   42,
	}
	rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNode, "default", nil, rootEventData)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithRootEvent(&rootEvent.ID).
		WithInput(map[string]any{triggerNode: rootEventData})

	configuration := map[string]any{
		"user":    "{{ root().user }}",
		"action":  "{{ root().action }}",
		"success": "{{ root().success }}",
		"count":   "{{ root().count }}",
	}

	result, err := builder.Build(configuration)
	require.NoError(t, err)
	assert.Equal(t, "john", result["user"])
	assert.Equal(t, "login", result["action"])
	assert.Equal(t, "true", result["success"])
	assert.Equal(t, "42", result["count"])
}

func Test_NodeConfigurationBuilder_WorkflowLevelNode_Root_ByName(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNode := "trigger-1"
	triggerName := "filter"
	componentNode := "component-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Name:   triggerName,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: componentNode,
				Name:   "processor",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: componentNode, Channel: "default"},
		},
	)

	rootEventData := map[string]any{
		"user":    "john",
		"action":  "login",
		"success": true,
		"count":   42,
	}
	rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNode, "default", nil, rootEventData)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithRootEvent(&rootEvent.ID).
		WithInput(map[string]any{triggerNode: rootEventData})

	configuration := map[string]any{
		"user":    "{{ $[\"" + triggerName + "\"].user }}",
		"action":  "{{ $[\"" + triggerName + "\"].action }}",
		"success": "{{ $[\"" + triggerName + "\"].success }}",
		"count":   "{{ $[\"" + triggerName + "\"].count }}",
	}

	result, err := builder.Build(configuration)
	require.NoError(t, err)
	assert.Equal(t, "john", result["user"])
	assert.Equal(t, "login", result["action"])
	assert.Equal(t, "true", result["success"])
	assert.Equal(t, "42", result["count"])
}

func Test_NodeConfigurationBuilder_NodeNameNotUnique_UsesClosestInChain(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	// Create a workflow with two nodes having the same name "filter"
	// node-1 (filter) -> node-2 (filter) -> node-3 (target)
	// When node-3 references "filter", it should get node-2's data (the closest one)
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Name:   "filter",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: "node-2",
				Name:   "filter",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: "node-3",
				Name:   "target",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{
			{SourceID: "node-1", TargetID: "node-2", Channel: "default"},
			{SourceID: "node-2", TargetID: "node-3", Channel: "default"},
		},
	)

	// Create executions for the chain
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)

	execution1 := support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", rootEvent.ID, rootEvent.ID)
	node1Data := map[string]any{"result": "first-filter"}
	event1 := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "node-1", "default", &execution1.ID, node1Data)

	execution2 := support.CreateNextNodeExecution(t, canvas.ID, "node-2", rootEvent.ID, event1.ID, &execution1.ID)
	node2Data := map[string]any{"result": "second-filter"}
	support.EmitCanvasEventForNodeWithData(t, canvas.ID, "node-2", "default", &execution2.ID, node2Data)

	// Build configuration from node-3's perspective - should get node-2's data (closest "filter")
	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithPreviousExecution(&execution2.ID).
		WithRootEvent(&rootEvent.ID).
		WithInput(map[string]any{"node-2": node2Data})

	configuration := map[string]any{
		"field": "{{ $[\"filter\"].result }}",
	}

	result, err := builder.Build(configuration)
	require.NoError(t, err)
	assert.Equal(t, "second-filter", result["field"])
}

// Test_NodeConfigurationBuilder_CyclicFeedback_UsesCurrentLineage guards against
// a loop reading a stale iteration's body output. A loop body forms a feedback
// cycle (loop -> body -> loop), so the body node is reachable "upstream" of the
// loop and has one execution per iteration. When the loop evaluates an
// expression (e.g. its until-condition) for the current feedback, $[body] must
// resolve to the current iteration's output, not an older one.
func Test_NodeConfigurationBuilder_CyclicFeedback_UsesCurrentLineage(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	// start -> loop -> body -> loop (feedback). The cycle makes "body" upstream
	// of "loop", so all of body's per-iteration executions are in scope.
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "start", Name: "start", Type: models.NodeTypeTrigger},
			{
				NodeID: "loop",
				Name:   "loop",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "loop"}}),
			},
			{
				NodeID: "body",
				Name:   "body",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{
			{SourceID: "start", TargetID: "loop", Channel: "default"},
			{SourceID: "loop", TargetID: "body", Channel: "next"},
			{SourceID: "body", TargetID: "loop", Channel: "success"},
		},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "start", "default", nil)

	// The long-lived loop session. Each body iteration branches off it (as in the
	// real feedback-model loop), so the body executions are siblings.
	loopExecution := support.CreateCanvasNodeExecution(t, canvas.ID, "loop", rootEvent.ID, rootEvent.ID)

	body1 := support.CreateNextNodeExecution(t, canvas.ID, "body", rootEvent.ID, rootEvent.ID, &loopExecution.ID)
	support.EmitCanvasEventForNodeWithData(t, canvas.ID, "body", "success", &body1.ID, map[string]any{"result": "tail"})

	body2 := support.CreateNextNodeExecution(t, canvas.ID, "body", rootEvent.ID, rootEvent.ID, &loopExecution.ID)
	support.EmitCanvasEventForNodeWithData(t, canvas.ID, "body", "success", &body2.ID, map[string]any{"result": "tail"})

	// Current iteration: body flips "head". The feedback into the loop is this event.
	body3 := support.CreateNextNodeExecution(t, canvas.ID, "body", rootEvent.ID, rootEvent.ID, &loopExecution.ID)
	body3Event := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "body", "success", &body3.ID, map[string]any{"result": "head"})

	// Evaluate from the loop's perspective for the current feedback (body3).
	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithNodeID("loop").
		WithRootEvent(&rootEvent.ID).
		WithPreviousExecution(&body3.ID).
		WithIncomingEventID(&body3Event.ID)

	// $[body] must be the current iteration (head), not the oldest (tail).
	value, err := builder.ResolveExpression(`$["body"].result`)
	require.NoError(t, err)
	assert.Equal(t, "head", value)

	done, err := builder.ResolveExpression(`$["body"].result == "head"`)
	require.NoError(t, err)
	assert.Equal(t, true, done)
}

func Test_NodeConfigurationBuilder_ReferencedUpstreamNodeWithNoOutputsResolvesToNil(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "start", Name: "start", Type: models.NodeTypeTrigger},
			{NodeID: "build-a", Name: "Build A", Type: models.NodeTypeComponent},
			{NodeID: "build-b", Name: "Build B", Type: models.NodeTypeComponent},
			{NodeID: "merge", Name: "merge", Type: models.NodeTypeComponent},
		},
		[]models.Edge{
			{SourceID: "start", TargetID: "build-a", Channel: "default"},
			{SourceID: "start", TargetID: "build-b", Channel: "default"},
			{SourceID: "build-a", TargetID: "merge", Channel: "passed"},
			{SourceID: "build-b", TargetID: "merge", Channel: "passed"},
		},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "start", "default", nil)
	buildAExecution := support.CreateCanvasNodeExecution(t, canvas.ID, "build-a", rootEvent.ID, rootEvent.ID)
	buildAEvent := support.EmitCanvasEventForNodeWithData(
		t,
		canvas.ID,
		"build-a",
		"passed",
		&buildAExecution.ID,
		map[string]any{"status": "succeeded"},
	)

	// The sibling execution exists in the same run but has not emitted an event yet.
	support.CreateCanvasNodeExecution(t, canvas.ID, "build-b", rootEvent.ID, rootEvent.ID)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithNodeID("merge").
		WithPreviousExecution(&buildAExecution.ID).
		WithIncomingEventID(&buildAEvent.ID).
		WithRootEvent(&rootEvent.ID).
		WithInput(map[string]any{"build-a": map[string]any{"status": "succeeded"}})

	missing, err := builder.ResolveExpression(`$["Build B"] == nil`)
	require.NoError(t, err)
	assert.Equal(t, true, missing)
}

func setCanvasEventCreatedAt(t *testing.T, eventID uuid.UUID, createdAt time.Time) {
	t.Helper()

	err := database.Conn().
		Model(&models.CanvasEvent{}).
		Where("id = ?", eventID).
		Update("created_at", createdAt).
		Error
	require.NoError(t, err)
}

func Test_NodeConfigurationBuilder_PreviousFallsBackToLatestUpstreamOutput(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "start", Name: "start", Type: models.NodeTypeTrigger},
			{NodeID: "build-a", Name: "Build A", Type: models.NodeTypeComponent},
			{NodeID: "build-b", Name: "Build B", Type: models.NodeTypeComponent},
			{NodeID: "merge", Name: "merge", Type: models.NodeTypeComponent},
		},
		[]models.Edge{
			{SourceID: "start", TargetID: "build-a", Channel: "default"},
			{SourceID: "start", TargetID: "build-b", Channel: "default"},
			{SourceID: "build-a", TargetID: "merge", Channel: "passed"},
			{SourceID: "build-b", TargetID: "merge", Channel: "passed"},
		},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "start", "default", nil)
	buildAExecution := support.CreateCanvasNodeExecution(t, canvas.ID, "build-a", rootEvent.ID, rootEvent.ID)
	buildAEvent := support.EmitCanvasEventForNodeWithData(
		t,
		canvas.ID,
		"build-a",
		"passed",
		&buildAExecution.ID,
		map[string]any{"status": "succeeded"},
	)
	buildBExecution := support.CreateCanvasNodeExecution(t, canvas.ID, "build-b", rootEvent.ID, rootEvent.ID)
	buildBEvent := support.EmitCanvasEventForNodeWithData(
		t,
		canvas.ID,
		"build-b",
		"failed",
		&buildBExecution.ID,
		map[string]any{"status": "failed"},
	)
	createdAt := time.Now()
	setCanvasEventCreatedAt(t, buildAEvent.ID, createdAt)
	setCanvasEventCreatedAt(t, buildBEvent.ID, createdAt.Add(time.Second))

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithNodeID("merge").
		WithPreviousExecution(&buildAExecution.ID).
		WithIncomingEventID(&buildBEvent.ID).
		WithRootEvent(&rootEvent.ID)

	status, err := builder.ResolveExpression(`previous().status`)
	require.NoError(t, err)
	assert.Equal(t, "failed", status)

	status, err = builder.ResolveExpression(`previous(2).status`)
	require.NoError(t, err)
	assert.Equal(t, "succeeded", status)
}

func Test_NodeConfigurationBuilder_PreviousPrefersCurrentInputOverLatestUpstreamOutput(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "start", Name: "start", Type: models.NodeTypeTrigger},
			{NodeID: "build-a", Name: "Build A", Type: models.NodeTypeComponent},
			{NodeID: "build-b", Name: "Build B", Type: models.NodeTypeComponent},
			{NodeID: "merge", Name: "merge", Type: models.NodeTypeComponent},
		},
		[]models.Edge{
			{SourceID: "start", TargetID: "build-a", Channel: "default"},
			{SourceID: "start", TargetID: "build-b", Channel: "default"},
			{SourceID: "build-a", TargetID: "merge", Channel: "passed"},
			{SourceID: "build-b", TargetID: "merge", Channel: "passed"},
		},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "start", "default", nil)
	buildAExecution := support.CreateCanvasNodeExecution(t, canvas.ID, "build-a", rootEvent.ID, rootEvent.ID)
	buildAEvent := support.EmitCanvasEventForNodeWithData(
		t,
		canvas.ID,
		"build-a",
		"passed",
		&buildAExecution.ID,
		map[string]any{"status": "succeeded"},
	)
	buildBExecution := support.CreateCanvasNodeExecution(t, canvas.ID, "build-b", rootEvent.ID, rootEvent.ID)
	buildBEvent := support.EmitCanvasEventForNodeWithData(
		t,
		canvas.ID,
		"build-b",
		"failed",
		&buildBExecution.ID,
		map[string]any{"status": "failed"},
	)
	createdAt := time.Now()
	setCanvasEventCreatedAt(t, buildAEvent.ID, createdAt)
	setCanvasEventCreatedAt(t, buildBEvent.ID, createdAt.Add(time.Second))

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithNodeID("merge").
		WithPreviousExecution(&buildAExecution.ID).
		WithIncomingEventID(&buildAEvent.ID).
		WithRootEvent(&rootEvent.ID).
		WithInput(map[string]any{"build-a": map[string]any{"status": "succeeded"}})

	status, err := builder.ResolveExpression(`previous().status`)
	require.NoError(t, err)
	assert.Equal(t, "succeeded", status)

	nextStatus, err := builder.ResolveExpression(`previous(2).status`)
	require.NoError(t, err)
	assert.Equal(t, "failed", nextStatus)
}

func Test_NodeConfigurationBuilder_PreviousFallbackUsesDirectParentsOnly(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "start", Name: "start", Type: models.NodeTypeTrigger},
			{NodeID: "prepare", Name: "Prepare", Type: models.NodeTypeComponent},
			{NodeID: "build-a", Name: "Build A", Type: models.NodeTypeComponent},
			{NodeID: "build-b", Name: "Build B", Type: models.NodeTypeComponent},
			{NodeID: "merge", Name: "merge", Type: models.NodeTypeComponent},
		},
		[]models.Edge{
			{SourceID: "start", TargetID: "prepare", Channel: "default"},
			{SourceID: "prepare", TargetID: "build-a", Channel: "default"},
			{SourceID: "prepare", TargetID: "build-b", Channel: "default"},
			{SourceID: "build-a", TargetID: "merge", Channel: "passed"},
			{SourceID: "build-b", TargetID: "merge", Channel: "passed"},
		},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "start", "default", nil)
	buildAExecution := support.CreateCanvasNodeExecution(t, canvas.ID, "build-a", rootEvent.ID, rootEvent.ID)
	buildAEvent := support.EmitCanvasEventForNodeWithData(
		t,
		canvas.ID,
		"build-a",
		"passed",
		&buildAExecution.ID,
		map[string]any{"status": "build-a"},
	)
	buildBExecution := support.CreateCanvasNodeExecution(t, canvas.ID, "build-b", rootEvent.ID, rootEvent.ID)
	buildBEvent := support.EmitCanvasEventForNodeWithData(
		t,
		canvas.ID,
		"build-b",
		"passed",
		&buildBExecution.ID,
		map[string]any{"status": "build-b"},
	)
	prepareExecution := support.CreateCanvasNodeExecution(t, canvas.ID, "prepare", rootEvent.ID, rootEvent.ID)
	prepareEvent := support.EmitCanvasEventForNodeWithData(
		t,
		canvas.ID,
		"prepare",
		"passed",
		&prepareExecution.ID,
		map[string]any{"status": "prepare"},
	)
	createdAt := time.Now()
	setCanvasEventCreatedAt(t, buildAEvent.ID, createdAt)
	setCanvasEventCreatedAt(t, buildBEvent.ID, createdAt.Add(time.Second))
	setCanvasEventCreatedAt(t, prepareEvent.ID, createdAt.Add(2*time.Second))

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithNodeID("merge").
		WithPreviousExecution(&buildAExecution.ID).
		WithRootEvent(&rootEvent.ID)

	status, err := builder.ResolveExpression(`previous().status`)
	require.NoError(t, err)
	assert.Equal(t, "build-b", status)

	status, err = builder.ResolveExpression(`previous(2).status`)
	require.NoError(t, err)
	assert.Equal(t, "build-a", status)
}

func Test_NodeConfigurationBuilder_PreviousFallbackUsesOutputLookupAmbiguityRules(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "start", Name: "start", Type: models.NodeTypeTrigger},
			{NodeID: "build-a", Name: "Build A", Type: models.NodeTypeComponent},
			{NodeID: "merge", Name: "merge", Type: models.NodeTypeComponent},
		},
		[]models.Edge{
			{SourceID: "start", TargetID: "build-a", Channel: "default"},
			{SourceID: "build-a", TargetID: "merge", Channel: "passed"},
		},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "start", "default", nil)
	buildAExecution := support.CreateCanvasNodeExecution(t, canvas.ID, "build-a", rootEvent.ID, rootEvent.ID)
	support.EmitCanvasEventForNodeWithData(
		t,
		canvas.ID,
		"build-a",
		"passed",
		&buildAExecution.ID,
		map[string]any{"status": "first"},
	)
	support.EmitCanvasEventForNodeWithData(
		t,
		canvas.ID,
		"build-a",
		"passed",
		&buildAExecution.ID,
		map[string]any{"status": "second"},
	)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithNodeID("merge").
		WithPreviousExecution(&buildAExecution.ID).
		WithRootEvent(&rootEvent.ID)

	_, err := builder.ResolveExpression(`previous().status`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous outputs")

	_, err = builder.ResolveExpression(`$["Build A"].status`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous outputs")
}

func Test_selectCurrentExecutionsByNode_PrefersLinearExecutionAndLatestFallback(t *testing.T) {
	now := time.Now()
	old := now.Add(-time.Minute)
	newer := now.Add(time.Minute)

	staleLoopExecution := models.CanvasNodeExecution{
		ID:        uuid.New(),
		NodeID:    "loop-body",
		CreatedAt: &newer,
	}
	currentLoopExecution := models.CanvasNodeExecution{
		ID:        uuid.New(),
		NodeID:    "loop-body",
		CreatedAt: &old,
	}
	oldBuildExecution := models.CanvasNodeExecution{
		ID:        uuid.New(),
		NodeID:    "build",
		CreatedAt: &old,
	}
	newBuildExecution := models.CanvasNodeExecution{
		ID:        uuid.New(),
		NodeID:    "build",
		CreatedAt: &newer,
	}

	selected := selectCurrentExecutionsByNode(
		[]models.CanvasNodeExecution{
			staleLoopExecution,
			currentLoopExecution,
			oldBuildExecution,
			newBuildExecution,
		},
		[]models.CanvasNodeExecution{currentLoopExecution},
	)

	selectedByNode := map[string]uuid.UUID{}
	for _, execution := range selected {
		selectedByNode[execution.NodeID] = execution.ID
	}

	assert.Equal(t, currentLoopExecution.ID, selectedByNode["loop-body"])
	assert.Equal(t, newBuildExecution.ID, selectedByNode["build"])
	assert.Len(t, selectedByNode, 2)
}

func Test_NodeConfigurationBuilder_NodeNameNotUnique_NoneInChain(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	// Create a workflow with two nodes having the same name but neither in the execution chain
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Name:   "filter",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: "node-2",
				Name:   "filter",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{},
	)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithInput(map[string]any{})

	configuration := map[string]any{
		"field": "{{ $[\"filter\"].data }}",
	}

	_, err := builder.Build(configuration)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "node name filter is not unique")
}

func Test_NodeConfigurationBuilder_NodeIDNotAllowed(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Name:   "filter",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{},
	)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithInput(map[string]any{"node-1": map[string]any{"data": "value"}})

	configuration := map[string]any{
		"field": "{{ $[\"node-1\"].data }}",
	}

	_, err := builder.Build(configuration)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "node name node-1 not found in execution chain")
}

func Test_NodeConfigurationBuilder_WorkflowLevelNode_Root_NoRootEvent(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "node-1", Name: "node-1", Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithInput(map[string]any{})

	configuration := map[string]any{
		"field": "{{ $[\"node-1\"].data }}",
	}

	_, err := builder.Build(configuration)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in execution chain")
}

func Test_NodeConfigurationBuilder_WorkflowLevelNode_Chain(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	// Create a workflow with three sequential nodes
	triggerNode := "trigger-1"
	node1 := "node-1"
	node2 := "node-2"
	node3 := "node-3"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Name:   triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: node1,
				Name:   node1,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: node2,
				Name:   node2,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: node3,
				Name:   node3,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{
			{SourceID: node1, TargetID: node2, Channel: "default"},
			{SourceID: node2, TargetID: node3, Channel: "default"},
		},
	)

	//
	// Create root event
	//
	rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNode, "default", nil, map[string]any{"root": "data"})

	//
	// Simulate execution for first node
	//
	execution1 := support.CreateCanvasNodeExecution(
		t,
		canvas.ID,
		node1,
		rootEvent.ID,
		rootEvent.ID,
	)
	node1Data := map[string]any{
		"step":   1,
		"result": "first",
	}
	event1 := support.EmitCanvasEventForNodeWithData(t, canvas.ID, node1, "default", &execution1.ID, node1Data)

	//
	// Simulate execution for second node
	//
	execution2 := support.CreateNextNodeExecution(
		t,
		canvas.ID,
		node2,
		rootEvent.ID,
		event1.ID,
		&execution1.ID,
	)

	node2Data := map[string]any{
		"step":   2,
		"result": "second",
	}
	support.EmitCanvasEventForNodeWithData(t, canvas.ID, node2, "default", &execution2.ID, node2Data)

	//
	// Now test message chain access from node3 perspective
	//
	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithPreviousExecution(&execution2.ID).
		WithRootEvent(&rootEvent.ID).
		WithInput(map[string]any{node2: node2Data})

	configuration := map[string]any{
		"from_node1": "{{ $[\"" + node1 + "\"].result }}",
		"from_node2": "{{ $[\"" + node2 + "\"].step }}",
	}

	result, err := builder.Build(configuration)
	require.NoError(t, err)
	assert.Equal(t, "first", result["from_node1"])
	assert.Equal(t, "2", result["from_node2"])
}

func Test_NodeConfigurationBuilder_Chain_IncludesParallelUpstreamExecutions(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	startNode := "start"
	action1Node := "action-1"
	action2Node := "action-2"
	action3Node := "action-3"
	mergeNode := "merge"

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: startNode,
				Name:   startNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: action1Node,
				Name:   action1Node,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: action2Node,
				Name:   action2Node,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: action3Node,
				Name:   action3Node,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: mergeNode,
				Name:   mergeNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "merge"}}),
			},
		},
		[]models.Edge{
			{SourceID: startNode, TargetID: action1Node, Channel: "default"},
			{SourceID: startNode, TargetID: action2Node, Channel: "default"},
			{SourceID: startNode, TargetID: action3Node, Channel: "default"},
			{SourceID: action1Node, TargetID: mergeNode, Channel: "default"},
			{SourceID: action2Node, TargetID: mergeNode, Channel: "default"},
			{SourceID: action3Node, TargetID: mergeNode, Channel: "default"},
		},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, startNode, "default", nil)

	execution1 := support.CreateCanvasNodeExecution(t, canvas.ID, action1Node, rootEvent.ID, rootEvent.ID)
	action1Data := map[string]any{"value": "from-action-1"}
	support.EmitCanvasEventForNodeWithData(t, canvas.ID, action1Node, "default", &execution1.ID, action1Data)

	execution2 := support.CreateCanvasNodeExecution(t, canvas.ID, action2Node, rootEvent.ID, rootEvent.ID)
	action2Data := map[string]any{"value": "from-action-2"}
	support.EmitCanvasEventForNodeWithData(t, canvas.ID, action2Node, "default", &execution2.ID, action2Data)

	execution3 := support.CreateCanvasNodeExecution(t, canvas.ID, action3Node, rootEvent.ID, rootEvent.ID)
	action3Data := map[string]any{"value": "from-action-3"}
	support.EmitCanvasEventForNodeWithData(t, canvas.ID, action3Node, "default", &execution3.ID, action3Data)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithNodeID(mergeNode).
		WithPreviousExecution(&execution1.ID).
		WithRootEvent(&rootEvent.ID).
		WithInput(map[string]any{action1Node: action1Data})

	configuration := map[string]any{
		"action1": "{{ $[\"action-1\"].value }}",
		"action2": "{{ $[\"action-2\"].value }}",
		"action3": "{{ $[\"action-3\"].value }}",
	}

	result, err := builder.Build(configuration)
	require.NoError(t, err)
	assert.Equal(t, "from-action-1", result["action1"])
	assert.Equal(t, "from-action-2", result["action2"])
	assert.Equal(t, "from-action-3", result["action3"])
}

func Test_NodeConfigurationBuilder_WorkflowLevelNode_Previous(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNode := "trigger-1"
	node1 := "node-1"
	node2 := "node-2"
	node3 := "node-3"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: node1,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: node2,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: node3,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{
			{SourceID: node1, TargetID: node2, Channel: "default"},
			{SourceID: node2, TargetID: node3, Channel: "default"},
		},
	)

	rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNode, "default", nil, map[string]any{"root": "data"})

	execution1 := support.CreateCanvasNodeExecution(
		t,
		canvas.ID,
		node1,
		rootEvent.ID,
		rootEvent.ID,
	)
	node1Data := map[string]any{
		"step":   1,
		"result": "first",
	}
	event1 := support.EmitCanvasEventForNodeWithData(t, canvas.ID, node1, "default", &execution1.ID, node1Data)

	execution2 := support.CreateNextNodeExecution(
		t,
		canvas.ID,
		node2,
		rootEvent.ID,
		event1.ID,
		&execution1.ID,
	)
	node2Data := map[string]any{
		"step":   2,
		"result": "second",
	}
	support.EmitCanvasEventForNodeWithData(t, canvas.ID, node2, "default", &execution2.ID, node2Data)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithPreviousExecution(&execution2.ID).
		WithRootEvent(&rootEvent.ID).
		WithInput(map[string]any{node2: node2Data})

	configuration := map[string]any{
		"immediate": "{{ previous().result }}",
		"upstream":  "{{ previous(2).result }}",
		"root":      "{{ previous(3).root }}",
	}

	result, err := builder.Build(configuration)
	require.NoError(t, err)
	assert.Equal(t, "second", result["immediate"])
	assert.Equal(t, "first", result["upstream"])
	assert.Equal(t, "data", result["root"])
}

func Test_NodeConfigurationBuilder_WorkflowLevelNode_Previous_MultipleInputs(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "node-1", Type: models.NodeTypeComponent},
			{NodeID: "node-2", Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithInput(map[string]any{
			"node-1": map[string]any{"value": "one"},
			"node-2": map[string]any{"value": "two"},
		})

	_, err := builder.Build(map[string]any{"field": "{{ previous().value }}"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "previous() is not available when multiple inputs are present")
}

func Test_NodeConfigurationBuilder_WorkflowLevelNode_Chain_NoPreviousExecution(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "node-1", Name: "node-1", Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithInput(map[string]any{})

	configuration := map[string]any{
		"field": "{{ $[\"node-1\"].data }}",
	}

	_, err := builder.Build(configuration)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in execution chain")
}

func Test_NodeConfigurationBuilder_WorkflowLevelNode_Chain_NodeNotInChain(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	node1 := "node-1"
	node2 := "node-2"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: node1, Name: node1, Type: models.NodeTypeComponent},
			{NodeID: node2, Name: node2, Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, node1, "default", nil)
	execution1 := support.CreateCanvasNodeExecution(t, canvas.ID, node1, rootEvent.ID, rootEvent.ID)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithPreviousExecution(&execution1.ID).
		WithInput(map[string]any{})

	configuration := map[string]any{
		"field": "{{ $[\"nonexistent-node\"].data }}",
	}

	_, err := builder.Build(configuration)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in execution chain")
}

func Test_NodeConfigurationBuilder_WorkflowLevelNode_Chain_KnownNodeWithoutExecutionResolvesNil(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	node1 := "node-1"
	node2 := "node-2"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: node1, Name: node1, Type: models.NodeTypeComponent},
			{NodeID: node2, Name: node2, Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, node1, "default", nil)
	execution1 := support.CreateCanvasNodeExecution(t, canvas.ID, node1, rootEvent.ID, rootEvent.ID)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithPreviousExecution(&execution1.ID).
		WithInput(map[string]any{})

	result, err := builder.ResolveExpression(`$["node-2"] == nil ? "missing" : "present"`)

	require.NoError(t, err)
	assert.Equal(t, "missing", result)
}

func Test_NodeConfigurationBuilder_ComplexNesting(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "node-1", Name: "node-1", Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	rootEventData := map[string]any{
		"user": map[string]any{
			"name":  "John",
			"email": "john@example.com",
		},
		"items":  []any{"apple", "banana", "cherry"},
		"prefix": "user",
	}
	rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "node-1", "default", nil, rootEventData)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithRootEvent(&rootEvent.ID).
		WithInput(map[string]any{"node-1": rootEventData})

	configuration := map[string]any{
		"nested": map[string]any{
			"user_name":  "{{ $[\"node-1\"].user.name }}",
			"user_email": "{{ $[\"node-1\"].user.email }}",
		},
		"array_field": []any{
			"{{ $[\"node-1\"].prefix }}",
			"{{ $[\"node-1\"].user.name }}",
			map[string]any{
				"inner": "{{ $[\"node-1\"].user.email }}",
			},
		},
	}

	result, err := builder.Build(configuration)
	require.NoError(t, err)

	// Check nested map
	nested := result["nested"].(map[string]any)
	assert.Equal(t, "John", nested["user_name"])
	assert.Equal(t, "john@example.com", nested["user_email"])

	// Check array with mixed types
	array := result["array_field"].([]any)
	assert.Equal(t, "user", array[0])
	assert.Equal(t, "John", array[1])
	innerMap := array[2].(map[string]any)
	assert.Equal(t, "john@example.com", innerMap["inner"])
}

func Test_NodeConfigurationBuilder_InputVariable(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "node-1", Name: "node-1", Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	inputData := map[string]any{
		"name":   "Alice",
		"age":    25,
		"active": true,
		"metadata": map[string]any{
			"role": "admin",
		},
	}

	rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "node-1", "default", nil, inputData)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithRootEvent(&rootEvent.ID).
		WithInput(map[string]any{"node-1": inputData})

	configuration := map[string]any{
		"name":     "{{ $[\"node-1\"].name }}",
		"age":      "{{ $[\"node-1\"].age }}",
		"active":   "{{ $[\"node-1\"].active }}",
		"role":     "{{ $[\"node-1\"].metadata.role }}",
		"combined": "User {{ $[\"node-1\"].name }} is {{ $[\"node-1\"].age }} years old",
	}

	result, err := builder.Build(configuration)
	require.NoError(t, err)
	assert.Equal(t, "Alice", result["name"])
	assert.Equal(t, "25", result["age"])
	assert.Equal(t, "true", result["active"])
	assert.Equal(t, "admin", result["role"])
	assert.Equal(t, "User Alice is 25 years old", result["combined"])
}

func Test_NodeConfigurationBuilder_NoExpression(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "node-1", Name: "node-1", Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithRootEvent(&rootEvent.ID).
		WithInput(map[string]any{})

	configuration := map[string]any{
		"plain_string": "hello world",
		"number":       42,
		"boolean":      true,
		"null_value":   nil,
	}

	result, err := builder.Build(configuration)
	require.NoError(t, err)
	assert.Equal(t, "hello world", result["plain_string"])
	assert.Equal(t, 42, result["number"])
	assert.Equal(t, true, result["boolean"])
	assert.Nil(t, result["null_value"])
}

func Test_NodeConfigurationBuilder_MemoryFind(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "node-1", Name: "node-1", Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	err := models.AddCanvasMemory(canvas.ID, "machines", map[string]any{"sandbox_id": "12121"})
	require.NoError(t, err)

	err = models.AddCanvasMemory(canvas.ID, "machines", map[string]any{"sandbox_id": "34343"})
	require.NoError(t, err)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).WithInput(map[string]any{})

	result, err := builder.ResolveExpression(`memory.find("machines", {"sandbox_id": "12121"})`)
	require.NoError(t, err)
	assert.Equal(t, []any{map[string]any{"sandbox_id": "12121"}}, result)
}

func Test_NodeConfigurationBuilder_MemoryFindFirst(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "node-1", Name: "node-1", Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	err := models.AddCanvasMemory(canvas.ID, "machines", map[string]any{"sandbox_id": "12121", "creator": "igor"})
	require.NoError(t, err)
	err = models.AddCanvasMemory(canvas.ID, "machines", map[string]any{"sandbox_id": "34343", "creator": "alex"})
	require.NoError(t, err)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).WithInput(map[string]any{})

	result, err := builder.ResolveExpression(`memory.findFirst("machines", {"sandbox_id": "12121"})`)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"sandbox_id": "12121", "creator": "igor"}, result)

	matched, err := builder.ResolveExpression(`memory.findFirst("machines", {"creator": "igor"}).sandbox_id`)
	require.NoError(t, err)
	assert.Equal(t, "12121", matched)

	missing, err := builder.ResolveExpression(`memory.findFirst("missing", {"sandbox_id": "does-not-exist"})`)
	require.NoError(t, err)
	assert.Nil(t, missing)
}

func Test_NodeConfigurationBuilder_Config_ViaNodeReference(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNode := "trigger-1"
	componentNode := "component-1"
	targetNode := "target-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Name:   triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: componentNode,
				Name:   componentNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: targetNode,
				Name:   targetNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: componentNode, Channel: "default"},
			{SourceID: componentNode, TargetID: targetNode, Channel: "default"},
		},
	)

	rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNode, "default", nil, map[string]any{"user": "alice"})

	componentConfig := map[string]any{"url": "https://example.com", "timeout": 30}
	execution1 := support.CreateCanvasNodeExecution(t, canvas.ID, componentNode, rootEvent.ID, rootEvent.ID)
	require.NoError(t, database.Conn().Model(execution1).Update("configuration", datatypes.NewJSONType(componentConfig)).Error)

	outputData := map[string]any{"status": "ok"}
	event1 := support.EmitCanvasEventForNodeWithData(t, canvas.ID, componentNode, "default", &execution1.ID, outputData)

	execution2 := support.CreateNextNodeExecution(t, canvas.ID, targetNode, rootEvent.ID, event1.ID, &execution1.ID)
	_ = execution2

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithPreviousExecution(&execution1.ID).
		WithRootEvent(&rootEvent.ID).
		WithInput(map[string]any{componentNode: outputData})

	result, err := builder.Build(map[string]any{
		"upstream_url": "{{ $[\"" + componentNode + "\"].config.url }}",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", result["upstream_url"])
}

func Test_NodeConfigurationBuilder_Config_ViaPrevious(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNode := "trigger-1"
	node1 := "node-1"
	node2 := "node-2"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Name:   triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: node1,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: node2,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: node1, Channel: "default"},
			{SourceID: node1, TargetID: node2, Channel: "default"},
		},
	)

	rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNode, "default", nil, map[string]any{"root": "data"})

	node1Config := map[string]any{"method": "POST", "endpoint": "/api/deploy"}
	execution1 := support.CreateCanvasNodeExecution(t, canvas.ID, node1, rootEvent.ID, rootEvent.ID)
	require.NoError(t, database.Conn().Model(execution1).Update("configuration", datatypes.NewJSONType(node1Config)).Error)

	node1Data := map[string]any{"result": "deployed"}
	event1 := support.EmitCanvasEventForNodeWithData(t, canvas.ID, node1, "default", &execution1.ID, node1Data)

	_ = support.CreateNextNodeExecution(t, canvas.ID, node2, rootEvent.ID, event1.ID, &execution1.ID)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithPreviousExecution(&execution1.ID).
		WithRootEvent(&rootEvent.ID).
		WithInput(map[string]any{node1: node1Data})

	result, err := builder.Build(map[string]any{
		"prev_method":   "{{ previous().config.method }}",
		"prev_endpoint": "{{ previous().config.endpoint }}",
	})
	require.NoError(t, err)
	assert.Equal(t, "POST", result["prev_method"])
	assert.Equal(t, "/api/deploy", result["prev_endpoint"])
}

func Test_NodeConfigurationBuilder_Config_ExistingExpressionsStillWork(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNode := "trigger-1"
	componentNode := "component-1"
	targetNode := "target-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Name:   triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: componentNode,
				Name:   componentNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: targetNode,
				Name:   targetNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: componentNode, Channel: "default"},
			{SourceID: componentNode, TargetID: targetNode, Channel: "default"},
		},
	)

	rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNode, "default", nil, map[string]any{"user": "alice"})

	componentConfig := map[string]any{"url": "https://example.com"}
	execution1 := support.CreateCanvasNodeExecution(t, canvas.ID, componentNode, rootEvent.ID, rootEvent.ID)
	require.NoError(t, database.Conn().Model(execution1).Update("configuration", datatypes.NewJSONType(componentConfig)).Error)

	outputData := map[string]any{"status": "ok", "code": 200}
	event1 := support.EmitCanvasEventForNodeWithData(t, canvas.ID, componentNode, "default", &execution1.ID, outputData)

	_ = support.CreateNextNodeExecution(t, canvas.ID, targetNode, rootEvent.ID, event1.ID, &execution1.ID)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithPreviousExecution(&execution1.ID).
		WithRootEvent(&rootEvent.ID).
		WithInput(map[string]any{componentNode: outputData})

	// Existing expressions accessing output data should still work
	result, err := builder.Build(map[string]any{
		"status":      "{{ $[\"" + componentNode + "\"].status }}",
		"root_user":   "{{ root().user }}",
		"prev_status": "{{ previous().status }}",
	})
	require.NoError(t, err)
	assert.Equal(t, "ok", result["status"])
	assert.Equal(t, "alice", result["root_user"])
	assert.Equal(t, "ok", result["prev_status"])
}

func Test_NodeConfigurationBuilder_Config_ViaPreviousDepthRootPath(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	node1 := "node-1"
	node2 := "node-2"
	node3 := "node-3"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: node1,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: node2,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: node3,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{
			{SourceID: node1, TargetID: node2, Channel: "default"},
			{SourceID: node2, TargetID: node3, Channel: "default"},
		},
	)

	rootEventData := map[string]any{"origin": "root"}
	bootstrapRoot := support.EmitCanvasEventForNodeWithData(t, canvas.ID, node1, "default", nil, map[string]any{"bootstrap": true})
	execution1 := support.CreateCanvasNodeExecution(t, canvas.ID, node1, bootstrapRoot.ID, bootstrapRoot.ID)
	node1Config := map[string]any{"region": "us-east-1"}
	require.NoError(t, database.Conn().Model(execution1).Update("configuration", datatypes.NewJSONType(node1Config)).Error)
	rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, node1, "default", &execution1.ID, rootEventData)
	require.NoError(t, database.Conn().Model(execution1).Updates(map[string]any{"root_event_id": rootEvent.ID, "event_id": rootEvent.ID}).Error)

	node2Data := map[string]any{"value": "node2"}
	execution2 := support.CreateNextNodeExecution(t, canvas.ID, node2, rootEvent.ID, rootEvent.ID, &execution1.ID)
	event2 := support.EmitCanvasEventForNodeWithData(t, canvas.ID, node2, "default", &execution2.ID, node2Data)

	execution3 := support.CreateNextNodeExecution(t, canvas.ID, node3, rootEvent.ID, event2.ID, &execution2.ID)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithPreviousExecution(&execution3.ID).
		WithRootEvent(&rootEvent.ID).
		WithInput(map[string]any{node3: map[string]any{"value": "node3"}})

	result, err := builder.Build(map[string]any{
		"root_config_region": "{{ previous(3).config.region }}",
	})
	require.NoError(t, err)
	assert.Equal(t, "us-east-1", result["root_config_region"])
}

func Test_NodeConfigurationBuilder_Config_DoesNotMutateInputPayload(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNode := "trigger-1"
	componentNode := "component-1"
	targetNode := "target-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Name:   triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: componentNode,
				Name:   componentNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: targetNode,
				Name:   targetNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: componentNode, Channel: "default"},
			{SourceID: componentNode, TargetID: targetNode, Channel: "default"},
		},
	)

	rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNode, "default", nil, map[string]any{"user": "alice"})

	componentConfig := map[string]any{"url": "https://example.com"}
	execution1 := support.CreateCanvasNodeExecution(t, canvas.ID, componentNode, rootEvent.ID, rootEvent.ID)
	require.NoError(t, database.Conn().Model(execution1).Update("configuration", datatypes.NewJSONType(componentConfig)).Error)

	inputPayload := map[string]any{"status": "ok"}
	input := map[string]any{componentNode: inputPayload}

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithPreviousExecution(&execution1.ID).
		WithRootEvent(&rootEvent.ID).
		WithInput(input)

	result, err := builder.Build(map[string]any{
		"upstream_url": "{{ $[\"" + componentNode + "\"].config.url }}",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", result["upstream_url"])
	assert.Equal(t, map[string]any{"status": "ok"}, inputPayload)
}

func Test_NodeConfigurationBuilder_Config_DoesNotOverwriteExistingConfigKey(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNode := "trigger-1"
	componentNode := "component-1"
	targetNode := "target-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Name:   triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: componentNode,
				Name:   componentNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: targetNode,
				Name:   targetNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: componentNode, Channel: "default"},
			{SourceID: componentNode, TargetID: targetNode, Channel: "default"},
		},
	)

	rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNode, "default", nil, map[string]any{"user": "alice"})

	componentConfig := map[string]any{"url": "https://internal.example.com"}
	execution1 := support.CreateCanvasNodeExecution(t, canvas.ID, componentNode, rootEvent.ID, rootEvent.ID)
	require.NoError(t, database.Conn().Model(execution1).Update("configuration", datatypes.NewJSONType(componentConfig)).Error)

	outputWithConfig := map[string]any{
		"status": "ok",
		"config": map[string]any{"api_url": "https://api.example.com", "version": "v2"},
	}
	event1 := support.EmitCanvasEventForNodeWithData(t, canvas.ID, componentNode, "default", &execution1.ID, outputWithConfig)

	_ = support.CreateNextNodeExecution(t, canvas.ID, targetNode, rootEvent.ID, event1.ID, &execution1.ID)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithPreviousExecution(&execution1.ID).
		WithRootEvent(&rootEvent.ID).
		WithInput(map[string]any{componentNode: outputWithConfig})

	result, err := builder.Build(map[string]any{
		"api_url":       "{{ $[\"" + componentNode + "\"].config.api_url }}",
		"prev_api_url":  "{{ previous().config.api_url }}",
		"output_status": "{{ $[\"" + componentNode + "\"].status }}",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com", result["api_url"])
	assert.Equal(t, "https://api.example.com", result["prev_api_url"])
	assert.Equal(t, "ok", result["output_status"])
}

func Test_NodeConfigurationBuilder_ForEachBranchPayload(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNode := "start"
	forEachNode := "forEach"
	waitNode := "wait"
	displayNode := "display"

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: triggerNode,
				Name:   triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: forEachNode,
				Name:   forEachNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "forEach"}}),
			},
			{
				NodeID: waitNode,
				Name:   waitNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "wait"}}),
			},
			{
				NodeID: displayNode,
				Name:   displayNode,
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "display"}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: forEachNode, Channel: "default"},
			{SourceID: forEachNode, TargetID: waitNode, Channel: "item"},
			{SourceID: waitNode, TargetID: displayNode, Channel: "default"},
		},
	)

	rootEvent := support.EmitCanvasEventForNodeWithData(
		t,
		canvas.ID,
		triggerNode,
		"default",
		nil,
		map[string]any{"items": []any{"a", "b", "c"}},
	)

	forEachExecution := support.CreateCanvasNodeExecution(t, canvas.ID, forEachNode, rootEvent.ID, rootEvent.ID)

	now := time.Now()
	emitItem := func(item string, index int) *models.CanvasEvent {
		return support.EmitCanvasEventForNodeWithData(
			t,
			canvas.ID,
			forEachNode,
			"item",
			&forEachExecution.ID,
			map[string]any{"item": item, "index": index, "totalCount": 3},
		)
	}

	_ = emitItem("a", 0)
	branchEvent := emitItem("b", 1)
	_ = emitItem("c", 2)

	require.NoError(t, database.Conn().Model(&models.CanvasEvent{}).
		Where("execution_id = ?", forEachExecution.ID).
		Update("created_at", now).Error)

	waitInput := map[string]any{"item": "b", "index": 1, "totalCount": 3}
	waitExecution := support.CreateNextNodeExecution(t, canvas.ID, waitNode, rootEvent.ID, branchEvent.ID, &forEachExecution.ID)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithPreviousExecution(&waitExecution.ID).
		WithRootEvent(&rootEvent.ID).
		WithInput(map[string]any{waitNode: waitInput})

	result, err := builder.Build(map[string]any{
		"item": "{{ $[\"" + forEachNode + "\"].item }}",
	})
	require.NoError(t, err)
	assert.Equal(t, "b", result["item"])
}
