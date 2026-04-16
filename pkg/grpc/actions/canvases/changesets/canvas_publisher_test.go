package changesets

import (
	"context"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__CanvasPublisherOptions_Validate(t *testing.T) {
	r := support.Setup(t)

	validOptions := canvasPublisherOptions(r)
	testCases := []struct {
		name          string
		mutate        func(*CanvasPublisherOptions)
		expectedError string
	}{
		{
			name: "missing registry",
			mutate: func(options *CanvasPublisherOptions) {
				options.Registry = nil
			},
			expectedError: "registry is required",
		},
		{
			name: "missing org id",
			mutate: func(options *CanvasPublisherOptions) {
				options.OrgID = uuid.Nil
			},
			expectedError: "org ID is required",
		},
		{
			name: "missing encryptor",
			mutate: func(options *CanvasPublisherOptions) {
				options.Encryptor = nil
			},
			expectedError: "encryptor is required",
		},
		{
			name: "missing auth service",
			mutate: func(options *CanvasPublisherOptions) {
				options.AuthService = nil
			},
			expectedError: "auth service is required",
		},
		{
			name: "missing webhook base url",
			mutate: func(options *CanvasPublisherOptions) {
				options.WebhookBaseURL = ""
			},
			expectedError: "webhook base URL is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			options := validOptions
			tc.mutate(&options)

			err := options.Validate()
			require.ErrorContains(t, err, tc.expectedError)
		})
	}
}

func Test__NewCanvasPublisher(t *testing.T) {
	r := support.Setup(t)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			componentCanvasNode("node-a", "Node A", "noop", map[string]any{"before": "value"}),
		},
		nil,
	)

	draft, err := models.SaveCanvasDraftInTransaction(
		database.Conn(),
		canvas.ID,
		r.User,
		[]models.Node{
			componentNode("node-a", "Node A", "noop", map[string]any{"before": "value"}),
		},
		nil,
	)
	require.NoError(t, err)

	publisher, err := NewCanvasPublisher(database.Conn(), draft, canvasPublisherOptions(r))

	require.Nil(t, publisher)
	require.ErrorContains(t, err, "no changes between live and draft version being applied")
}

func Test__CanvasPublisher_Publish(t *testing.T) {
	t.Run("publishes mixed changes and promotes draft to live", func(t *testing.T) {
		r := support.Setup(t)

		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				componentCanvasNode("node-a", "Node A", "noop", map[string]any{"value": "before"}),
				componentCanvasNode("node-b", "Node B", "noop", map[string]any{"value": "remove"}),
			},
			[]models.Edge{
				{SourceID: "node-a", TargetID: "node-b", Channel: "default"},
			},
		)

		draft, err := models.SaveCanvasDraftInTransaction(
			database.Conn(),
			canvas.ID,
			r.User,
			[]models.Node{
				componentNode("node-a", "Node A Updated", "noop", map[string]any{"value": "after"}),
				componentNode("node-c", "Node C", "noop", map[string]any{"value": "new"}),
			},
			[]models.Edge{
				{SourceID: "node-a", TargetID: "node-c", Channel: "default"},
			},
		)
		require.NoError(t, err)

		publisher, err := NewCanvasPublisher(database.Conn(), draft, canvasPublisherOptions(r))
		require.NoError(t, err)

		err = publisher.Publish(context.Background())
		require.NoError(t, err)

		updatedCanvas, err := models.FindCanvasWithoutOrgScope(canvas.ID)
		require.NoError(t, err)
		require.NotNil(t, updatedCanvas.LiveVersionID)
		require.Equal(t, draft.ID, *updatedCanvas.LiveVersionID)

		publishedVersion, err := models.FindCanvasVersionInTransaction(database.Conn(), canvas.ID, draft.ID)
		require.NoError(t, err)
		require.Equal(t, models.CanvasVersionStatePublished, publishedVersion.State)
		require.NotNil(t, publishedVersion.PublishedAt)
		require.Equal(
			t,
			datatypes.NewJSONSlice([]models.Edge{{SourceID: "node-a", TargetID: "node-c", Channel: "default"}}),
			publishedVersion.Edges,
		)

		versionNodeA := findVersionNode(t, publishedVersion.Nodes, "node-a")
		versionNodeC := findVersionNode(t, publishedVersion.Nodes, "node-c")
		require.Equal(t, "Node A Updated", versionNodeA.Name)
		require.Equal(t, map[string]any{"value": "after"}, versionNodeA.Configuration)
		require.Equal(t, "Node C", versionNodeC.Name)
		require.Equal(t, map[string]any{"value": "new"}, versionNodeC.Configuration)

		activeNodes, err := models.FindCanvasNodes(canvas.ID)
		require.NoError(t, err)
		require.Len(t, activeNodes, 2)

		activeNodeA := findCanvasNode(t, activeNodes, "node-a")
		activeNodeC := findCanvasNode(t, activeNodes, "node-c")
		require.Equal(t, "Node A Updated", activeNodeA.Name)
		require.Equal(t, map[string]any{"value": "after"}, activeNodeA.Configuration.Data())
		require.Equal(t, models.CanvasNodeStateReady, activeNodeA.State)
		require.Equal(t, "Node C", activeNodeC.Name)
		require.Equal(t, map[string]any{"value": "new"}, activeNodeC.Configuration.Data())
		require.Equal(t, models.CanvasNodeStateReady, activeNodeC.State)

		allNodes, err := models.FindCanvasNodesUnscoped(canvas.ID)
		require.NoError(t, err)
		deletedNode := findCanvasNode(t, allNodes, "node-b")
		require.True(t, deletedNode.DeletedAt.Valid)
	})

	t.Run("setup errors are persisted in node state and published version", func(t *testing.T) {
		r := support.Setup(t)

		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				componentCanvasNode("node-a", "Node A", "noop", map[string]any{"value": "before"}),
			},
			nil,
		)

		draft, err := models.SaveCanvasDraftInTransaction(
			database.Conn(),
			canvas.ID,
			r.User,
			[]models.Node{
				componentNode("node-a", "Node A", "noop", map[string]any{"value": "before"}),
				componentNode("node-broken", "Node Broken", "missingcomponent", map[string]any{}),
			},
			nil,
		)
		require.NoError(t, err)

		publisher, err := NewCanvasPublisher(database.Conn(), draft, canvasPublisherOptions(r))
		require.NoError(t, err)

		err = publisher.Publish(context.Background())
		require.NoError(t, err)

		activeNodes, err := models.FindCanvasNodes(canvas.ID)
		require.NoError(t, err)
		brokenNode := findCanvasNode(t, activeNodes, "node-broken")
		require.Equal(t, models.CanvasNodeStateError, brokenNode.State)
		require.NotNil(t, brokenNode.StateReason)
		require.Contains(t, *brokenNode.StateReason, "component missingcomponent not registered")

		publishedVersion, err := models.FindCanvasVersionInTransaction(database.Conn(), canvas.ID, draft.ID)
		require.NoError(t, err)
		brokenVersionNode := findVersionNode(t, publishedVersion.Nodes, "node-broken")
		require.NotNil(t, brokenVersionNode.ErrorMessage)
		require.Contains(t, *brokenVersionNode.ErrorMessage, "component missingcomponent not registered")
	})

	t.Run("add node skips setup when node already has error", func(t *testing.T) {
		r := support.Setup(t)

		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				componentCanvasNode("node-a", "Node A", "noop", map[string]any{"value": "before"}),
			},
			nil,
		)

		existingError := "invalid configuration from previous validation"
		draft, err := models.SaveCanvasDraftInTransaction(
			database.Conn(),
			canvas.ID,
			r.User,
			[]models.Node{
				componentNode("node-a", "Node A", "noop", map[string]any{"value": "before"}),
				{
					ID:            "node-broken",
					Name:          "Node Broken",
					Type:          models.NodeTypeComponent,
					Ref:           models.NodeRef{Component: &models.ComponentRef{Name: "missingcomponent"}},
					Configuration: map[string]any{},
					Metadata:      map[string]any{},
					Position:      models.Position{X: 10, Y: 20},
					ErrorMessage:  &existingError,
				},
			},
			nil,
		)
		require.NoError(t, err)

		publisher, err := NewCanvasPublisher(database.Conn(), draft, canvasPublisherOptions(r))
		require.NoError(t, err)

		err = publisher.Publish(context.Background())
		require.NoError(t, err)

		activeNodes, err := models.FindCanvasNodes(canvas.ID)
		require.NoError(t, err)
		brokenNode := findCanvasNode(t, activeNodes, "node-broken")
		require.Equal(t, models.CanvasNodeStateError, brokenNode.State)
		require.NotNil(t, brokenNode.StateReason)
		require.Equal(t, existingError, *brokenNode.StateReason)

		publishedVersion, err := models.FindCanvasVersionInTransaction(database.Conn(), canvas.ID, draft.ID)
		require.NoError(t, err)
		brokenVersionNode := findVersionNode(t, publishedVersion.Nodes, "node-broken")
		require.NotNil(t, brokenVersionNode.ErrorMessage)
		require.Equal(t, existingError, *brokenVersionNode.ErrorMessage)
	})

	t.Run("update node skips setup when node already has error", func(t *testing.T) {
		r := support.Setup(t)

		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				componentCanvasNode("node-a", "Node A", "noop", map[string]any{"value": "before"}),
			},
			nil,
		)

		existingError := "node has invalid setup data"
		draft, err := models.SaveCanvasDraftInTransaction(
			database.Conn(),
			canvas.ID,
			r.User,
			[]models.Node{
				{
					ID:            "node-a",
					Name:          "Node A Updated",
					Type:          models.NodeTypeComponent,
					Ref:           models.NodeRef{Component: &models.ComponentRef{Name: "missingcomponent"}},
					Configuration: map[string]any{"value": "after"},
					Metadata:      map[string]any{},
					Position:      models.Position{X: 10, Y: 20},
					ErrorMessage:  &existingError,
				},
			},
			nil,
		)
		require.NoError(t, err)

		publisher, err := NewCanvasPublisher(database.Conn(), draft, canvasPublisherOptions(r))
		require.NoError(t, err)

		err = publisher.Publish(context.Background())
		require.NoError(t, err)

		activeNodes, err := models.FindCanvasNodes(canvas.ID)
		require.NoError(t, err)
		updatedNode := findCanvasNode(t, activeNodes, "node-a")
		require.Equal(t, "Node A Updated", updatedNode.Name)
		require.Equal(t, map[string]any{"value": "after"}, updatedNode.Configuration.Data())
		require.Equal(t, models.CanvasNodeStateError, updatedNode.State)
		require.NotNil(t, updatedNode.StateReason)
		require.Equal(t, existingError, *updatedNode.StateReason)

		publishedVersion, err := models.FindCanvasVersionInTransaction(database.Conn(), canvas.ID, draft.ID)
		require.NoError(t, err)
		updatedVersionNode := findVersionNode(t, publishedVersion.Nodes, "node-a")
		require.NotNil(t, updatedVersionNode.ErrorMessage)
		require.Equal(t, existingError, *updatedVersionNode.ErrorMessage)
	})

	t.Run("add node with conflicting id rewrites id in db and published version", func(t *testing.T) {
		r := support.Setup(t)

		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{
				componentCanvasNode("node-a", "Node A", "noop", map[string]any{"value": "before"}),
			},
			nil,
		)

		conflictingID := "node-conflict"
		legacyNode := componentCanvasNode(conflictingID, "Legacy Node", "noop", map[string]any{"value": "legacy"})
		legacyNode.WorkflowID = canvas.ID
		legacyNode.State = models.CanvasNodeStateReady
		require.NoError(t, database.Conn().Create(&legacyNode).Error)
		require.NoError(t, database.Conn().Delete(&legacyNode).Error)

		draft, err := models.SaveCanvasDraftInTransaction(
			database.Conn(),
			canvas.ID,
			r.User,
			[]models.Node{
				componentNode("node-a", "Node A", "noop", map[string]any{"value": "before"}),
				componentNode(conflictingID, "Node Conflict", "noop", map[string]any{"value": "new"}),
			},
			nil,
		)
		require.NoError(t, err)

		publisher, err := NewCanvasPublisher(database.Conn(), draft, canvasPublisherOptions(r))
		require.NoError(t, err)

		err = publisher.Publish(context.Background())
		require.NoError(t, err)

		activeNodes, err := models.FindCanvasNodes(canvas.ID)
		require.NoError(t, err)

		index := slices.IndexFunc(activeNodes, func(node models.CanvasNode) bool {
			return node.Name == "Node Conflict"
		})
		require.True(t, index != -1, "expected added node with conflicting ID")

		addedNode := activeNodes[index]
		require.NotEqual(t, conflictingID, addedNode.NodeID)
		require.Equal(t, "Node Conflict", addedNode.Name)

		publishedVersion, err := models.FindCanvasVersionInTransaction(database.Conn(), canvas.ID, draft.ID)
		require.NoError(t, err)

		versionHasOldID := slices.ContainsFunc(publishedVersion.Nodes, func(node models.Node) bool {
			return node.ID == conflictingID
		})
		require.False(t, versionHasOldID)

		versionHasNewID := slices.ContainsFunc(publishedVersion.Nodes, func(node models.Node) bool {
			return node.ID == addedNode.NodeID && node.Name == "Node Conflict"
		})
		require.True(t, versionHasNewID)
	})
}

func componentCanvasNode(nodeID string, name string, component string, configuration map[string]any) models.CanvasNode {
	return models.CanvasNode{
		NodeID:        nodeID,
		Name:          name,
		Type:          models.NodeTypeComponent,
		Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: component}}),
		Configuration: datatypes.NewJSONType(configuration),
		Metadata:      datatypes.NewJSONType(map[string]any{}),
		Position:      datatypes.NewJSONType(models.Position{X: 10, Y: 20}),
	}
}

func componentNode(nodeID string, name string, component string, configuration map[string]any) models.Node {
	return models.Node{
		ID:            nodeID,
		Name:          name,
		Type:          models.NodeTypeComponent,
		Ref:           models.NodeRef{Component: &models.ComponentRef{Name: component}},
		Configuration: configuration,
		Metadata:      map[string]any{},
		Position:      models.Position{X: 10, Y: 20},
	}
}

func canvasPublisherOptions(r *support.ResourceRegistry) CanvasPublisherOptions {
	return CanvasPublisherOptions{
		Registry:       r.Registry,
		OrgID:          r.Organization.ID,
		Encryptor:      r.Encryptor,
		AuthService:    r.AuthService,
		WebhookBaseURL: "https://example.com/webhooks",
	}
}

func findCanvasNode(t *testing.T, nodes []models.CanvasNode, nodeID string) models.CanvasNode {
	t.Helper()

	index := slices.IndexFunc(nodes, func(node models.CanvasNode) bool {
		return node.NodeID == nodeID
	})

	require.True(t, index != -1, "expected node %s", nodeID)
	return nodes[index]
}

func findVersionNode(t *testing.T, nodes []models.Node, nodeID string) models.Node {
	t.Helper()

	index := slices.IndexFunc(nodes, func(node models.Node) bool {
		return node.ID == nodeID
	})

	require.True(t, index != -1, "expected node %s", nodeID)
	return nodes[index]
}
