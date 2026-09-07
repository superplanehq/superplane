package factories

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/yaml"
)

const (
	prFeedbackPullRequestTriggerNodeID = "on-pull-request"
	prFeedbackWaitChecksNodeID         = "wait-pr-checks"
	prFeedbackMarkPassedNodeID         = "mark-checks-passed"
	prFeedbackStartRepairNodeID        = "start-check-repair"
	prFeedbackPauseFixesNodeID         = "pause-automatic-fixes"
	prFeedbackAnnounceLimitNodeID      = "set-fixes-paused-note"
	prFeedbackStopWaitingNodeID        = "stop-waiting-for-checks"
	prFeedbackRecordTimeoutNodeID      = "record-check-timeout"

	prFeedbackWaitChecksComponent     = "github.waitForPullRequestChecks"
	prFeedbackUpdateActivityComponent = "updatePullRequestActivity"
	prFeedbackAddRunErrorComponent    = "addRunError"
	prFeedbackSetStatusNoteComponent  = "setWorkOrderStatusNote"
	prFeedbackStatusNoteKey           = "pr-closure"

	prFeedbackChecksDefaultName        = "Fix pull request checks"
	prFeedbackChecksDefaultDescription = "Wait for pull request checks and start one agent run when selected checks fail."
	prFeedbackWaitChecksNodeName       = "Wait For Pull Request Checks"
)

func buildChecksPRFeedbackCanvas(request prFeedbackBuildRequest) *yaml.Canvas {
	name := prFeedbackCanvasName(request, prFeedbackChecksDefaultName)

	return &yaml.Canvas{
		APIVersion: yaml.APIVersion,
		Kind:       yaml.KindCanvas,
		Metadata: &yaml.CanvasMetadata{
			Name:        name,
			Description: prFeedbackChecksDefaultDescription,
		},
		Spec: &yaml.CanvasSpec{
			Edges: []yaml.Edge{
				{Channel: "default", SourceID: prFeedbackPullRequestTriggerNodeID, TargetID: prFeedbackFindNodeID},
				{Channel: "found", SourceID: prFeedbackFindNodeID, TargetID: prFeedbackActivityNodeID},
				{Channel: "default", SourceID: prFeedbackActivityNodeID, TargetID: prFeedbackWaitChecksNodeID},
				{Channel: "passed", SourceID: prFeedbackWaitChecksNodeID, TargetID: prFeedbackMarkPassedNodeID},
				{Channel: "failed", SourceID: prFeedbackWaitChecksNodeID, TargetID: prFeedbackStartRepairNodeID},
				{Channel: "timedOut", SourceID: prFeedbackWaitChecksNodeID, TargetID: prFeedbackStopWaitingNodeID},
				{Channel: "default", SourceID: prFeedbackStartRepairNodeID, TargetID: prFeedbackRunnerNodeID},
				{Channel: "limitReached", SourceID: prFeedbackStartRepairNodeID, TargetID: prFeedbackPauseFixesNodeID},
				{Channel: "default", SourceID: prFeedbackPauseFixesNodeID, TargetID: prFeedbackAnnounceLimitNodeID},
				{Channel: "default", SourceID: prFeedbackStopWaitingNodeID, TargetID: prFeedbackRecordTimeoutNodeID},
			},
			Nodes: []yaml.Node{
				{
					ID:        prFeedbackPullRequestTriggerNodeID,
					Name:      "On Pull Request",
					Type:      yaml.NodeTypeTrigger,
					Component: "github.onPullRequest",
					Configuration: map[string]any{
						"repository": request.Repository,
						"actions":    []any{"opened", "reopened", "synchronize"},
					},
					Integration: request.Binding.integrationRef(),
					Position:    yaml.Position{X: 80, Y: 260},
				},
				{
					ID:        prFeedbackFindNodeID,
					Name:      "Find Pull Request",
					Type:      yaml.NodeTypeAction,
					Component: prFeedbackFindComponent,
					Configuration: map[string]any{
						"provider":   "github",
						"repository": "{{ root().data.repository.full_name }}",
						"number":     "{{ root().data.pull_request.number }}",
						"url":        "{{ root().data.pull_request.html_url }}",
					},
					Position: yaml.Position{X: 360, Y: 260},
				},
				{
					ID:        prFeedbackActivityNodeID,
					Name:      "Add Pull Request Activity",
					Type:      yaml.NodeTypeAction,
					Component: prFeedbackActivityComponent,
					Configuration: map[string]any{
						"pullRequestId": `{{ $["Find Pull Request"].data.pullRequest.id }}`,
						"revision":      prFeedbackPRHeadSHAExpression(),
						"access":        "concurrent",
						"description":   prFeedbackChecksWaitingDescriptionExpression(),
					},
					Position: yaml.Position{X: 500, Y: 260},
				},
				{
					ID:            prFeedbackWaitChecksNodeID,
					Name:          prFeedbackWaitChecksNodeName,
					Type:          yaml.NodeTypeAction,
					Component:     prFeedbackWaitChecksComponent,
					Configuration: prFeedbackWaitChecksConfiguration(request),
					Integration:   request.Binding.integrationRef(),
					Position:      yaml.Position{X: 640, Y: 260},
				},
				{
					ID:        prFeedbackMarkPassedNodeID,
					Name:      "Mark Checks Passed",
					Type:      yaml.NodeTypeAction,
					Component: prFeedbackUpdateActivityComponent,
					Configuration: map[string]any{
						"description": prFeedbackChecksPassedDescriptionExpression(),
					},
					Position: yaml.Position{X: 820, Y: 80},
				},
				{
					ID:        prFeedbackStartRepairNodeID,
					Name:      "Start Check Repair",
					Type:      yaml.NodeTypeAction,
					Component: prFeedbackUpdateActivityComponent,
					Configuration: map[string]any{
						"access":      "exclusive",
						"description": prFeedbackChecksRepairDescriptionExpression(),
					},
					Position: yaml.Position{X: 820, Y: 260},
				},
				{
					ID:            prFeedbackRunnerNodeID,
					Name:          "Fix Failed Checks",
					Type:          yaml.NodeTypeAction,
					Component:     request.Agent.component(),
					Configuration: prFeedbackChecksRunnerConfiguration(request),
					Position:      yaml.Position{X: 1000, Y: 260},
				},
				{
					ID:        prFeedbackPauseFixesNodeID,
					Name:      "Pause Automatic Fixes",
					Type:      yaml.NodeTypeAction,
					Component: prFeedbackUpdateActivityComponent,
					Configuration: map[string]any{
						"description": prFeedbackChecksLimitDescriptionExpression(request.MaximumAttempts),
					},
					Position: yaml.Position{X: 1000, Y: 400},
				},
				{
					ID:            prFeedbackAnnounceLimitNodeID,
					Name:          "Set Fixes Paused Note",
					Type:          yaml.NodeTypeAction,
					Component:     prFeedbackSetStatusNoteComponent,
					Configuration: prFeedbackChecksLimitStatusNoteConfiguration(request.MaximumAttempts),
					Position:      yaml.Position{X: 1180, Y: 400},
				},
				{
					ID:        prFeedbackStopWaitingNodeID,
					Name:      "Stop Waiting For Checks",
					Type:      yaml.NodeTypeAction,
					Component: prFeedbackUpdateActivityComponent,
					Configuration: map[string]any{
						"description": prFeedbackChecksTimeoutDescriptionExpression(),
					},
					Position: yaml.Position{X: 820, Y: 440},
				},
				{
					ID:        prFeedbackRecordTimeoutNodeID,
					Name:      "Record Check Timeout",
					Type:      yaml.NodeTypeAction,
					Component: prFeedbackAddRunErrorComponent,
					Configuration: map[string]any{
						"message": "Stopped waiting for checks on {{ root().data.pull_request.head.sha[:7] }}",
					},
					Position: yaml.Position{X: 1000, Y: 440},
				},
			},
		},
	}
}

func prFeedbackWaitChecksConfiguration(request prFeedbackBuildRequest) map[string]any {
	configuration := map[string]any{
		"repository": request.Repository,
		"ref":        prFeedbackPRHeadSHAExpression(),
	}
	if names := checkNamesNodeValue(request.CheckNames); len(names) > 0 {
		configuration["checkNames"] = names
	}
	return configuration
}

func prFeedbackChecksRunnerConfiguration(request prFeedbackBuildRequest) map[string]any {
	configuration := map[string]any{
		"machineType":             prFeedbackMachineType,
		"executionTimeoutSeconds": prFeedbackTimeoutSeconds,
		"steps":                   prFeedbackChecksRunnerSteps(),
		"environmentFrom":         prFeedbackEnvironmentFrom(request.Binding, request.RunnerIntegrationNames),
		"environment": []any{
			map[string]any{
				"name":        "REPO",
				"value":       "{{ root().data.repository.full_name }}",
				"valueSource": "literal",
			},
			map[string]any{
				"name":        "PR_NUMBER",
				"value":       "{{ root().data.pull_request.number }}",
				"valueSource": "literal",
			},
			map[string]any{
				"name":        "PR_HEAD",
				"value":       "{{ root().data.pull_request.head.ref }}",
				"valueSource": "literal",
			},
			map[string]any{
				"name":        "PR_REVISION",
				"value":       prFeedbackPRHeadSHAExpression(),
				"valueSource": "literal",
			},
			map[string]any{
				"name":        "FAILED_CHECKS",
				"value":       prFeedbackFailedChecksExpression(),
				"valueSource": "literal",
			},
			map[string]any{
				"name":        "COAUTHORS",
				"value":       prFeedbackCoauthorsExpression(),
				"valueSource": "literal",
			},
		},
	}

	if credentials := request.Agent.credentials(); credentials != nil {
		configuration["credentials"] = credentials
	}
	if model := request.Agent.model(); model != "" {
		configuration["model"] = model
	}

	return configuration
}

func prFeedbackChecksRunnerSteps() []any {
	return []any{
		map[string]any{
			"name": "Set Up Git User",
			"type": "bash",
			"command": strings.Join([]string{
				"git config --global user.email \"superplaneagent@superplane.com\"",
				"git config --global user.name \"SuperPlane Agent\"",
			}, "\n"),
		},
		map[string]any{
			"name": "Checkout Pull Request",
			"type": "bash",
			"command": strings.Join([]string{
				"set -euo pipefail",
				`git clone --depth 1 "https://x-access-token:${GITHUB_TOKEN}@github.com/${REPO}.git" repo`,
				"cd repo",
				`if [ -z "${PR_HEAD:-}" ]; then`,
				`  PR_HEAD=$(curl -fsSL -H "Authorization: Bearer ${GITHUB_TOKEN}" -H "Accept: application/vnd.github+json" "https://api.github.com/repos/${REPO}/pulls/${PR_NUMBER}" | jq -r .head.ref)`,
				"fi",
				`if [ -z "${PR_HEAD}" ] || [ "${PR_HEAD}" = "null" ]; then`,
				`  echo "Could not resolve the pull request head branch." >&2`,
				"  exit 1",
				"fi",
				`git fetch origin "pull/${PR_NUMBER}/head:${PR_HEAD}"`,
				`git checkout "${PR_HEAD}"`,
			}, "\n"),
		},
		map[string]any{
			"name":             "Set Up DCO Signing",
			"type":             "bash",
			"workingDirectory": "repo",
			"command": strings.Join([]string{
				"cat > .git/hooks/prepare-commit-msg <<'HOOK'",
				`git interpret-trailers --in-place --if-exists doNothing \`,
				`  --trailer "Signed-off-by: SuperPlane Agent <superplaneagent@superplane.com>" "$1"`,
				"",
				`printf '%s\n' "${COAUTHORS:-}" | while IFS= read -r trailer; do`,
				`  if [ -n "$trailer" ]; then`,
				`    git interpret-trailers --in-place --if-exists addIfDifferent --trailer "$trailer" "$1"`,
				"  fi",
				"done",
				"",
				"exit 0",
				"HOOK",
				"",
				"chmod +x .git/hooks/prepare-commit-msg",
			}, "\n"),
		},
		map[string]any{
			"name":             "Fix Failed Checks",
			"type":             "prompt",
			"workingDirectory": "repo",
			"prompt":           prFeedbackChecksPrompt(),
		},
		map[string]any{
			"name":             "Commit and Push",
			"type":             "bash",
			"workingDirectory": "repo",
			"command": strings.Join([]string{
				"set -euo pipefail",
				`REMOTE_HEAD=$(curl -fsSL -H "Authorization: Bearer ${GITHUB_TOKEN}" -H "Accept: application/vnd.github+json" "https://api.github.com/repos/${REPO}/pulls/${PR_NUMBER}" | jq -r .head.sha)`,
				`if [ -z "${REMOTE_HEAD}" ] || [ "${REMOTE_HEAD}" = "null" ]; then`,
				`  echo "Could not read the remote pull request head." >&2`,
				"  exit 1",
				"fi",
				`if [ "${REMOTE_HEAD}" != "${PR_REVISION}" ]; then`,
				`  echo "Remote pull request head changed. Stop without pushing."`,
				"  exit 0",
				"fi",
				"git add -A",
				"if ! git diff --cached --quiet; then",
				`  git commit -s -m "fix: repair failing checks on PR #${PR_NUMBER}"`,
				"  git push origin HEAD",
				"fi",
			}, "\n"),
		},
	}
}

func prFeedbackChecksPrompt() string {
	return strings.Join([]string{
		"You repair failed pull request checks for a SuperPlane work order.",
		"The repository is already checked out in the current working directory.",
		"Stay on this branch. Push at most one commit to this branch. Do not create a new branch.",
		"",
		"Repository: {{ root().data.repository.full_name }}",
		"Pull request: #{{ root().data.pull_request.number }}",
		"Revision: {{ root().data.pull_request.head.sha }}",
		"",
		"Failed checks are in FAILED_CHECKS.",
		"Address every failed check in that list in this run.",
		"Read logs from GitHub or from an available external integration.",
		"Use the GitHub token in GITHUB_TOKEN.",
		"",
		"Verify the remote pull request head before you push.",
		"Stop without pushing when the remote head differs from this revision.",
		"Keep the change focused. Add tests where they are needed.",
		"Do not report a work-order check or add a work-order comment.",
	}, "\n")
}

func prFeedbackPRHeadSHAExpression() string {
	return "{{ root().data.pull_request.head.sha }}"
}

func prFeedbackChecksWaitingDescriptionExpression() string {
	return "Waiting for checks on {{ root().data.pull_request.head.sha[:7] }}"
}

func prFeedbackChecksPassedDescriptionExpression() string {
	return "Checks passed on {{ root().data.pull_request.head.sha[:7] }}"
}

func prFeedbackChecksRepairDescriptionExpression() string {
	return "Fixing failed checks on {{ root().data.pull_request.head.sha[:7] }}"
}

func prFeedbackChecksTimeoutDescriptionExpression() string {
	return "Stopped waiting for checks on {{ root().data.pull_request.head.sha[:7] }}"
}

func prFeedbackChecksLimitDescriptionExpression(maximumAttempts int) string {
	if maximumAttempts < 1 {
		maximumAttempts = prFeedbackDefaultMaximumAttempts
	}
	return "Automatic fixes paused after " + attemptCountLabel(maximumAttempts)
}

func prFeedbackChecksLimitStatusNoteConfiguration(maximumAttempts int) map[string]any {
	return map[string]any{
		"orderId":             prFeedbackWorkOrderIDExpression(),
		"noteKey":             prFeedbackStatusNoteKey,
		"headline":            "Automatic fixes did not succeed",
		"body":                prFeedbackChecksLimitStatusNoteBody(maximumAttempts),
		"ctaLabel":            prFeedbackReviewPRCtaLabelExpression(),
		"ctaUrl":              prFeedbackReviewPRCtaURLExpression(),
		"showOnlyWhenWaiting": true,
	}
}

func prFeedbackChecksLimitStatusNoteBody(maximumAttempts int) string {
	if maximumAttempts < 1 {
		maximumAttempts = prFeedbackDefaultMaximumAttempts
	}
	return "SuperPlane paused automatic fixes after " + attemptCountLabel(maximumAttempts) +
		". Review the pull request and fix the remaining checks."
}

func prFeedbackWorkOrderIDExpression() string {
	return `{{ $["Find Pull Request"].data.workOrder.id }}`
}

func prFeedbackReviewPRCtaLabelExpression() string {
	return "Review PR #{{ root().data.pull_request.number }}"
}

func prFeedbackReviewPRCtaURLExpression() string {
	return "{{ root().data.pull_request.html_url }}"
}

func attemptCountLabel(count int) string {
	if count == 1 {
		return "1 attempt"
	}
	return fmt.Sprintf("%d attempts", count)
}

func prFeedbackFailedChecksExpression() string {
	return `{{ join(map($["Wait For Pull Request Checks"].data.failedChecks ?? [], .name + " " + .conclusion + " " + (.detailsUrl ?? "")), "\n") }}`
}

func checkNamesNodeValue(names []string) []any {
	values := make([]any, 0, len(names))
	for _, name := range names {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		values = append(values, trimmed)
	}
	return values
}
