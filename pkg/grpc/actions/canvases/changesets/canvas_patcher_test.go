package changesets

import (
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/datatypes"
)

func Test__CanvasPatcher(t *testing.T) {
	r := support.Setup(t)

	t.Run("applies mixed operations", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry, orgID: r.Organization.ID}
		steps.givenCanvasVersion(
			[]models.Node{
				{
					ID:            "node-a",
					Name:          "Node A",
					Configuration: map[string]any{"expression": "true"},
					Type:          models.NodeTypeComponent,
					Ref: models.NodeRef{
						Component: &models.ComponentRef{Name: "if"},
					},
				},
				{
					ID:            "node-b",
					Name:          "Node B",
					Configuration: map[string]any{"expression": "false"},
					Type:          models.NodeTypeComponent,
					Ref: models.NodeRef{
						Component: &models.ComponentRef{Name: "if"},
					},
				},
			},
			[]models.Edge{{SourceID: "node-a", TargetID: "node-b", Channel: "true"}},
		)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_ADD_NODE,
					Node: &pb.CanvasChangeset_Change_Node{
						Id:            "node-c",
						Name:          "Node C",
						Block:         "noop",
						Configuration: structFromMap(t, map[string]any{}),
					},
				},
				{
					Type: pb.CanvasChangeset_Change_UPDATE_NODE,
					Node: &pb.CanvasChangeset_Change_Node{
						Id:            "node-a",
						Name:          "Node A Updated",
						Configuration: structFromMap(t, map[string]any{"expression": "false"}),
					},
				},
				{
					Type: pb.CanvasChangeset_Change_ADD_EDGE,
					Edge: &pb.CanvasChangeset_Change_Edge{
						SourceId: "node-a",
						TargetId: "node-c",
						Channel:  "true",
					},
				},
				{
					Type: pb.CanvasChangeset_Change_DELETE_NODE,
					Node: &pb.CanvasChangeset_Change_Node{Id: "node-b"},
				},
			},
		}, nil)

		steps.assertNoError()
		steps.assertHasNode("node-a", "Node A Updated", map[string]any{"expression": "false"})
		steps.assertHasNode("node-c", "Node C", map[string]any{})
		steps.assertHasNodeBlock("node-c", "noop")
		steps.assertHasNoNodeIntegrationID("node-c")
		steps.assertNodeCount(2)
		steps.assertHasEdge("node-a", "node-c", "true")
		steps.assertEdgeCount(1)
		steps.assertGraphIsValid()
	})

	t.Run("builds deterministic node and edge ordering", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(
			[]models.Node{
				{ID: "node-c", Name: "Node C"},
				{ID: "node-a", Name: "Node A"},
				{ID: "node-b", Name: "Node B"},
			},
			[]models.Edge{
				{SourceID: "node-c", TargetID: "node-a", Channel: "default"},
				{SourceID: "node-a", TargetID: "node-b", Channel: "default"},
				{SourceID: "node-a", TargetID: "node-b", Channel: "alpha"},
			},
		)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_UPDATE_NODE,
					Node: &pb.CanvasChangeset_Change_Node{
						Id:   "node-a",
						Name: "Node A Updated",
					},
				},
			},
		}, nil)

		steps.assertNoError()
		steps.assertNodeOrder([]string{"node-a", "node-b", "node-c"})
		steps.assertEdgeOrder([]models.Edge{
			{SourceID: "node-a", TargetID: "node-b", Channel: "alpha"},
			{SourceID: "node-a", TargetID: "node-b", Channel: "default"},
			{SourceID: "node-c", TargetID: "node-a", Channel: "default"},
		})
	})

	t.Run("returns error when auto layout is invalid", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(
			[]models.Node{
				{
					ID:   "node-a",
					Name: "Node A",
					Type: models.NodeTypeComponent,
					Ref: models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					},
				},
			},
			nil,
		)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_UPDATE_NODE,
					Node: &pb.CanvasChangeset_Change_Node{
						Id:   "node-a",
						Name: "Node A Updated",
					},
				},
			},
		}, &pb.CanvasAutoLayout{
			Algorithm: pb.CanvasAutoLayout_ALGORITHM_UNSPECIFIED,
		})

		steps.assertHasError()
		steps.assertErrorContains("layout algorithm is required")
		require.Nil(t, steps.finalVersion)
	})

	t.Run("does not apply layout when auto layout is omitted", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(
			[]models.Node{
				{
					ID:   "node-a",
					Name: "Node A",
					Type: models.NodeTypeComponent,
					Ref: models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					},
					Position: models.Position{X: 125, Y: 240},
				},
				{
					ID:   "node-b",
					Name: "Node B",
					Type: models.NodeTypeComponent,
					Ref: models.NodeRef{
						Component: &models.ComponentRef{Name: "noop"},
					},
					Position: models.Position{X: 780, Y: 95},
				},
			},
			nil,
		)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_ADD_EDGE,
					Edge: &pb.CanvasChangeset_Change_Edge{
						SourceId: "node-a",
						TargetId: "node-b",
						Channel:  "default",
					},
				},
			},
		}, nil)

		steps.assertNoError()
		steps.assertNodePosition("node-a", 125, 240)
		steps.assertNodePosition("node-b", 780, 95)
	})

	t.Run("rejects edges that reference an undefined source output channel", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry, orgID: r.Organization.ID}
		steps.givenCanvasVersion(
			[]models.Node{
				{
					ID:   "http-1",
					Name: "HTTP Request",
					Type: models.NodeTypeComponent,
					Ref: models.NodeRef{
						Component: &models.ComponentRef{Name: "http"},
					},
					Configuration: map[string]any{
						"method": "GET",
						"url":    "https://example.com",
					},
				},
				{
					ID:   "if-1",
					Name: "If",
					Type: models.NodeTypeComponent,
					Ref: models.NodeRef{
						Component: &models.ComponentRef{Name: "if"},
					},
					Configuration: map[string]any{
						"expression": "true",
					},
				},
			},
			nil,
		)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_ADD_EDGE,
					Edge: &pb.CanvasChangeset_Change_Edge{
						SourceId: "http-1",
						TargetId: "if-1",
						Channel:  "default",
					},
				},
			},
		}, nil)

		steps.assertHasError()
		steps.assertErrorContains(`source node http-1 does not have output channel "default"`)
	})

	t.Run("returns error when change object is misconfigured", func(t *testing.T) {
		testCases := []struct {
			name            string
			changeset       *pb.CanvasChangeset
			expectedMessage string
		}{
			{
				name:            "changeset is nil",
				changeset:       nil,
				expectedMessage: "changeset is required",
			},
			{
				name:            "changeset has no changes",
				changeset:       &pb.CanvasChangeset{},
				expectedMessage: "changeset is required",
			},
			{
				name: "changeset has nil change",
				changeset: &pb.CanvasChangeset{
					Changes: []*pb.CanvasChangeset_Change{nil},
				},
				expectedMessage: "change is required",
			},
			{
				name: "add node change has no node payload",
				changeset: &pb.CanvasChangeset{
					Changes: []*pb.CanvasChangeset_Change{
						{Type: pb.CanvasChangeset_Change_ADD_NODE},
					},
				},
				expectedMessage: "node is required for ADD_NODE",
			},
			{
				name: "add node change has empty id",
				changeset: &pb.CanvasChangeset{
					Changes: []*pb.CanvasChangeset_Change{
						{
							Type: pb.CanvasChangeset_Change_ADD_NODE,
							Node: &pb.CanvasChangeset_Change_Node{Name: "Node A", Block: "noop"},
						},
					},
				},
				expectedMessage: "target node id is required for ADD_NODE",
			},
			{
				name: "add node change has empty name",
				changeset: &pb.CanvasChangeset{
					Changes: []*pb.CanvasChangeset_Change{
						{
							Type: pb.CanvasChangeset_Change_ADD_NODE,
							Node: &pb.CanvasChangeset_Change_Node{Id: "node-a", Block: "noop"},
						},
					},
				},
				expectedMessage: "target node name is required for ADD_NODE",
			},
			{
				name: "update node change has no node payload",
				changeset: &pb.CanvasChangeset{
					Changes: []*pb.CanvasChangeset_Change{
						{Type: pb.CanvasChangeset_Change_UPDATE_NODE},
					},
				},
				expectedMessage: "node is required for UPDATE_NODE",
			},
			{
				name: "update node change has empty id",
				changeset: &pb.CanvasChangeset{
					Changes: []*pb.CanvasChangeset_Change{
						{
							Type: pb.CanvasChangeset_Change_UPDATE_NODE,
							Node: &pb.CanvasChangeset_Change_Node{Name: "Node A"},
						},
					},
				},
				expectedMessage: "node id is required for UPDATE_NODE",
			},
			{
				name: "delete node change has no node payload",
				changeset: &pb.CanvasChangeset{
					Changes: []*pb.CanvasChangeset_Change{
						{Type: pb.CanvasChangeset_Change_DELETE_NODE},
					},
				},
				expectedMessage: "target is required for DELETE_NODE",
			},
			{
				name: "delete node change has empty id",
				changeset: &pb.CanvasChangeset{
					Changes: []*pb.CanvasChangeset_Change{
						{
							Type: pb.CanvasChangeset_Change_DELETE_NODE,
							Node: &pb.CanvasChangeset_Change_Node{},
						},
					},
				},
				expectedMessage: "target node id is required for DELETE_NODE",
			},
			{
				name: "add edge change has no edge payload",
				changeset: &pb.CanvasChangeset{
					Changes: []*pb.CanvasChangeset_Change{
						{Type: pb.CanvasChangeset_Change_ADD_EDGE},
					},
				},
				expectedMessage: "edge is required for ADD_EDGE",
			},
			{
				name: "add edge change has empty source id",
				changeset: &pb.CanvasChangeset{
					Changes: []*pb.CanvasChangeset_Change{
						{
							Type: pb.CanvasChangeset_Change_ADD_EDGE,
							Edge: &pb.CanvasChangeset_Change_Edge{TargetId: "node-b", Channel: "default"},
						},
					},
				},
				expectedMessage: "source id is required for ADD_EDGE",
			},
			{
				name: "delete edge change has no edge payload",
				changeset: &pb.CanvasChangeset{
					Changes: []*pb.CanvasChangeset_Change{
						{Type: pb.CanvasChangeset_Change_DELETE_EDGE},
					},
				},
				expectedMessage: "edge is required for DELETE_EDGE",
			},
			{
				name: "delete edge change has empty channel",
				changeset: &pb.CanvasChangeset{
					Changes: []*pb.CanvasChangeset_Change{
						{
							Type: pb.CanvasChangeset_Change_DELETE_EDGE,
							Edge: &pb.CanvasChangeset_Change_Edge{SourceId: "node-a", TargetId: "node-b"},
						},
					},
				},
				expectedMessage: "channel is required for DELETE_EDGE",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
				steps.givenCanvasVersion(nil, nil)

				steps.whenHandling(tc.changeset, nil)

				steps.assertHasError()
				steps.assertErrorContains(tc.expectedMessage)
				require.Nil(t, steps.finalVersion)
			})
		}
	})

	t.Run("update node -> no configuration provided, previous configuration is preserved", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(
			[]models.Node{{
				ID:            "node-a",
				Name:          "Node A",
				Configuration: map[string]any{"expression": "true"},
				Type:          models.NodeTypeComponent,
				Ref: models.NodeRef{
					Component: &models.ComponentRef{Name: "if"},
				},
			}},
			nil,
		)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_UPDATE_NODE,
					Node: &pb.CanvasChangeset_Change_Node{Id: "node-a", Name: "Node A Updated"},
				},
			},
		}, nil)

		steps.assertNoError()
		steps.assertHasNode("node-a", "Node A Updated", map[string]any{"expression": "true"})
	})

	t.Run("update node -> no collapsed change provided, previous collapsed state is preserved", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(
			[]models.Node{{
				ID:            "node-a",
				Name:          "Node A",
				Configuration: map[string]any{"expression": "true"},
				Type:          models.NodeTypeComponent,
				IsCollapsed:   true,
				Ref: models.NodeRef{
					Component: &models.ComponentRef{Name: "if"},
				},
			}},
			nil,
		)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_UPDATE_NODE,
					Node: &pb.CanvasChangeset_Change_Node{Id: "node-a", Name: "Node A Updated"},
				},
			},
		}, nil)

		steps.assertNoError()
		steps.assertNodeCollapsed("node-a", true)
	})

	t.Run("update node -> explicit false collapsed change uncollapses node", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(
			[]models.Node{{
				ID:            "node-a",
				Name:          "Node A",
				Configuration: map[string]any{"expression": "true"},
				Type:          models.NodeTypeComponent,
				IsCollapsed:   true,
				Ref: models.NodeRef{
					Component: &models.ComponentRef{Name: "if"},
				},
			}},
			nil,
		)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_UPDATE_NODE,
					Node: &pb.CanvasChangeset_Change_Node{
						Id:          "node-a",
						Name:        "Node A Updated",
						IsCollapsed: proto.Bool(false),
					},
				},
			},
		}, nil)

		steps.assertNoError()
		steps.assertNodeCollapsed("node-a", false)
	})

	t.Run("update node -> invalid configuration sets node error without returning error", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(
			[]models.Node{
				{
					ID:            "node-a",
					Name:          "Node A",
					Configuration: map[string]any{"expression": "true"},
					Type:          models.NodeTypeComponent,
					Ref: models.NodeRef{
						Component: &models.ComponentRef{Name: "if"},
					},
				},
			},
			nil,
		)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_UPDATE_NODE,
					Node: &pb.CanvasChangeset_Change_Node{
						Id:            "node-a",
						Name:          "Node A",
						Configuration: structFromMap(t, map[string]any{"expression": nil}),
					},
				},
			},
		}, nil)

		steps.assertNoError()
		steps.assertHasNode("node-a", "Node A", map[string]any{"expression": nil})
		steps.assertNodeErrorContains("node-a", "field 'expression' is required")
	})

	t.Run("rejects self-loop edge", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(
			[]models.Node{{ID: "node-a", Name: "Node A"}},
			nil,
		)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_ADD_EDGE,
					Edge: &pb.CanvasChangeset_Change_Edge{
						SourceId: "node-a",
						TargetId: "node-a",
						Channel:  "default",
					},
				},
			},
		}, nil)
		steps.assertHasError()
		steps.assertErrorContains("self-loop edges are not allowed")
	})

	t.Run("rejects block that does not exist", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(nil, nil)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_ADD_NODE,
					Node: &pb.CanvasChangeset_Change_Node{
						Id:    "node-a",
						Name:  "Node A",
						Block: "core.hello",
					},
				},
			},
		}, nil)

		steps.assertHasError()
		steps.assertErrorContains("block core.hello not found in registry")
	})

	t.Run("rejects unknown operation type", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(nil, nil)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_UNSPECIFIED,
				},
			},
		}, nil)

		steps.assertHasError()
	})

	t.Run("rejects graph with cycles", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(
			[]models.Node{
				{ID: "node-a", Name: "Node A"},
				{ID: "node-b", Name: "Node B"},
			},
			[]models.Edge{
				{SourceID: "node-a", TargetID: "node-b", Channel: "default"},
			},
		)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_ADD_EDGE,
					Edge: &pb.CanvasChangeset_Change_Edge{
						SourceId: "node-b",
						TargetId: "node-a",
						Channel:  "default",
					},
				},
			},
		}, nil)
		steps.assertHasError()
	})

	t.Run("add component node -> invalid configuration sets node error without returning error", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(nil, nil)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_ADD_NODE,
					Node: &pb.CanvasChangeset_Change_Node{
						Id:    "node-a",
						Name:  "Node A",
						Block: "if",
					},
				},
			},
		}, nil)

		steps.assertNoError()
		steps.assertNodeCount(1)
		steps.assertHasNode("node-a", "Node A", nil)
		steps.assertHasNodeBlock("node-a", "if")
		steps.assertNodeErrorContains("node-a", "field 'expression' is required")
	})

	t.Run("add trigger node -> invalid configuration sets node error without returning error", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(nil, nil)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_ADD_NODE,
					Node: &pb.CanvasChangeset_Change_Node{
						Id:    "node-a",
						Name:  "Node A",
						Block: "schedule",
					},
				},
			},
		}, nil)

		steps.assertNoError()
		steps.assertNodeCount(1)
		steps.assertHasNode("node-a", "Node A", nil)
		steps.assertHasNodeBlock("node-a", "schedule")
		steps.assertNodeErrorContains("node-a", "field 'type' is required")
	})

	t.Run("add widget node -> invalid configuration sets node error without returning error", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry}
		steps.givenCanvasVersion(nil, nil)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_ADD_NODE,
					Node: &pb.CanvasChangeset_Change_Node{
						Id:    "node-a",
						Name:  "Node A",
						Block: "annotation",
					},
				},
			},
		}, nil)
		steps.assertNoError()
		steps.assertNodeCount(1)
		steps.assertHasNode("node-a", "Node A", nil)
		steps.assertHasNodeBlock("node-a", "annotation")
		steps.assertNodeErrorContains("node-a", "field 'text' is required")
	})

	t.Run("add integration component without integration id -> sets node error without returning error", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry, orgID: r.Organization.ID}
		steps.givenCanvasVersion(nil, nil)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_ADD_NODE,
					Node: &pb.CanvasChangeset_Change_Node{
						Id:    "node-a",
						Name:  "Node A",
						Block: "github.getIssue",
						Configuration: structFromMap(t, map[string]any{
							"repository":  "superplanehq/superplane",
							"issueNumber": "1",
						}),
					},
				},
			},
		}, nil)

		steps.assertNoError()
		steps.assertNodeCount(1)
		steps.assertHasNode("node-a", "Node A", nil)
		steps.assertHasNodeBlock("node-a", "github.getIssue")
		steps.assertNodeErrorContains("node-a", "integration is required for github.getIssue")
	})

	t.Run("add integration component with invalid integration id -> sets node error without returning error", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry, orgID: r.Organization.ID}
		steps.givenCanvasVersion(nil, nil)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_ADD_NODE,
					Node: &pb.CanvasChangeset_Change_Node{
						Id:            "node-a",
						Name:          "Node A",
						Block:         "github.getIssue",
						IntegrationId: "not-a-uuid",
						Configuration: structFromMap(t, map[string]any{
							"repository":  "superplanehq/superplane",
							"issueNumber": "1",
						}),
					},
				},
			},
		}, nil)

		steps.assertNoError()
		steps.assertNodeCount(1)
		steps.assertHasNode("node-a", "Node A", nil)
		steps.assertHasNodeBlock("node-a", "github.getIssue")
		steps.assertNodeErrorContains("node-a", "invalid integration id")
	})

	t.Run("add integration component with integration id that does not exist -> sets node error without returning error", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry, orgID: r.Organization.ID}
		steps.givenCanvasVersion(nil, nil)

		missingIntegrationID := uuid.New().String()

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_ADD_NODE,
					Node: &pb.CanvasChangeset_Change_Node{
						Id:            "node-a",
						Name:          "Node A",
						Block:         "github.getIssue",
						IntegrationId: missingIntegrationID,
						Configuration: structFromMap(t, map[string]any{
							"repository":  "superplanehq/superplane",
							"issueNumber": "1",
						}),
					},
				},
			},
		}, nil)

		steps.assertNoError()
		steps.assertNodeCount(1)
		steps.assertHasNode("node-a", "Node A", nil)
		steps.assertHasNodeBlock("node-a", "github.getIssue")
		steps.assertNodeErrorContains("node-a", "integration "+missingIntegrationID+" not found")
	})

	t.Run("accepts integration component with existing integration id", func(t *testing.T) {
		steps := &CanvasPatcherSteps{t: t, registry: r.Registry, orgID: r.Organization.ID}
		steps.givenCanvasVersion(nil, nil)

		integration, err := models.CreateIntegration(
			uuid.New(),
			r.Organization.ID,
			"github",
			support.RandomName("integration"),
			nil,
		)
		require.NoError(t, err)

		steps.whenHandling(&pb.CanvasChangeset{
			Changes: []*pb.CanvasChangeset_Change{
				{
					Type: pb.CanvasChangeset_Change_ADD_NODE,
					Node: &pb.CanvasChangeset_Change_Node{
						Id:            "node-a",
						Name:          "Node A",
						Block:         "github.getIssue",
						IntegrationId: integration.ID.String(),
						Configuration: structFromMap(t, map[string]any{
							"repository":  "superplanehq/superplane",
							"issueNumber": "1",
						}),
					},
				},
			},
		}, nil)

		steps.assertNoError()
		steps.assertNodeCount(1)
		steps.assertHasNode("node-a", "Node A", map[string]any{
			"repository":  "superplanehq/superplane",
			"issueNumber": "1",
		})
		steps.assertHasNodeBlock("node-a", "github.getIssue")
		steps.assertHasNodeIntegrationID("node-a", integration.ID.String())
	})
}

type CanvasPatcherSteps struct {
	t            *testing.T
	registry     *registry.Registry
	orgID        uuid.UUID
	patcher      *CanvasPatcher
	err          error
	finalVersion *models.CanvasVersion
}

func (s *CanvasPatcherSteps) givenCanvasVersion(nodes []models.Node, edges []models.Edge) {
	s.patcher = NewCanvasPatcher(database.Conn(), s.orgID, s.registry, &models.CanvasVersion{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
		Nodes:      datatypes.NewJSONSlice(nodes),
		Edges:      datatypes.NewJSONSlice(edges),
	})
}

func (s *CanvasPatcherSteps) whenHandling(operations *pb.CanvasChangeset, autoLayout *pb.CanvasAutoLayout) {
	s.err = s.patcher.ApplyChangeset(operations, autoLayout)
	s.finalVersion = s.patcher.GetVersion()
}

func (s *CanvasPatcherSteps) assertNoError() {
	require.NoError(s.t, s.err)
}

func (s *CanvasPatcherSteps) assertHasError() {
	require.Error(s.t, s.err)
}

func (s *CanvasPatcherSteps) assertErrorContains(text string) {
	require.ErrorContains(s.t, s.err, text)
}

func (s *CanvasPatcherSteps) assertHasNode(nodeID string, name string, configuration map[string]any) {
	i := slices.IndexFunc(s.finalVersion.Nodes, func(node models.Node) bool {
		return node.ID == nodeID
	})

	require.True(s.t, i != -1, "expected node %s", nodeID)
	require.Equal(s.t, name, s.finalVersion.Nodes[i].Name)
	require.Equal(s.t, configuration, s.finalVersion.Nodes[i].Configuration)
}

func (s *CanvasPatcherSteps) assertNodeCount(count int) {
	require.NotNil(s.t, s.finalVersion)
	require.Len(s.t, s.finalVersion.Nodes, count)
}

func (s *CanvasPatcherSteps) assertHasNodeBlock(nodeID string, block string) {
	i := slices.IndexFunc(s.finalVersion.Nodes, func(node models.Node) bool {
		return node.ID == nodeID
	})

	require.True(s.t, i != -1, "expected node %s", nodeID)

	nodeBlock := s.findBlockName(s.finalVersion.Nodes[i])
	require.Equal(s.t, block, nodeBlock)
}

func (s *CanvasPatcherSteps) assertHasNodeIntegrationID(nodeID string, integrationID string) {
	i := slices.IndexFunc(s.finalVersion.Nodes, func(node models.Node) bool {
		return node.ID == nodeID
	})

	require.True(s.t, i != -1, "expected node %s", nodeID)
	require.NotNil(s.t, s.finalVersion.Nodes[i].IntegrationID)
	require.Equal(s.t, integrationID, *s.finalVersion.Nodes[i].IntegrationID)
}

func (s *CanvasPatcherSteps) assertHasNoNodeIntegrationID(nodeID string) {
	i := slices.IndexFunc(s.finalVersion.Nodes, func(node models.Node) bool {
		return node.ID == nodeID
	})

	require.True(s.t, i != -1, "expected node %s", nodeID)
	require.Nil(s.t, s.finalVersion.Nodes[i].IntegrationID)
}

func (s *CanvasPatcherSteps) assertNodeCollapsed(nodeID string, expected bool) {
	i := slices.IndexFunc(s.finalVersion.Nodes, func(node models.Node) bool {
		return node.ID == nodeID
	})

	require.True(s.t, i != -1, "expected node %s", nodeID)
	require.Equal(s.t, expected, s.finalVersion.Nodes[i].IsCollapsed)
}

func (s *CanvasPatcherSteps) assertNodeErrorContains(nodeID string, text string) {
	i := slices.IndexFunc(s.finalVersion.Nodes, func(node models.Node) bool {
		return node.ID == nodeID
	})

	require.True(s.t, i != -1, "expected node %s", nodeID)
	require.NotNil(s.t, s.finalVersion.Nodes[i].ErrorMessage)
	require.Contains(s.t, *s.finalVersion.Nodes[i].ErrorMessage, text)
}

func (s *CanvasPatcherSteps) assertNodePosition(nodeID string, x int, y int) {
	i := slices.IndexFunc(s.finalVersion.Nodes, func(node models.Node) bool {
		return node.ID == nodeID
	})

	require.True(s.t, i != -1, "expected node %s", nodeID)
	require.Equal(s.t, x, s.finalVersion.Nodes[i].Position.X)
	require.Equal(s.t, y, s.finalVersion.Nodes[i].Position.Y)
}

func (s *CanvasPatcherSteps) findBlockName(node models.Node) string {
	if node.Ref.Component != nil && node.Ref.Component.Name != "" {
		return node.Ref.Component.Name
	}

	if node.Ref.Trigger != nil && node.Ref.Trigger.Name != "" {
		return node.Ref.Trigger.Name
	}

	if node.Ref.Widget != nil && node.Ref.Widget.Name != "" {
		return node.Ref.Widget.Name
	}

	return ""
}

func (s *CanvasPatcherSteps) assertHasEdge(sourceID string, targetID string, channel string) {
	i := slices.IndexFunc(s.finalVersion.Edges, func(edge models.Edge) bool {
		return edge.SourceID == sourceID && edge.TargetID == targetID && edge.Channel == channel
	})

	require.True(s.t, i != -1, "expected edge %s -> %s on channel %s", sourceID, targetID, channel)
	require.Equal(s.t, sourceID, s.finalVersion.Edges[i].SourceID)
	require.Equal(s.t, targetID, s.finalVersion.Edges[i].TargetID)
	require.Equal(s.t, channel, s.finalVersion.Edges[i].Channel)
}

func (s *CanvasPatcherSteps) assertEdgeCount(count int) {
	require.Len(s.t, s.finalVersion.Edges, count)
}

func (s *CanvasPatcherSteps) assertNodeOrder(nodeIDs []string) {
	orderedNodeIDs := make([]string, 0, len(s.finalVersion.Nodes))
	for _, node := range s.finalVersion.Nodes {
		orderedNodeIDs = append(orderedNodeIDs, node.ID)
	}

	require.Equal(s.t, nodeIDs, orderedNodeIDs)
}

func (s *CanvasPatcherSteps) assertEdgeOrder(edges []models.Edge) {
	require.Equal(s.t, datatypes.NewJSONSlice(edges), s.finalVersion.Edges)
}

func (s *CanvasPatcherSteps) assertGraphIsValid() {
	require.NoError(s.t, CheckForCycles(s.finalVersion.Nodes, s.finalVersion.Edges))
}

func structFromMap(t *testing.T, value map[string]any) *structpb.Struct {
	t.Helper()

	result, err := structpb.NewStruct(value)
	require.NoError(t, err)

	return result
}
