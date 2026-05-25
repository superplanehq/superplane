package guardrails

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// secretRule is a simple regex-based secret detection rule.
type secretRule struct {
	id       string
	priority int
	pattern  *regexp.Regexp
	severity Severity
	evidence string
}

func (r *secretRule) ID() string                       { return r.id }
func (r *secretRule) Priority() int                    { return r.priority }
func (r *secretRule) AppliesToProvider(_ string) bool  { return true }
func (r *secretRule) AppliesToFieldType(_ string) bool { return true }
func (r *secretRule) AppliesToSystemField() bool       { return true }
func (r *secretRule) DefaultSeverity() Severity        { return r.severity }

func (r *secretRule) Evaluate(content string, _ ScanContext) ([]Finding, error) {
	loc := r.pattern.FindStringIndex(content)
	if loc == nil {
		return nil, nil
	}
	match := content[loc[0]:loc[1]]
	redacted := fmt.Sprintf("[REDACTED:%s]", r.id)
	_ = match
	return []Finding{
		{
			RuleID:      r.id,
			Severity:    r.severity,
			Confidence:  0.99,
			Category:    CategorySecret,
			Evidence:    r.evidence,
			MatchOffset: loc[0],
			MatchLen:    loc[1] - loc[0],
			Redacted:    true,
			Match:       redacted,
		},
	}, nil
}

func awsAccessKey() Rule {
	return &secretRule{
		id:       "secret.aws_access_key",
		priority: 10,
		pattern:  regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
		severity: SeverityCritical,
		evidence: "AWS Access Key ID pattern detected",
	}
}

func gitHubPAT() Rule {
	return &secretRule{
		id:       "secret.github_pat",
		priority: 10,
		pattern:  regexp.MustCompile(`(?:ghp_[A-Za-z0-9]{36}|github_pat_[A-Za-z0-9_]{82})`),
		severity: SeverityCritical,
		evidence: "GitHub personal access token detected",
	}
}

func openAIKey() Rule {
	return &secretRule{
		id:       "secret.openai_key",
		priority: 10,
		pattern:  regexp.MustCompile(`sk-[A-Za-z0-9]{48}`),
		severity: SeverityCritical,
		evidence: "OpenAI API key detected",
	}
}

func connectionString() Rule {
	return &secretRule{
		id:       "secret.connection_string",
		priority: 15,
		pattern:  regexp.MustCompile(`(?i)(?:postgres(?:ql)?|mysql|mongodb|redis):\/\/[^:]+:[^@\s]+@`),
		severity: SeverityCritical,
		evidence: "Database connection string with embedded credentials detected",
	}
}

func jwtBearer() Rule {
	return &secretRule{
		id:       "secret.jwt_bearer",
		priority: 20,
		pattern:  regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`),
		severity: SeverityHigh,
		evidence: "JWT token detected",
	}
}

// highEntropyRule detects high-entropy strings adjacent to credential keywords.
type highEntropyRule struct {
	priority  int
	severity  Severity
	threshold float64
	keywords  []string
}

func (r *highEntropyRule) ID() string                       { return "secret.generic_high_entropy" }
func (r *highEntropyRule) Priority() int                    { return r.priority }
func (r *highEntropyRule) AppliesToProvider(_ string) bool  { return true }
func (r *highEntropyRule) AppliesToFieldType(_ string) bool { return true }
func (r *highEntropyRule) AppliesToSystemField() bool       { return true }
func (r *highEntropyRule) DefaultSeverity() Severity        { return r.severity }

func (r *highEntropyRule) Evaluate(content string, _ ScanContext) ([]Finding, error) {
	lower := strings.ToLower(content)
	hasKeyword := false
	for _, kw := range r.keywords {
		if strings.Contains(lower, kw) {
			hasKeyword = true
			break
		}
	}
	if !hasKeyword {
		return nil, nil
	}

	for _, length := range []int{40, 32, 20, 64} {
		if length > len(content) {
			continue
		}
		for i := 0; i <= len(content)-length; i++ {
			window := content[i : i+length]
			e := shannonEntropy(window)
			if e >= r.threshold {
				return []Finding{
					{
						RuleID:      r.ID(),
						Severity:    r.severity,
						Confidence:  0.75,
						Category:    CategorySecret,
						Evidence:    fmt.Sprintf("High-entropy string (%.2f bits/char) adjacent to credential keyword", e),
						MatchOffset: i,
						MatchLen:    length,
						Redacted:    true,
						Match:       "[REDACTED:secret.generic_high_entropy]",
					},
				}, nil
			}
		}
	}

	return nil, nil
}

func shannonEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}
	var freq [256]float64
	for _, b := range []byte(s) {
		freq[b]++
	}
	l := float64(len(s))
	var entropy float64
	for _, count := range freq {
		if count > 0 {
			p := count / l
			entropy -= p * math.Log2(p)
		}
	}
	return entropy
}

func genericHighEntropy() Rule {
	return &highEntropyRule{
		priority:  30,
		severity:  SeverityHigh,
		threshold: 4.2,
		keywords: []string{
			"password", "passwd", "secret", "api_key", "apikey",
			"token", "credential", "auth", "bearer", "private_key",
		},
	}
}
