package agents

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	goyaml "gopkg.in/yaml.v3"
)

const canvasYAMLSnapshotPath = "canvas.yaml"

type canvasNodeSummary struct {
	ID        string
	Name      string
	Type      string
	Component string
	Issue     string
}

type snapshotGraph struct {
	source string
	nodes  []models.Node
	edges  []models.Edge
}

func buildCanvasSnapshot(ctx context.Context, session *models.AgentSession) string {
	canvas, err := models.FindCanvas(session.OrganizationID, session.CanvasID)
	if err != nil {
		log.WithError(err).Warn("failed to load canvas for agent snapshot")
		return "[Canvas Snapshot]\nUnable to load current canvas snapshot."
	}

	var builder strings.Builder
	builder.WriteString("[Canvas Snapshot]\n")
	builder.WriteString(fmt.Sprintf("canvas_id: %s\n", canvas.ID.String()))
	builder.WriteString(fmt.Sprintf("name: %s\n", canvas.Name))

	if canvas.LiveVersionID != nil {
		builder.WriteString(fmt.Sprintf("live_version_id: %s\n", canvas.LiveVersionID.String()))
	}

	version, err := models.FindLiveCanvasVersion(canvas.ID)
	if err != nil {
		log.WithError(err).Warn("failed to load live canvas version for agent snapshot")
		builder.WriteString("snapshot_source: unavailable\n")
		builder.WriteString("nodes: unavailable\n")
		return strings.TrimRight(builder.String(), "\n")
	}

	if version == nil {
		builder.WriteString("snapshot_source: live\n")
		builder.WriteString("nodes: unavailable\n")
		return strings.TrimRight(builder.String(), "\n")
	}

	graph := loadEffectiveSnapshotGraph(ctx, session, version)
	builder.WriteString(fmt.Sprintf("snapshot_source: %s\n", graph.source))
	builder.WriteString(fmt.Sprintf("node_count: %d\n", len(graph.nodes)))
	builder.WriteString(fmt.Sprintf("edge_count: %d\n", len(graph.edges)))

	nodes := summarizeSnapshotNodes(graph.nodes, 12)
	if len(nodes) == 0 {
		builder.WriteString("node_summaries: []\n")
		return strings.TrimRight(builder.String(), "\n")
	}

	builder.WriteString("node_summaries:\n")
	for _, node := range nodes {
		component := node.Component
		if component == "" {
			component = "unknown"
		}
		name := node.Name
		if name == "" {
			name = node.ID
		}
		line := fmt.Sprintf("  - id=%s name=%q type=%s component=%s", node.ID, name, node.Type, component)
		if node.Issue != "" {
			line += fmt.Sprintf(" issue=%q", node.Issue)
		}
		builder.WriteString(line + "\n")
	}

	if len(graph.nodes) > len(nodes) {
		builder.WriteString(fmt.Sprintf("  - ... %d more nodes omitted\n", len(graph.nodes)-len(nodes)))
	}

	return strings.TrimRight(builder.String(), "\n")
}

// loadEffectiveSnapshotGraph returns the graph the UI editor shows: staged
// canvas.yaml when the user has pending edits, otherwise the published live version.
func loadEffectiveSnapshotGraph(ctx context.Context, session *models.AgentSession, live *models.CanvasVersion) snapshotGraph {
	graph := snapshotGraph{source: "live", nodes: live.Nodes, edges: live.Edges}
	files, err := models.ListStagedFilesForUser(database.DB(ctx), session.CanvasID, session.UserID)
	if err != nil {
		log.WithError(err).Warn("failed to load staged files for agent snapshot")
		return graph
	}

	for _, file := range files {
		if file.Path != canvasYAMLSnapshotPath {
			continue
		}
		if file.Deleted {
			return snapshotGraph{source: "staging"}
		}
		parsed, parseErr := parseStagedSnapshotGraph(file.Content)
		if parseErr != nil {
			log.WithError(parseErr).Warn("failed to parse staged canvas yaml for agent snapshot")
			return graph
		}
		return parsed
	}

	return graph
}

func summarizeSnapshotNodes(nodes []models.Node, limit int) []canvasNodeSummary {
	count := len(nodes)
	if count > limit {
		count = limit
	}

	result := make([]canvasNodeSummary, 0, count)
	for i := 0; i < count; i++ {
		result = append(result, canvasNodeSummary{
			ID:        nodes[i].ID,
			Name:      nodes[i].Name,
			Type:      nodes[i].Type,
			Component: snapshotNodeRefName(nodes[i].Ref),
			Issue: firstSnapshotNonEmptyString(
				snapshotStringPtrValue(nodes[i].ErrorMessage),
				snapshotStringPtrValue(nodes[i].WarningMessage),
			),
		})
	}

	return result
}

func snapshotNodeRefName(ref models.NodeRef) string {
	switch {
	case ref.Component != nil:
		return ref.Component.Name
	case ref.Trigger != nil:
		return ref.Trigger.Name
	case ref.Widget != nil:
		return ref.Widget.Name
	default:
		return ""
	}
}

func snapshotStringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func firstSnapshotNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// Local YAML parse only. pkg/yaml.CanvasFromYAML pulls canvases/changesets and
// creates a test import cycle through test/support → agents.
type stagedCanvasDocument struct {
	Spec *stagedCanvasSpec `yaml:"spec"`
}

type stagedCanvasSpec struct {
	Nodes []stagedCanvasNode `yaml:"nodes"`
	Edges []stagedCanvasEdge `yaml:"edges"`
}

type stagedCanvasNode struct {
	ID             string  `yaml:"id"`
	Name           string  `yaml:"name"`
	Type           string  `yaml:"type"`
	Component      string  `yaml:"component"`
	ErrorMessage   *string `yaml:"errorMessage"`
	WarningMessage *string `yaml:"warningMessage"`
}

type stagedCanvasEdge struct {
	SourceID string `yaml:"sourceId"`
	TargetID string `yaml:"targetId"`
	Channel  string `yaml:"channel"`
}

func parseStagedSnapshotGraph(content string) (snapshotGraph, error) {
	var document stagedCanvasDocument
	if err := goyaml.Unmarshal([]byte(content), &document); err != nil {
		return snapshotGraph{}, err
	}
	if document.Spec == nil {
		return snapshotGraph{}, fmt.Errorf("staged canvas yaml has no spec")
	}

	nodes := make([]models.Node, 0, len(document.Spec.Nodes))
	for _, node := range document.Spec.Nodes {
		nodes = append(nodes, stagedCanvasNodeToModel(node))
	}

	edges := make([]models.Edge, 0, len(document.Spec.Edges))
	for _, edge := range document.Spec.Edges {
		edges = append(edges, models.Edge{
			SourceID: edge.SourceID,
			TargetID: edge.TargetID,
			Channel:  edge.Channel,
		})
	}

	return snapshotGraph{source: "staging", nodes: nodes, edges: edges}, nil
}

func stagedCanvasNodeToModel(node stagedCanvasNode) models.Node {
	return models.Node{
		ID:             node.ID,
		Name:           node.Name,
		Type:           stagedCanvasNodeType(node.Type),
		Ref:            stagedCanvasNodeRef(node.Type, node.Component),
		ErrorMessage:   node.ErrorMessage,
		WarningMessage: node.WarningMessage,
	}
}

func stagedCanvasNodeType(yamlType string) string {
	switch yamlType {
	case "TYPE_TRIGGER":
		return models.NodeTypeTrigger
	case "TYPE_WIDGET":
		return models.NodeTypeWidget
	case "TYPE_ACTION":
		return models.NodeTypeComponent
	default:
		return yamlType
	}
}

func stagedCanvasNodeRef(yamlType, component string) models.NodeRef {
	if component == "" {
		return models.NodeRef{}
	}
	switch yamlType {
	case "TYPE_TRIGGER":
		return models.NodeRef{Trigger: &models.TriggerRef{Name: component}}
	case "TYPE_WIDGET":
		return models.NodeRef{Widget: &models.WidgetRef{Name: component}}
	default:
		return models.NodeRef{Component: &models.ComponentRef{Name: component}}
	}
}
