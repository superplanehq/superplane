package actions

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/agents"
	canvasyaml "github.com/superplanehq/superplane/pkg/canvas/yaml"
	"github.com/superplanehq/superplane/pkg/database"
	grpcactions "github.com/superplanehq/superplane/pkg/grpc/actions"
	canvasRepository "github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	canvasLayout "github.com/superplanehq/superplane/pkg/grpc/actions/canvases/layout"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"google.golang.org/protobuf/types/known/structpb"
)

const patchDraftActionName = "patch_draft"

type patchDraftAction struct {
	deps Dependencies
}

func newPatchDraftAction(deps Dependencies) patchDraftAction {
	return patchDraftAction{deps: deps}
}

func (a patchDraftAction) Name() string {
	return patchDraftActionName
}

func (a patchDraftAction) Execute(ctx context.Context, session agents.AgentSessionContext, input Input) (any, error) {
	canvasID, err := uuid.Parse(session.CanvasID)
	if err != nil {
		return updateResult{}, fmt.Errorf("invalid session canvas id: %w", err)
	}
	organizationID, err := uuid.Parse(session.OrganizationID)
	if err != nil {
		return updateResult{}, fmt.Errorf("invalid session organization id: %w", err)
	}

	userID := uuid.MustParse(session.UserID)
	draft, err := resolveTargetDraftVersion(canvasID, userID, input)
	if err != nil {
		return updateResult{}, fmt.Errorf("ensure draft: %w", err)
	}

	changeset, err := buildDraftChangeset(input.PatchOperations)
	if err != nil {
		return updateResult{}, err
	}

	canvasYAML, err := canvasRepository.ReadRepositorySpecFileStaged(
		ctx,
		session.OrganizationID,
		session.CanvasID,
		draft.ID.String(),
		canvasRepository.CanvasYAMLRepositoryPath,
	)
	if err != nil {
		return updateResult{}, fmt.Errorf("read staged canvas yaml: %w", err)
	}

	pbCanvas, err := canvasyaml.ParseCanvasResource([]byte(strings.TrimSpace(canvasYAML)))
	if err != nil {
		return updateResult{}, fmt.Errorf("parse staged canvas yaml: %w", err)
	}

	nodes, edges, err := canvasRepository.ParseCanvas(a.deps.Registry, session.OrganizationID, pbCanvas)
	if err != nil {
		return updateResult{}, fmt.Errorf("validate staged canvas yaml: %w", err)
	}

	patchedDraft := *draft
	patchedDraft.Name = pbCanvas.GetMetadata().GetName()
	patchedDraft.Description = pbCanvas.GetMetadata().GetDescription()
	patchedDraft.Nodes = nodes
	patchedDraft.Edges = edges

	patcher := changesets.NewCanvasPatcher(database.Conn(), organizationID, a.deps.Registry, &patchedDraft)
	if err := patcher.ApplyChangeset(changeset, nil); err != nil {
		return updateResult{}, err
	}

	patched := patcher.GetVersion()
	autoLayout := resolvePatchDraftAutoLayout(input.AutoLayout, changeset, edges, patched.Nodes)
	if autoLayout != nil {
		nodes, edges, err := canvasLayout.ApplyLayout(patched.Nodes, patched.Edges, autoLayout)
		if err != nil {
			return updateResult{}, err
		}
		patched.Nodes = nodes
		patched.Edges = edges
	}

	patchedYAML, err := serializePatchedDraftYAML(patched, session.CanvasID)
	if err != nil {
		return updateResult{}, fmt.Errorf("serialize patched canvas yaml: %w", err)
	}

	if _, err := canvasRepository.StageRepositorySpecFileOperations(
		ctx,
		session.OrganizationID,
		session.CanvasID,
		draft.ID.String(),
		[]*pb.CanvasRepositoryFileOperation{{
			Path:    canvasRepository.CanvasYAMLRepositoryPath,
			Content: []byte(patchedYAML),
		}},
	); err != nil {
		return updateResult{}, err
	}

	return updateResult{
		Action:     patchDraftActionName,
		CanvasID:   session.CanvasID,
		VersionID:  draft.ID.String(),
		Draft:      draftResult{VersionID: draft.ID.String(), DisplayName: draft.DisplayName, BranchName: draft.GitBranch},
		NodeIssues: collectNodeIssues(patched.Nodes),
		Summary:    summarizeParsedCanvas(patched.Name, patched.Nodes, patched.Edges),
	}, nil
}

func buildDraftChangeset(operations []PatchOperation) (*changesets.CanvasChangeset, error) {
	if len(operations) == 0 {
		return nil, fmt.Errorf("patch_operations is required for patch_draft")
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
	if operation.Node == nil {
		return nil, fmt.Errorf("node is required")
	}

	node := &changesets.ChangeNode{
		ID:            strings.TrimSpace(operation.Node.ID),
		Name:          operation.Node.Name,
		Block:         strings.TrimSpace(operation.Node.Component),
		IntegrationID: strings.TrimSpace(operation.Node.IntegrationID),
		IsCollapsed:   operation.Node.IsCollapsed,
	}
	if node.ID == "" {
		node.ID = strings.TrimSpace(operation.NodeID)
	}

	if operation.Node.Configuration != nil {
		configuration, err := structpb.NewStruct(operation.Node.Configuration)
		if err != nil {
			return nil, fmt.Errorf("invalid node configuration: %w", err)
		}
		node.Configuration = configuration
	}
	if operation.Node.Position != nil {
		node.Position = &componentpb.Position{
			X: int32(operation.Node.Position.X),
			Y: int32(operation.Node.Position.Y),
		}
	}

	return node, nil
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

func resolvePatchDraftAutoLayout(
	input *AutoLayoutInput,
	changeset *changesets.CanvasChangeset,
	originalEdges []models.Edge,
	finalNodes []models.Node,
) *pb.CanvasAutoLayout {
	if input == nil || isEmptyAutoLayoutInput(input) {
		nodeIDs := defaultPatchDraftAutoLayoutNodeIDs(changeset, originalEdges, finalNodes)
		if len(nodeIDs) == 0 {
			return nil
		}
		return &pb.CanvasAutoLayout{
			Algorithm: pb.CanvasAutoLayout_ALGORITHM_HORIZONTAL,
			Scope:     pb.CanvasAutoLayout_SCOPE_CONNECTED_COMPONENT,
			NodeIds:   nodeIDs,
		}
	}
	return resolveCustomToolAutoLayout(input, true)
}

func isEmptyAutoLayoutInput(input *AutoLayoutInput) bool {
	return input != nil &&
		strings.TrimSpace(input.Scope) == "" &&
		len(input.NodeIDs) == 0
}

func defaultPatchDraftAutoLayoutNodeIDs(
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

func serializePatchedDraftYAML(version *models.CanvasVersion, canvasID string) (string, error) {
	positioned := &pb.CanvasVersion{
		Metadata: &pb.CanvasVersion_Metadata{
			Name:        version.Name,
			Description: version.Description,
		},
		Spec: &pb.Canvas_Spec{
			Nodes: grpcactions.NodesToProto(version.Nodes),
			Edges: grpcactions.EdgesToProto(version.Edges),
		},
	}
	return canvasyaml.CanvasResourceYAML(positioned, canvasID)
}
