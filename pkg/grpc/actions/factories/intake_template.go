package factories

import (
	"fmt"
	"regexp"
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
		triggerConfiguration: map[string]any{"actions": []any{"opened"}},
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

// intakeAnalysisConfiguration sets the machine and steps. BYOK agents also
// receive credentials and a model.
func intakeAnalysisConfiguration(spec intakeSpec, agent *intakeAgent) map[string]any {
	configuration := map[string]any{
		"machineType": intakeAnalysisMachineType,
		"steps": []any{
			map[string]any{
				"name":   "Analyze and score",
				"type":   "prompt",
				"prompt": intakeAnalysisPrompt(spec.analysisSubject),
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

func intakeAnalysisPrompt(subject string) string {
	return strings.Join([]string{
		fmt.Sprintf("Analyze this %s and decide whether it is suitable for an engineering work order.", subject),
		"Consider impact, clarity, feasibility, and whether an agent on this factory line can take a concrete action.",
		fmt.Sprintf("Write one JSON object to %s. Do not write the result to another file.", intakeAnalysisOutputFile),
		fmt.Sprintf("The file must parse with jq. Run `jq empty %s` and keep editing until it succeeds.", intakeAnalysisOutputFile),
		"Keys:",
		`- "score": integer from 0 through 100. A higher value means greater confidence.`,
		`- "summary": one sentence on how suitable the work is for an agent on this factory line.`,
		`- "reasons": exactly three short sentences that explain the score.`,
		"Write three reasons: what the item names, what already exists, and whether an agent can do the work.",
		"",
		"Event:",
		"{{ root().data }}",
	}, "\n")
}

// intakeAnalysisOutputCommand promotes the file the agent wrote to the node's
// result, so the rest of the graph reads fields instead of parsing text. The
// prompt asks for an exact shape, but this step accepts what an agent really
// produces: a quoted number, a missing summary, or a different number of
// reasons. Only the score is required, because the report check cannot run
// without it.
func intakeAnalysisOutputCommand() string {
	return fmt.Sprintf(`if ! jq -ce '{
  score: (.score | tonumber | floor),
  summary: ((.summary // "") | tostring),
  reasons: [(if (.reasons | type) == "array" then .reasons[] else empty end) | tostring]
}' %s > "$SUPERPLANE_RESULT_FILE"; then
  echo "The analysis at %s has no readable score" >&2
  exit 1
fi`, intakeAnalysisOutputFile, intakeAnalysisOutputFile)
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
