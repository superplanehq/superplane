package factories

import (
	"strings"

	"github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/yaml"
)

const (
	prFeedbackConflictTriggerNodeID       = "on-pull-request-conflict"
	prFeedbackConflictRunnerNodeID        = "resolve-merge-conflict"
	prFeedbackConflictsDefaultName        = "Fix merge conflicts"
	prFeedbackConflictsDefaultDescription = "Start one agent run when a pull request has merge conflicts."
)

func buildConflictsPRFeedbackCanvas(request prFeedbackBuildRequest) *yaml.Canvas {
	name := prFeedbackCanvasName(request, prFeedbackConflictsDefaultName)

	return withPRFeedbackConcurrency(&yaml.Canvas{
		APIVersion: yaml.APIVersion,
		Kind:       yaml.KindCanvas,
		Metadata: &yaml.CanvasMetadata{
			Name:        name,
			Description: prFeedbackConflictsDefaultDescription,
		},
		Spec: &yaml.CanvasSpec{
			Edges: []yaml.Edge{
				{Channel: "default", SourceID: prFeedbackConflictTriggerNodeID, TargetID: prFeedbackFindNodeID},
				{Channel: "found", SourceID: prFeedbackFindNodeID, TargetID: prFeedbackActivityNodeID},
				{Channel: "default", SourceID: prFeedbackActivityNodeID, TargetID: prFeedbackStartRepairNodeID},
				{Channel: "default", SourceID: prFeedbackStartRepairNodeID, TargetID: prFeedbackConflictRunnerNodeID},
				{Channel: "limitReached", SourceID: prFeedbackStartRepairNodeID, TargetID: prFeedbackPauseFixesNodeID},
			},
			Nodes: []yaml.Node{
				{
					ID:        prFeedbackConflictTriggerNodeID,
					Name:      "On Merge Conflict",
					Type:      yaml.NodeTypeTrigger,
					Component: factory.OnPullRequestConflictTriggerName,
					Configuration: map[string]any{
						"repository": request.Repository,
					},
					Position: yaml.Position{X: 80, Y: 260},
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
						"title":         prFeedbackConflictDetectedTitleExpression(),
					},
					Position: yaml.Position{X: 500, Y: 260},
				},
				{
					ID:        prFeedbackStartRepairNodeID,
					Name:      "Start Conflict Repair",
					Type:      yaml.NodeTypeAction,
					Component: prFeedbackUpdateActivityComponent,
					Configuration: map[string]any{
						"access": "exclusive",
						"title":  prFeedbackConflictRepairTitleExpression(),
					},
					Position: yaml.Position{X: 820, Y: 260},
				},
				{
					ID:            prFeedbackConflictRunnerNodeID,
					Name:          "Resolve Merge Conflict",
					Type:          yaml.NodeTypeAction,
					Component:     request.Agent.component(),
					Configuration: prFeedbackConflictsRunnerConfiguration(request),
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
			},
		},
	})
}

func prFeedbackConflictsRunnerConfiguration(request prFeedbackBuildRequest) map[string]any {
	configuration := map[string]any{
		"machineType":             prFeedbackMachineType,
		"executionTimeoutSeconds": prFeedbackTimeoutSeconds,
		"includeVisualEvidence":   request.IncludeVisualEvidence,
		"steps":                   prFeedbackConflictsRunnerSteps(),
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
				"name":        "BASE_REF",
				"value":       "{{ root().data.pull_request.base.ref }}",
				"valueSource": "literal",
			},
			map[string]any{
				"name":        "HEAD_REPO",
				"value":       "{{ root().data.pull_request.head.repo.full_name }}",
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

func prFeedbackConflictsRunnerSteps() []any {
	return []any{
		map[string]any{
			"name": "Set Up Git User",
			"type": "bash",
			"command": strings.Join([]string{
				"git config --global user.email \"" + runner.FactoryAgentEmail + "\"",
				"git config --global user.name \"" + runner.FactoryAgentName + "\"",
			}, "\n"),
		},
		map[string]any{
			"name": "Checkout Pull Request",
			"type": "bash",
			"command": strings.Join([]string{
				"set -euo pipefail",
				"gh auth setup-git --hostname github.com --force",
				`git clone --filter=blob:none "https://github.com/${REPO}.git" repo`,
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
				`if [ -n "${BASE_REF:-}" ]; then`,
				`  git fetch origin "${BASE_REF}"`,
				"fi",
			}, "\n"),
		},
		map[string]any{
			"name":             "Set Up DCO Signing",
			"type":             "bash",
			"workingDirectory": "repo",
			"command":          runner.FactoryRepoCommitSetup(),
		},
		map[string]any{
			"name":             "Resolve Merge Conflict",
			"type":             "prompt",
			"workingDirectory": "repo",
			"prompt":           prFeedbackConflictsPrompt(),
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
				`HEAD_REPOSITORY="${HEAD_REPO:-}"`,
				`if [ -z "${HEAD_REPOSITORY}" ] || [ "${HEAD_REPOSITORY}" = "null" ]; then`,
				`  HEAD_REPOSITORY=$(curl -fsSL -H "Authorization: Bearer ${GITHUB_TOKEN}" -H "Accept: application/vnd.github+json" "https://api.github.com/repos/${REPO}/pulls/${PR_NUMBER}" | jq -r '.head.repo.full_name // empty')`,
				"fi",
				`if [ -z "${HEAD_REPOSITORY}" ] || [ "${HEAD_REPOSITORY}" = "null" ]; then`,
				`  echo "Could not resolve the pull request head repository." >&2`,
				"  exit 1",
				"fi",
				"git add -A",
				"if ! git diff --cached --quiet; then",
				`  git commit -s -m "fix: resolve merge conflict on PR #${PR_NUMBER}"`,
				"fi",
				`LOCAL_HEAD=$(git rev-parse HEAD)`,
				`if [ "${LOCAL_HEAD}" = "${REMOTE_HEAD}" ]; then`,
				`  echo "No commit to push."`,
				"  exit 0",
				"fi",
				`if [ "$(printf '%s' "${HEAD_REPOSITORY}" | tr '[:upper:]' '[:lower:]')" = "$(printf '%s' "${REPO}" | tr '[:upper:]' '[:lower:]')" ]; then`,
				`  git push origin "HEAD:refs/heads/${PR_HEAD}"`,
				"else",
				`  git push "https://github.com/${HEAD_REPOSITORY}.git" "HEAD:refs/heads/${PR_HEAD}"`,
				"fi",
			}, "\n"),
		},
	}
}

func prFeedbackConflictsPrompt() string {
	return strings.Join([]string{
		"You resolve merge conflicts on a pull request.",
		"The repository is already checked out on the pull request branch.",
		"Stay on this branch. Create at most one commit. Do not push.",
		"Do not create a new branch. Do not rebase. Do not force-push.",
		"",
		"Repository: {{ root().data.repository.full_name }}",
		"Pull request: #{{ root().data.pull_request.number }}",
		"Base branch: {{ root().data.pull_request.base.ref }}",
		"Revision: {{ root().data.pull_request.head.sha }}",
		"",
		"Fetch the base branch and merge it into this branch:",
		"git fetch origin {{ root().data.pull_request.base.ref }}",
		"git merge origin/{{ root().data.pull_request.base.ref }}",
		"",
		"Resolve every conflict. Keep the intent of both sides.",
		"Run the relevant tests.",
		"Create one signed commit. Do not push it.",
		"The next step pushes that commit to the pull request head repository.",
		"Do not push to the base repository when the head repository is a fork.",
		"Do not report a work-order check or add a work-order comment.",
		runner.FactoryCommitIdentityPrompt,
	}, "\n")
}

func prFeedbackConflictDetectedTitleExpression() string {
	return prFeedbackChecksSHATitleExpression("Merge conflict on ")
}

func prFeedbackConflictRepairTitleExpression() string {
	return prFeedbackChecksSHATitleExpression("Resolving merge conflict on ")
}
