package canvases

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

func Test__ListCanvases__ReturnsEmptyListWhenNoCanvasesExist(t *testing.T) {
	r := support.Setup(t)

	response, err := ListCanvases(context.Background(), r.Registry, r.Organization.ID.String(), false)
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Empty(t, response.Canvases)
}

func Test__ListCanvases__ReturnsAllCanvasesForAnOrganization(t *testing.T) {
	r := support.Setup(t)

	//
	// Create multiple canvases
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
	// List canvases
	//
	response, err := ListCanvases(context.Background(), r.Registry, r.Organization.ID.String(), false)
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Len(t, response.Canvases, 2)

	//
	// Verify both canvases are returned
	//
	canvasIDs := []string{response.Canvases[0].Metadata.Id, response.Canvases[1].Metadata.Id}
	assert.Contains(t, canvasIDs, workflow1.ID.String())
	assert.Contains(t, canvasIDs, workflow2.ID.String())

	//
	// List of canvases returned is ordered by name
	//
	canvasNames := make([]string, len(response.Canvases))
	for i, canvas := range response.Canvases {
		canvasNames[i] = canvas.Metadata.Name
	}

	assert.True(t, sort.StringsAreSorted(canvasNames), "canvases should be sorted by name in ascending order")
}

func Test__ListCanvases__DoesNotReturnCanvasesFromOtherOrganizations(t *testing.T) {
	r := support.Setup(t)

	//
	// Create canvas in the test organization
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
	// List canvases for original organization
	//
	response, err := ListCanvases(context.Background(), r.Registry, r.Organization.ID.String(), false)
	require.NoError(t, err)
	require.NotNil(t, response)

	//
	// Should only return the canvas from the original organization
	//
	assert.Len(t, response.Canvases, 1)
	assert.Equal(t, workflow.ID.String(), response.Canvases[0].Metadata.Id)
	assert.Equal(t, r.Organization.ID.String(), response.Canvases[0].Metadata.OrganizationId)
	assert.NotEqual(t, otherWorkflow.ID.String(), response.Canvases[0].Metadata.Id)
}

func Test__ListCanvases__ReturnsCanvasesWithoutStatusInformation(t *testing.T) {
	r := support.Setup(t)

	//
	// Create canvas with nodes
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
	// List canvases
	//
	response, err := ListCanvases(context.Background(), r.Registry, r.Organization.ID.String(), false)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Canvases, 1)

	//
	// Verify status is nil (not loaded)
	//
	assert.Nil(t, response.Canvases[0].Status)
}

func Test__ListCanvases__ReturnsCanvasesWithMetadataAndSpec(t *testing.T) {
	r := support.Setup(t)

	//
	// Create canvas with nodes and edges
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
	// List canvases
	//
	response, err := ListCanvases(context.Background(), r.Registry, r.Organization.ID.String(), false)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Canvases, 1)

	listedCanvas := response.Canvases[0]

	//
	// Verify metadata is present
	//
	require.NotNil(t, listedCanvas.Metadata)
	assert.Equal(t, workflow.ID.String(), listedCanvas.Metadata.Id)
	assert.Equal(t, workflow.OrganizationID.String(), listedCanvas.Metadata.OrganizationId)
	assert.Equal(t, workflow.Name, listedCanvas.Metadata.Name)
	assert.Equal(t, workflow.Description, listedCanvas.Metadata.Description)
	assert.NotNil(t, listedCanvas.Metadata.CreatedAt)
	assert.NotNil(t, listedCanvas.Metadata.UpdatedAt)
	assert.NotNil(t, listedCanvas.Metadata.CreatedBy)

	//
	// Verify spec is present with nodes and edges
	//
	require.NotNil(t, listedCanvas.Spec)
	assert.Len(t, listedCanvas.Spec.Nodes, 2)
	assert.Equal(t, "node-1", listedCanvas.Spec.Nodes[0].Id)
	assert.Equal(t, "First Node", listedCanvas.Spec.Nodes[0].Name)
	assert.Equal(t, "node-2", listedCanvas.Spec.Nodes[1].Id)
	assert.Equal(t, "Second Node", listedCanvas.Spec.Nodes[1].Name)

	assert.Len(t, listedCanvas.Spec.Edges, 1)
	assert.Equal(t, "node-1", listedCanvas.Spec.Edges[0].SourceId)
	assert.Equal(t, "node-2", listedCanvas.Spec.Edges[0].TargetId)
	assert.Equal(t, "default", listedCanvas.Spec.Edges[0].Channel)

	//
	// Verify status is NOT present
	//
	assert.Nil(t, listedCanvas.Status)
}

func Test__ListCanvases__DoesNotReturnSoftDeletedCanvasesWhenIncludingTemplates(t *testing.T) {
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

	response, err := ListCanvases(context.Background(), r.Registry, r.Organization.ID.String(), true)
	require.NoError(t, err)
	require.NotNil(t, response)

	canvasIDs := make([]string, len(response.Canvases))
	for i, canvas := range response.Canvases {
		canvasIDs[i] = canvas.Metadata.Id
	}

	assert.True(t, slices.Contains(canvasIDs, activeWorkflow.ID.String()))
	assert.True(t, slices.Contains(canvasIDs, templateWorkflow.ID.String()))
	assert.False(t, slices.Contains(canvasIDs, deletedWorkflow.ID.String()))
}
