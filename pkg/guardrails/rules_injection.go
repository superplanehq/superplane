package guardrails

import (
	"regexp"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

type injectionRule struct {
	id       string
	priority int
	patterns []*regexp.Regexp
	severity Severity
	evidence string
}

func (r *injectionRule) ID() string                       { return r.id }
func (r *injectionRule) Priority() int                    { return r.priority }
func (r *injectionRule) AppliesToProvider(_ string) bool  { return true }
func (r *injectionRule) AppliesToFieldType(_ string) bool { return true }
func (r *injectionRule) AppliesToSystemField() bool       { return false }
func (r *injectionRule) DefaultSeverity() Severity        { return r.severity }

func (r *injectionRule) Evaluate(content string, ctx ScanContext) ([]Finding, error) {
	if ctx.IsSystemField {
		return nil, nil
	}

	normalized := norm.NFKC.String(content)
	normalized = collapseWhitespace(normalized)

	for _, pat := range r.patterns {
		loc := pat.FindStringIndex(normalized)
		if loc == nil {
			continue
		}
		match := normalized[loc[0]:loc[1]]
		if len(match) > 80 {
			match = match[:80] + "..."
		}
		return []Finding{
			{
				RuleID:      r.id,
				Severity:    r.severity,
				Confidence:  0.85,
				Category:    CategoryInjection,
				Evidence:    r.evidence,
				MatchOffset: loc[0],
				MatchLen:    loc[1] - loc[0],
				Redacted:    false,
				Match:       match,
			},
		}, nil
	}

	return nil, nil
}

func collapseWhitespace(s string) string {
	runes := []rune(s)
	out := make([]rune, 0, len(runes))
	inSpace := false
	for _, r := range runes {
		if unicode.IsSpace(r) {
			if !inSpace {
				out = append(out, ' ')
				inSpace = true
			}
		} else {
			out = append(out, r)
			inSpace = false
		}
	}
	return string(out)
}

func instructionOverride() Rule {
	return &injectionRule{
		id:       "injection.instruction_override",
		priority: 25,
		severity: SeverityHigh,
		evidence: "Prompt injection: instruction-override pattern detected",
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)ignore\s+(?:(?:all|the|your|prior|previous|above)\s+){1,3}instructions?`),
			regexp.MustCompile(`(?i)disregard\s+(?:(?:your|the|all|prior|previous|above)\s+){1,3}`),
			regexp.MustCompile(`(?i)forget\s+(?:everything|all)\s+(?:you|i|we)\s+(?:told|said|mentioned)`),
			regexp.MustCompile(`(?i)your\s+new\s+(?:instructions?|role|purpose|mission)\s+(?:are|is)`),
			regexp.MustCompile(`(?i)(?:DAN|developer|jailbreak|unrestricted)\s+mode`),
		},
	}
}

func roleDelimiterInjection() Rule {
	return &injectionRule{
		id:       "injection.role_delimiter",
		priority: 20,
		severity: SeverityHigh,
		evidence: "Prompt injection: role-delimiter injection pattern detected",
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\n\s*(?:User|Assistant|Human|AI)\s*:`),
			regexp.MustCompile(`<\|im_start\|>|<\|im_end\|>`),
			regexp.MustCompile(`<!--\s*SYSTEM\s*[:>]`),
		},
	}
}
