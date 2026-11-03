package mergeconfidence

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models/factory"
)

func TestUpdateDescription(t *testing.T) {
	const section = "<!-- superplane:merge-confidence:start -->\n## SuperPlane merge confidence\nnew\n<!-- superplane:merge-confidence:end -->"
	const old = "<!-- superplane:merge-confidence:start -->old<!-- superplane:merge-confidence:end -->"
	const greptile = "<!-- greptile_comment -->\r\nConfidence: 4/5\r\n<!-- /greptile_comment -->"
	for _, tc := range []struct {
		name, body, want string
		invalid          bool
	}{
		{name: "empty", want: section},
		{name: "append", body: "Original\r\n" + greptile, want: "Original\r\n" + greptile + "\n\n" + section},
		{name: "replace preserves bytes", body: " \r\n" + greptile + "\r\n" + old + "\n trailing \t", want: " \r\n" + greptile + "\r\n" + section + "\n trailing \t"},
		{name: "unchanged", body: section, want: section},
		{name: "missing end", body: StartMarker, invalid: true},
		{name: "missing start", body: EndMarker, invalid: true},
		{name: "reversed", body: EndMarker + StartMarker, invalid: true},
		{name: "duplicate", body: old + old, invalid: true},
		{name: "duplicate start", body: StartMarker + old, invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := UpdateDescription(tc.body, section)
			if tc.invalid {
				require.Error(t, err)
				assert.Equal(t, tc.body, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRender(t *testing.T) {
	for _, tc := range []struct {
		name       string
		passed     bool
		enabled    []string
		scores     map[string]float64
		want       string
		incomplete bool
	}{
		{name: "invert risk and drift and take weakest", passed: true, enabled: []string{"risk", "performance", "security", "drift", "reversibility"}, scores: map[string]float64{"risk": 2, "performance": 5, "security": 4, "drift": 4, "reversibility": 5}, want: "**Overall score: 2/5**"},
		{name: "disabled checks and custom score", passed: true, enabled: []string{"performance", "custom"}, scores: map[string]float64{"performance": 4, "custom": 1, "risk": 5}, want: "**Overall score: 4/5**"},
		{name: "low blast radius", passed: true, enabled: []string{"risk"}, scores: map[string]float64{"risk": 1}, want: "**Overall score: 5/5**"},
		{name: "high blast radius", passed: true, enabled: []string{"risk"}, scores: map[string]float64{"risk": 5}, want: "**Overall score: 1/5**"},
		{name: "low drift", passed: true, enabled: []string{"drift"}, scores: map[string]float64{"drift": 1}, want: "**Overall score: 5/5**"},
		{name: "high drift", passed: true, enabled: []string{"drift"}, scores: map[string]float64{"drift": 5}, want: "**Overall score: 1/5**"},
		{name: "custom only", passed: true, enabled: []string{"custom"}, scores: map[string]float64{"custom": 2}, want: "No built-in checks are enabled."},
		{name: "all disabled", passed: true, want: "No checks are enabled."},
		{name: "missing result", passed: true, enabled: []string{"risk", "security"}, scores: map[string]float64{"risk": 1}, incomplete: true},
		{name: "failed with all results", enabled: []string{"risk"}, scores: map[string]float64{"risk": 1}, incomplete: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checks := map[string]factory.CheckRef{}
			for id, score := range tc.scores {
				checks[id+"-review"] = factory.CheckRef{Score: score, MaxScore: 5, Summary: "available result"}
			}
			got := Render(Assessment{Passed: tc.passed, Enabled: tc.enabled, Checks: checks, ReviewedCommit: "abc123", TaskURL: "https://example.com/task"})
			if tc.incomplete {
				assert.Contains(t, got, "**Incomplete**")
				assert.NotContains(t, got, "Overall score:")
			} else {
				assert.Contains(t, got, tc.want)
			}
			assert.Contains(t, got, "Reviewed commit: abc123")
			assert.Contains(t, got, "[View task](<https://example.com/task>)")
			if tc.name == "missing result" {
				assert.Contains(t, got, "| Security | Missing |")
				assert.Contains(t, got, "available result")
			}
			if tc.name == "disabled checks and custom score" {
				assert.NotContains(t, got, "Blast radius")
			}
		})
	}
}

func TestRenderEscapesGeneratedContent(t *testing.T) {
	got := Render(Assessment{Passed: true, Enabled: []string{"custom"}, Labels: map[string]string{"custom": "a|b\n" + EndMarker}, Checks: map[string]factory.CheckRef{"custom-review": {Score: 2, MaxScore: 5, Summary: "<script> & **bold** [link](url) `code` \\ |\r\n" + StartMarker}}})
	assert.Equal(t, 1, strings.Count(got, StartMarker))
	assert.Equal(t, 1, strings.Count(got, EndMarker))
	assert.NotContains(t, got, "<script>")
	assert.NotContains(t, got, "**bold**")
	assert.Contains(t, got, "&#124;")
	assert.Equal(t, 1, strings.Count(got, "| a&#124;b"))
}
