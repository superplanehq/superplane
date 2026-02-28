package contexts

import (
	"testing"

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

	execution1 := support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", rootEvent.ID, rootEvent.ID, nil)
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
		nil,
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

	execution1 := support.CreateCanvasNodeExecution(t, canvas.ID, action1Node, rootEvent.ID, rootEvent.ID, nil)
	action1Data := map[string]any{"value": "from-action-1"}
	support.EmitCanvasEventForNodeWithData(t, canvas.ID, action1Node, "default", &execution1.ID, action1Data)

	execution2 := support.CreateCanvasNodeExecution(t, canvas.ID, action2Node, rootEvent.ID, rootEvent.ID, nil)
	action2Data := map[string]any{"value": "from-action-2"}
	support.EmitCanvasEventForNodeWithData(t, canvas.ID, action2Node, "default", &execution2.ID, action2Data)

	execution3 := support.CreateCanvasNodeExecution(t, canvas.ID, action3Node, rootEvent.ID, rootEvent.ID, nil)
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
		nil,
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
	execution1 := support.CreateCanvasNodeExecution(t, canvas.ID, node1, rootEvent.ID, rootEvent.ID, nil)

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

func Test_NodeConfigurationBuilder_BlueprintLevelNode_Root(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	// Create a blueprint with a component node
	blueprint := support.CreateBlueprint(
		t,
		r.Organization.ID,
		[]models.Node{
			{
				ID:   "bp-node-1",
				Name: "bp-node-1",
				Type: models.NodeTypeComponent,
				Ref:  models.NodeRef{Component: &models.ComponentRef{Name: "noop"}},
			},
		},
		[]models.Edge{},
		[]models.BlueprintOutputChannel{
			{
				Name:              "default",
				NodeID:            "bp-node-1",
				NodeOutputChannel: "default",
			},
		},
	)

	// Create a workflow with a trigger and a blueprint node
	triggerNode := "trigger-1"
	blueprintNode := "blueprint-1"
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
				NodeID: blueprintNode,
				Name:   blueprintNode,
				Type:   models.NodeTypeBlueprint,
				Ref:    datatypes.NewJSONType(models.NodeRef{Blueprint: &models.BlueprintRef{ID: blueprint.ID.String()}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: blueprintNode, Channel: "default"},
		},
	)

	// Create a root event with test data
	rootEventData := map[string]any{
		"username": "alice",
		"email":    "alice@example.com",
		"age":      30,
	}
	rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNode, "default", nil, rootEventData)

	// Create a blueprint execution
	blueprintExecution := support.CreateCanvasNodeExecution(
		t,
		canvas.ID,
		blueprintNode,
		rootEvent.ID,
		rootEvent.ID,
		nil,
	)

	// Get the blueprint node for testing
	blueprintCanvasNode, err := models.FindCanvasNode(database.Conn(), canvas.ID, blueprintNode)
	require.NoError(t, err)

	// Build configuration for a node inside the blueprint
	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		ForBlueprintNode(blueprintCanvasNode).
		WithRootEvent(&rootEvent.ID).
		WithPreviousExecution(&blueprintExecution.ID).
		WithInput(map[string]any{triggerNode: rootEventData})

	configuration := map[string]any{
		"username": "{{ $[\"" + triggerNode + "\"].username }}",
		"email":    "{{ $[\"" + triggerNode + "\"].email }}",
		"age":      "{{ $[\"" + triggerNode + "\"].age }}",
	}

	result, err := builder.Build(configuration)
	require.NoError(t, err)
	assert.Equal(t, "alice", result["username"])
	assert.Equal(t, "alice@example.com", result["email"])
	assert.Equal(t, "30", result["age"])
}

func Test_NodeConfigurationBuilder_BlueprintLevelNode_Chain(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	// Create a blueprint with two sequential nodes
	blueprint := support.CreateBlueprint(
		t,
		r.Organization.ID,
		[]models.Node{
			{
				ID:   "bp-node-1",
				Name: "bp-node-1",
				Type: models.NodeTypeComponent,
				Ref:  models.NodeRef{Component: &models.ComponentRef{Name: "noop"}},
			},
			{
				ID:   "bp-node-2",
				Name: "bp-node-2",
				Type: models.NodeTypeComponent,
				Ref:  models.NodeRef{Component: &models.ComponentRef{Name: "noop"}},
			},
		},
		[]models.Edge{
			{SourceID: "bp-node-1", TargetID: "bp-node-2", Channel: "default"},
		},
		[]models.BlueprintOutputChannel{
			{
				Name:              "default",
				NodeID:            "bp-node-2",
				NodeOutputChannel: "default",
			},
		},
	)

	// Create a workflow with a trigger and a blueprint node
	triggerNode := "trigger-1"
	blueprintNode := "blueprint-1"
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
				NodeID: blueprintNode,
				Name:   blueprintNode,
				Type:   models.NodeTypeBlueprint,
				Ref:    datatypes.NewJSONType(models.NodeRef{Blueprint: &models.BlueprintRef{ID: blueprint.ID.String()}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: blueprintNode, Channel: "default"},
		},
	)

	rootEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, triggerNode, "default", nil, map[string]any{"root": "data"})

	//
	// Create parent blueprint execution
	//
	blueprintExecution := support.CreateCanvasNodeExecution(
		t,
		canvas.ID,
		blueprintNode,
		rootEvent.ID,
		rootEvent.ID,
		nil,
	)

	//
	// Create first blueprint node child execution with outputs
	//
	bpNode1ID := blueprintNode + ":bp-node-1"
	bpNode2ID := blueprintNode + ":bp-node-2"
	bpNode1Execution := support.CreateCanvasNodeExecution(
		t,
		canvas.ID,
		bpNode1ID,
		rootEvent.ID,
		rootEvent.ID,
		&blueprintExecution.ID,
	)

	bpNode1Data := map[string]any{
		"processed": true,
		"value":     "from-first-node",
	}
	event1 := support.EmitCanvasEventForNodeWithData(t, canvas.ID, bpNode1ID, "default", &bpNode1Execution.ID, bpNode1Data)

	//
	// Create second blueprint node child execution
	//
	bpNode2Execution := support.CreateCanvasNodeExecution(
		t,
		canvas.ID,
		bpNode2ID,
		rootEvent.ID,
		event1.ID,
		&blueprintExecution.ID,
	)
	bpNode2Execution.PreviousExecutionID = &bpNode1Execution.ID
	require.NoError(t, database.Conn().Save(&bpNode2Execution).Error)

	//
	// Test message chain access from bp-node-2 perspective
	//
	blueprintCanvasNode, err := models.FindCanvasNode(database.Conn(), canvas.ID, blueprintNode)
	require.NoError(t, err)

	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		ForBlueprintNode(blueprintCanvasNode).
		WithPreviousExecution(&bpNode2Execution.ID).
		WithRootEvent(&rootEvent.ID).
		WithInput(map[string]any{bpNode1ID: bpNode1Data})

	configuration := map[string]any{
		"processed": "{{ $[\"bp-node-1\"].processed }}",
		"value":     "{{ $[\"bp-node-1\"].value }}",
	}

	result, err := builder.Build(configuration)
	require.NoError(t, err)
	assert.Equal(t, "true", result["processed"])
	assert.Equal(t, "from-first-node", result["value"])
}

func Test_NodeConfigurationBuilder_BlueprintLevelNode_Config(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	// Create a blueprint
	blueprint := support.CreateBlueprint(
		t,
		r.Organization.ID,
		[]models.Node{
			{
				ID:   "bp-node-1",
				Name: "bp-node-1",
				Type: models.NodeTypeComponent,
				Ref:  models.NodeRef{Component: &models.ComponentRef{Name: "noop"}},
			},
		},
		[]models.Edge{},
		[]models.BlueprintOutputChannel{
			{
				Name:              "default",
				NodeID:            "bp-node-1",
				NodeOutputChannel: "default",
			},
		},
	)

	// Create a canvas with a blueprint node that has configuration
	blueprintNode := "blueprint-1"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: blueprintNode,
				Name:   blueprintNode,
				Type:   models.NodeTypeBlueprint,
				Ref:    datatypes.NewJSONType(models.NodeRef{Blueprint: &models.BlueprintRef{ID: blueprint.ID.String()}}),
				Configuration: datatypes.NewJSONType(map[string]any{
					"api_key":     "secret-key-123",
					"endpoint":    "https://api.example.com",
					"timeout":     30,
					"retry_count": 3,
					"nested": map[string]any{
						"value": "nested-data",
					},
				}),
			},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, blueprintNode, "default", nil)

	// Get the blueprint node
	blueprintCanvasNode, err := models.FindCanvasNode(database.Conn(), canvas.ID, blueprintNode)
	require.NoError(t, err)

	// Build configuration accessing parent blueprint node config
	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		ForBlueprintNode(blueprintCanvasNode).
		WithRootEvent(&rootEvent.ID).
		WithInput(map[string]any{blueprintNode: map[string]any{"input": "data"}})

	configuration := map[string]any{
		"key":      "{{ config.api_key }}",
		"url":      "{{ config.endpoint }}",
		"timeout":  "{{ config.timeout }}",
		"retries":  "{{ config.retry_count }}",
		"nested":   "{{ config.nested.value }}",
		"combined": "API: {{ config.endpoint }}, Key: {{ config.api_key }}",
	}

	result, err := builder.Build(configuration)
	require.NoError(t, err)
	assert.Equal(t, "secret-key-123", result["key"])
	assert.Equal(t, "https://api.example.com", result["url"])
	assert.Equal(t, "30", result["timeout"])
	assert.Equal(t, "3", result["retries"])
	assert.Equal(t, "nested-data", result["nested"])
	assert.Equal(t, "API: https://api.example.com, Key: secret-key-123", result["combined"])
}

func Test_NodeConfigurationBuilder_BlueprintLevelNode_Config_NotAvailableForWorkflowNodes(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Name:   "node-1",
				Type:   models.NodeTypeComponent,
				Configuration: datatypes.NewJSONType(map[string]any{
					"field": "value",
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// Build without ForBlueprintNode - this is a canvas-level node
	//
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	builder := NewNodeConfigurationBuilder(database.Conn(), canvas.ID).
		WithRootEvent(&rootEvent.ID).
		WithInput(map[string]any{"node-1": rootEvent.Data.Data()})

	_, err := builder.Build(map[string]any{"field": "{{ config.field }}"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error resolving field field: unknown name config")
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

func Test_NodeConfigurationBuilder_DisallowExpression_ListItems(t *testing.T) {
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
		WithInput(map[string]any{
			"node-1": map[string]any{
				"allowed":    "resolved",
				"disallowed": "raw",
			},
		}).
		WithConfigurationFields([]configuration.Field{
			{
				Name: "items",
				Type: configuration.FieldTypeList,
				TypeOptions: &configuration.TypeOptions{
					List: &configuration.ListTypeOptions{
						ItemDefinition: &configuration.ListItemDefinition{
							Type: configuration.FieldTypeObject,
							Schema: []configuration.Field{
								{Name: "allowed", Type: configuration.FieldTypeString},
								{Name: "disallowed", Type: configuration.FieldTypeString, DisallowExpression: true},
							},
						},
					},
				},
			},
		})

	configuration := map[string]any{
		"items": []any{
			map[string]any{
				"allowed":    "{{ $[\"node-1\"].allowed }}",
				"disallowed": "{{ $[\"node-1\"].disallowed }}",
			},
		},
	}

	result, err := builder.Build(configuration)
	require.NoError(t, err)

	items := result["items"].([]any)
	item := items[0].(map[string]any)
	assert.Equal(t, "resolved", item["allowed"])
	assert.Equal(t, "{{ $[\"node-1\"].disallowed }}", item["disallowed"])
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

	result, err := builder.resolveExpression(`memory.find("machines", {"sandbox_id": "12121"})`)
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

	result, err := builder.resolveExpression(`memory.findFirst("machines", {"sandbox_id": "12121"})`)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"sandbox_id": "12121", "creator": "igor"}, result)

	matched, err := builder.resolveExpression(`memory.findFirst("machines", {"creator": "igor"}).sandbox_id`)
	require.NoError(t, err)
	assert.Equal(t, "12121", matched)

	missing, err := builder.resolveExpression(`memory.findFirst("missing", {"sandbox_id": "does-not-exist"})`)
	require.NoError(t, err)
	assert.Nil(t, missing)
}
