package factories

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
)

const (
	// Node identifiers of a generated intake graph. An intake owns its whole
	// canvas, so the identifiers are fixed rather than derived from the source.
	intakeTriggerNodeID = "trigger"
	intakeFilterNodeID  = "filter"
	intakeCreateNodeID  = "create-work-order"

	intakeAuthorPermissionNodeID = "get-author-permission"
	intakeAuthorFilterNodeID     = "author-has-repository-access"

	// Legacy node identifiers. A graph generated before intake became
	// create-only still resolves so settings and health keep working.
	intakeAnalysisNodeID         = "analyze"
	intakeThresholdNodeID        = "threshold"
	intakeReportConfidenceNodeID = "report-confidence"

	intakeCreateNodeName = "Create Task"

	intakeFilterComponent           = "if"
	intakeAuthorPermissionComponent = "github.getRepositoryPermission"
	intakeThresholdComponent        = intakeFilterComponent
	intakeCreateComponent           = "createWorkOrder"

	intakeAddRunErrorNodeID    = "add-run-error"
	intakeAddRunErrorNodeName  = "Record Analysis Failure"
	intakeAddRunErrorComponent = "addRunError"
	intakeAddRunErrorMessage   = "The analysis agent failed. Open the agent logs to find the cause."

	intakeAnalysisTimeoutSeconds = 1800

	// intakeConcurrencyMax is how many items an intake node works on at once.
	// A node runs one execution at a time by default, which makes a batch of
	// items wait for each other: a seeded batch, or a source that reports many
	// items in a burst, would take as long as the sum of its analyses.
	intakeConcurrencyMax = 100

	DefaultIntakeConfidencePct = 65
)

// intakeAnalysisMachineType is the machine the analysis runner asks for. The
// runner components reject a node without one, so the generated graph has to
// name it.
const intakeAnalysisMachineType = runner.MachineTypeE1LargeAMD64

// intakeAnalysisComponents are the runners a backlog canvas can score with.
// Creation picks the runner of the workspace agent, but a graph the user
// re-pointed at another runner still has to resolve.
var intakeAnalysisComponents = intakeAgentComponents()

func intakeAgentComponents() []string {
	components := make([]string, 0, len(intakeAgentSpecs)+1)
	components = append(components, models.SuperPlaneRunnerComponent)
	for _, spec := range intakeAgentSpecs {
		components = append(components, spec.component)
	}

	return components
}

type intakeSpec struct {
	name                 string
	description          string
	triggerComponent     string
	triggerName          string
	triggerConfiguration map[string]any
	createTitle          string
	createDescription    string
}

var intakeSpecsBySource = map[string]intakeSpec{
	models.FactoryIntakeSourceGitHubIssues: {
		name:                 "GitHub issues",
		description:          "Create a work order when a GitHub issue is opened.",
		triggerComponent:     "github.onIssue",
		triggerName:          "On Issue",
		triggerConfiguration: map[string]any{"actions": intakeTriggerActionsFor(defaultIntakeSettings())},
		createTitle:          "{{ root().data.issue.title }}",
		createDescription:    "{{ root().data.issue.body }}",
	},
	models.FactoryIntakeSourceSentryExceptions: {
		name:                 "Sentry exceptions",
		description:          "Create a work order when a Sentry exception is reported.",
		triggerComponent:     "sentry.onIssue",
		triggerName:          "On Issue Event",
		triggerConfiguration: map[string]any{"actions": intakeSentryActionsFor(defaultSentryIntakeSettings())},
		createTitle:          "{{ root().data.data.issue.title }}",
		createDescription:    "{{ root().data.description }}",
	},
	models.FactoryIntakeSourcePagerDutyIncidents: {
		name:             "PagerDuty incidents",
		description:      "Create a work order when a PagerDuty incident is triggered.",
		triggerComponent: "pagerduty.onIncident",
		triggerName:      "On Incident",
		triggerConfiguration: map[string]any{
			"events":    []any{"incident.triggered"},
			"urgencies": []any{"high", "low"},
		},
		createTitle:       "{{ root().data.incident.title }}",
		createDescription: "{{ root().data.incident.html_url }}",
	},
	models.FactoryIntakeSourceProductiveTasks: {
		name:                 "Productive tasks",
		description:          "Create a work order when a Productive task is created.",
		triggerComponent:     "productive.onTask",
		triggerName:          "On Task",
		triggerConfiguration: map[string]any{"actions": []any{"created"}},
		createTitle:          "{{ root().data.data.attributes.title }}",
		createDescription:    "{{ root().data.data.attributes.description }}",
	},
	models.FactoryIntakeSourceJiraIssues: {
		name:             "Jira issues",
		description:      "Create a work order when a Jira issue is created or updated.",
		triggerComponent: "jira.onIssue",
		triggerName:      "On Issue",
		triggerConfiguration: map[string]any{
			"events": intakeTriggerEventsFor(defaultJiraIntakeSettings()),
		},
		createTitle: `{{ root().data.issue.key }}: {{ root().data.issue.fields.summary }}`,
		// The raw description field holds an Atlassian Document Format
		// object, so the work order reads the plain text copy the trigger
		// reports next to it.
		createDescription: `{{ root().data.description }}`,
	},
	models.FactoryIntakeSourceDependabotAlerts: {
		name:             "Dependabot alerts",
		description:      "Create a task when GitHub reports a Dependabot alert. Turn on Dependabot alerts for the repository.",
		triggerComponent: "github.onDependabotAlert",
		triggerName:      "On Dependabot Alert",
		triggerConfiguration: map[string]any{
			"actions": dependabotIntakeActions(),
		},
		createTitle:       dependabotAlertCreateTitle,
		createDescription: dependabotAlertCreateDescription,
	},
}

// dependabotAlertCreateTitle names the package, not the manifest, because
// every alert for one package lands on one task. It must match
// dependabot.TaskTitle.
const dependabotAlertCreateTitle = `{{ "Fix Dependabot alerts for " + (root().data.alert.dependency.package.name ?? "a dependency") + ((root().data.alert.dependency.package.ecosystem ?? "") != "" ? " (" + root().data.alert.dependency.package.ecosystem + ")" : "") }}`

// dependabotAlertCreateDescription opens the task with the first alert. A
// later alert for the same package is merged in by the Create Task
// component, so the alert block must match dependabot.AlertSection. One
// expression keeps a missing field from failing the whole description.
const dependabotAlertCreateDescription = `{{ "Fix every open Dependabot alert for " + (root().data.alert.dependency.package.name ?? "a dependency") + ((root().data.alert.dependency.package.ecosystem ?? "") != "" ? " (" + root().data.alert.dependency.package.ecosystem + ")" : "") + ".\n\n## Alerts\n\n### #" + string(root().data.alert.number ?? 0) + " " + (root().data.alert.security_advisory.summary ?? "") + "\nSeverity: " + (root().data.alert.security_advisory.severity ?? "") + "\nManifest: " + (root().data.alert.dependency.manifest_path ?? "") + "\nVulnerable versions: " + (root().data.alert.security_vulnerability.vulnerable_version_range ?? "") + "\nPatched version: " + (root().data.alert.security_vulnerability.first_patched_version?.identifier ?? "") + ((root().data.alert.dependency.relationship ?? "") in ["direct", "transitive"] ? "\nRelationship: " + root().data.alert.dependency.relationship : "") + "\n" + (root().data.alert.html_url ?? "") }}`

func intakeSourceByTriggerComponent(component string) (string, bool) {
	for source, spec := range intakeSpecsBySource {
		if spec.triggerComponent == component {
			return source, true
		}
	}
	return "", false
}

func intakeDefaultName(source string) string {
	return intakeSpecsBySource[source].name
}

func intakeDefaultDescription(source string) string {
	return intakeSpecsBySource[source].description
}

// buildIntakeCanvas returns the canvas document for a new intake: listen on the
// source and create a work order. GitHub intakes keep a filter node so label
// and assignment settings have somewhere to live. Planning happens
// on the factory Backlog canvas after the work order exists.
func buildIntakeCanvas(request intakeCanvasRequest) (*yaml.Canvas, error) {
	spec, ok := intakeSpecsBySource[request.Source]
	if !ok {
		return nil, models.ErrFactoryIntakeSourceInvalid
	}

	name := strings.TrimSpace(request.Name)
	if name == "" {
		name = spec.name
	}

	settings := intakeSettingsOrDefault(request.Source, request.Settings)
	nodes := []yaml.Node{
		{
			ID:            intakeTriggerNodeID,
			Name:          spec.triggerName,
			Type:          yaml.NodeTypeTrigger,
			Component:     spec.triggerComponent,
			Configuration: intakeTriggerConfiguration(spec, request.Binding),
			Metadata:      intakeTriggerMetadata(request.Source, request.Settings),
			Integration:   request.Binding.integrationRef(),
			Position:      yaml.Position{X: 160, Y: 80},
		},
	}
	edges := []yaml.Edge{}
	createY := 260

	if intakeSourceHasFilterNode(request.Source) {
		nodes = append(nodes, yaml.Node{
			ID:        intakeFilterNodeID,
			Name:      "Matches filters?",
			Type:      yaml.NodeTypeAction,
			Component: intakeFilterComponent,
			Configuration: map[string]any{
				"expression": intakeFilterExpressionFor(request.Source, settings),
			},
			Concurrency: intakeConcurrency(),
			Position:    yaml.Position{X: 160, Y: 260},
		})
		edges = append(edges,
			yaml.Edge{Channel: "default", SourceID: intakeTriggerNodeID, TargetID: intakeFilterNodeID},
			yaml.Edge{Channel: "true", SourceID: intakeFilterNodeID, TargetID: intakeCreateNodeID},
		)
		createY = 440
	} else {
		edges = append(edges, yaml.Edge{Channel: "default", SourceID: intakeTriggerNodeID, TargetID: intakeCreateNodeID})
	}

	nodes = append(nodes, yaml.Node{
		ID:            intakeCreateNodeID,
		Name:          intakeCreateNodeName,
		Type:          yaml.NodeTypeAction,
		Component:     intakeCreateComponent,
		Configuration: intakeCreateConfiguration(spec, settings),
		Concurrency:   intakeConcurrency(),
		Position:      yaml.Position{X: 160, Y: createY},
	})

	return &yaml.Canvas{
		APIVersion: yaml.APIVersion,
		Kind:       yaml.KindCanvas,
		Metadata: &yaml.CanvasMetadata{
			Name:        name,
			Description: spec.description,
		},
		Spec: &yaml.CanvasSpec{
			Edges: edges,
			Nodes: nodes,
		},
	}, nil
}

// intakeCreateConfiguration is the Create Task node configuration. The
// instructions are stored as the user wrote them; only the title and the
// description are expressions.
func intakeCreateConfiguration(spec intakeSpec, settings intakeSettings) map[string]any {
	configuration := map[string]any{
		"title":       spec.createTitle,
		"description": spec.createDescription,
	}
	if instructions := strings.TrimSpace(settings.Instructions); instructions != "" {
		configuration[intakeInstructionsConfigurationKey] = instructions
	}
	return configuration
}

// intakeConcurrency returns the concurrency of one intake node. Each node owns
// its spec, so a later edit to one node cannot reach the others. Only action
// nodes take a spec; a trigger has no queue to widen.
func intakeConcurrency() *yaml.ConcurrencySpec {
	max := intakeConcurrencyMax
	return &yaml.ConcurrencySpec{Max: &max}
}

func ensureIntakeFilterNode(
	nodes []models.Node,
	edges []models.Edge,
	graph intakeGraph,
) ([]models.Node, []models.Edge, intakeGraph, error) {
	if graph.FilterNodeID != "" {
		return nodes, edges, graph, nil
	}
	if graph.TriggerNodeID == "" || graph.CreateNodeID == "" {
		return nil, nil, graph, fmt.Errorf("intake automation has no filter to update")
	}

	nodes = upsertIntakeNode(nodes, models.Node{
		ID:   intakeFilterNodeID,
		Name: "Matches filters?",
		Type: models.NodeTypeComponent,
		Ref: models.NodeRef{
			Component: &models.ComponentRef{Name: intakeFilterComponent},
		},
		Configuration: map[string]any{
			"expression": "true",
		},
		Position:    models.Position{X: 160, Y: 260},
		Concurrency: intakeModelConcurrency(),
	})
	graph.FilterNodeID = intakeFilterNodeID

	edges = slices.DeleteFunc(edges, func(edge models.Edge) bool {
		return edge.SourceID == graph.TriggerNodeID && edge.TargetID == graph.CreateNodeID
	})
	edges = ensureIntakeEdge(edges, models.Edge{
		Channel:  "default",
		SourceID: graph.TriggerNodeID,
		TargetID: intakeFilterNodeID,
	})
	edges = ensureIntakeEdge(edges, models.Edge{
		Channel:  "true",
		SourceID: intakeFilterNodeID,
		TargetID: graph.CreateNodeID,
	})
	return nodes, edges, graph, nil
}

func configureIntakeAuthorAccess(
	nodes []models.Node,
	edges []models.Edge,
	graph intakeGraph,
	enabled bool,
) ([]models.Node, []models.Edge, error) {
	if !enabled {
		nodes = slices.DeleteFunc(nodes, func(node models.Node) bool {
			return node.ID == graph.AuthorPermissionNodeID || node.ID == graph.AuthorFilterNodeID
		})
		edges = slices.DeleteFunc(edges, func(edge models.Edge) bool {
			return edge.SourceID == graph.AuthorPermissionNodeID ||
				edge.TargetID == graph.AuthorPermissionNodeID ||
				edge.SourceID == graph.AuthorFilterNodeID ||
				edge.TargetID == graph.AuthorFilterNodeID
		})
		return nodes, ensureIntakeEdge(edges, models.Edge{
			Channel:  "true",
			SourceID: graph.FilterNodeID,
			TargetID: graph.CreateNodeID,
		}), nil
	}

	trigger := findIntakeNode(nodes, graph.TriggerNodeID)
	if trigger == nil {
		return nil, nil, fmt.Errorf("intake automation has no GitHub trigger")
	}
	repository, _ := trigger.Configuration["repository"].(string)
	if strings.TrimSpace(repository) == "" {
		return nil, nil, fmt.Errorf("intake automation has no GitHub repository")
	}
	if trigger.IntegrationID == nil || strings.TrimSpace(*trigger.IntegrationID) == "" {
		return nil, nil, fmt.Errorf("intake automation has no GitHub integration")
	}

	permissionNode := models.Node{
		ID:   intakeAuthorPermissionNodeID,
		Name: "Get Author Repository Permission",
		Type: models.NodeTypeComponent,
		Ref: models.NodeRef{
			Component: &models.ComponentRef{Name: intakeAuthorPermissionComponent},
		},
		Configuration: map[string]any{
			"repository": repository,
			"username":   "{{ root().data.issue.user.login }}",
		},
		Position:      models.Position{X: 160, Y: 440},
		Concurrency:   intakeModelConcurrency(),
		IntegrationID: trigger.IntegrationID,
	}
	authorFilterNode := models.Node{
		ID:   intakeAuthorFilterNodeID,
		Name: "Author Has Repository Access?",
		Type: models.NodeTypeComponent,
		Ref: models.NodeRef{
			Component: &models.ComponentRef{Name: intakeFilterComponent},
		},
		Configuration: map[string]any{
			"expression": `root().data.permission != "none"`,
		},
		Position:    models.Position{X: 160, Y: 620},
		Concurrency: intakeModelConcurrency(),
	}
	nodes = upsertIntakeNode(nodes, permissionNode)
	nodes = upsertIntakeNode(nodes, authorFilterNode)

	edges = slices.DeleteFunc(edges, func(edge models.Edge) bool {
		return edge.SourceID == graph.FilterNodeID &&
			(edge.TargetID == graph.CreateNodeID || edge.TargetID == intakeAuthorPermissionNodeID)
	})
	edges = ensureIntakeEdge(edges, models.Edge{
		Channel:  "true",
		SourceID: graph.FilterNodeID,
		TargetID: intakeAuthorPermissionNodeID,
	})
	edges = ensureIntakeEdge(edges, models.Edge{
		Channel:  "default",
		SourceID: intakeAuthorPermissionNodeID,
		TargetID: intakeAuthorFilterNodeID,
	})
	edges = ensureIntakeEdge(edges, models.Edge{
		Channel:  "true",
		SourceID: intakeAuthorFilterNodeID,
		TargetID: graph.CreateNodeID,
	})
	return nodes, edges, nil
}

func intakeModelConcurrency() *models.ConcurrencySpec {
	max := intakeConcurrencyMax
	return &models.ConcurrencySpec{Max: &max}
}

func upsertIntakeNode(nodes []models.Node, updated models.Node) []models.Node {
	for i := range nodes {
		if nodes[i].ID == updated.ID {
			nodes[i] = updated
			return nodes
		}
	}
	return append(nodes, updated)
}

func ensureIntakeEdge(edges []models.Edge, expected models.Edge) []models.Edge {
	if slices.Contains(edges, expected) {
		return edges
	}
	return append(edges, expected)
}

// intakeTriggerConfiguration lays the binding over the template so the trigger
// listens on a concrete resource. The template map is shared between intakes,
// so it is copied rather than written to.
func intakeTriggerConfiguration(spec intakeSpec, binding *intakeBinding) map[string]any {
	configuration := make(map[string]any, len(spec.triggerConfiguration)+len(binding.configuration()))
	for name, value := range spec.triggerConfiguration {
		configuration[name] = value
	}
	for name, value := range binding.configuration() {
		configuration[name] = value
	}

	return configuration
}

func intakeTriggerMetadata(source string, settings intakeSettings) map[string]any {
	if source != models.FactoryIntakeSourceJiraIssues {
		return nil
	}
	return jiraCompletionMetadata(intakeSettingsOrDefault(source, settings))
}

func intakeSettingsOrDefault(source string, settings intakeSettings) intakeSettings {
	if settings.ConfidencePct != 0 {
		return settings
	}
	return defaultIntakeSettingsFor(source)
}

func intakeRefinementConfiguration(agent *intakeAgent, githubName string) map[string]any {
	configuration := intakeRunnerConfiguration(agent, githubName)
	configuration["steps"] = []any{
		map[string]any{
			"name":    "Clone repository",
			"type":    runner.AgentStepBash,
			"command": intakeAnalysisCloneCommand(),
		},
		map[string]any{
			"name":             "Refine Task",
			"type":             "prompt",
			"workingDirectory": "repo",
			"prompt":           intakeRefinementPrompt(),
		},
	}
	return configuration
}

func intakeRunnerConfiguration(agent *intakeAgent, githubName string) map[string]any {
	if strings.TrimSpace(githubName) == "" {
		githubName = intakeGitHubAppName
	}

	configuration := map[string]any{
		"machineType":             intakeAnalysisMachineType,
		"executionTimeoutSeconds": intakeAnalysisTimeoutSeconds,
		"environmentFrom": []any{
			map[string]any{
				"source": "integration",
				"integration": map[string]any{
					"name": githubName,
				},
			},
		},
		"environment": []any{
			map[string]any{
				"name":        "REPO_URL",
				"value":       "{{ root().data.workOrder.repository_url }}",
				"valueSource": "literal",
			},
			map[string]any{
				"name":        "BASE",
				"value":       "{{ root().data.workOrder.default_branch }}",
				"valueSource": "literal",
			},
		},
	}

	if credentials := agent.credentials(); credentials != nil {
		configuration["credentials"] = credentials
	}
	if model := agent.model(); model != "" {
		configuration["model"] = model
	}

	return configuration
}

func intakeRefinementPrompt() string {
	return runner.PlanningSessionUserPromptMarkdown() + "\n\nTask:\n{{ root().data.workOrder }}"
}

func intakeAnalysisCloneCommand() string {
	return strings.Join([]string{
		"set -euo pipefail",
		`if [ -z "${REPO_URL:-}" ]; then`,
		`  echo "This workspace has no repository to analyze." >&2`,
		"  exit 1",
		"fi",
		`git config --global url."https://x-access-token:${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"`,
		"rm -rf repo",
		`git clone --depth 1 --branch "${BASE:-main}" "${REPO_URL}" repo`,
	}, "\n")
}

var intakeThresholdPattern = regexp.MustCompile(`>=\s*(\d+)`)

// intakeConfidenceFromExpression reads a legacy score gate back out of a
// generated expression. A hand-edited expression that no longer matches
// reports false so callers can leave the value alone instead of guessing.
func intakeConfidenceFromExpression(expression string) (int, bool) {
	match := intakeThresholdPattern.FindStringSubmatch(expression)
	if match == nil {
		return 0, false
	}

	confidence, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, false
	}

	return clampIntakeConfidence(confidence), true
}

func clampIntakeConfidence(confidencePct int) int {
	if confidencePct < 0 {
		return 0
	}
	if confidencePct > 100 {
		return 100
	}
	return confidencePct
}
