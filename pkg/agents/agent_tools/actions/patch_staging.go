package actions

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/database"
	canvasRepository "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/layout"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/pkg/yaml"
	"google.golang.org/protobuf/types/known/structpb"
)

const patchStagingActionName = "patch_staging"

type patchStagingAction struct {
	deps Dependencies
}

type patchStagingTarget struct {
	organizationID  uuid.UUID
	draft           *models.CanvasVersion
	changeset       *changesets.CanvasChangeset
	consoleYAML     string
	autoLayoutInput *AutoLayoutInput
}

type stagedDraftCanvas struct {
	nodes []models.Node
	edges []models.Edge
}

func newPatchStagingAction(deps Dependencies) patchStagingAction {
	return patchStagingAction{deps: deps}
}

func (a patchStagingAction) Name() string {
	return patchStagingActionName
}

func (a patchStagingAction) Execute(ctx context.Context, session agents.AgentSessionContext, input Input) (any, error) {
	target, err := resolvePatchStagingTarget(session, input)
	if err != nil {
		return updateResult{}, err
	}

	canvasID, err := uuid.Parse(session.CanvasID)
	if err != nil {
		return updateResult{}, fmt.Errorf("invalid session canvas id: %w", err)
	}

	tx := database.DB(ctx)
	canvas, err := models.FindCanvasInTransaction(tx, target.organizationID, canvasID)
	if err != nil {
		return updateResult{}, fmt.Errorf("load canvas: %w", err)
	}

	stagedCanvas, err := a.readStagedCanvas(ctx, session, canvas, target.draft)
	if err != nil {
		return updateResult{}, err
	}

	patched, err := a.applyPatchToStagedCanvas(target, stagedCanvas)
	if err != nil {
		return updateResult{}, err
	}

	if err := stagePatchedDraftFiles(ctx, session, target, canvas, patched); err != nil {
		return updateResult{}, err
	}

	return newPatchStagingResult(session, target.draft, canvas, patched), nil
}

func resolvePatchStagingTarget(session agents.AgentSessionContext, input Input) (patchStagingTarget, error) {
	canvasID, err := uuid.Parse(session.CanvasID)
	if err != nil {
		return patchStagingTarget{}, fmt.Errorf("invalid session canvas id: %w", err)
	}

	organizationID, err := uuid.Parse(session.OrganizationID)
	if err != nil {
		return patchStagingTarget{}, fmt.Errorf("invalid session organization id: %w", err)
	}

	liveVersion, err := resolveLiveCanvasVersion(canvasID, input)
	if err != nil {
		return patchStagingTarget{}, fmt.Errorf("resolve live version: %w", err)
	}

	changeset, err := buildDraftChangeset(input.PatchOperations)
	if err != nil {
		return patchStagingTarget{}, err
	}

	consoleYAML := strings.TrimSpace(input.ConsoleYAML)
	if changeset == nil && consoleYAML == "" && input.AutoLayout == nil {
		return patchStagingTarget{}, fmt.Errorf("patch_operations, console_yaml, or auto_layout is required for patch_staging")
	}
	if consoleYAML != "" {
		_, err := yaml.ConsoleFromYML([]byte(consoleYAML))
		if err != nil {
			return patchStagingTarget{}, err
		}
	}

	return patchStagingTarget{
		organizationID:  organizationID,
		draft:           liveVersion,
		changeset:       changeset,
		consoleYAML:     input.ConsoleYAML,
		autoLayoutInput: input.AutoLayout,
	}, nil
}

func (a patchStagingAction) readStagedCanvas(ctx context.Context, session agents.AgentSessionContext, canvas *models.Canvas, version *models.CanvasVersion) (stagedDraftCanvas, error) {
	canvasYAML, err := canvasRepository.ReadRepositorySpecFileStaged(
		ctx,
		canvas,
		version,
		canvasRepository.CanvasYAMLRepositoryPath,
	)
	if err != nil {
		return stagedDraftCanvas{}, fmt.Errorf("read staged canvas yaml: %w", err)
	}

	c, err := yaml.CanvasFromYAML([]byte(strings.TrimSpace(canvasYAML)))
	if err != nil {
		return stagedDraftCanvas{}, fmt.Errorf("parse staged canvas yaml: %w", err)
	}

	nodes, edges, err := c.Parse(a.deps.Registry, session.OrganizationID)
	if err != nil {
		return stagedDraftCanvas{}, fmt.Errorf("parse staged canvas yaml: %w", err)
	}

	return stagedDraftCanvas{
		nodes: nodes,
		edges: edges,
	}, nil
}

func (a patchStagingAction) applyPatchToStagedCanvas(
	target patchStagingTarget,
	stagedCanvas stagedDraftCanvas,
) (*models.CanvasVersion, error) {
	patchedDraft := *target.draft
	patchedDraft.Nodes = stagedCanvas.nodes
	patchedDraft.Edges = stagedCanvas.edges

	patched := &patchedDraft
	if target.changeset != nil {
		patcher := changesets.NewCanvasPatcher(database.Conn(), target.organizationID, a.deps.Registry, &patchedDraft)
		if err := patcher.ApplyChangeset(target.changeset); err != nil {
			return nil, fmt.Errorf("apply patch changeset: %w", err)
		}
		patched = patcher.GetVersion()
	}

	autoLayout := resolvePatchStagingAutoLayout(target.autoLayoutInput, target.changeset, stagedCanvas.edges, patched.Nodes)
	if autoLayout == nil {
		return patched, nil
	}

	nodes := []layout.N{}
	for _, node := range patched.Nodes {
		if node.Type == models.NodeTypeWidget {
			continue
		}
		nodes = append(nodes, layout.N{
			ID:       node.ID,
			Type:     node.Type,
			Position: layout.Position{X: node.Position.X, Y: node.Position.Y},
		})
	}

	edges := []layout.E{}
	for _, edge := range patched.Edges {
		edges = append(edges, layout.E{
			SourceID: edge.SourceID,
			TargetID: edge.TargetID,
			Channel:  edge.Channel,
		})
	}

	positionedNodes, _, err := layout.ApplyLayout(nodes, edges, autoLayout)
	if err != nil {
		return nil, fmt.Errorf("apply patch auto-layout: %w", err)
	}

	for _, positionedNode := range positionedNodes {
		i := slices.IndexFunc(patched.Nodes, func(node models.Node) bool {
			return node.ID == positionedNode.ID
		})

		if i == -1 {
			continue
		}

		patched.Nodes[i].Position.X = positionedNode.Position.X
		patched.Nodes[i].Position.Y = positionedNode.Position.Y
	}

	return patched, nil
}

func stagePatchedDraftFiles(ctx context.Context, session agents.AgentSessionContext, target patchStagingTarget, canvas *models.Canvas, patched *models.CanvasVersion) error {
	operations := make([]*pb.CanvasRepositoryFileOperation, 0, 2)
	if target.changeset != nil || target.autoLayoutInput != nil {
		patchedYAML, err := yaml.VersionToCanvasYAML(canvas.Name, canvas.Description, patched)
		if err != nil {
			return fmt.Errorf("serialize patched canvas yaml: %w", err)
		}
		operations = append(operations, &pb.CanvasRepositoryFileOperation{
			Path:    canvasRepository.CanvasYAMLRepositoryPath,
			Content: []byte(patchedYAML),
		})
	}
	if strings.TrimSpace(target.consoleYAML) != "" {
		operations = append(operations, &pb.CanvasRepositoryFileOperation{
			Path:    canvasRepository.ConsoleYAMLRepositoryPath,
			Content: []byte(target.consoleYAML),
		})
	}
	if len(operations) == 0 {
		return nil
	}

	if _, err := canvasRepository.PutCanvasStaging(
		ctx,
		session.OrganizationID,
		session.CanvasID,
		operations,
	); err != nil {
		return fmt.Errorf("stage patched draft files: %w", err)
	}

	return nil
}

func newPatchStagingResult(session agents.AgentSessionContext, draft *models.CanvasVersion, canvas *models.Canvas, patched *models.CanvasVersion) updateResult {
	return updateResult{
		Action:     patchStagingActionName,
		CanvasID:   session.CanvasID,
		VersionID:  draft.ID.String(),
		Draft:      draftResult{VersionID: draft.ID.String()},
		NodeIssues: collectNodeIssues(patched.Nodes),
		Summary:    summarizeParsedCanvas(canvas.Name, patched.Nodes, patched.Edges),
	}
}

func buildDraftChangeset(operations []PatchOperation) (*changesets.CanvasChangeset, error) {
	if len(operations) == 0 {
		return nil, nil
	}

	changes := make([]*changesets.Change, 0, len(operations))
	for i, operation := range operations {
		change, err := buildDraftChange(operation)
		if err != nil {
			return nil, fmt.Errorf("patch_operations[%d]: %w", i, err)
		}
		changes = append(changes, change)
	}

	return &changesets.CanvasChangeset{Changes: changes}, nil
}

func buildDraftChange(operation PatchOperation) (*changesets.Change, error) {
	switch normalizePatchOp(operation.Op) {
	case "add_node":
		node, err := patchChangeNode(operation)
		if err != nil {
			return nil, err
		}
		return &changesets.Change{Type: changesets.ChangeTypeAddNode, Node: node}, nil

	case "update_node":
		node, err := patchChangeNode(operation)
		if err != nil {
			return nil, err
		}
		return &changesets.Change{Type: changesets.ChangeTypeUpdateNode, Node: node}, nil

	case "delete_node":
		nodeID := patchNodeID(operation)
		if nodeID == "" {
			return nil, fmt.Errorf("node_id is required")
		}
		return &changesets.Change{
			Type: changesets.ChangeTypeDeleteNode,
			Node: &changesets.ChangeNode{ID: nodeID},
		}, nil

	case "add_edge":
		edge, err := patchChangeEdge(operation)
		if err != nil {
			return nil, err
		}
		return &changesets.Change{Type: changesets.ChangeTypeAddEdge, Edge: edge}, nil

	case "delete_edge":
		edge, err := patchChangeEdge(operation)
		if err != nil {
			return nil, err
		}
		return &changesets.Change{Type: changesets.ChangeTypeDeleteEdge, Edge: edge}, nil
	}

	return nil, fmt.Errorf("unsupported op %q", operation.Op)
}

func normalizePatchOp(op string) string {
	switch strings.TrimSpace(op) {
	case "replace_node":
		return "update_node"
	case "remove_node":
		return "delete_node"
	case "remove_edge":
		return "delete_edge"
	default:
		return strings.TrimSpace(op)
	}
}

func patchChangeNode(operation PatchOperation) (*changesets.ChangeNode, error) {
	if operation.Node == nil && operation.Position == nil {
		return nil, fmt.Errorf("node is required")
	}

	node := &changesets.ChangeNode{
		ID: strings.TrimSpace(operation.NodeID),
	}

	if operation.Node != nil {
		node.ID = strings.TrimSpace(operation.Node.ID)
		node.Name = operation.Node.Name
		node.Block = strings.TrimSpace(operation.Node.Component)
		node.IntegrationID = strings.TrimSpace(operation.Node.IntegrationID)
		node.IsCollapsed = operation.Node.IsCollapsed
		if node.ID == "" {
			node.ID = strings.TrimSpace(operation.NodeID)
		}
	}

	if operation.Node != nil && operation.Node.Configuration != nil {
		configuration, err := structpb.NewStruct(operation.Node.Configuration)
		if err != nil {
			return nil, fmt.Errorf("invalid node configuration: %w", err)
		}
		node.Configuration = configuration
	}
	if position := patchNodePosition(operation); position != nil {
		node.Position = &componentpb.Position{
			X: int32(position.X),
			Y: int32(position.Y),
		}
	}

	if strings.TrimSpace(node.ID) == "" {
		return nil, fmt.Errorf("node_id is required")
	}

	return node, nil
}

func patchNodePosition(operation PatchOperation) *PatchPosition {
	if operation.Position != nil {
		return operation.Position
	}
	if operation.Node != nil {
		return operation.Node.Position
	}
	return nil
}

func patchNodeID(operation PatchOperation) string {
	if operation.NodeID != "" {
		return strings.TrimSpace(operation.NodeID)
	}
	if operation.Node != nil {
		return strings.TrimSpace(operation.Node.ID)
	}
	return ""
}

func patchChangeEdge(operation PatchOperation) (*changesets.ChangeEdge, error) {
	if operation.Edge == nil {
		return nil, fmt.Errorf("edge is required")
	}

	edge := &changesets.ChangeEdge{
		SourceID: strings.TrimSpace(operation.Edge.SourceID),
		TargetID: strings.TrimSpace(operation.Edge.TargetID),
		Channel:  strings.TrimSpace(operation.Edge.Channel),
	}
	if edge.Channel == "" {
		edge.Channel = "default"
	}
	return edge, nil
}

func resolvePatchStagingAutoLayout(
	input *AutoLayoutInput,
	changeset *changesets.CanvasChangeset,
	originalEdges []models.Edge,
	finalNodes []models.Node,
) *layout.AutoLayout {
	if input != nil && input.Enabled != nil && !*input.Enabled {
		return nil
	}

	if input == nil {
		nodeIDs := defaultPatchStagingAutoLayoutNodeIDs(changeset, originalEdges, finalNodes)
		if len(nodeIDs) == 0 {
			return nil
		}
		return &layout.AutoLayout{
			Algorithm: layout.AlgorithmHorizontal,
			Scope:     layout.ScopeConnectedComponent,
			NodeIDs:   nodeIDs,
		}
	}

	if isEmptyAutoLayoutInput(input) {
		nodeIDs := defaultPatchStagingAutoLayoutNodeIDs(changeset, originalEdges, finalNodes)
		if len(nodeIDs) > 0 {
			return &layout.AutoLayout{
				Algorithm: layout.AlgorithmHorizontal,
				Scope:     layout.ScopeConnectedComponent,
				NodeIDs:   nodeIDs,
			}
		}
		return &layout.AutoLayout{
			Algorithm: layout.AlgorithmHorizontal,
			Scope:     layout.ScopeFullCanvas,
		}
	}

	return resolveToolAutoLayoutInput(input)
}

func isEmptyAutoLayoutInput(input *AutoLayoutInput) bool {
	return input != nil &&
		(input.Enabled == nil || *input.Enabled) &&
		strings.TrimSpace(input.Scope) == "" &&
		len(input.NodeIDs) == 0
}

func defaultPatchStagingAutoLayoutNodeIDs(
	changeset *changesets.CanvasChangeset,
	originalEdges []models.Edge,
	finalNodes []models.Node,
) []string {
	if changeset == nil {
		return nil
	}

	finalNodeIDs := make(map[string]struct{}, len(finalNodes))
	for _, node := range finalNodes {
		if node.ID != "" {
			finalNodeIDs[node.ID] = struct{}{}
		}
	}

	seedNodeIDs := map[string]struct{}{}
	deletedNodeIDs := map[string]struct{}{}
	addSeed := func(nodeID string) {
		nodeID = strings.TrimSpace(nodeID)
		if nodeID == "" {
			return
		}
		if _, exists := finalNodeIDs[nodeID]; !exists {
			return
		}
		seedNodeIDs[nodeID] = struct{}{}
	}

	for _, change := range changeset.Changes {
		if change == nil {
			continue
		}

		switch change.Type {
		case changesets.ChangeTypeAddNode, changesets.ChangeTypeUpdateNode:
			if change.Node != nil {
				addSeed(change.Node.ID)
			}
		case changesets.ChangeTypeDeleteNode:
			if change.Node != nil && strings.TrimSpace(change.Node.ID) != "" {
				deletedNodeIDs[strings.TrimSpace(change.Node.ID)] = struct{}{}
			}
		case changesets.ChangeTypeAddEdge, changesets.ChangeTypeDeleteEdge:
			if change.Edge != nil {
				addSeed(change.Edge.SourceID)
				addSeed(change.Edge.TargetID)
			}
		}
	}

	for deletedNodeID := range deletedNodeIDs {
		for _, edge := range originalEdges {
			if edge.SourceID == deletedNodeID {
				addSeed(edge.TargetID)
			}
			if edge.TargetID == deletedNodeID {
				addSeed(edge.SourceID)
			}
		}
	}

	nodeIDs := make([]string, 0, len(seedNodeIDs))
	for nodeID := range seedNodeIDs {
		nodeIDs = append(nodeIDs, nodeID)
	}
	sort.Strings(nodeIDs)
	return nodeIDs
}

func resolveToolAutoLayoutInput(input *AutoLayoutInput) *layout.AutoLayout {
	if input == nil {
		return nil
	}

	autoLayout := &layout.AutoLayout{
		Algorithm: layout.AlgorithmHorizontal,
		NodeIDs:   append([]string(nil), input.NodeIDs...),
	}

	switch strings.TrimSpace(input.Scope) {
	case "full_canvas", "full-canvas":
		autoLayout.Scope = layout.ScopeFullCanvas
	case "connected_component", "connected-component":
		autoLayout.Scope = layout.ScopeConnectedComponent
	default:
		if len(autoLayout.NodeIDs) > 0 {
			autoLayout.Scope = layout.ScopeConnectedComponent
		}
	}

	return autoLayout
}
