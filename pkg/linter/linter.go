package linter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

// Severity indicates how critical a lint issue is.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// LintIssue represents a single problem detected during linting.
type LintIssue struct {
	Severity Severity `json:"severity"`
	Rule     string   `json:"rule"`
	NodeID   string   `json:"nodeId"`
	NodeName string   `json:"nodeName"`
	Message  string   `json:"message"`
}

// QualityGrade represents an A-F quality rating.
type QualityGrade string

const (
	GradeA QualityGrade = "A"
	GradeB QualityGrade = "B"
	GradeC QualityGrade = "C"
	GradeD QualityGrade = "D"
	GradeF QualityGrade = "F"
)

// LintSummary provides aggregate counts of the lint results.
type LintSummary struct {
	TotalNodes   int `json:"totalNodes"`
	TotalEdges   int `json:"totalEdges"`
	ErrorCount   int `json:"errorCount"`
	WarningCount int `json:"warningCount"`
	InfoCount    int `json:"infoCount"`
}

// LintResult is the complete output of running the linter on a canvas.
type LintResult struct {
	Status       string       `json:"status"` // "pass" or "fail"
	Errors       []LintIssue  `json:"errors"`
	Warnings     []LintIssue  `json:"warnings"`
	Info         []LintIssue  `json:"info"`
	Summary      LintSummary  `json:"summary"`
	QualityScore int          `json:"qualityScore"` // 0-100
	QualityGrade QualityGrade `json:"qualityGrade"` // A-F
}

// computeQualityScore returns a score from 0-100 and a letter grade.
// Each error deducts 15 points (max 60 total), each warning deducts 5 (max 30),
// each info deducts 1 (max 10). This prevents scores from bottoming out too quickly.
func computeQualityScore(errors, warnings, info int) (int, QualityGrade) {
	errorPenalty := errors * 15
	if errorPenalty > 60 {
		errorPenalty = 60
	}
	warningPenalty := warnings * 5
	if warningPenalty > 30 {
		warningPenalty = 30
	}
	infoPenalty := info * 1
	if infoPenalty > 10 {
		infoPenalty = 10
	}
	score := 100 - errorPenalty - warningPenalty - infoPenalty
	if score < 0 {
		score = 0
	}

	var grade QualityGrade
	switch {
	case score >= 90:
		grade = GradeA
	case score >= 75:
		grade = GradeB
	case score >= 60:
		grade = GradeC
	case score >= 40:
		grade = GradeD
	default:
		grade = GradeF
	}

	return score, grade
}

// terminalComponents are components that naturally end a workflow and
// should not be flagged as dead-ends.
var terminalComponents = map[string]bool{
	"approval":                    true,
	"slack.sendTextMessage":       true,
	"slack.waitForButtonClick":    true,
	"github.createIssue":          true,
	"github.createIssueComment":   true,
	"github.createRelease":        true,
	"github.updateIssue":          true,
	"github.publishCommitStatus":  true,
	"github.addReaction":          true,
	"pagerduty.createIncident":    true,
	"pagerduty.resolveIncident":   true,
	"pagerduty.escalateIncident":  true,
	"pagerduty.annotateIncident":  true,
	"pagerduty.acknowledgeIncident": true,
}

// destructiveComponents are components that perform irreversible or
// high-impact actions and should require an upstream approval gate.
var destructiveComponents = map[string]bool{
	"pagerduty.resolveIncident":  true,
	"pagerduty.escalateIncident": true,
	"github.deleteRelease":       true,
	"github.createRelease":       true,
}

// nodeRefDoubleQuotePattern matches $["Node Name"] references in expressions.
var nodeRefDoubleQuotePattern = regexp.MustCompile(`\$\["([^"]+)"\]`)

// nodeRefSingleQuotePattern matches $['Node Name'] references in expressions.
var nodeRefSingleQuotePattern = regexp.MustCompile(`\$\['([^']+)'\]`)

// LintCanvas performs static analysis on a canvas defined by nodes and edges.
// The registry parameter is accepted for future use and may be nil.
func LintCanvas(nodes []models.Node, edges []models.Edge, _ *registry.Registry) *LintResult {
	result := &LintResult{
		Errors:   []LintIssue{},
		Warnings: []LintIssue{},
		Info:     []LintIssue{},
	}

	// Build lookup maps.
	nodeByID := make(map[string]models.Node, len(nodes))
	nodeByName := make(map[string]bool, len(nodes))
	outgoing := make(map[string][]models.Edge)
	incoming := make(map[string][]models.Edge)
	triggers := make([]models.Node, 0)
	widgets := make(map[string]bool)

	for _, n := range nodes {
		nodeByID[n.ID] = n
		nodeByName[n.Name] = true

		if n.Type == "TYPE_TRIGGER" {
			triggers = append(triggers, n)
		}
		if n.Type == "TYPE_WIDGET" {
			widgets[n.ID] = true
		}
	}

	for _, e := range edges {
		outgoing[e.SourceID] = append(outgoing[e.SourceID], e)
		incoming[e.TargetID] = append(incoming[e.TargetID], e)
	}

	// Run all rule checkers.
	checkDuplicateNodes(nodes, result)
	checkEdgeValidity(edges, nodeByID, widgets, result)
	checkCycles(nodes, edges, widgets, result)
	checkOrphanNodes(nodes, triggers, outgoing, widgets, result)
	checkDeadEnds(nodes, outgoing, widgets, result)
	checkMissingApprovalGate(nodes, incoming, nodeByID, widgets, result)
	checkMissingRequiredConfig(nodes, incoming, result)
	checkExpressionSyntax(nodes, nodeByName, widgets, result)
	checkUnreachableBranches(nodes, outgoing, result)

	// Compute summary.
	result.Summary = LintSummary{
		TotalNodes:   len(nodes),
		TotalEdges:   len(edges),
		ErrorCount:   len(result.Errors),
		WarningCount: len(result.Warnings),
		InfoCount:    len(result.Info),
	}

	if len(result.Errors) > 0 {
		result.Status = "fail"
	} else {
		result.Status = "pass"
	}

	result.QualityScore, result.QualityGrade = computeQualityScore(
		len(result.Errors), len(result.Warnings), len(result.Info),
	)

	return result
}

// checkDuplicateNodes detects duplicate node IDs and duplicate node names.
func checkDuplicateNodes(nodes []models.Node, result *LintResult) {
	seenIDs := make(map[string]bool, len(nodes))
	seenNames := make(map[string]bool, len(nodes))

	for _, n := range nodes {
		if seenIDs[n.ID] {
			result.Errors = append(result.Errors, LintIssue{
				Severity: SeverityError,
				Rule:     "duplicate-node-id",
				NodeID:   n.ID,
				NodeName: n.Name,
				Message:  fmt.Sprintf("Duplicate node ID %q", n.ID),
			})
		}
		seenIDs[n.ID] = true

		if n.Type == "TYPE_WIDGET" {
			continue
		}
		if seenNames[n.Name] {
			result.Warnings = append(result.Warnings, LintIssue{
				Severity: SeverityWarning,
				Rule:     "duplicate-node-name",
				NodeID:   n.ID,
				NodeName: n.Name,
				Message:  fmt.Sprintf("Duplicate node name %q — expression references may be ambiguous", n.Name),
			})
		}
		seenNames[n.Name] = true
	}
}

// checkEdgeValidity validates edges for dangling references, self-loops,
// duplicate edges, and edges involving widget nodes.
func checkEdgeValidity(edges []models.Edge, nodeByID map[string]models.Node, widgets map[string]bool, result *LintResult) {
	type edgeKey struct{ src, tgt, ch string }
	seen := make(map[edgeKey]bool, len(edges))

	for i, e := range edges {
		// Dangling source/target.
		if _, ok := nodeByID[e.SourceID]; !ok {
			result.Errors = append(result.Errors, LintIssue{
				Severity: SeverityError,
				Rule:     "invalid-edge",
				NodeID:   e.SourceID,
				Message:  fmt.Sprintf("Edge %d references nonexistent source node %q", i, e.SourceID),
			})
		}
		if _, ok := nodeByID[e.TargetID]; !ok {
			result.Errors = append(result.Errors, LintIssue{
				Severity: SeverityError,
				Rule:     "invalid-edge",
				NodeID:   e.TargetID,
				Message:  fmt.Sprintf("Edge %d references nonexistent target node %q", i, e.TargetID),
			})
		}

		// Self-loop.
		if e.SourceID == e.TargetID {
			result.Errors = append(result.Errors, LintIssue{
				Severity: SeverityError,
				Rule:     "invalid-edge",
				NodeID:   e.SourceID,
				NodeName: nodeByID[e.SourceID].Name,
				Message:  fmt.Sprintf("Edge %d is a self-loop on node %q", i, e.SourceID),
			})
		}

		// Duplicate edge.
		key := edgeKey{e.SourceID, e.TargetID, e.Channel}
		if seen[key] {
			result.Warnings = append(result.Warnings, LintIssue{
				Severity: SeverityWarning,
				Rule:     "duplicate-edge",
				NodeID:   e.SourceID,
				NodeName: nodeByID[e.SourceID].Name,
				Message:  fmt.Sprintf("Duplicate edge from %q to %q on channel %q", e.SourceID, e.TargetID, e.Channel),
			})
		}
		seen[key] = true

		// Widget as edge endpoint.
		if widgets[e.SourceID] {
			result.Errors = append(result.Errors, LintIssue{
				Severity: SeverityError,
				Rule:     "invalid-edge",
				NodeID:   e.SourceID,
				NodeName: nodeByID[e.SourceID].Name,
				Message:  fmt.Sprintf("Edge %d uses widget node %q as source", i, e.SourceID),
			})
		}
		if widgets[e.TargetID] {
			result.Errors = append(result.Errors, LintIssue{
				Severity: SeverityError,
				Rule:     "invalid-edge",
				NodeID:   e.TargetID,
				NodeName: nodeByID[e.TargetID].Name,
				Message:  fmt.Sprintf("Edge %d uses widget node %q as target", i, e.TargetID),
			})
		}
	}
}

// checkCycles detects cycles in the non-widget node graph using Kahn's algorithm.
func checkCycles(nodes []models.Node, edges []models.Edge, widgets map[string]bool, result *LintResult) {
	// Build adjacency for non-widget nodes only.
	inDegree := make(map[string]int)
	adj := make(map[string][]string)

	for _, n := range nodes {
		if widgets[n.ID] {
			continue
		}
		inDegree[n.ID] = 0
	}

	for _, e := range edges {
		if widgets[e.SourceID] || widgets[e.TargetID] {
			continue
		}
		adj[e.SourceID] = append(adj[e.SourceID], e.TargetID)
		inDegree[e.TargetID]++
	}

	// Kahn's: start from nodes with in-degree 0.
	queue := make([]string, 0)
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	visited := 0
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		visited++

		for _, next := range adj[current] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	totalNonWidget := 0
	for _, n := range nodes {
		if !widgets[n.ID] {
			totalNonWidget++
		}
	}

	if visited < totalNonWidget {
		// Find nodes that are part of cycles (those with remaining in-degree > 0).
		var cycleNodes []string
		for id, deg := range inDegree {
			if deg > 0 {
				cycleNodes = append(cycleNodes, id)
			}
		}
		result.Errors = append(result.Errors, LintIssue{
			Severity: SeverityError,
			Rule:     "cycle-detected",
			Message:  fmt.Sprintf("Cycle detected involving %d node(s): %v", len(cycleNodes), cycleNodes),
		})
	}
}

// checkOrphanNodes finds non-widget nodes that are not reachable from any trigger via BFS.
func checkOrphanNodes(
	nodes []models.Node,
	triggers []models.Node,
	outgoing map[string][]models.Edge,
	widgets map[string]bool,
	result *LintResult,
) {
	reachable := make(map[string]bool)

	// BFS from all trigger nodes.
	queue := make([]string, 0, len(triggers))
	for _, t := range triggers {
		queue = append(queue, t.ID)
		reachable[t.ID] = true
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, e := range outgoing[current] {
			if !reachable[e.TargetID] {
				reachable[e.TargetID] = true
				queue = append(queue, e.TargetID)
			}
		}
	}

	for _, n := range nodes {
		if widgets[n.ID] {
			continue
		}
		if !reachable[n.ID] {
			result.Warnings = append(result.Warnings, LintIssue{
				Severity: SeverityWarning,
				Rule:     "orphan-node",
				NodeID:   n.ID,
				NodeName: n.Name,
				Message:  fmt.Sprintf("Node %q is not reachable from any trigger", n.Name),
			})
		}
	}
}

// checkDeadEnds finds non-widget, non-trigger nodes with no outgoing edges
// that are not known terminal components.
func checkDeadEnds(
	nodes []models.Node,
	outgoing map[string][]models.Edge,
	widgets map[string]bool,
	result *LintResult,
) {
	for _, n := range nodes {
		if widgets[n.ID] || n.Type == "TYPE_TRIGGER" {
			continue
		}

		if len(outgoing[n.ID]) > 0 {
			continue
		}

		compName := getComponentName(n)
		if terminalComponents[compName] {
			continue
		}

		result.Warnings = append(result.Warnings, LintIssue{
			Severity: SeverityWarning,
			Rule:     "dead-end",
			NodeID:   n.ID,
			NodeName: n.Name,
			Message:  fmt.Sprintf("Node %q has no outgoing edges and is not a terminal component", n.Name),
		})
	}
}

// checkMissingApprovalGate verifies that every destructive component has
// an upstream approval node reachable by walking backwards through edges.
func checkMissingApprovalGate(
	nodes []models.Node,
	incoming map[string][]models.Edge,
	nodeByID map[string]models.Node,
	widgets map[string]bool,
	result *LintResult,
) {
	for _, n := range nodes {
		if widgets[n.ID] {
			continue
		}

		compName := getComponentName(n)
		if !destructiveComponents[compName] {
			continue
		}

		if !hasUpstreamApproval(n.ID, incoming, nodeByID) {
			result.Errors = append(result.Errors, LintIssue{
				Severity: SeverityError,
				Rule:     "missing-approval-gate",
				NodeID:   n.ID,
				NodeName: n.Name,
				Message:  fmt.Sprintf("Destructive action %q in node %q has no upstream approval gate", compName, n.Name),
			})
		}
	}
}

// hasUpstreamApproval does a reverse BFS from the given node looking for
// an approval component in its ancestors.
func hasUpstreamApproval(
	startID string,
	incoming map[string][]models.Edge,
	nodeByID map[string]models.Node,
) bool {
	visited := make(map[string]bool)
	queue := []string{startID}
	visited[startID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, e := range incoming[current] {
			if visited[e.SourceID] {
				continue
			}
			visited[e.SourceID] = true

			source, ok := nodeByID[e.SourceID]
			if !ok {
				continue
			}

			if getComponentName(source) == "approval" {
				return true
			}

			queue = append(queue, e.SourceID)
		}
	}

	return false
}

// checkMissingRequiredConfig checks specific component types for
// required or recommended configuration fields.
func checkMissingRequiredConfig(
	nodes []models.Node,
	incoming map[string][]models.Edge,
	result *LintResult,
) {
	for _, n := range nodes {
		config := n.Configuration
		if config == nil {
			config = map[string]any{}
		}

		compName := getComponentName(n)

		switch compName {
		case "claude.textPrompt":
			prompt, _ := config["prompt"].(string)
			if strings.TrimSpace(prompt) == "" {
				result.Errors = append(result.Errors, LintIssue{
					Severity: SeverityError,
					Rule:     "missing-required-config",
					NodeID:   n.ID,
					NodeName: n.Name,
					Message:  fmt.Sprintf("Node %q (claude.textPrompt) is missing required \"prompt\" configuration", n.Name),
				})
			}

		case "slack.sendTextMessage":
			if _, ok := config["channel"]; !ok {
				result.Warnings = append(result.Warnings, LintIssue{
					Severity: SeverityWarning,
					Rule:     "missing-required-config",
					NodeID:   n.ID,
					NodeName: n.Name,
					Message:  fmt.Sprintf("Node %q (slack.sendTextMessage) is missing \"channel\" configuration", n.Name),
				})
			}

		case "merge":
			incomingCount := len(incoming[n.ID])
			if incomingCount < 2 {
				result.Info = append(result.Info, LintIssue{
					Severity: SeverityInfo,
					Rule:     "missing-required-config",
					NodeID:   n.ID,
					NodeName: n.Name,
					Message:  fmt.Sprintf("Node %q (merge) has %d incoming edge(s); merge typically expects 2 or more", n.Name, incomingCount),
				})
			}

		case "filter":
			expr, _ := config["expression"].(string)
			if strings.TrimSpace(expr) == "" {
				result.Errors = append(result.Errors, LintIssue{
					Severity: SeverityError,
					Rule:     "missing-required-config",
					NodeID:   n.ID,
					NodeName: n.Name,
					Message:  fmt.Sprintf("Node %q (filter) is missing required \"expression\" configuration", n.Name),
				})
			}

		case "http":
			if _, ok := config["url"]; !ok {
				result.Warnings = append(result.Warnings, LintIssue{
					Severity: SeverityWarning,
					Rule:     "missing-required-config",
					NodeID:   n.ID,
					NodeName: n.Name,
					Message:  fmt.Sprintf("Node %q (http) is missing \"url\" configuration", n.Name),
				})
			}
		}
	}
}

// checkExpressionSyntax scans all string values in every node's Configuration
// for unbalanced {{ }} delimiters and invalid $["Node Name"] references.
func checkExpressionSyntax(
	nodes []models.Node,
	nodeByName map[string]bool,
	widgets map[string]bool,
	result *LintResult,
) {
	for _, n := range nodes {
		if widgets[n.ID] {
			continue
		}

		config := n.Configuration
		if config == nil {
			continue
		}

		for _, val := range collectStringValues(config) {
			// Check balanced {{ }} delimiters.
			openCount := strings.Count(val, "{{")
			closeCount := strings.Count(val, "}}")
			if openCount != closeCount {
				result.Errors = append(result.Errors, LintIssue{
					Severity: SeverityError,
					Rule:     "invalid-expression",
					NodeID:   n.ID,
					NodeName: n.Name,
					Message:  fmt.Sprintf("Node %q has unbalanced expression delimiters: %d opening '{{' vs %d closing '}}'", n.Name, openCount, closeCount),
				})
			}

			// Check $["Node Name"] references point to real nodes.
			// Use separate patterns for double-quoted and single-quoted
			// to correctly handle node names containing the other quote type.
			for _, pat := range []*regexp.Regexp{nodeRefDoubleQuotePattern, nodeRefSingleQuotePattern} {
				matches := pat.FindAllStringSubmatch(val, -1)
				for _, match := range matches {
					refName := match[1]
					if !nodeByName[refName] {
						result.Warnings = append(result.Warnings, LintIssue{
							Severity: SeverityWarning,
							Rule:     "invalid-expression",
							NodeID:   n.ID,
							NodeName: n.Name,
							Message:  fmt.Sprintf("Node %q references unknown node %q", n.Name, refName),
						})
					}
				}
			}
		}
	}
}

// collectStringValues recursively extracts all string values from a map.
func collectStringValues(m map[string]any) []string {
	if m == nil {
		return nil
	}
	var result []string
	for _, v := range m {
		switch val := v.(type) {
		case string:
			result = append(result, val)
		case map[string]any:
			result = append(result, collectStringValues(val)...)
		case []any:
			for _, item := range val {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
				if sub, ok := item.(map[string]any); ok {
					result = append(result, collectStringValues(sub)...)
				}
			}
		}
	}
	return result
}

// checkUnreachableBranches checks that filter components have at least one
// "default" channel outgoing edge, ensuring the matching path has somewhere to go.
func checkUnreachableBranches(
	nodes []models.Node,
	outgoing map[string][]models.Edge,
	result *LintResult,
) {
	for _, n := range nodes {
		compName := getComponentName(n)
		if compName != "filter" {
			continue
		}

		hasDefault := false
		for _, e := range outgoing[n.ID] {
			if e.Channel == "default" {
				hasDefault = true
				break
			}
		}

		if !hasDefault {
			result.Info = append(result.Info, LintIssue{
				Severity: SeverityInfo,
				Rule:     "unreachable-branch",
				NodeID:   n.ID,
				NodeName: n.Name,
				Message:  fmt.Sprintf("Filter node %q has no \"default\" channel outgoing edge; matched events have nowhere to go", n.Name),
			})
		}
	}
}

// getComponentName returns the component or trigger name for a node.
func getComponentName(node models.Node) string {
	if node.Ref.Component != nil {
		return node.Ref.Component.Name
	}
	if node.Ref.Trigger != nil {
		return node.Ref.Trigger.Name
	}
	return ""
}
