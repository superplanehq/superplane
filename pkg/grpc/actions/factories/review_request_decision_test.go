package factories

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
)

func readReviewDecision(t *testing.T, script string, decision []byte, environment map[string]string) ([]byte, error) {
	t.Helper()
	directory := t.TempDir()
	input, output := filepath.Join(directory, "decision.json"), filepath.Join(directory, "result.json")
	if decision != nil {
		require.NoError(t, os.WriteFile(input, decision, 0600))
	}
	command := exec.CommandContext(t.Context(), "bash", "-c", strings.ReplaceAll(script, "/tmp/review-request.json", input))
	command.Env = append(os.Environ(), "SUPERPLANE_RESULT_FILE="+output)
	for name, value := range environment {
		command.Env = append(command.Env, name+"="+value)
	}
	logs, err := command.CombinedOutput()
	if err != nil {
		_, readErr := os.Stat(output)
		require.True(t, os.IsNotExist(readErr), "invalid decisions must not write a result")
		return nil, fmt.Errorf("read review decision: %w: %s", err, logs)
	}
	result, err := os.ReadFile(output)
	require.NoError(t, err)
	return result, nil
}

func TestReviewRequestReadDecision(t *testing.T) {
	result, err := materializeFactoryTemplate(reviewRequestTemplateID, models.ProviderGitHub, factoryTemplateInput{})
	require.NoError(t, err)
	doc, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
	require.NoError(t, err)
	steps := findYAMLNode(t, doc, reviewRequestAgentNodeID).Configuration["steps"].([]any)
	script := steps[2].(map[string]any)["command"].(string)
	valid := `{"reviewRequired":true,"rationale":"Security findings need review.","publicationSummary":"Security findings require human review.","noCandidate":false,"reviewers":[{"id":2,"login":"octocat","reason":"Owns the changed code.","publicationReason":"Selected for code ownership."}]}`
	empty := `{"reviewRequired":true,"rationale":"No suitable owner.","publicationSummary":"Ownership changes require human review.","noCandidate":true,"reviewers":[]}`
	for _, tc := range []struct {
		name, decision, fallback, login, lookupError string
		fails, fallbackUsed                          bool
		report                                       string
	}{
		{name: "waive positive report", decision: `{"reviewRequired":false,"rationale":"All checks are positive.","publicationSummary":"All checks are positive.","noCandidate":false,"reviewers":[]}`, report: `{"complete":true,"checks":[{"level":"positive"}]}`},
		{name: "cannot waive concerning report", decision: `{"reviewRequired":false,"rationale":"Skip.","publicationSummary":"Skip.","noCandidate":false,"reviewers":[]}`, fails: true},
		{name: "cannot waive empty report", decision: `{"reviewRequired":false,"rationale":"Skip.","publicationSummary":"Skip.","noCandidate":false,"reviewers":[]}`, report: `{"complete":true,"checks":[]}`, fails: true},
		{name: "cannot waive missing check", decision: `{"reviewRequired":false,"rationale":"Skip.","publicationSummary":"Skip.","noCandidate":false,"reviewers":[]}`, report: `{"complete":true,"checks":[{"level":"positive","missing":true}]}`, fails: true},
		{name: "cannot waive incomplete report", decision: `{"reviewRequired":false,"rationale":"Skip.","publicationSummary":"Skip.","noCandidate":false,"reviewers":[]}`, report: `{"complete":false,"checks":[{"level":"positive"}]}`, fails: true},
		{name: "selection", decision: valid, login: "octocat"},
		{name: "selection takes precedence", decision: valid, fallback: "fallback-user", login: "octocat"},
		{name: "selection ignores supplied error status", decision: strings.Replace(valid, `"reviewRequired":true`, `"reviewRequired":true,"status":"selection_error"`, 1), login: "octocat"},
		{name: "fallback", decision: empty, fallback: " FALLBACK-USER ", login: "fallback-user", fallbackUsed: true},
		{name: "no fallback", decision: empty},
		{name: "ineligible fallback", decision: empty, fallback: "author"},
		{name: "missing file", fails: true},
		{name: "invalid JSON", decision: "broken", fails: true},
		{name: "array", decision: "[]", fails: true},
		{name: "null", decision: "null", fails: true},
		{name: "missing fields", decision: `{"reviewRequired":true}`, fails: true},
		{name: "review required", decision: strings.Replace(valid, `"reviewRequired":true`, `"reviewRequired":false`, 1), fails: true},
		{name: "blank rationale", decision: strings.Replace(valid, "Security findings need review.", " ", 1), fails: true},
		{name: "missing publication summary", decision: strings.Replace(valid, `"publicationSummary":"Security findings require human review.",`, "", 1), fails: true},
		{name: "blank publication summary", decision: strings.Replace(valid, "Security findings require human review.", " ", 1), fails: true},
		{name: "non-string publication summary", decision: strings.Replace(valid, `"publicationSummary":"Security findings require human review."`, `"publicationSummary":123`, 1), fails: true},
		{name: "missing no-candidate summary", decision: strings.Replace(empty, `"publicationSummary":"Ownership changes require human review.",`, "", 1), fails: true},
		{name: "inconsistent selection", decision: strings.Replace(valid, `"noCandidate":false`, `"noCandidate":true`, 1), fails: true},
		{name: "empty selection", decision: strings.Replace(empty, `"noCandidate":true`, `"noCandidate":false`, 1), fails: true},
		{name: "duplicate identity", decision: strings.Replace(valid, `}]`, `},{"id":2,"login":"octocat","reason":"Same owner.","publicationReason":"Same owner."}]`, 1), fails: true},
		{name: "ineligible identity", decision: strings.Replace(valid, `"id":2`, `"id":1`, 1), fails: true},
		{name: "changed login", decision: strings.Replace(valid, "octocat", "another-user", 1), fails: true},
		{name: "blank reason", decision: strings.Replace(valid, "Owns the changed code.", " ", 1), fails: true},
		{name: "missing publication reason", decision: strings.Replace(valid, `,"publicationReason":"Selected for code ownership."`, "", 1), fails: true},
		{name: "blank publication reason", decision: strings.Replace(valid, "Selected for code ownership.", " ", 1), fails: true},
		{name: "non-string publication reason", decision: strings.Replace(valid, `"publicationReason":"Selected for code ownership."`, `"publicationReason":123`, 1), fails: true},
		{name: "lookup error only", decision: `{"error":"lookup unavailable"}`, fallback: "fallback-user", lookupError: "lookup unavailable"},
		{name: "lookup error overrides selection", decision: strings.Replace(valid, `"reviewRequired":true`, `"reviewRequired":true,"error":"lookup unavailable","fallbackUsed":true`, 1), fallback: "fallback-user", lookupError: "lookup unavailable"},
		{name: "lookup error overrides no candidate", decision: strings.Replace(empty, `"reviewRequired":true`, `"reviewRequired":true,"error":"lookup unavailable"`, 1), fallback: "fallback-user", lookupError: "lookup unavailable"},
		{name: "blank lookup error", decision: `{"error":" "}`, fails: true},
		{name: "non-string lookup error", decision: `{"error":{"message":"unavailable"}}`, fails: true},
		{name: "null lookup error", decision: `{"error":null}`, fails: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.report == "" {
				tc.report = `{"complete":true,"checks":[{"level":"concerning"}]}`
			}
			var input []byte
			if tc.decision != "" {
				input = []byte(tc.decision)
			}
			output, err := readReviewDecision(t, script, input, map[string]string{
				"SUPERPLANE_REVIEW_REQUEST_CANDIDATES": `[{"id":2,"login":"octocat"},{"id":3,"login":"fallback-user"}]`,
				"SUPERPLANE_REVIEW_REQUEST_FALLBACK":   tc.fallback,
				"SUPERPLANE_REVIEW_REQUEST_REPORT":     tc.report,
			})
			if tc.fails {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			var decision map[string]any
			require.NoError(t, json.Unmarshal(output, &decision))
			require.Equal(t, tc.fallbackUsed, decision["fallbackUsed"])
			if tc.lookupError != "" {
				require.Equal(t, "selection_error", decision["status"])
				require.Equal(t, tc.lookupError, decision["error"])
				require.Equal(t, false, decision["noCandidate"])
			} else {
				require.NotContains(t, decision, "status")
				var input map[string]any
				require.NoError(t, json.Unmarshal([]byte(tc.decision), &input))
				require.Equal(t, input["rationale"], decision["rationale"], "detailed evidence is retained")
				require.Equal(t, input["publicationSummary"], decision["publicationSummary"])
			}
			reviewers := decision["reviewers"].([]any)
			if tc.login == "" {
				require.Empty(t, reviewers)
				return
			}
			require.Len(t, reviewers, 1)
			require.Equal(t, tc.login, reviewers[0].(map[string]any)["login"])
			selected := reviewers[0].(map[string]any)
			require.NotEmpty(t, selected["reason"])
			if tc.fallbackUsed {
				require.Equal(t, "Used the configured fallback because no suitable candidate was found.", selected["publicationReason"])
				return
			}
			require.Equal(t, "Owns the changed code.", selected["reason"])
			require.Equal(t, "Selected for code ownership.", selected["publicationReason"])
		})
	}
}
