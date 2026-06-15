package canvases

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__ListCanvases__ReturnsEmptyListWhenNoCanvasesExist(t *testing.T) {
	r := support.Setup(t)

	response, err := ListCanvases(context.Background(), r.Registry, r.Organization.ID.String())
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Empty(t, response.Canvases)
}

func Test__ListCanvases__ReturnsAllCanvasesForAnOrganization(t *testing.T) {
	r := support.Setup(t)

	//
	// Create multiple canvases
	//
	canvas1, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
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

	canvas2, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
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
	response, err := ListCanvases(context.Background(), r.Registry, r.Organization.ID.String())
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Len(t, response.Canvases, 2)

	//
	// Verify both canvases are returned
	//
	canvasIDs := []string{response.Canvases[0].Id, response.Canvases[1].Id}
	assert.Contains(t, canvasIDs, canvas1.ID.String())
	assert.Contains(t, canvasIDs, canvas2.ID.String())

	//
	// List of canvases returned is ordered by name
	//
	canvasNames := make([]string, len(response.Canvases))
	for i, canvas := range response.Canvases {
		canvasNames[i] = canvas.Name
	}

	assert.True(t, sort.StringsAreSorted(canvasNames), "canvases should be sorted by name in ascending order")
}

func Test__ListCanvases__DoesNotReturnCanvasesFromOtherOrganizations(t *testing.T) {
	r := support.Setup(t)

	//
	// Create canvas in the test organization
	//
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
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
	otherCanvas, _ := support.CreateCanvas(
		t,
		otherOrg.ID,
		r.User,
		[]models.CanvasNode{
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
	response, err := ListCanvases(context.Background(), r.Registry, r.Organization.ID.String())
	require.NoError(t, err)
	require.NotNil(t, response)

	//
	// Should only return the canvas from the original organization
	//
	assert.Len(t, response.Canvases, 1)
	assert.Equal(t, canvas.ID.String(), response.Canvases[0].Id)
	assert.NotEqual(t, otherCanvas.ID.String(), response.Canvases[0].Id)
}

func Test__ListCanvases__ReturnsCanvasesWithoutStatusInformation(t *testing.T) {
	r := support.Setup(t)

	//
	// Create canvas with nodes
	//
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
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
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	event := support.EmitCanvasEventForNode(t, canvas.ID, "node-1", "default", nil)
	support.CreateCanvasNodeExecution(t, canvas.ID, "node-1", rootEvent.ID, event.ID)
	support.CreateQueueItem(t, canvas.ID, "node-1", rootEvent.ID, event.ID)

	//
	// List canvases
	//
	response, err := ListCanvases(context.Background(), r.Registry, r.Organization.ID.String())
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Canvases, 1)
}

func Test__ListCanvases__ReturnsSummaries(t *testing.T) {
	r := support.Setup(t)

	//
	// Create canvas with nodes and edges
	//
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
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
	response, err := ListCanvases(context.Background(), r.Registry, r.Organization.ID.String())
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Canvases, 1)

	listedCanvas := response.Canvases[0]

	//
	// Verify summary is returned
	//
	require.NotNil(t, listedCanvas)
	assert.Equal(t, canvas.ID.String(), listedCanvas.Id)
	assert.Equal(t, canvas.Name, listedCanvas.Name)
	assert.Equal(t, canvas.Description, listedCanvas.Description)
	assert.NotNil(t, listedCanvas.CreatedAt)
	assert.NotNil(t, listedCanvas.UpdatedAt)
	assert.NotNil(t, listedCanvas.CreatedBy.Id)
	assert.NotNil(t, listedCanvas.CreatedBy.Name)
	assert.NotNil(t, listedCanvas.FolderId)
}
