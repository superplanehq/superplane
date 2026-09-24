package factories

import (
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

// intakeGraph locates the nodes of an intake inside its canvas. Node
// identifiers are an implementation detail of the generated graph, so they are
// resolved on read and never stored on the intake row or sent over the API.
type intakeGraph struct {
	TriggerNodeID          string
	AnalysisNodeID         string
	FilterNodeID           string
	AuthorPermissionNodeID string
	AuthorFilterNodeID     string
	CreateNodeID           string
	ConfidencePct          int
}

// Healthy reports whether the graph can still do the intake's job: receive an
// item and reach the node that creates the work order. Analysis is optional:
// a generated graph creates first, and a legacy graph that still scores stays
// healthy as long as create is reachable.
func (g intakeGraph) Healthy(edges []models.Edge) bool {
	if g.TriggerNodeID == "" || g.CreateNodeID == "" {
		return false
	}

	return hasCanvasPath(edges, g.TriggerNodeID, g.CreateNodeID)
}

func (g intakeGraph) TriggerIntegrationID(spec models.LiveCanvasSpec) string {
	trigger := findIntakeNode(spec.Nodes, g.TriggerNodeID)
	if trigger == nil || trigger.IntegrationID == nil {
		return ""
	}

	return strings.TrimSpace(*trigger.IntegrationID)
}

func (g intakeGraph) TriggerResourceID(spec models.LiveCanvasSpec) string {
	trigger := findIntakeNode(spec.Nodes, g.TriggerNodeID)
	if trigger == nil {
		return ""
	}

	if repository, ok := trigger.Configuration["repository"].(string); ok {
		if value := strings.TrimSpace(repository); value != "" {
			return value
		}
	}
	if project, ok := trigger.Configuration["project"].(string); ok {
		return strings.TrimSpace(project)
	}

	return ""
}

func intakeHealth(
	tx *gorm.DB,
	intake *models.FactoryIntake,
	graph intakeGraph,
	spec models.LiveCanvasSpec,
	states map[string]string,
) pb.FactoryIntake_Health {
	if !graph.Healthy(spec.Edges) {
		return pb.FactoryIntake_HEALTH_GRAPH_BROKEN
	}

	integrationID := graph.TriggerIntegrationID(spec)
	if integrationID == "" && intakeSourceAllowsRebind(intake.Source) {
		return pb.FactoryIntake_HEALTH_MISSING_INTEGRATION
	}
	if integrationID != "" {
		state, found := states[integrationID]
		if !found {
			return pb.FactoryIntake_HEALTH_MISSING_INTEGRATION
		}
		if state != models.IntegrationStateReady {
			return pb.FactoryIntake_HEALTH_INTEGRATION_NOT_READY
		}
	}

	if intake.Source == models.FactoryIntakeSourceJiraIssues {
		if health := jiraIntakeWebhookHealth(tx, intake.CanvasID, graph.TriggerNodeID); health != pb.FactoryIntake_HEALTH_OK {
			return health
		}
	}
	if intake.Source == models.FactoryIntakeSourceProductiveTasks {
		if health := productiveIntakeWebhookHealth(tx, intake.CanvasID, graph.TriggerNodeID); health != pb.FactoryIntake_HEALTH_OK {
			return health
		}
	}

	return pb.FactoryIntake_HEALTH_OK
}

func intakeSourceAllowsRebind(source string) bool {
	return source == models.FactoryIntakeSourceJiraIssues ||
		source == models.FactoryIntakeSourceSentryExceptions ||
		source == models.FactoryIntakeSourceProductiveTasks
}

// jiraIntakeWebhookHealth reports whether the intake trigger can receive Jira
// issue events. A pending, failed, or unregistered webhook looks like a live
// intake in the canvas, but Atlassian never POSTs until the shared webhook is
// ready and has a remote id. A failed webhook is reported apart from a pending
// one, because it is out of retries and waiting does not repair it.
func jiraIntakeWebhookHealth(tx *gorm.DB, canvasID uuid.UUID, triggerNodeID string) pb.FactoryIntake_Health {
	return intakeWebhookHealth(tx, canvasID, triggerNodeID, jiraWebhookHasRemoteID)
}

func productiveIntakeWebhookHealth(tx *gorm.DB, canvasID uuid.UUID, triggerNodeID string) pb.FactoryIntake_Health {
	return intakeWebhookHealth(tx, canvasID, triggerNodeID, productiveWebhookHasRemoteID)
}

func intakeWebhookHealth(
	tx *gorm.DB,
	canvasID uuid.UUID,
	triggerNodeID string,
	hasRemoteID func(any) bool,
) pb.FactoryIntake_Health {
	if tx == nil || triggerNodeID == "" {
		return pb.FactoryIntake_HEALTH_WEBHOOK_NOT_READY
	}

	node, err := models.FindCanvasNode(tx, canvasID, triggerNodeID)
	if err != nil || node.WebhookID == nil {
		return pb.FactoryIntake_HEALTH_WEBHOOK_NOT_READY
	}

	webhook, err := models.FindWebhookInTransaction(tx, *node.WebhookID)
	if err != nil {
		return pb.FactoryIntake_HEALTH_WEBHOOK_NOT_READY
	}

	if webhook.State == models.WebhookStateFailed {
		return pb.FactoryIntake_HEALTH_WEBHOOK_FAILED
	}

	if webhook.State != models.WebhookStateReady || !hasRemoteID(webhook.Metadata.Data()) {
		return pb.FactoryIntake_HEALTH_WEBHOOK_NOT_READY
	}

	return pb.FactoryIntake_HEALTH_OK
}

func jiraWebhookHasRemoteID(metadata any) bool {
	asMap, ok := metadata.(map[string]any)
	if !ok || asMap == nil {
		return false
	}
	id, present := asMap["webhookId"]
	return present && id != nil
}

func productiveWebhookHasRemoteID(metadata any) bool {
	asMap, ok := metadata.(map[string]any)
	if !ok || asMap == nil {
		return false
	}
	if webhookRemoteIDsPresent(asMap["ids"]) {
		return true
	}
	id, present := asMap["id"]
	return present && id != nil && strings.TrimSpace(fmt.Sprint(id)) != ""
}

func webhookRemoteIDsPresent(value any) bool {
	switch ids := value.(type) {
	case []string:
		return slices.ContainsFunc(ids, func(id string) bool {
			return strings.TrimSpace(id) != ""
		})
	case []any:
		return slices.ContainsFunc(ids, func(id any) bool {
			return id != nil && strings.TrimSpace(fmt.Sprint(id)) != ""
		})
	default:
		return false
	}
}

// resolveIntakeGraph matches the generated node identifiers first, then falls
// back to component names so a renamed node or a different agent runner does
// not make the intake unreadable.
func resolveIntakeGraph(source string, spec models.LiveCanvasSpec) intakeGraph {
	nodes := spec.Nodes
	graph := intakeGraph{
		ConfidencePct: DefaultIntakeConfidencePct,
	}

	triggerComponent := intakeSpecsBySource[source].triggerComponent
	graph.TriggerNodeID = resolveIntakeNode(nodes, intakeTriggerNodeID, func(node *models.Node) bool {
		return node.ComponentName() == triggerComponent
	})
	graph.AnalysisNodeID = resolveIntakeNode(nodes, intakeAnalysisNodeID, func(node *models.Node) bool {
		return slices.Contains(intakeAnalysisComponents, node.ComponentName())
	})
	graph.FilterNodeID = resolveIntakeFilterNode(nodes)
	if findIntakeNode(nodes, intakeAuthorPermissionNodeID) != nil {
		graph.AuthorPermissionNodeID = intakeAuthorPermissionNodeID
	}
	if findIntakeNode(nodes, intakeAuthorFilterNodeID) != nil {
		graph.AuthorFilterNodeID = intakeAuthorFilterNodeID
	}
	graph.CreateNodeID = resolveIntakeNode(nodes, intakeCreateNodeID, func(node *models.Node) bool {
		return node.ComponentName() == intakeCreateComponent
	})

	if filter := findIntakeNode(nodes, graph.FilterNodeID); filter != nil {
		if expression, ok := filter.Configuration["expression"].(string); ok {
			if confidence, ok := intakeConfidenceFromExpression(expression); ok {
				graph.ConfidencePct = confidence
			}
		}
	}

	return graph
}

func resolveIntakeFilterNode(nodes []models.Node) string {
	if id := resolveIntakeNode(nodes, intakeFilterNodeID, func(node *models.Node) bool {
		return node.ComponentName() == intakeFilterComponent
	}); id != "" {
		return id
	}

	return resolveIntakeNode(nodes, intakeThresholdNodeID, func(node *models.Node) bool {
		return node.ComponentName() == intakeFilterComponent
	})
}

func resolveIntakeNode(nodes []models.Node, preferredID string, matches func(*models.Node) bool) string {
	if node := findIntakeNode(nodes, preferredID); node != nil {
		return node.ID
	}

	for i := range nodes {
		if matches(&nodes[i]) {
			return nodes[i].ID
		}
	}

	return ""
}

func findIntakeNode(nodes []models.Node, nodeID string) *models.Node {
	if nodeID == "" {
		return nil
	}

	for i := range nodes {
		if nodes[i].ID == nodeID {
			return &nodes[i]
		}
	}

	return nil
}

func hasCanvasPath(edges []models.Edge, sourceID, targetID string) bool {
	pending := []string{sourceID}
	visited := map[string]bool{sourceID: true}

	for len(pending) > 0 {
		current := pending[0]
		pending = pending[1:]
		for _, edge := range edges {
			if edge.SourceID != current {
				continue
			}
			if edge.TargetID == targetID {
				return true
			}
			if !visited[edge.TargetID] {
				visited[edge.TargetID] = true
				pending = append(pending, edge.TargetID)
			}
		}
	}

	return false
}
