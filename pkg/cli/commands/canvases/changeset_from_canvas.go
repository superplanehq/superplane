package canvases

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// canvasNodeForChangeset holds the fields needed to diff two canvas specs and
// build a REST changeset (same semantics as pkg/grpc/actions/canvases/changesets).
type canvasNodeForChangeset struct {
	ID            string
	Name          string
	Block         string
	Configuration map[string]interface{}
	Position      openapi_client.ComponentsPosition
	IsCollapsed   bool
	IntegrationID string
}

func blockNameFromOpenAPINode(node openapi_client.SuperplaneComponentsNode) string {
	if node.Component != nil && strings.TrimSpace(node.Component.GetName()) != "" {
		return strings.TrimSpace(node.Component.GetName())
	}
	if node.Trigger != nil && strings.TrimSpace(node.Trigger.GetName()) != "" {
		return strings.TrimSpace(node.Trigger.GetName())
	}
	if node.Blueprint != nil && strings.TrimSpace(node.Blueprint.GetId()) != "" {
		return strings.TrimSpace(node.Blueprint.GetId())
	}
	if node.Widget != nil && strings.TrimSpace(node.Widget.GetName()) != "" {
		return strings.TrimSpace(node.Widget.GetName())
	}
	return ""
}

func positionFromOpenAPINode(n openapi_client.SuperplaneComponentsNode) openapi_client.ComponentsPosition {
	if n.Position != nil {
		return *n.Position
	}
	p := openapi_client.NewComponentsPosition()
	p.SetX(0)
	p.SetY(0)
	return *p
}

func nodeFromOpenAPI(n openapi_client.SuperplaneComponentsNode) (canvasNodeForChangeset, error) {
	id := strings.TrimSpace(n.GetId())
	if id == "" {
		return canvasNodeForChangeset{}, fmt.Errorf("node id is required")
	}
	block := blockNameFromOpenAPINode(n)
	if block == "" {
		return canvasNodeForChangeset{}, fmt.Errorf("block name is required for node %s", id)
	}
	pos := positionFromOpenAPINode(n)
	out := canvasNodeForChangeset{
		ID:            id,
		Name:          n.GetName(),
		Block:         block,
		Configuration: n.Configuration,
		Position:      pos,
		IsCollapsed:   n.GetIsCollapsed(),
	}
	if n.Integration != nil {
		out.IntegrationID = strings.TrimSpace(n.Integration.GetId())
	}
	return out, nil
}

func buildNodeMap(nodes []openapi_client.SuperplaneComponentsNode) (map[string]canvasNodeForChangeset, error) {
	out := make(map[string]canvasNodeForChangeset, len(nodes))
	for _, raw := range nodes {
		node, err := nodeFromOpenAPI(raw)
		if err != nil {
			return nil, err
		}
		out[node.ID] = node
	}
	return out, nil
}

type canvasEdgeKey struct {
	SourceID string
	TargetID string
	Channel  string
}

func edgeKey(e openapi_client.SuperplaneComponentsEdge) canvasEdgeKey {
	return canvasEdgeKey{
		SourceID: strings.TrimSpace(e.GetSourceId()),
		TargetID: strings.TrimSpace(e.GetTargetId()),
		Channel:  strings.TrimSpace(e.GetChannel()),
	}
}

func buildEdgeMap(edges []openapi_client.SuperplaneComponentsEdge) map[canvasEdgeKey]openapi_client.SuperplaneComponentsEdge {
	out := make(map[canvasEdgeKey]openapi_client.SuperplaneComponentsEdge, len(edges))
	for _, e := range edges {
		out[edgeKey(e)] = e
	}
	return out
}

func changeNodeForAdd(node canvasNodeForChangeset) openapi_client.CanvasChangesetChangeNode {
	n := openapi_client.NewCanvasChangesetChangeNode()
	n.SetId(node.ID)
	n.SetName(node.Name)
	n.SetBlock(node.Block)
	n.SetIsCollapsed(node.IsCollapsed)
	pos := openapi_client.NewComponentsPosition()
	pos.SetX(int32(node.Position.GetX()))
	pos.SetY(int32(node.Position.GetY()))
	n.SetPosition(*pos)
	if node.Configuration != nil {
		n.SetConfiguration(node.Configuration)
	}
	if node.IntegrationID != "" {
		n.SetIntegrationId(node.IntegrationID)
	}
	return *n
}

func changeNodeForUpdate(current, proposed canvasNodeForChangeset) openapi_client.CanvasChangesetChangeNode {
	n := openapi_client.NewCanvasChangesetChangeNode()
	n.SetId(proposed.ID)
	n.SetName(proposed.Name)
	n.SetBlock(proposed.Block)

	if proposed.Configuration != nil && !reflect.DeepEqual(current.Configuration, proposed.Configuration) {
		n.SetConfiguration(proposed.Configuration)
	}
	if proposed.Position.GetX() != current.Position.GetX() || proposed.Position.GetY() != current.Position.GetY() {
		pos := openapi_client.NewComponentsPosition()
		pos.SetX(int32(proposed.Position.GetX()))
		pos.SetY(int32(proposed.Position.GetY()))
		n.SetPosition(*pos)
	}
	if proposed.IsCollapsed != current.IsCollapsed {
		n.SetIsCollapsed(proposed.IsCollapsed)
	}
	return *n
}

func changeEdge(e openapi_client.SuperplaneComponentsEdge) openapi_client.CanvasChangesetChangeEdge {
	out := openapi_client.NewCanvasChangesetChangeEdge()
	out.SetSourceId(e.GetSourceId())
	out.SetTargetId(e.GetTargetId())
	out.SetChannel(e.GetChannel())
	return *out
}

// buildCanvasChangesetFromSpecs computes the same changeset the server would use
// when applying a full canvas snapshot to an existing draft version.
func buildCanvasChangesetFromSpecs(
	currentNodes []openapi_client.SuperplaneComponentsNode,
	currentEdges []openapi_client.SuperplaneComponentsEdge,
	proposedNodes []openapi_client.SuperplaneComponentsNode,
	proposedEdges []openapi_client.SuperplaneComponentsEdge,
) (*openapi_client.CanvasesCanvasChangeset, error) {
	curN, err := buildNodeMap(currentNodes)
	if err != nil {
		return nil, err
	}
	propN, err := buildNodeMap(proposedNodes)
	if err != nil {
		return nil, err
	}
	curE := buildEdgeMap(currentEdges)
	propE := buildEdgeMap(proposedEdges)

	var changes []openapi_client.CanvasChangesetChange

	for id := range curN {
		if _, ok := propN[id]; !ok {
			ch := openapi_client.NewCanvasChangesetChange()
			ch.SetType(openapi_client.CANVASCHANGESETCHANGETYPE_DELETE_NODE)
			del := openapi_client.NewCanvasChangesetChangeNode()
			del.SetId(id)
			ch.SetNode(*del)
			changes = append(changes, *ch)
		}
	}

	for _, node := range propN {
		if _, ok := curN[node.ID]; ok {
			continue
		}
		ch := openapi_client.NewCanvasChangesetChange()
		ch.SetType(openapi_client.CANVASCHANGESETCHANGETYPE_ADD_NODE)
		add := changeNodeForAdd(node)
		ch.SetNode(add)
		changes = append(changes, *ch)
	}

	for _, proposed := range propN {
		current, ok := curN[proposed.ID]
		if !ok {
			continue
		}
		if reflect.DeepEqual(current, proposed) {
			continue
		}
		ch := openapi_client.NewCanvasChangesetChange()
		ch.SetType(openapi_client.CANVASCHANGESETCHANGETYPE_UPDATE_NODE)
		ch.SetNode(changeNodeForUpdate(current, proposed))
		changes = append(changes, *ch)
	}

	for k, edge := range curE {
		if _, ok := propE[k]; !ok {
			ch := openapi_client.NewCanvasChangesetChange()
			ch.SetType(openapi_client.CANVASCHANGESETCHANGETYPE_DELETE_EDGE)
			ch.SetEdge(changeEdge(edge))
			changes = append(changes, *ch)
		}
	}

	for k, edge := range propE {
		if _, ok := curE[k]; !ok {
			ch := openapi_client.NewCanvasChangesetChange()
			ch.SetType(openapi_client.CANVASCHANGESETCHANGETYPE_ADD_EDGE)
			ch.SetEdge(changeEdge(edge))
			changes = append(changes, *ch)
		}
	}

	cs := openapi_client.NewCanvasesCanvasChangeset()
	cs.SetChanges(changes)
	return cs, nil
}

func sortNodeIssuesForDisplay(issues []canvasNodeIssue) {
	sort.Slice(issues, func(i, j int) bool {
		if issues[i].NodeID != issues[j].NodeID {
			return issues[i].NodeID < issues[j].NodeID
		}
		return issues[i].Kind < issues[j].Kind
	})
}

type canvasNodeIssue struct {
	NodeID   string
	NodeName string
	Kind     string // "error" or "warning"
	Message  string
}

func collectCanvasNodeIssues(spec openapi_client.CanvasesCanvasSpec) []canvasNodeIssue {
	nodes := spec.GetNodes()
	out := make([]canvasNodeIssue, 0)
	for _, n := range nodes {
		id := strings.TrimSpace(n.GetId())
		name := strings.TrimSpace(n.GetName())
		if msg, ok := n.GetErrorMessageOk(); ok && strings.TrimSpace(*msg) != "" {
			out = append(out, canvasNodeIssue{
				NodeID:   id,
				NodeName: name,
				Kind:     "error",
				Message:  strings.TrimSpace(*msg),
			})
		}
		if msg, ok := n.GetWarningMessageOk(); ok && strings.TrimSpace(*msg) != "" {
			out = append(out, canvasNodeIssue{
				NodeID:   id,
				NodeName: name,
				Kind:     "warning",
				Message:  strings.TrimSpace(*msg),
			})
		}
	}
	sortNodeIssuesForDisplay(out)
	return out
}

func formatNodeIssuesLine(issue canvasNodeIssue) string {
	label := issue.NodeID
	if issue.NodeName != "" {
		label = fmt.Sprintf("%s (%s)", issue.NodeID, issue.NodeName)
	}
	return fmt.Sprintf("  - %s [%s]: %s", label, issue.Kind, issue.Message)
}
