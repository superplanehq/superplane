package factories

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
)

const (
	// Node identifiers of a generated intake graph. An intake owns its whole
	// canvas, so the identifiers are fixed rather than derived from the source.
	intakeTriggerNodeID   = "trigger"
	intakeAnalysisNodeID  = "analyze"
	intakeThresholdNodeID = "threshold"
	intakeCreateNodeID    = "create-work-order"

	// The threshold expression reads the analysis result by node name, so the
	// name is part of the generated graph's contract.
	intakeAnalysisNodeName = "Analyze intake"

	intakeThresholdComponent = "if"
	intakeCreateComponent    = "createWorkOrder"

	DefaultIntakeConfidencePct = 65
)

// intakeAnalysisComponents are the runners an intake can score with. Creation
// always picks Claude Code, but a graph the user re-pointed at another runner
// still has to resolve.
var intakeAnalysisComponents = []string{
	"runnerClaudeCode",
	"runnerCodex",
	"runnerOpenRouter",
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
		description:          "Analyze new GitHub issues and create work orders for suitable changes.",
		triggerComponent:     "github.onIssue",
		triggerName:          "On Issue",
		triggerConfiguration: map[string]any{"actions": []any{"opened"}},
		analysisSubject:      "GitHub issue",
		createTitle:          "{{ root().data.issue.title }}",
		createDescription:    "{{ root().data.issue.body }}",
	},
	models.FactoryIntakeSourceSentryExceptions: {
		name:                 "Sentry exceptions",
		description:          "Analyze new Sentry exceptions and create work orders for suitable fixes.",
		triggerComponent:     "sentry.onIssue",
		triggerName:          "On Issue Event",
		triggerConfiguration: map[string]any{"actions": []any{"created", "unresolved"}},
		analysisSubject:      "Sentry exception",
		createTitle:          "{{ root().data.data.issue.title }}",
		createDescription:    "{{ root().data.data.issue.permalink }}",
	},
	models.FactoryIntakeSourcePagerDutyIncidents: {
		name:             "PagerDuty incidents",
		description:      "Analyze triggered PagerDuty incidents and create work orders for suitable follow-up work.",
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
// source, score the item with an agent, and create a work order when the score
// clears the threshold. The binding tells the trigger which installation and
// resource to listen on; without one the user completes the trigger by hand.
func buildIntakeCanvas(source, name string, confidencePct int, binding *intakeBinding) (*yaml.Canvas, error) {
	spec, ok := intakeSpecsBySource[source]
	if !ok {
		return nil, models.ErrFactoryIntakeSourceInvalid
	}

	if strings.TrimSpace(name) == "" {
		name = spec.name
	}

	return &yaml.Canvas{
		APIVersion: yaml.APIVersion,
		Kind:       yaml.KindCanvas,
		Metadata: &yaml.CanvasMetadata{
			Name:        name,
			Description: spec.description,
		},
		Spec: &yaml.CanvasSpec{
			Edges: []yaml.Edge{
				{Channel: "default", SourceID: intakeTriggerNodeID, TargetID: intakeAnalysisNodeID},
				{Channel: "passed", SourceID: intakeAnalysisNodeID, TargetID: intakeThresholdNodeID},
				{Channel: "true", SourceID: intakeThresholdNodeID, TargetID: intakeCreateNodeID},
			},
			Nodes: []yaml.Node{
				{
					ID:            intakeTriggerNodeID,
					Name:          spec.triggerName,
					Type:          yaml.NodeTypeTrigger,
					Component:     spec.triggerComponent,
					Configuration: intakeTriggerConfiguration(spec, binding),
					Integration:   binding.integrationRef(),
					Position:      yaml.Position{X: 160, Y: 80},
				},
				{
					ID:        intakeAnalysisNodeID,
					Name:      intakeAnalysisNodeName,
					Type:      yaml.NodeTypeAction,
					Component: intakeAnalysisComponents[0],
					Configuration: map[string]any{
						"steps": []any{
							map[string]any{
								"name":   "Analyze and score",
								"type":   "prompt",
								"prompt": intakeAnalysisPrompt(spec.analysisSubject),
							},
						},
					},
					Position: yaml.Position{X: 160, Y: 260},
				},
				{
					ID:        intakeThresholdNodeID,
					Name:      "Meets confidence threshold?",
					Type:      yaml.NodeTypeAction,
					Component: intakeThresholdComponent,
					Configuration: map[string]any{
						"expression": intakeThresholdExpression(confidencePct),
					},
					Position: yaml.Position{X: 160, Y: 440},
				},
				{
					ID:        intakeCreateNodeID,
					Name:      "Create Work Order",
					Type:      yaml.NodeTypeAction,
					Component: intakeCreateComponent,
					Configuration: map[string]any{
						"title":       spec.createTitle,
						"description": spec.createDescription,
					},
					Position: yaml.Position{X: 160, Y: 620},
				},
			},
		},
	}, nil
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

func intakeAnalysisPrompt(subject string) string {
	return strings.Join([]string{
		fmt.Sprintf("Analyze this %s and decide whether it is suitable for an engineering work order.", subject),
		"Consider impact, clarity, feasibility, and whether an engineer can take a concrete action.",
		"Return only one integer from 0 through 100. A higher value means greater confidence.",
		"",
		"Event:",
		"{{ root().data }}",
	}, "\n")
}

func intakeThresholdExpression(confidencePct int) string {
	return fmt.Sprintf(`int($[%q].data[0].result.result) >= %d`, intakeAnalysisNodeName, clampIntakeConfidence(confidencePct))
}

var intakeThresholdPattern = regexp.MustCompile(`>=\s*(\d+)`)

// intakeConfidenceFromExpression reads the threshold back out of a generated
// expression. A hand-edited expression that no longer matches reports false so
// callers can leave the value alone instead of guessing.
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
