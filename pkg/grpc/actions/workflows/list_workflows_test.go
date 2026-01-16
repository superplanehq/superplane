package workflows

import (
	"context"
	"slices"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__ListWorkflows__ReturnsEmptyListWhenNoWorkflowsExist(t *testing.T) {
	r := support.Setup(t)

	response, err := ListWorkflows(context.Background(), r.Registry, r.Organization.ID.String(), false)
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Empty(t, response.Workflows)
}

func Test__ListWorkflows__ReturnsAllWorkflowsForAnOrganization(t *testing.T) {
	r := support.Setup(t)

	//
	// Create multiple workflows
	//
	workflow1, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	workflow2, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: "node-2",
				Name:   "Node 2",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// List workflows
	//
	response, err := ListWorkflows(context.Background(), r.Registry, r.Organization.ID.String(), false)
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Len(t, response.Workflows, 2)

	//
	// Verify both workflows are returned
	//
	workflowIDs := []string{response.Workflows[0].Metadata.Id, response.Workflows[1].Metadata.Id}
	assert.Contains(t, workflowIDs, workflow1.ID.String())
	assert.Contains(t, workflowIDs, workflow2.ID.String())

	//
	// List of workflows returned is ordered by name
	//
	workflowNames := make([]string, len(response.Workflows))
	for i, wf := range response.Workflows {
		workflowNames[i] = wf.Metadata.Name
	}

	assert.True(t, sort.StringsAreSorted(workflowNames), "workflows should be sorted by name in ascending order")
}

func Test__ListWorkflows__DoesNotReturnWorkflowsFromOtherOrganizations(t *testing.T) {
	r := support.Setup(t)

	//
	// Create workflow in the test organization
	//
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// Create another organization and workflow
	//
	otherOrg := support.CreateOrganization(t, r, r.User)
	otherWorkflow, _ := support.CreateWorkflow(
		t,
		otherOrg.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: "other-node",
				Name:   "Other Node",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// List workflows for original organization
	//
	response, err := ListWorkflows(context.Background(), r.Registry, r.Organization.ID.String(), false)
	require.NoError(t, err)
	require.NotNil(t, response)

	//
	// Should only return the workflow from the original organization
	//
	assert.Len(t, response.Workflows, 1)
	assert.Equal(t, workflow.ID.String(), response.Workflows[0].Metadata.Id)
	assert.Equal(t, r.Organization.ID.String(), response.Workflows[0].Metadata.OrganizationId)
	assert.NotEqual(t, otherWorkflow.ID.String(), response.Workflows[0].Metadata.Id)
}

func Test__ListWorkflows__ReturnsWorkflowsWithoutStatusInformation(t *testing.T) {
	r := support.Setup(t)

	//
	// Create workflow with nodes
	//
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	//
	// Create executions and queue items
	//
	rootEvent := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)
	event := support.EmitWorkflowEventForNode(t, workflow.ID, "node-1", "default", nil)
	support.CreateWorkflowNodeExecution(t, workflow.ID, "node-1", rootEvent.ID, event.ID, nil)
	support.CreateWorkflowQueueItem(t, workflow.ID, "node-1", rootEvent.ID, event.ID)

	//
	// List workflows
	//
	response, err := ListWorkflows(context.Background(), r.Registry, r.Organization.ID.String(), false)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Workflows, 1)

	//
	// Verify status is nil (not loaded)
	//
	assert.Nil(t, response.Workflows[0].Status)
}

func Test__ListWorkflows__ReturnsWorkflowsWithMetadataAndSpec(t *testing.T) {
	r := support.Setup(t)

	//
	// Create workflow with nodes and edges
	//
	workflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: "node-1",
				Name:   "First Node",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
			{
				NodeID: "node-2",
				Name:   "Second Node",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{
			{
				SourceID: "node-1",
				TargetID: "node-2",
				Channel:  "default",
			},
		},
	)

	//
	// List workflows
	//
	response, err := ListWorkflows(context.Background(), r.Registry, r.Organization.ID.String(), false)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Workflows, 1)

	listedWorkflow := response.Workflows[0]

	//
	// Verify metadata is present
	//
	require.NotNil(t, listedWorkflow.Metadata)
	assert.Equal(t, workflow.ID.String(), listedWorkflow.Metadata.Id)
	assert.Equal(t, workflow.OrganizationID.String(), listedWorkflow.Metadata.OrganizationId)
	assert.Equal(t, workflow.Name, listedWorkflow.Metadata.Name)
	assert.Equal(t, workflow.Description, listedWorkflow.Metadata.Description)
	assert.NotNil(t, listedWorkflow.Metadata.CreatedAt)
	assert.NotNil(t, listedWorkflow.Metadata.UpdatedAt)
	assert.NotNil(t, listedWorkflow.Metadata.CreatedBy)

	//
	// Verify spec is present with nodes and edges
	//
	require.NotNil(t, listedWorkflow.Spec)
	assert.Len(t, listedWorkflow.Spec.Nodes, 2)
	assert.Equal(t, "node-1", listedWorkflow.Spec.Nodes[0].Id)
	assert.Equal(t, "First Node", listedWorkflow.Spec.Nodes[0].Name)
	assert.Equal(t, "node-2", listedWorkflow.Spec.Nodes[1].Id)
	assert.Equal(t, "Second Node", listedWorkflow.Spec.Nodes[1].Name)

	assert.Len(t, listedWorkflow.Spec.Edges, 1)
	assert.Equal(t, "node-1", listedWorkflow.Spec.Edges[0].SourceId)
	assert.Equal(t, "node-2", listedWorkflow.Spec.Edges[0].TargetId)
	assert.Equal(t, "default", listedWorkflow.Spec.Edges[0].Channel)

	//
	// Verify status is NOT present
	//
	assert.Nil(t, listedWorkflow.Status)
}

func Test__ListWorkflows__DoesNotReturnSoftDeletedWorkflowsWhenIncludingTemplates(t *testing.T) {
	r := support.Setup(t)

	activeWorkflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: "node-1",
				Name:   "Node 1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	deletedWorkflow, _ := support.CreateWorkflow(
		t,
		r.Organization.ID,
		r.User,
		[]models.WorkflowNode{
			{
				NodeID: "node-2",
				Name:   "Node 2",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	require.NoError(t, deletedWorkflow.SoftDelete())

	now := time.Now()
	templateWorkflow := &models.Workflow{
		ID:             uuid.New(),
		OrganizationID: models.TemplateOrganizationID,
		IsTemplate:     true,
		Name:           support.RandomName("template"),
		Description:    "Template workflow",
		Nodes:          datatypes.NewJSONSlice([]models.Node{}),
		Edges:          datatypes.NewJSONSlice([]models.Edge{}),
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	require.NoError(t, database.Conn().Create(templateWorkflow).Error)

	response, err := ListWorkflows(context.Background(), r.Registry, r.Organization.ID.String(), true)
	require.NoError(t, err)
	require.NotNil(t, response)

	workflowIDs := make([]string, len(response.Workflows))
	for i, wf := range response.Workflows {
		workflowIDs[i] = wf.Metadata.Id
	}

	assert.True(t, slices.Contains(workflowIDs, activeWorkflow.ID.String()))
	assert.True(t, slices.Contains(workflowIDs, templateWorkflow.ID.String()))
	assert.False(t, slices.Contains(workflowIDs, deletedWorkflow.ID.String()))
}
