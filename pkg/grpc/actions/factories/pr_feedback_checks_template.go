package factories

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
)

const (
	prFeedbackPullRequestTriggerNodeID = "on-pull-request"
	prFeedbackWaitChecksNodeID         = "wait-pr-checks"
	prFeedbackMarkPassedNodeID         = "mark-checks-passed"
	prFeedbackStartRepairNodeID        = "start-check-repair"
	prFeedbackPauseFixesNodeID         = "pause-automatic-fixes"
	prFeedbackStopWaitingNodeID        = "stop-waiting-for-checks"
	prFeedbackRecordTimeoutNodeID      = "record-check-timeout"

	prFeedbackWaitChecksComponent      = "github.waitForPullRequestChecks"
	prFeedbackUpdateActivityComponent  = "updatePullRequestActivity"
	prFeedbackAddRunErrorComponent     = "addRunError"
	prFeedbackChecksDefaultName        = "Fix pull request checks"
	prFeedbackChecksDefaultDescription = "Wait for pull request checks and start one agent run when selected checks fail."
	prFeedbackWaitChecksNodeName       = "Wait For Pull Request Checks"
)

func buildChecksPRFeedbackCanvas(request prFeedbackBuildRequest) *yaml.Canvas {
	name := prFeedbackCanvasName(request, prFeedbackChecksDefaultName)
	provider := prFeedbackVCSProvider(request)

	return withPRFeedbackConcurrency(&yaml.Canvas{
		APIVersion: yaml.APIVersion,
		Kind:       yaml.KindCanvas,
		Metadata: &yaml.CanvasMetadata{
			Name:        name,
			Description: prFeedbackChecksDefaultDescription,
		},
		Spec: &yaml.CanvasSpec{
			Edges: []yaml.Edge{
				{Channel: "default", SourceID: prFeedbackPullRequestTriggerNodeID, TargetID: prFeedbackActivityNodeID},
				{Channel: "default", SourceID: prFeedbackActivityNodeID, TargetID: prFeedbackWaitChecksNodeID},
				{Channel: "passed", SourceID: prFeedbackWaitChecksNodeID, TargetID: prFeedbackMarkPassedNodeID},
				{Channel: "failed", SourceID: prFeedbackWaitChecksNodeID, TargetID: prFeedbackStartRepairNodeID},
				{Channel: "timedOut", SourceID: prFeedbackWaitChecksNodeID, TargetID: prFeedbackStopWaitingNodeID},
				{Channel: "default", SourceID: prFeedbackStartRepairNodeID, TargetID: prFeedbackRunnerNodeID},
				{Channel: "limitReached", SourceID: prFeedbackStartRepairNodeID, TargetID: prFeedbackPauseFixesNodeID},
				{Channel: "default", SourceID: prFeedbackStopWaitingNodeID, TargetID: prFeedbackRecordTimeoutNodeID},
			},
			Nodes: []yaml.Node{
				{
					ID:        prFeedbackPullRequestTriggerNodeID,
					Name:      "On Pull Request",
					Type:      yaml.NodeTypeTrigger,
					Component: prFeedbackTriggerComponent(provider),
					Configuration: map[string]any{
						"repository":              request.Repository,
						"actions":                 prFeedbackTriggerActions(provider),
						"onlyFactoryPullRequests": true,
					},
					Integration: request.Binding.integrationRef(),
					Position:    yaml.Position{X: 80, Y: 260},
				},
				{
					ID:        prFeedbackActivityNodeID,
					Name:      "Add Pull Request Activity",
					Type:      yaml.NodeTypeAction,
					Component: prFeedbackActivityComponent,
					Configuration: map[string]any{
						"pullRequestId": `{{ root().data.pullRequest.id }}`,
						"revision":      prFeedbackPRHeadSHAExpression(),
						"access":        "concurrent",
						"title":         prFeedbackChecksWaitingTitleExpressionFor(prFeedbackVCSProvider(request)),
					},
					Position: yaml.Position{X: 500, Y: 260},
				},
				{
					ID:            prFeedbackWaitChecksNodeID,
					Name:          prFeedbackWaitChecksNodeName,
					Type:          yaml.NodeTypeAction,
					Component:     prFeedbackWaitComponent(provider),
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
						"title":       prFeedbackChecksPassedTitleExpressionFor(prFeedbackVCSProvider(request)),
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
						"title":       prFeedbackChecksRepairTitleExpressionFor(prFeedbackVCSProvider(request)),
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
						"title": prFeedbackChecksLimitDescriptionExpression(request.MaximumAttempts),
					},
					Position: yaml.Position{X: 1000, Y: 400},
				},
				{
					ID:        prFeedbackStopWaitingNodeID,
					Name:      "Stop Waiting For Checks",
					Type:      yaml.NodeTypeAction,
					Component: prFeedbackUpdateActivityComponent,
					Configuration: map[string]any{
						"title": prFeedbackChecksTimeoutDescriptionExpression(),
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
	})
}

func prFeedbackTriggerComponent(provider string) string {
	if provider == models.ProviderBitbucket {
		return "bitbucket.onPullRequest"
	}
	return "github.onPullRequest"
}

func prFeedbackTriggerActions(provider string) []any {
	if provider == models.ProviderBitbucket {
		return []any{"created", "updated"}
	}
	return []any{"opened", "reopened", "synchronize"}
}

func prFeedbackWaitComponent(provider string) string {
	if provider == models.ProviderBitbucket {
		return "bitbucket.waitForBuilds"
	}
	return prFeedbackWaitChecksComponent
}

func isPrFeedbackWaitComponent(component string) bool {
	return component == prFeedbackWaitChecksComponent || component == "bitbucket.waitForBuilds"
}

func prFeedbackWaitChecksConfiguration(request prFeedbackBuildRequest) map[string]any {
	if prFeedbackVCSProvider(request) == models.ProviderBitbucket {
		return map[string]any{
			"repository": request.Repository,
			"ref":        prFeedbackPRHeadSHAExpression(),
			"buildKeys":  checkNamesNodeValue(request.CheckNames),
		}
	}
	return map[string]any{
		"repository": request.Repository,
		"ref":        prFeedbackPRHeadSHAExpression(),
		"checkNames": checkNamesNodeValue(request.CheckNames),
	}
}

func prFeedbackChecksRunnerConfiguration(request prFeedbackBuildRequest) map[string]any {
	configuration := map[string]any{
		"machineType":             prFeedbackMachineType,
		"executionTimeoutSeconds": prFeedbackTimeoutSeconds,
		"includeVisualEvidence":   request.IncludeVisualEvidence,
		"steps":                   prFeedbackChecksRunnerSteps(request),
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
	request.Agent.applyLLMProvider(configuration)

	return configuration
}

func prFeedbackChecksRunnerSteps(request prFeedbackBuildRequest) []any {
	steps := []any{
		map[string]any{
			"name": "Set Up Git User",
			"type": "bash",
			"command": strings.Join([]string{
				"git config --global user.email \"" + runner.FactoryAgentEmail + "\"",
				"git config --global user.name \"" + runner.FactoryAgentName + "\"",
			}, "\n"),
		},
	}
	if prFeedbackVCSProvider(request) == models.ProviderBitbucket {
		return append(steps,
			map[string]any{
				"name": "Checkout Pull Request",
				"type": "bash",
				"command": strings.Join([]string{
					"set -euo pipefail",
					`git clone "https://bitbucket.org/${REPO}.git" repo`,
					"cd repo",
					`if [ -z "${PR_HEAD:-}" ]; then`,
					`  PR_HEAD=$(curl -fsSL -H "Authorization: Bearer ${BITBUCKET_TOKEN}" "https://api.bitbucket.org/2.0/repositories/${REPO}/pullrequests/${PR_NUMBER}" | jq -r .source.branch.name)`,
					"fi",
					`if [ -z "${PR_HEAD}" ] || [ "${PR_HEAD}" = "null" ]; then`,
					`  echo "Could not resolve the pull request head branch." >&2`,
					"  exit 1",
					"fi",
					`git fetch origin "${PR_HEAD}:${PR_HEAD}"`,
					`git checkout "${PR_HEAD}"`,
				}, "\n"),
			},
			map[string]any{
				"name":             "Set Up DCO Signing",
				"type":             "bash",
				"workingDirectory": "repo",
				"command":          runner.FactoryRepoCommitSetup(),
			},
			map[string]any{
				"name":             "Fix Failed Builds",
				"type":             "prompt",
				"workingDirectory": "repo",
				"prompt":           prFeedbackChecksPrompt(request),
			},
			prFeedbackBitbucketCommitPushStep(),
		)
	}
	return append(steps,
		map[string]any{
			"name": "Checkout Pull Request",
			"type": "bash",
			"command": strings.Join([]string{
				"set -euo pipefail",
				"gh auth setup-git --hostname github.com --force",
				`git clone --depth 1 "https://github.com/${REPO}.git" repo`,
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
			"command":          runner.FactoryRepoCommitSetup(),
		},
		map[string]any{
			"name":             "Fix Failed Checks",
			"type":             "prompt",
			"workingDirectory": "repo",
			"prompt":           prFeedbackChecksPrompt(request),
		},
		prFeedbackGitHubCommitPushStep(),
	)
}

func prFeedbackGitHubCommitPushStep() map[string]any {
	return map[string]any{
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
	}
}

func prFeedbackBitbucketCommitPushStep() map[string]any {
	return map[string]any{
		"name":             "Commit and Push",
		"type":             "bash",
		"workingDirectory": "repo",
		"command": strings.Join([]string{
			"set -euo pipefail",
			`REMOTE_HEAD=$(curl -fsSL -H "Authorization: Bearer ${BITBUCKET_TOKEN}" "https://api.bitbucket.org/2.0/repositories/${REPO}/pullrequests/${PR_NUMBER}" | jq -r .source.commit.hash)`,
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
			`  git commit -s -m "fix: repair failing builds on PR #${PR_NUMBER}"`,
			"  git push origin HEAD",
			"fi",
		}, "\n"),
	}
}

func prFeedbackChecksPrompt(request prFeedbackBuildRequest) string {
	if prFeedbackVCSProvider(request) == models.ProviderBitbucket {
		return strings.Join([]string{
			"You repair failed pull request builds for a SuperPlane work order.",
			"The repository is already checked out in the current working directory.",
			"Stay on this branch. Push at most one commit to this branch. Do not create a new branch.",
			"",
			"Repository: {{ root().data.repository.full_name }}",
			"Pull request: #{{ root().data.pull_request.number }}",
			"Revision: {{ root().data.pull_request.head.sha }}",
			"",
			"Failed builds are in FAILED_CHECKS.",
			"Address every failed build in that list in this run.",
			"Read logs from Bitbucket Pipelines or from the build URL in FAILED_CHECKS.",
			"Use the Bitbucket token in BITBUCKET_TOKEN.",
			"",
			"Verify the remote pull request head before you push.",
			"Stop without pushing when the remote head differs from this revision.",
			"Keep the change focused. Add tests where they are needed.",
			"Do not report a work-order check or add a work-order comment.",
			runner.FactoryCommitIdentityPrompt,
		}, "\n")
	}
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
		runner.FactoryCommitIdentityPrompt,
	}, "\n")
}

func prFeedbackPRHeadSHAExpression() string {
	return "{{ root().data.pull_request.head.sha }}"
}

func prFeedbackChecksWaitingTitleExpression() string {
	return prFeedbackChecksWaitingTitleExpressionFor("")
}

func prFeedbackChecksWaitingTitleExpressionFor(provider string) string {
	if provider == models.ProviderBitbucket {
		return prFeedbackChecksSHATitleExpressionFor("Waiting for builds on ", provider)
	}
	return prFeedbackChecksSHATitleExpressionFor("Waiting for checks on ", "")
}

func prFeedbackChecksPassedTitleExpression() string {
	return prFeedbackChecksPassedTitleExpressionFor("")
}

func prFeedbackChecksPassedTitleExpressionFor(provider string) string {
	if provider == models.ProviderBitbucket {
		return prFeedbackChecksSHATitleExpressionFor("Builds passed on ", provider)
	}
	return prFeedbackChecksSHATitleExpressionFor("Checks passed on ", "")
}

func prFeedbackChecksRepairTitleExpression() string {
	return prFeedbackChecksRepairTitleExpressionFor("")
}

func prFeedbackChecksRepairTitleExpressionFor(provider string) string {
	if provider == models.ProviderBitbucket {
		return prFeedbackChecksSHATitleExpressionFor("Fixing failed builds on ", provider)
	}
	return prFeedbackChecksSHATitleExpressionFor("Fixing failed checks on ", "")
}

func prFeedbackChecksSHATitleExpression(prefix string) string {
	return prFeedbackChecksSHATitleExpressionFor(prefix, "")
}

func prFeedbackChecksSHATitleExpressionFor(prefix, provider string) string {
	return `{{ "` + prefix + `[" + root().data.pull_request.head.sha[:7] + "](" + ` +
		prFeedbackCommitURLSourceFor(provider) + ` + ")" }}`
}

func prFeedbackCommitURLSource() string {
	return prFeedbackCommitURLSourceFor("")
}

func prFeedbackCommitURLSourceFor(provider string) string {
	if provider == models.ProviderBitbucket {
		return `(root().data.repository.links?.html.href ?? ("https://bitbucket.org/" + root().data.repository.full_name))` +
			` + "/commits/" + root().data.pull_request.head.sha`
	}
	return `(root().data.repository.html_url ?? ("https://github.com/" + root().data.repository.full_name))` +
		` + "/commit/" + root().data.pull_request.head.sha`
}

func prFeedbackChecksPassedDescriptionExpression() string {
	return `{{ ` + prFeedbackChecksMarkdownListSource(prFeedbackWaitChecksSelectedSource()) + ` }}`
}

func prFeedbackChecksRepairDescriptionExpression() string {
	return `{{ "Failed checks\n" + ` + prFeedbackChecksMarkdownListSource(prFeedbackWaitChecksFailedSource()) + ` }}`
}

func prFeedbackWaitChecksSelectedSource() string {
	return `$["` + prFeedbackWaitChecksNodeName + `"].data.selectedChecks ?? []`
}

func prFeedbackWaitChecksFailedSource() string {
	return `$["` + prFeedbackWaitChecksNodeName + `"].data.failedChecks ?? []`
}

func prFeedbackChecksMarkdownListSource(checksExpr string) string {
	return `join(map(` + checksExpr + `, ` + prFeedbackCheckMarkdownItemSource() + `), "\n")`
}

func prFeedbackCheckMarkdownItemSource() string {
	label := `.name + ((.description ?? "") != "" ? ": " + .description : "")`
	linked := `((.detailsUrl ?? "") != "" ? "[" + (` + label + `) + "](" + .detailsUrl + ")" : (` + label + `))`
	return `"· " + ` + linked + ` + ((.summary ?? "") != "" ? ": " + .summary : "")`
}

func prFeedbackChecksActivityExpressions(nodeID string) (string, string, bool) {
	return prFeedbackChecksActivityExpressionsFor(nodeID, "")
}

func prFeedbackChecksActivityExpressionsFor(nodeID, provider string) (string, string, bool) {
	switch nodeID {
	case prFeedbackActivityNodeID:
		return prFeedbackChecksWaitingTitleExpressionFor(provider), "", true
	case prFeedbackMarkPassedNodeID:
		return prFeedbackChecksPassedTitleExpressionFor(provider), prFeedbackChecksPassedDescriptionExpression(), true
	case prFeedbackStartRepairNodeID:
		return prFeedbackChecksRepairTitleExpressionFor(provider), prFeedbackChecksRepairDescriptionExpression(), true
	default:
		return "", "", false
	}
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

// prFeedbackWaitNamesKey names the required-builds field on wait nodes.
// Bitbucket waits select build keys; GitHub waits select check names.
func prFeedbackWaitNamesKey(factory *models.Factory) string {
	if factory != nil && factory.OnboardingConfigValue().EffectiveVCSProvider() == models.ProviderBitbucket {
		return "buildKeys"
	}
	return "checkNames"
}
