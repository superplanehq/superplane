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
	// Create a simple workflow with a trigger and a component node
	//
	triggerNode := "trigger-1"
	componentNode := "component-1"
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: componentNode,
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
	rootEvent := support.EmitWorkflowEventForNodeWithData(t, workflow.ID, triggerNode, "default", nil, rootEventData)

	//
	// Use message chain access to get information from the root event
	//
	builder := NewNodeConfigurationBuilder(database.Conn(), workflow.ID).
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

func Test_NodeConfigurationBuilder_WorkflowLevelNode_Root_ByName(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	triggerNode := "trigger-1"
	componentNode := "component-1"
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: triggerNode,
				Name:   "triggerNode",
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: componentNode,
				Name:   "componentNode",
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
	rootEvent := support.EmitWorkflowEventForNodeWithData(t, workflow.ID, triggerNode, "default", nil, rootEventData)

	builder := NewNodeConfigurationBuilder(database.Conn(), workflow.ID).
		WithRootEvent(&rootEvent.ID).
		WithInput(map[string]any{triggerNode: rootEventData})

	configuration := map[string]any{
		"user":    "{{ $.triggerNode.user }}",
		"action":  "{{ $.triggerNode.action }}",
		"success": "{{ $.triggerNode.success }}",
		"count":   "{{ $.triggerNode.count }}",
	}

	result, err := builder.Build(configuration)
	require.NoError(t, err)
	assert.Equal(t, "john", result["user"])
	assert.Equal(t, "login", result["action"])
	assert.Equal(t, "true", result["success"])
	assert.Equal(t, "42", result["count"])
}

func Test_NodeConfigurationBuilder_WorkflowLevelNode_DuplicateNames(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: "node-1",
				Name:   "dup",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
			{
				NodeID: "node-2",
				Name:   "dup",
				Type:   models.NodeTypeComponent,
				Ref:    datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "noop"}}),
			},
		},
		[]models.Edge{},
	)

	builder := NewNodeConfigurationBuilder(database.Conn(), workflow.ID).
		WithInput(map[string]any{"node-1": map[string]any{"value": "ok"}})

	configuration := map[string]any{
		"value": "{{ $.dup.value }}",
	}

	_, err := builder.Build(configuration)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "node name dup is not unique")
}

func Test_NodeConfigurationBuilder_WorkflowLevelNode_Root_NoRootEvent(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{NodeID: "node-1", Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	builder := NewNodeConfigurationBuilder(database.Conn(), workflow.ID).
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
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
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

	//
	// Create root event
	//
	rootEvent := support.EmitWorkflowEventForNodeWithData(t, workflow.ID, triggerNode, "default", nil, map[string]any{"root": "data"})

	//
	// Simulate execution for first node
	//
	execution1 := support.CreateWorkflowNodeExecution(
		t,
		workflow.ID,
		node1,
		rootEvent.ID,
		rootEvent.ID,
		nil,
	)
	node1Data := map[string]any{
		"step":   1,
		"result": "first",
	}
	event1 := support.EmitWorkflowEventForNodeWithData(t, workflow.ID, node1, "default", &execution1.ID, node1Data)

	//
	// Simulate execution for second node
	//
	execution2 := support.CreateNextNodeExecution(
		t,
		workflow.ID,
		node2,
		rootEvent.ID,
		event1.ID,
		&execution1.ID,
	)

	node2Data := map[string]any{
		"step":   2,
		"result": "second",
	}
	support.EmitWorkflowEventForNodeWithData(t, workflow.ID, node2, "default", &execution2.ID, node2Data)

	//
	// Now test message chain access from node3 perspective
	//
	builder := NewNodeConfigurationBuilder(database.Conn(), workflow.ID).
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

func Test_NodeConfigurationBuilder_WorkflowLevelNode_Chain_NoPreviousExecution(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{NodeID: "node-1", Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	builder := NewNodeConfigurationBuilder(database.Conn(), workflow.ID).
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
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{NodeID: node1, Type: models.NodeTypeComponent},
			{NodeID: node2, Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitWorkflowEventForNode(t, workflow.ID, node1, "default", nil)
	execution1 := support.CreateWorkflowNodeExecution(t, workflow.ID, node1, rootEvent.ID, rootEvent.ID, nil)

	builder := NewNodeConfigurationBuilder(database.Conn(), workflow.ID).
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
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: blueprintNode,
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
	rootEvent := support.EmitWorkflowEventForNodeWithData(t, workflow.ID, triggerNode, "default", nil, rootEventData)

	// Create a blueprint execution
	blueprintExecution := support.CreateWorkflowNodeExecution(
		t,
		workflow.ID,
		blueprintNode,
		rootEvent.ID,
		rootEvent.ID,
		nil,
	)

	// Get the blueprint node for testing
	blueprintWorkflowNode, err := models.FindWorkflowNode(database.Conn(), workflow.ID, blueprintNode)
	require.NoError(t, err)

	// Build configuration for a node inside the blueprint
	builder := NewNodeConfigurationBuilder(database.Conn(), workflow.ID).
		ForBlueprintNode(blueprintWorkflowNode).
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
				Type: models.NodeTypeComponent,
				Ref:  models.NodeRef{Component: &models.ComponentRef{Name: "noop"}},
			},
			{
				ID:   "bp-node-2",
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
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: triggerNode,
				Type:   models.NodeTypeTrigger,
				Ref:    datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "start"}}),
			},
			{
				NodeID: blueprintNode,
				Type:   models.NodeTypeBlueprint,
				Ref:    datatypes.NewJSONType(models.NodeRef{Blueprint: &models.BlueprintRef{ID: blueprint.ID.String()}}),
			},
		},
		[]models.Edge{
			{SourceID: triggerNode, TargetID: blueprintNode, Channel: "default"},
		},
	)

	rootEvent := support.EmitWorkflowEventForNodeWithData(t, workflow.ID, triggerNode, "default", nil, map[string]any{"root": "data"})

	//
	// Create parent blueprint execution
	//
	blueprintExecution := support.CreateWorkflowNodeExecution(
		t,
		workflow.ID,
		blueprintNode,
		rootEvent.ID,
		rootEvent.ID,
		nil,
	)

	//
	// Create first blueprint node child execution with outputs
	//
	bpNode1Execution := support.CreateWorkflowNodeExecution(
		t,
		workflow.ID,
		"bp-node-1",
		rootEvent.ID,
		rootEvent.ID,
		&blueprintExecution.ID,
	)

	bpNode1Data := map[string]any{
		"processed": true,
		"value":     "from-first-node",
	}
	event1 := support.EmitWorkflowEventForNodeWithData(t, workflow.ID, "bp-node-1", "default", &bpNode1Execution.ID, bpNode1Data)

	//
	// Create second blueprint node child execution
	//
	bpNode2Execution := support.CreateWorkflowNodeExecution(
		t,
		workflow.ID,
		"bp-node-2",
		rootEvent.ID,
		event1.ID,
		&blueprintExecution.ID,
	)
	bpNode2Execution.PreviousExecutionID = &bpNode1Execution.ID
	require.NoError(t, database.Conn().Save(&bpNode2Execution).Error)

	//
	// Test message chain access from bp-node-2 perspective
	//
	blueprintWorkflowNode, err := models.FindWorkflowNode(database.Conn(), workflow.ID, blueprintNode)
	require.NoError(t, err)

	builder := NewNodeConfigurationBuilder(database.Conn(), workflow.ID).
		ForBlueprintNode(blueprintWorkflowNode).
		WithPreviousExecution(&bpNode2Execution.ID).
		WithRootEvent(&rootEvent.ID).
		WithInput(map[string]any{"bp-node-1": bpNode1Data})

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

	// Create a workflow with a blueprint node that has configuration
	blueprintNode := "blueprint-1"
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: blueprintNode,
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

	rootEvent := support.EmitWorkflowEventForNode(t, workflow.ID, blueprintNode, "default", nil)

	// Get the blueprint node
	blueprintWorkflowNode, err := models.FindWorkflowNode(database.Conn(), workflow.ID, blueprintNode)
	require.NoError(t, err)

	// Build configuration accessing parent blueprint node config
	builder := NewNodeConfigurationBuilder(database.Conn(), workflow.ID).
		ForBlueprintNode(blueprintWorkflowNode).
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

	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: "node-1",
				Type:   models.NodeTypeComponent,
				Configuration: datatypes.NewJSONType(map[string]any{
					"field": "value",
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// Build without ForBlueprintNode - this is a workflow-level node
	//
	rootEvent := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)
	builder := NewNodeConfigurationBuilder(database.Conn(), workflow.ID).
		WithRootEvent(&rootEvent.ID).
		WithInput(map[string]any{"node-1": rootEvent.Data.Data()})

	_, err := builder.Build(map[string]any{"field": "{{ config.field }}"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error resolving field field: unknown name config")
}

func Test_NodeConfigurationBuilder_ComplexNesting(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{NodeID: "node-1", Type: models.NodeTypeComponent},
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
	rootEvent := support.EmitWorkflowEventForNodeWithData(t, workflow.ID, "node-1", "default", nil, rootEventData)

	builder := NewNodeConfigurationBuilder(database.Conn(), workflow.ID).
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

	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{NodeID: "node-1", Type: models.NodeTypeComponent},
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

	rootEvent := support.EmitWorkflowEventForNodeWithData(t, workflow.ID, "node-1", "default", nil, inputData)

	builder := NewNodeConfigurationBuilder(database.Conn(), workflow.ID).
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

	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{NodeID: "node-1", Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	rootEvent := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)

	builder := NewNodeConfigurationBuilder(database.Conn(), workflow.ID).
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

	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{NodeID: "node-1", Type: models.NodeTypeComponent},
		},
		[]models.Edge{},
	)

	builder := NewNodeConfigurationBuilder(database.Conn(), workflow.ID).
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
