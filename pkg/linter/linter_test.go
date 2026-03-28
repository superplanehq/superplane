package linter

import (
	"os"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func triggerNode(id, name string) models.Node {
	return models.Node{
		ID:            id,
		Name:          name,
		Type:          "TYPE_TRIGGER",
		Ref:           models.NodeRef{Trigger: &models.TriggerRef{Name: "pagerduty.onIncident"}},
		Configuration: map[string]any{},
	}
}

func componentNode(id, name, componentName string, config map[string]any) models.Node {
	return models.Node{
		ID:            id,
		Name:          name,
		Type:          "TYPE_COMPONENT",
		Ref:           models.NodeRef{Component: &models.ComponentRef{Name: componentName}},
		Configuration: config,
	}
}

func widgetNode(id, name string) models.Node {
	return models.Node{
		ID:            id,
		Name:          name,
		Type:          "TYPE_WIDGET",
		Ref:           models.NodeRef{Widget: &models.WidgetRef{Name: "annotation"}},
		Configuration: map[string]any{},
	}
}

func edge(src, tgt, channel string) models.Edge {
	return models.Edge{SourceID: src, TargetID: tgt, Channel: channel}
}

// countIssuesByRule returns the number of issues with the given rule name.
func countIssuesByRule(issues []LintIssue, rule string) int {
	count := 0
	for _, i := range issues {
		if i.Rule == rule {
			count++
		}
	}
	return count
}

// ---------------------------------------------------------------------------
// Original tests (existing rules)
// ---------------------------------------------------------------------------

func TestLintCanvas_HealthyCanvas(t *testing.T) {
	// Full valid flow: trigger -> filter -> 3 parallel -> merge -> claude -> slack -> approval
	nodes := []models.Node{
		triggerNode("t1", "Listen for incidents"),
		componentNode("f1", "Is it P1", "filter", map[string]any{
			"expression": `$["Listen for incidents"].data.priority == "P1"`,
		}),
		componentNode("c1", "Get deploy", "github.getRelease", nil),
		componentNode("c2", "Get metrics", "http", map[string]any{"url": "https://api.example.com"}),
		componentNode("c3", "Get logs", "pagerduty.listLogEntries", nil),
		componentNode("m1", "Wait for all", "merge", nil),
		componentNode("ai", "AI Assessment", "claude.textPrompt", map[string]any{
			"prompt": "Analyze the incident: {{ $[\"Listen for incidents\"].data.title }}",
		}),
		componentNode("sl", "Notify Slack", "slack.sendTextMessage", map[string]any{
			"channel": "#incidents",
		}),
		componentNode("ap", "Approve", "approval", nil),
	}
	edges := []models.Edge{
		edge("t1", "f1", "default"),
		edge("f1", "c1", "default"),
		edge("f1", "c2", "default"),
		edge("f1", "c3", "default"),
		edge("c1", "m1", "default"),
		edge("c2", "m1", "default"),
		edge("c3", "m1", "default"),
		edge("m1", "ai", "success"),
		edge("ai", "sl", "default"),
		edge("sl", "ap", "default"),
	}

	result := LintCanvas(nodes, edges, nil)

	assert.Equal(t, "pass", result.Status)
	assert.Empty(t, result.Errors)
	assert.Equal(t, 9, result.Summary.TotalNodes)
	assert.Equal(t, 10, result.Summary.TotalEdges)
}

func TestLintCanvas_EmptyCanvas(t *testing.T) {
	result := LintCanvas(nil, nil, nil)

	assert.Equal(t, "pass", result.Status)
	assert.Empty(t, result.Errors)
	assert.Empty(t, result.Warnings)
	assert.Empty(t, result.Info)
	assert.Equal(t, 0, result.Summary.TotalNodes)
	assert.Equal(t, 0, result.Summary.TotalEdges)
}

func TestLintCanvas_OrphanNode(t *testing.T) {
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
		componentNode("c1", "Connected", "http", map[string]any{"url": "https://example.com"}),
		componentNode("orphan", "Orphaned Node", "http", map[string]any{"url": "https://example.com"}),
	}
	edges := []models.Edge{
		edge("t1", "c1", "default"),
	}

	result := LintCanvas(nodes, edges, nil)

	assert.Equal(t, 1, countIssuesByRule(result.Warnings, "orphan-node"))
	found := false
	for _, w := range result.Warnings {
		if w.Rule == "orphan-node" {
			assert.Equal(t, "Orphaned Node", w.NodeName)
			found = true
		}
	}
	assert.True(t, found)
}

func TestLintCanvas_DeadEnd(t *testing.T) {
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
		componentNode("c1", "Dead End Node", "http", map[string]any{"url": "https://example.com"}),
	}
	edges := []models.Edge{
		edge("t1", "c1", "default"),
	}

	result := LintCanvas(nodes, edges, nil)

	deadEnds := countIssuesByRule(result.Warnings, "dead-end")
	assert.Equal(t, 1, deadEnds)

	for _, w := range result.Warnings {
		if w.Rule == "dead-end" {
			assert.Equal(t, "Dead End Node", w.NodeName)
		}
	}
}

func TestLintCanvas_DeadEnd_TerminalOK(t *testing.T) {
	// All terminal components should not produce dead-end warnings.
	terminals := []struct {
		name      string
		component string
	}{
		{"Approve", "approval"},
		{"Slack", "slack.sendTextMessage"},
		{"Create Issue", "github.createIssue"},
		{"Create PD", "pagerduty.createIncident"},
		{"Resolve PD", "pagerduty.resolveIncident"},
	}

	for _, tc := range terminals {
		t.Run(tc.component, func(t *testing.T) {
			nodes := []models.Node{
				triggerNode("t1", "Trigger"),
				componentNode("term", tc.name, tc.component, nil),
			}
			edges := []models.Edge{
				edge("t1", "term", "default"),
			}

			result := LintCanvas(nodes, edges, nil)
			assert.Equal(t, 0, countIssuesByRule(result.Warnings, "dead-end"))
		})
	}
}

func TestLintCanvas_MissingApprovalGate(t *testing.T) {
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
		componentNode("d1", "Resolve Incident", "pagerduty.resolveIncident", nil),
	}
	edges := []models.Edge{
		edge("t1", "d1", "default"),
	}

	result := LintCanvas(nodes, edges, nil)

	assert.Equal(t, "fail", result.Status)
	require.Equal(t, 1, countIssuesByRule(result.Errors, "missing-approval-gate"))

	for _, e := range result.Errors {
		if e.Rule == "missing-approval-gate" {
			assert.Equal(t, "Resolve Incident", e.NodeName)
			assert.Contains(t, e.Message, "pagerduty.resolveIncident")
		}
	}
}

func TestLintCanvas_ApprovalGatePresent(t *testing.T) {
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
		componentNode("ap", "Approve First", "approval", nil),
		componentNode("d1", "Resolve Incident", "pagerduty.resolveIncident", nil),
	}
	edges := []models.Edge{
		edge("t1", "ap", "default"),
		edge("ap", "d1", "default"),
	}

	result := LintCanvas(nodes, edges, nil)

	assert.Equal(t, 0, countIssuesByRule(result.Errors, "missing-approval-gate"))
}

func TestLintCanvas_MissingConfig_EmptyPrompt(t *testing.T) {
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
		componentNode("ai", "AI Node", "claude.textPrompt", map[string]any{
			"prompt": "",
		}),
	}
	edges := []models.Edge{
		edge("t1", "ai", "default"),
	}

	result := LintCanvas(nodes, edges, nil)

	configErrors := 0
	for _, e := range result.Errors {
		if e.Rule == "missing-required-config" && e.NodeName == "AI Node" {
			configErrors++
			assert.Contains(t, e.Message, "prompt")
		}
	}
	assert.Equal(t, 1, configErrors)
}

func TestLintCanvas_MissingConfig_MergeSingleInput(t *testing.T) {
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
		componentNode("m1", "Solo Merge", "merge", nil),
	}
	edges := []models.Edge{
		edge("t1", "m1", "default"),
	}

	result := LintCanvas(nodes, edges, nil)

	mergeInfo := 0
	for _, i := range result.Info {
		if i.Rule == "missing-required-config" && i.NodeName == "Solo Merge" {
			mergeInfo++
			assert.Contains(t, i.Message, "1 incoming edge")
		}
	}
	assert.Equal(t, 1, mergeInfo)
}

func TestLintCanvas_MissingConfig_FilterNoExpression(t *testing.T) {
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
		componentNode("f1", "Empty Filter", "filter", map[string]any{
			"expression": "",
		}),
	}
	edges := []models.Edge{
		edge("t1", "f1", "default"),
	}

	result := LintCanvas(nodes, edges, nil)

	filterErrors := 0
	for _, e := range result.Errors {
		if e.Rule == "missing-required-config" && e.NodeName == "Empty Filter" {
			filterErrors++
			assert.Contains(t, e.Message, "expression")
		}
	}
	assert.Equal(t, 1, filterErrors)
}

func TestLintCanvas_InvalidExpression_UnbalancedBraces(t *testing.T) {
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
		componentNode("c1", "Bad Expr", "http", map[string]any{
			"url": "{{ no closing",
		}),
	}
	edges := []models.Edge{
		edge("t1", "c1", "default"),
	}

	result := LintCanvas(nodes, edges, nil)

	exprErrors := 0
	for _, e := range result.Errors {
		if e.Rule == "invalid-expression" {
			exprErrors++
			assert.Contains(t, e.Message, "unbalanced")
		}
	}
	assert.Equal(t, 1, exprErrors)
}

func TestLintCanvas_InvalidExpression_BadNodeRef(t *testing.T) {
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
		componentNode("c1", "Bad Ref", "http", map[string]any{
			"url": `{{ $["Nonexistent Node"].data }}`,
		}),
	}
	edges := []models.Edge{
		edge("t1", "c1", "default"),
	}

	result := LintCanvas(nodes, edges, nil)

	refWarnings := 0
	for _, w := range result.Warnings {
		if w.Rule == "invalid-expression" {
			refWarnings++
			assert.Contains(t, w.Message, "Nonexistent Node")
		}
	}
	assert.Equal(t, 1, refWarnings)
}

func TestLintCanvas_ValidExpression(t *testing.T) {
	nodes := []models.Node{
		triggerNode("t1", "Listen for incidents"),
		componentNode("c1", "Use Data", "http", map[string]any{
			"url": `{{ $["Listen for incidents"].data.field }}`,
		}),
	}
	edges := []models.Edge{
		edge("t1", "c1", "default"),
	}

	result := LintCanvas(nodes, edges, nil)

	assert.Equal(t, 0, countIssuesByRule(result.Warnings, "invalid-expression"))
	assert.Equal(t, 0, countIssuesByRule(result.Errors, "invalid-expression"))
}

func TestLintCanvas_UnreachableBranch(t *testing.T) {
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
		componentNode("f1", "Filter Without Default", "filter", map[string]any{
			"expression": "true",
		}),
		componentNode("c1", "On Match", "http", map[string]any{"url": "https://example.com"}),
	}
	edges := []models.Edge{
		edge("t1", "f1", "default"),
		edge("f1", "c1", "match"), // not "default"
	}

	result := LintCanvas(nodes, edges, nil)

	branchInfo := 0
	for _, i := range result.Info {
		if i.Rule == "unreachable-branch" {
			branchInfo++
			assert.Equal(t, "Filter Without Default", i.NodeName)
		}
	}
	assert.Equal(t, 1, branchInfo)
}

func TestLintCanvas_WidgetsIgnored(t *testing.T) {
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
		componentNode("c1", "Connected", "approval", nil),
		widgetNode("w1", "My Annotation"),
		widgetNode("w2", "Another Note"),
	}
	edges := []models.Edge{
		edge("t1", "c1", "default"),
		// Widgets are not connected to anything — they should not produce warnings.
	}

	result := LintCanvas(nodes, edges, nil)

	for _, w := range result.Warnings {
		assert.NotEqual(t, "orphan-node", w.Rule, "widgets should not produce orphan-node warnings")
		assert.NotEqual(t, "dead-end", w.Rule, "widgets should not produce dead-end warnings")
	}
}

// ---------------------------------------------------------------------------
// New tests for C1: Cycle detection
// ---------------------------------------------------------------------------

func TestLintCanvas_CycleDetected(t *testing.T) {
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
		componentNode("a", "Node A", "http", map[string]any{"url": "https://a.com"}),
		componentNode("b", "Node B", "http", map[string]any{"url": "https://b.com"}),
		componentNode("c", "Node C", "http", map[string]any{"url": "https://c.com"}),
	}
	edges := []models.Edge{
		edge("t1", "a", "default"),
		edge("a", "b", "default"),
		edge("b", "c", "default"),
		edge("c", "a", "default"), // cycle: a -> b -> c -> a
	}

	result := LintCanvas(nodes, edges, nil)

	assert.Equal(t, "fail", result.Status)
	assert.Equal(t, 1, countIssuesByRule(result.Errors, "cycle-detected"))
}

func TestLintCanvas_NoCycle(t *testing.T) {
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
		componentNode("a", "Node A", "approval", nil),
	}
	edges := []models.Edge{
		edge("t1", "a", "default"),
	}

	result := LintCanvas(nodes, edges, nil)

	assert.Equal(t, 0, countIssuesByRule(result.Errors, "cycle-detected"))
}

// ---------------------------------------------------------------------------
// New tests for C6: Duplicate node detection
// ---------------------------------------------------------------------------

func TestLintCanvas_DuplicateNodeID(t *testing.T) {
	nodes := []models.Node{
		triggerNode("dup", "Trigger One"),
		componentNode("dup", "Trigger Two", "approval", nil),
	}
	edges := []models.Edge{
		edge("dup", "dup", "default"),
	}

	result := LintCanvas(nodes, edges, nil)

	assert.Equal(t, "fail", result.Status)
	assert.GreaterOrEqual(t, countIssuesByRule(result.Errors, "duplicate-node-id"), 1)
}

func TestLintCanvas_DuplicateNodeName(t *testing.T) {
	nodes := []models.Node{
		triggerNode("t1", "Same Name"),
		componentNode("c1", "Same Name", "approval", nil),
	}
	edges := []models.Edge{
		edge("t1", "c1", "default"),
	}

	result := LintCanvas(nodes, edges, nil)

	assert.Equal(t, 1, countIssuesByRule(result.Warnings, "duplicate-node-name"))
}

// ---------------------------------------------------------------------------
// New tests for C7: Edge validation
// ---------------------------------------------------------------------------

func TestLintCanvas_DanglingEdge(t *testing.T) {
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
	}
	edges := []models.Edge{
		edge("t1", "nonexistent", "default"),
	}

	result := LintCanvas(nodes, edges, nil)

	assert.Equal(t, "fail", result.Status)
	assert.GreaterOrEqual(t, countIssuesByRule(result.Errors, "invalid-edge"), 1)
}

func TestLintCanvas_SelfLoop(t *testing.T) {
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
		componentNode("c1", "Self Looper", "http", map[string]any{"url": "https://example.com"}),
	}
	edges := []models.Edge{
		edge("t1", "c1", "default"),
		edge("c1", "c1", "default"), // self-loop
	}

	result := LintCanvas(nodes, edges, nil)

	assert.GreaterOrEqual(t, countIssuesByRule(result.Errors, "invalid-edge"), 1)
	found := false
	for _, e := range result.Errors {
		if e.Rule == "invalid-edge" && e.NodeID == "c1" {
			assert.Contains(t, e.Message, "self-loop")
			found = true
		}
	}
	assert.True(t, found)
}

func TestLintCanvas_DuplicateEdge(t *testing.T) {
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
		componentNode("c1", "Target", "approval", nil),
	}
	edges := []models.Edge{
		edge("t1", "c1", "default"),
		edge("t1", "c1", "default"), // duplicate
	}

	result := LintCanvas(nodes, edges, nil)

	assert.Equal(t, 1, countIssuesByRule(result.Warnings, "duplicate-edge"))
}

func TestLintCanvas_WidgetAsEdgeEndpoint(t *testing.T) {
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
		widgetNode("w1", "Annotation"),
	}
	edges := []models.Edge{
		edge("t1", "w1", "default"),
	}

	result := LintCanvas(nodes, edges, nil)

	assert.GreaterOrEqual(t, countIssuesByRule(result.Errors, "invalid-edge"), 1)
}

// ---------------------------------------------------------------------------
// New tests for C9: Multiple destructive components
// ---------------------------------------------------------------------------

func TestLintCanvas_MultipleDestructiveComponents_SingleApproval(t *testing.T) {
	// One approval should cover all downstream destructive actions.
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
		componentNode("ap", "Approve", "approval", nil),
		componentNode("d1", "Resolve", "pagerduty.resolveIncident", nil),
		componentNode("d2", "Escalate", "pagerduty.escalateIncident", nil),
	}
	edges := []models.Edge{
		edge("t1", "ap", "default"),
		edge("ap", "d1", "default"),
		edge("d1", "d2", "default"),
	}

	result := LintCanvas(nodes, edges, nil)

	assert.Equal(t, 0, countIssuesByRule(result.Errors, "missing-approval-gate"),
		"single upstream approval should satisfy both destructive nodes")
}

func TestLintCanvas_MultipleDestructiveComponents_OneWithout(t *testing.T) {
	// One destructive action has approval, the other does not.
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
		componentNode("ap", "Approve", "approval", nil),
		componentNode("d1", "Resolve", "pagerduty.resolveIncident", nil),
		componentNode("d2", "Delete Release", "github.deleteRelease", nil),
	}
	edges := []models.Edge{
		edge("t1", "ap", "default"),
		edge("ap", "d1", "default"),
		edge("t1", "d2", "default"), // d2 bypasses approval
	}

	result := LintCanvas(nodes, edges, nil)

	assert.Equal(t, 1, countIssuesByRule(result.Errors, "missing-approval-gate"))
	for _, e := range result.Errors {
		if e.Rule == "missing-approval-gate" {
			assert.Equal(t, "Delete Release", e.NodeName)
		}
	}
}

// ---------------------------------------------------------------------------
// New test for C10: Nil Configuration map
// ---------------------------------------------------------------------------

func TestLintCanvas_NilConfiguration(t *testing.T) {
	// Nodes with nil Configuration should not panic.
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
		{
			ID:            "nil-config",
			Name:          "Nil Config Claude",
			Type:          "TYPE_COMPONENT",
			Ref:           models.NodeRef{Component: &models.ComponentRef{Name: "claude.textPrompt"}},
			Configuration: nil, // explicitly nil
		},
	}
	edges := []models.Edge{
		edge("t1", "nil-config", "default"),
	}

	// Should not panic.
	result := LintCanvas(nodes, edges, nil)

	// Should report missing prompt config error.
	assert.Equal(t, "fail", result.Status)
	configErrors := 0
	for _, e := range result.Errors {
		if e.Rule == "missing-required-config" && e.NodeName == "Nil Config Claude" {
			configErrors++
		}
	}
	assert.Equal(t, 1, configErrors)
}

// ---------------------------------------------------------------------------
// New test for C11: Deeply nested configuration values
// ---------------------------------------------------------------------------

func TestLintCanvas_NestedConfigExpression(t *testing.T) {
	nodes := []models.Node{
		triggerNode("t1", "Listen for incidents"),
		componentNode("c1", "HTTP with headers", "http", map[string]any{
			"url": "https://api.example.com",
			"headers": map[string]any{
				"Authorization": `Bearer {{ $["Listen for incidents"].data.token }}`,
			},
		}),
	}
	edges := []models.Edge{
		edge("t1", "c1", "default"),
	}

	result := LintCanvas(nodes, edges, nil)

	// Should find the valid reference in nested config — no warnings.
	assert.Equal(t, 0, countIssuesByRule(result.Warnings, "invalid-expression"))
}

func TestLintCanvas_NestedConfigBadRef(t *testing.T) {
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
		componentNode("c1", "HTTP nested bad ref", "http", map[string]any{
			"url": "https://api.example.com",
			"headers": map[string]any{
				"X-Custom": `{{ $["Ghost Node"].data.value }}`,
			},
		}),
	}
	edges := []models.Edge{
		edge("t1", "c1", "default"),
	}

	result := LintCanvas(nodes, edges, nil)

	assert.Equal(t, 1, countIssuesByRule(result.Warnings, "invalid-expression"))
}

// ---------------------------------------------------------------------------
// New test for C3: Expression regex with quotes in node names
// ---------------------------------------------------------------------------

func TestLintCanvas_ExpressionSingleQuoteRef(t *testing.T) {
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
		componentNode("c1", "Node's Data", "http", map[string]any{"url": "https://example.com"}),
		componentNode("c2", "Consumer", "http", map[string]any{
			// Double-quote reference to a node name containing a single quote
			"url": `{{ $["Node's Data"].data.field }}`,
		}),
	}
	edges := []models.Edge{
		edge("t1", "c1", "default"),
		edge("c1", "c2", "default"),
	}

	result := LintCanvas(nodes, edges, nil)

	// Should correctly parse the double-quoted reference containing a single quote.
	assert.Equal(t, 0, countIssuesByRule(result.Warnings, "invalid-expression"))
}

// ---------------------------------------------------------------------------
// YAML parsing types for the dogfood test
// ---------------------------------------------------------------------------

type canvasYAML struct {
	Spec struct {
		Nodes []nodeYAML `json:"nodes"`
		Edges []edgeYAML `json:"edges"`
	} `json:"spec"`
}

type nodeYAML struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Type          string         `json:"type"`
	Configuration map[string]any `json:"configuration"`
	Component     *struct {
		Name string `json:"name"`
	} `json:"component"`
	Trigger *struct {
		Name string `json:"name"`
	} `json:"trigger"`
	Widget *struct {
		Name string `json:"name"`
	} `json:"widget"`
	Blueprint *struct {
		ID string `json:"id"`
	} `json:"blueprint"`
}

type edgeYAML struct {
	SourceID string `json:"sourceId"`
	TargetID string `json:"targetId"`
	Channel  string `json:"channel"`
}

// C8 fix: dogfood test now asserts specific expected warnings and rejects unexpected ones.
func TestLintCanvas_IncidentCopilotTemplate(t *testing.T) {
	data, err := os.ReadFile("../../templates/canvases/incident-copilot.yaml")
	require.NoError(t, err, "failed to read incident-copilot.yaml template")

	var canvas canvasYAML
	err = yaml.Unmarshal(data, &canvas)
	require.NoError(t, err, "failed to parse incident-copilot.yaml")

	// Convert YAML nodes to models.Node.
	nodes := make([]models.Node, 0, len(canvas.Spec.Nodes))
	for _, yn := range canvas.Spec.Nodes {
		n := models.Node{
			ID:            yn.ID,
			Name:          yn.Name,
			Type:          yn.Type,
			Configuration: yn.Configuration,
		}
		if n.Configuration == nil {
			n.Configuration = map[string]any{}
		}
		if yn.Component != nil {
			n.Ref.Component = &models.ComponentRef{Name: yn.Component.Name}
		}
		if yn.Trigger != nil {
			n.Ref.Trigger = &models.TriggerRef{Name: yn.Trigger.Name}
		}
		if yn.Widget != nil {
			n.Ref.Widget = &models.WidgetRef{Name: yn.Widget.Name}
		}
		if yn.Blueprint != nil {
			n.Ref.Blueprint = &models.BlueprintRef{ID: yn.Blueprint.ID}
		}
		nodes = append(nodes, n)
	}

	// Convert YAML edges to models.Edge.
	edges := make([]models.Edge, 0, len(canvas.Spec.Edges))
	for _, ye := range canvas.Spec.Edges {
		edges = append(edges, models.Edge{
			SourceID: ye.SourceID,
			TargetID: ye.TargetID,
			Channel:  ye.Channel,
		})
	}

	result := LintCanvas(nodes, edges, nil)

	// The incident-copilot template should pass the linter with zero errors.
	assert.Equal(t, "pass", result.Status, "incident-copilot template should pass lint")
	assert.Empty(t, result.Errors, "incident-copilot template should have zero errors")

	// Verify we actually parsed a non-trivial canvas.
	assert.Greater(t, result.Summary.TotalNodes, 5, "should have parsed multiple nodes")
	assert.Greater(t, result.Summary.TotalEdges, 5, "should have parsed multiple edges")

	// With channel configured, there should be no warnings.
	assert.Empty(t, result.Warnings, "copilot template should have zero warnings with channel configured")

	// Assert no orphan nodes, no dead ends, no cycles.
	assert.Equal(t, 0, countIssuesByRule(result.Warnings, "orphan-node"), "no orphan nodes expected")
	assert.Equal(t, 0, countIssuesByRule(result.Warnings, "dead-end"), "no dead ends expected")
	assert.Equal(t, 0, countIssuesByRule(result.Errors, "cycle-detected"), "no cycles expected")

	// Assert info section is reasonable.
	for _, info := range result.Info {
		t.Logf("INFO: [%s] %s: %s", info.Rule, info.NodeName, info.Message)
	}

	// Quality score assertions.
	assert.GreaterOrEqual(t, result.QualityScore, 90, "copilot template should score >= 90")
	assert.Equal(t, GradeA, result.QualityGrade, "copilot template should be grade A")
	t.Logf("Quality: score=%d grade=%s", result.QualityScore, result.QualityGrade)
}

// ---------------------------------------------------------------------------
// Quality scoring tests
// ---------------------------------------------------------------------------

func TestQualityScore_Perfect(t *testing.T) {
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
		componentNode("c1", "End", "approval", nil),
	}
	edges := []models.Edge{
		edge("t1", "c1", "default"),
	}

	result := LintCanvas(nodes, edges, nil)

	assert.Equal(t, 100, result.QualityScore)
	assert.Equal(t, GradeA, result.QualityGrade)
}

func TestQualityScore_WithErrors(t *testing.T) {
	// Destructive component without approval = 1 error.
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
		componentNode("d1", "Resolve", "pagerduty.resolveIncident", nil),
	}
	edges := []models.Edge{
		edge("t1", "d1", "default"),
	}

	result := LintCanvas(nodes, edges, nil)

	assert.Equal(t, "fail", result.Status)
	// 1 error = -15 points -> score 85, grade B
	assert.Equal(t, 85, result.QualityScore)
	assert.Equal(t, GradeB, result.QualityGrade)
}

func TestQualityScore_ManyIssues(t *testing.T) {
	// 4 errors (missing-approval-gate) + 1 warning (dead-end on github.deleteRelease,
	// which is not in terminalComponents).
	// Error penalty: 4*15=60, capped at 60. Warning penalty: 1*5=5. Total: 65.
	// Score: 100-65=35, grade F.
	nodes := []models.Node{
		triggerNode("t1", "Trigger"),
		componentNode("d1", "Resolve", "pagerduty.resolveIncident", nil),
		componentNode("d2", "Escalate", "pagerduty.escalateIncident", nil),
		componentNode("d3", "Delete", "github.deleteRelease", nil),
		componentNode("d4", "Release", "github.createRelease", nil),
	}
	edges := []models.Edge{
		edge("t1", "d1", "default"),
		edge("t1", "d2", "default"),
		edge("t1", "d3", "default"),
		edge("t1", "d4", "default"),
	}

	result := LintCanvas(nodes, edges, nil)

	assert.Equal(t, "fail", result.Status)
	assert.LessOrEqual(t, result.QualityScore, 40, "many issues should produce low score")
	assert.GreaterOrEqual(t, result.Summary.ErrorCount, 4)
}

// ---------------------------------------------------------------------------
// Dogfood tests for existing templates
// ---------------------------------------------------------------------------

func loadTemplateForTest(t *testing.T, path string) ([]models.Node, []models.Edge) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read template: %s", path)

	var canvas canvasYAML
	err = yaml.Unmarshal(data, &canvas)
	require.NoError(t, err, "failed to parse template: %s", path)

	nodes := make([]models.Node, 0, len(canvas.Spec.Nodes))
	for _, yn := range canvas.Spec.Nodes {
		n := models.Node{
			ID:            yn.ID,
			Name:          yn.Name,
			Type:          yn.Type,
			Configuration: yn.Configuration,
		}
		if n.Configuration == nil {
			n.Configuration = map[string]any{}
		}
		if yn.Component != nil {
			n.Ref.Component = &models.ComponentRef{Name: yn.Component.Name}
		}
		if yn.Trigger != nil {
			n.Ref.Trigger = &models.TriggerRef{Name: yn.Trigger.Name}
		}
		if yn.Widget != nil {
			n.Ref.Widget = &models.WidgetRef{Name: yn.Widget.Name}
		}
		if yn.Blueprint != nil {
			n.Ref.Blueprint = &models.BlueprintRef{ID: yn.Blueprint.ID}
		}
		nodes = append(nodes, n)
	}

	edges := make([]models.Edge, 0, len(canvas.Spec.Edges))
	for _, ye := range canvas.Spec.Edges {
		edges = append(edges, models.Edge{
			SourceID: ye.SourceID,
			TargetID: ye.TargetID,
			Channel:  ye.Channel,
		})
	}

	return nodes, edges
}

func TestLintCanvas_IncidentDataCollectionTemplate(t *testing.T) {
	nodes, edges := loadTemplateForTest(t, "../../templates/canvases/incident-data-collection.yaml")

	result := LintCanvas(nodes, edges, nil)

	assert.Equal(t, "pass", result.Status, "incident-data-collection template should pass lint")
	assert.Empty(t, result.Errors, "incident-data-collection template should have zero errors")
	assert.Greater(t, result.Summary.TotalNodes, 3, "should have parsed multiple nodes")
	assert.Equal(t, 0, countIssuesByRule(result.Warnings, "orphan-node"))
	assert.Equal(t, 0, countIssuesByRule(result.Errors, "cycle-detected"))
	assert.Equal(t, GradeA, result.QualityGrade, "incident-data-collection should be grade A")

	t.Logf("Quality: score=%d grade=%s errors=%d warnings=%d",
		result.QualityScore, result.QualityGrade, result.Summary.ErrorCount, result.Summary.WarningCount)
}

func TestLintCanvas_IncidentRouterTemplate(t *testing.T) {
	nodes, edges := loadTemplateForTest(t, "../../templates/canvases/incident-router.yaml")

	result := LintCanvas(nodes, edges, nil)

	assert.Equal(t, "pass", result.Status, "incident-router template should pass lint")
	assert.Empty(t, result.Errors, "incident-router template should have zero errors")
	assert.Greater(t, result.Summary.TotalNodes, 3, "should have parsed multiple nodes")
	assert.Equal(t, 0, countIssuesByRule(result.Warnings, "orphan-node"))
	assert.Equal(t, 0, countIssuesByRule(result.Errors, "cycle-detected"))
	assert.Equal(t, GradeA, result.QualityGrade, "incident-router should be grade A")

	t.Logf("Quality: score=%d grade=%s errors=%d warnings=%d",
		result.QualityScore, result.QualityGrade, result.Summary.ErrorCount, result.Summary.WarningCount)
}
