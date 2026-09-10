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

	// The analysis node name is part of the generated backlog graph's contract:
	// the report-check fields read the score by this name.
	intakeAnalysisNodeName = "Analyze intake"
	intakeCreateNodeName   = "Create Task"

	intakeFilterComponent           = "if"
	intakeAuthorPermissionComponent = "github.getRepositoryPermission"
	intakeThresholdComponent        = intakeFilterComponent
	intakeCreateComponent           = "createWorkOrder"
	intakeReportConfidenceComponent = "reportWorkOrderCheck"

	intakeConfidenceCheckKey  = "confidence"
	intakeConfidenceCheckName = "Confidence score"
	intakeConfidenceScoreMax  = 5
	intakeConfidenceFormat    = "fraction"
	intakeConfidenceDirection = "higherIsBetter"

	// Band edges of the confidence meter, which reads High from 4, Medium at
	// 3, and Low below 3. The check has no neutral threshold, so Medium maps
	// to caution and Low maps to critical.
	intakeConfidenceCautionAt  = 3
	intakeConfidenceCriticalAt = 2

	intakeAnalysisOutputFile = "/tmp/intake-analysis.json"
	intakeIntentOutputFile   = "/tmp/spec.md"

	intakeIntentArtifactNodeID   = "attach-intent"
	intakeIntentArtifactNodeName = "Add spec"
	intakeIntentArtifactTitle    = "spec.md"

	intakeAddRunErrorNodeID    = "add-run-error"
	intakeAddRunErrorNodeName  = "Record Analysis Failure"
	intakeAddRunErrorComponent = "addRunError"
	intakeAddRunErrorMessage   = "The analysis agent failed. Open the agent logs to find the cause."

	intakeAnalysisTimeoutSeconds = 3600

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
	analysisSubject      string
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
		analysisSubject:      "GitHub issue",
		createTitle:          "{{ root().data.issue.title }}",
		createDescription:    "{{ root().data.issue.body }}",
	},
	models.FactoryIntakeSourceSentryExceptions: {
		name:                 "Sentry exceptions",
		description:          "Create a work order when a Sentry exception is reported.",
		triggerComponent:     "sentry.onIssue",
		triggerName:          "On Issue Event",
		triggerConfiguration: map[string]any{"actions": []any{"created", "unresolved"}},
		analysisSubject:      "Sentry exception",
		createTitle:          "{{ root().data.data.issue.title }}",
		createDescription:    "{{ root().data.data.issue.permalink }}",
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
		analysisSubject:   "PagerDuty incident",
		createTitle:       "{{ root().data.incident.title }}",
		createDescription: "{{ root().data.incident.html_url }}",
	},
	models.FactoryIntakeSourceProductiveTasks: {
		name:                 "Productive.io tasks",
		description:          "Create a work order when a Productive.io task is created.",
		triggerComponent:     "productive.onTask",
		triggerName:          "On Task",
		triggerConfiguration: map[string]any{"actions": []any{"created"}},
		analysisSubject:      "Productive.io task",
		createTitle:          "{{ root().data.data.attributes.title }}",
		createDescription:    "{{ root().data.data.attributes.description }}",
	},
}

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
// and assignment settings have somewhere to live. Confidence scoring happens
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

	nodes := []yaml.Node{
		{
			ID:            intakeTriggerNodeID,
			Name:          spec.triggerName,
			Type:          yaml.NodeTypeTrigger,
			Component:     spec.triggerComponent,
			Configuration: intakeTriggerConfiguration(spec, request.Binding),
			Integration:   request.Binding.integrationRef(),
			Position:      yaml.Position{X: 160, Y: 80},
		},
	}
	edges := []yaml.Edge{}
	createY := 260

	if request.Source == models.FactoryIntakeSourceGitHubIssues {
		nodes = append(nodes, yaml.Node{
			ID:        intakeFilterNodeID,
			Name:      "Matches filters?",
			Type:      yaml.NodeTypeAction,
			Component: intakeFilterComponent,
			Configuration: map[string]any{
				"expression": intakeFilterExpressionFor(request.Source, defaultIntakeSettings()),
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
		ID:        intakeCreateNodeID,
		Name:      intakeCreateNodeName,
		Type:      yaml.NodeTypeAction,
		Component: intakeCreateComponent,
		Configuration: map[string]any{
			"title":       spec.createTitle,
			"description": spec.createDescription,
		},
		Concurrency: intakeConcurrency(),
		Position:    yaml.Position{X: 160, Y: createY},
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

// intakeConcurrency returns the concurrency of one intake node. Each node owns
// its spec, so a later edit to one node cannot reach the others. Only action
// nodes take a spec; a trigger has no queue to widen.
func intakeConcurrency() *yaml.ConcurrencySpec {
	max := intakeConcurrencyMax
	return &yaml.ConcurrencySpec{Max: &max}
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

// intakeAnalysisConfiguration sets the machine, checkout, and steps. BYOK
// agents also receive credentials and a model.
func intakeAnalysisConfiguration(spec intakeSpec, agent *intakeAgent, githubName string) map[string]any {
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
		"steps": []any{
			map[string]any{
				"name":    "Clone repository",
				"type":    runner.AgentStepBash,
				"command": intakeAnalysisCloneCommand(),
			},
			map[string]any{
				"name":             "Analyze and score",
				"type":             "prompt",
				"workingDirectory": "repo",
				"prompt":           intakeAnalysisPrompt(spec.analysisSubject),
			},
			map[string]any{
				"name":    "Use analysis as output",
				"type":    runner.AgentStepBash,
				"command": intakeAnalysisOutputCommand(),
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

func intakeAnalysisPrompt(subject string) string {
	return strings.Join([]string{
		fmt.Sprintf("Analyze this %s against the repository checked out in the working directory.", subject),
		"Read the ticket and the code. Score how well an agent on this factory line can complete the work.",
		"Do not score from the title and description alone.",
		"",
		fmt.Sprintf("Write one JSON object to %s.", intakeAnalysisOutputFile),
		fmt.Sprintf("The file must parse with jq. Run `jq empty %s` and keep editing until it succeeds.", intakeAnalysisOutputFile),
		"Keys:",
		`- "score": integer from 0 through 100. A higher value means greater confidence.`,
		`- "summary": one sentence on how suitable the work is for an agent on this factory line.`,
		`- "reasons": exactly three short sentences that explain the score.`,
		"Write three reasons: what the item names, what already exists in this repository, and whether an agent can do the work.",
		"",
		fmt.Sprintf("Also write %s. This file is the specification a person reads before they start.", intakeIntentOutputFile),
		"After you write the files, call propose_spec with the full spec.md markdown. Call propose_confidence with the 0-5 score and the summary sentence.",
		"Do not edit the original request. Later turns update only the spec and the confidence score.",
		"The SuperPlane UI shows Executive summary as the short view. It shows the later headings as the Plan.",
		"Write a specification, not a chat note. Use short sentences and plain words. Use American English. Do not use contractions.",
		"Ground every claim in this repository. Name files, types, and functions that exist.",
		"Do not invent files or APIs. Do not start with a line such as Intent:.",
		"Do not add an Open questions section. Use a mermaid fence only when a diagram clarifies a flow.",
		"Map your 0-100 score to a 0-5 confidence with round(score / 20).",
		"",
		"Always start with this shape:",
		"# <outcome in 8 words or fewer>",
		"## Executive summary",
		"Copy this markdown shape. Do not add extra paragraphs before, between, or after the headings.",
		"<one sentence that names only the change>",
		"### Need",
		"- <what is missing today>",
		"- <second gap, if needed>",
		"### Result",
		"- <what a person will notice when the work is done>",
		"- <second result, if needed>",
		"### Stay the same",
		"- <one hard limit>",
		"Example:",
		"Show a next action on the empty billing page.",
		"### Need",
		"- The empty view shows a title and no action.",
		"### Result",
		"- The empty view tells the user how to add a payment method.",
		"### Stay the same",
		"- Do not change the page after a card exists.",
		"Rules for this section:",
		"Use 1 lead sentence, or 2 at most. Then only the three ### headings and their bullets.",
		"Need: 2 to 4 bullets. Result: 2 to 4 bullets. Stay the same: 1 to 3 bullets.",
		"Do not explain what the app is, what stack it uses, or how the repository is organized.",
		"Do not name files or APIs here. Put those in the Plan.",
		"Do not write why the change matters for the product or for later work.",
		"Do not write first person. Do not use ## headings inside this section.",
		"",
		"If confidence is 2 through 5, write 80 to 150 lines after the title. Add these markdown headings after the executive summary, in this order:",
		"## Problem",
		"What is missing or broken. Name the current types, functions, and files. Quote current behavior when it helps.",
		"## Scope",
		"In scope: what this change must do. Out of scope: what this change must not do. Use two short bullet lists.",
		"## Outcome",
		"What done looks like for a user and for the code. Be concrete. Cover create, edit, list, and show when those surfaces exist.",
		"## Approach",
		"Numbered steps an agent can follow. Write at least 5 steps. Each step names the file or seam to change and what to change there.",
		"## Files and seams",
		"A bullet list of existing paths to change, and new paths only when you must add a file. One reason per path. Include tests.",
		"## Acceptance",
		"A numbered list of checks. Include tests to add or run, and the command when you know it.",
		"## Risks",
		"Hard limits and likely failure points. Name what must stay the same. Do not write only keep the change small.",
		"",
		"If confidence is 2 or 3, the Risks section must also say what is uncertain and why. Do not pretend the work is clear.",
		"",
		"If confidence is 0 or 1, do not write Problem, Scope, Outcome, Approach, Files and seams, or Acceptance.",
		"After the executive summary, use only these headings:",
		"## Why not start",
		"Tell the reader not to start implementation until they refine the task. Say what is missing.",
		"## What would make this clear",
		"The top 3 changes that would make the task clear enough to start. Use a numbered list.",
		"",
		"Task:",
		"{{ root().data.workOrder }}",
	}, "\n")
}

// intakeAnalysisOutputCommand promotes the files the agent wrote to the node's
// result, so the rest of the graph reads fields instead of parsing text. The
// prompt asks for an exact shape, but this step accepts what an agent really
// produces: a quoted number, a missing summary, or a different number of
// reasons. Only the score and intent body are required.
func intakeAnalysisOutputCommand() string {
	return fmt.Sprintf(`if [ ! -s %s ]; then
  echo "The analysis wrote no spec.md" >&2
  exit 1
fi
if ! jq -ce --rawfile intent %s '{
  score: (.score | tonumber | floor),
  summary: ((.summary // "") | tostring),
  reasons: [(if (.reasons | type) == "array" then .reasons[] else empty end) | tostring],
  intent: ($intent | tostring)
}' %s > "$SUPERPLANE_RESULT_FILE"; then
  echo "The analysis at %s has no readable score" >&2
  exit 1
fi`, intakeIntentOutputFile, intakeIntentOutputFile, intakeAnalysisOutputFile, intakeAnalysisOutputFile)
}

func intakeIntentBodyExpression() string {
	return fmt.Sprintf(`{{ $[%q].data.result.intent }}`, intakeAnalysisNodeName)
}

func intakeIntentArtifactConfiguration() map[string]any {
	return map[string]any{
		"orderId":      intakeWorkOrderIDFromRootExpression(),
		"artifactType": "markdown",
		"title":        intakeIntentArtifactTitle,
		"body":         intakeIntentBodyExpression(),
	}
}

func intakeAnalysisScorePath() string {
	return fmt.Sprintf(`$[%q].data.result.score`, intakeAnalysisNodeName)
}

func intakeWorkOrderIDFromRootExpression() string {
	return `{{ root().data.workOrder.id }}`
}

func intakeConfidenceSummaryExpression() string {
	return fmt.Sprintf(`{{ $[%q].data.result.summary }}`, intakeAnalysisNodeName)
}

func intakeConfidenceWriteupExpression(subject string) string {
	intro := fmt.Sprintf(
		"The automation read this %s. It scored how suitable the work is for an agent on this factory line.",
		subject,
	)
	return fmt.Sprintf(
		`{{ %q + "\n\n### Why this score\n- " + join($[%q].data.result.reasons, "\n- ") }}`,
		intro,
		intakeAnalysisNodeName,
	)
}

// intakeConfidenceScoreExpression maps the analysis percentage to the 0–5
// scale of the work-order confidence meter. The meter rounds the score it
// receives, so the expression rounds too and both agree on the bar count.
func intakeConfidenceScoreExpression() string {
	pctPerPoint := 100 / intakeConfidenceScoreMax
	return fmt.Sprintf(
		`{{ int(round(int(%s) / %d.0)) }}`,
		intakeAnalysisScorePath(), pctPerPoint,
	)
}

func intakeConfidenceReportConfiguration(subject string) map[string]any {
	return map[string]any{
		"orderId":    intakeWorkOrderIDFromRootExpression(),
		"checkKey":   intakeConfidenceCheckKey,
		"name":       intakeConfidenceCheckName,
		"score":      intakeConfidenceScoreExpression(),
		"maxScore":   strconv.Itoa(intakeConfidenceScoreMax),
		"format":     intakeConfidenceFormat,
		"direction":  intakeConfidenceDirection,
		"cautionAt":  float64(intakeConfidenceCautionAt),
		"criticalAt": float64(intakeConfidenceCriticalAt),
		"summary":    intakeConfidenceSummaryExpression(),
		"analysis":   intakeConfidenceWriteupExpression(subject),
	}
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
