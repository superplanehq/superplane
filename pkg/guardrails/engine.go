package guardrails

import (
	"sort"
	"time"

	log "github.com/sirupsen/logrus"
)

const engineTimeoutMs = 40

// Engine runs a sorted chain of rules against a prompt string.
type Engine struct {
	rules []Rule
}

// NewEngine returns an Engine with the given rules sorted by priority.
func NewEngine(rules []Rule) *Engine {
	sorted := make([]Rule, len(rules))
	copy(sorted, rules)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority() < sorted[j].Priority()
	})
	return &Engine{rules: sorted}
}

// Scan evaluates all applicable rules against content and returns findings.
// If the total evaluation exceeds engineTimeoutMs it returns whatever findings
// have been collected so far with a TIMEOUT finding appended.
func (e *Engine) Scan(content string, ctx ScanContext) []Finding {
	deadline := time.Now().Add(engineTimeoutMs * time.Millisecond)
	var findings []Finding

	for _, rule := range e.rules {
		if time.Now().After(deadline) {
			log.Warnf("guardrail engine timeout scanning execution=%s field=%s", ctx.ExecutionID, ctx.FieldName)
			findings = append(findings, Finding{
				RuleID:   "engine.timeout",
				Severity: SeverityInfo,
				Category: CategoryInjection,
				Evidence: "Rule engine timed out; scan is incomplete",
			})
			return findings
		}

		if !rule.AppliesToProvider(ctx.Provider) {
			continue
		}
		if ctx.IsSystemField && !rule.AppliesToSystemField() {
			continue
		}

		rulefindings, err := rule.Evaluate(content, ctx)
		if err != nil {
			log.Warnf("guardrail rule %s error on execution=%s: %v", rule.ID(), ctx.ExecutionID, err)
			continue
		}
		findings = append(findings, rulefindings...)

		// Short-circuit: hard-block conditions no longer need cheap rules
		// (policy decides the action, but early exit saves CPU).
		for _, f := range rulefindings {
			if f.Severity == SeverityCritical && f.Confidence >= 0.95 {
				return findings
			}
		}
	}

	return findings
}

// Score returns a 0-100 risk score from a slice of findings.
func Score(findings []Finding) int {
	if len(findings) == 0 {
		return 0
	}
	score := 0
	for _, f := range findings {
		score += severityScore(f.Severity, f.Confidence)
	}
	if score > 100 {
		score = 100
	}
	return score
}

func severityScore(s Severity, confidence float64) int {
	var base int
	switch s {
	case SeverityCritical:
		base = 80
	case SeverityHigh:
		base = 40
	case SeverityMedium:
		base = 20
	case SeverityLow:
		base = 10
	default:
		base = 5
	}
	return int(float64(base) * confidence)
}
