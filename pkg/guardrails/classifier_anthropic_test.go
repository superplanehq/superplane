package guardrails

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAnthropicClassifier_MissingKey(t *testing.T) {
	_, err := NewAnthropicClassifier(AnthropicClassifierConfig{APIKey: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "APIKey is required")
}

func TestNewAnthropicClassifier_DefaultsApplied(t *testing.T) {
	c, err := NewAnthropicClassifier(AnthropicClassifierConfig{APIKey: "sk-test"})
	require.NoError(t, err)
	assert.Equal(t, defaultClassifierModel, c.cfg.Model)
	assert.Equal(t, anthropicAPIBase, c.cfg.BaseURL)
}

func TestParseClassifierJSON_WellFormed(t *testing.T) {
	findings := []Finding{
		{RuleID: "secret.aws_access_key", Severity: SeverityCritical, Confidence: 0.99},
		{RuleID: "injection.instruction_override", Severity: SeverityHigh, Confidence: 0.85},
	}
	text := `{"risk_score": 82, "confirmed_findings": ["secret.aws_access_key"], "analysis": "Real AWS key detected"}`
	score, ids := parseClassifierJSON(text, findings)
	assert.Equal(t, 82, score)
	assert.True(t, ids["secret.aws_access_key"])
	assert.False(t, ids["injection.instruction_override"])
}

func TestParseClassifierJSON_ScoreClamped(t *testing.T) {
	findings := []Finding{{RuleID: "r1", Severity: SeverityHigh, Confidence: 0.9}}
	text := `{"risk_score": 150, "confirmed_findings": ["r1"], "analysis": "over limit"}`
	score, _ := parseClassifierJSON(text, findings)
	assert.Equal(t, 100, score)
}

func TestParseClassifierJSON_Fallback_InvalidJSON(t *testing.T) {
	findings := []Finding{
		{RuleID: "r1", Severity: SeverityHigh, Confidence: 0.9},
	}
	// Malformed JSON — should fall back to confirming all findings.
	score, ids := parseClassifierJSON("not json at all", findings)
	assert.Equal(t, Score(findings), score)
	assert.True(t, ids["r1"])
}

func TestFilterConfirmedFindings_FiltersCorrectly(t *testing.T) {
	findings := []Finding{
		{RuleID: "r1"}, {RuleID: "r2"}, {RuleID: "r3"},
	}
	confirmed := map[string]bool{"r1": true, "r3": true}
	result := filterConfirmedFindings(findings, confirmed)
	require.Len(t, result, 2)
	assert.Equal(t, "r1", result[0].RuleID)
	assert.Equal(t, "r3", result[1].RuleID)
}

func TestFilterConfirmedFindings_EmptyConfirmed_ReturnsAll(t *testing.T) {
	findings := []Finding{{RuleID: "r1"}, {RuleID: "r2"}}
	result := filterConfirmedFindings(findings, map[string]bool{})
	assert.Len(t, result, 2)
}

func TestBuildClassifierUserMessage_IncludesFindings(t *testing.T) {
	req := ClassificationRequest{
		ContentHash: "abc123",
		Findings: []Finding{
			{RuleID: "secret.aws_access_key", Category: CategorySecret, Severity: SeverityCritical, Evidence: "AWS key detected", Confidence: 0.99},
		},
	}
	msg := buildClassifierUserMessage(req)
	assert.Contains(t, msg, "secret.aws_access_key")
	assert.Contains(t, msg, "abc123")
	assert.Contains(t, msg, "CRITICAL")
}

func TestNoOpClassifier_ReturnsNil(t *testing.T) {
	c := NewNoOpClassifier()
	assert.Equal(t, "noop", c.Model())
	result, err := c.Classify(nil, ClassificationRequest{})
	assert.NoError(t, err)
	assert.Nil(t, result)
}
