package factories

import (
	"strings"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
)

const (
	prFeedbackPushTriggerNodeID         = "on-base-push"
	prFeedbackListNodeID                = "list-pull-requests"
	prFeedbackForEachNodeID             = "for-each-pull-request"
	prFeedbackWaitMergeableNodeID       = "wait-pr-mergeable"
	prFeedbackStartConflictRepairNodeID = "start-conflict-repair"
	prFeedbackConflictsRunnerNodeID     = "resolve-pr-conflicts"

	prFeedbackListComponent          = "listPullRequests"
	prFeedbackForEachComponent       = "forEach"
	prFeedbackWaitMergeableComponent = "github.waitForPullRequestMergeable"

	prFeedbackConflictsDefaultName        = "Resolve pull request conflicts"
	prFeedbackConflictsDefaultDescription = "Wait for merge conflicts and start one agent run when GitHub reports a conflict."
	prFeedbackConflictsDefaultBaseBranch  = "main"

	prFeedbackWaitMergeableNodeName = "Wait For Pull Request Mergeable"
	prFeedbackForEachNodeName       = "For Each Pull Request"
)

func buildConflictsPRFeedbackCanvas(request prFeedbackBuildRequest) *yaml.Canvas {
	name := prFeedbackCanvasName(request, prFeedbackConflictsDefaultName)
	baseBranch := conflictsBaseBranch(request.BaseBranch)

	return &yaml.Canvas{
		APIVersion: yaml.APIVersion,
		Kind:       yaml.KindCanvas,
		Metadata: &yaml.CanvasMetadata{
			Name:        name,
			Description: prFeedbackConflictsDefaultDescription,
		},
		Spec: &yaml.CanvasSpec{
			Edges: []yaml.Edge{
				{Channel: "default", SourceID: prFeedbackPullRequestTriggerNodeID, TargetID: prFeedbackFindNodeID},
				{Channel: "found", SourceID: prFeedbackFindNodeID, TargetID: prFeedbackWaitMergeableNodeID},
				{Channel: "default", SourceID: prFeedbackPushTriggerNodeID, TargetID: prFeedbackListNodeID},
				{Channel: "default", SourceID: prFeedbackListNodeID, TargetID: prFeedbackForEachNodeID},
				{Channel: "item", SourceID: prFeedbackForEachNodeID, TargetID: prFeedbackWaitMergeableNodeID},
				{Channel: "conflicted", SourceID: prFeedbackWaitMergeableNodeID, TargetID: prFeedbackStartConflictRepairNodeID},
				{Channel: "default", SourceID: prFeedbackStartConflictRepairNodeID, TargetID: prFeedbackConflictsRunnerNodeID},
				{Channel: "limitReached", SourceID: prFeedbackStartConflictRepairNodeID, TargetID: prFeedbackPauseFixesNodeID},
				{Channel: "default", SourceID: prFeedbackPauseFixesNodeID, TargetID: prFeedbackAnnounceLimitNodeID},
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
					Position:    yaml.Position{X: 80, Y: 140},
				},
				{
					ID:        prFeedbackPushTriggerNodeID,
					Name:      "On Base Push",
					Type:      yaml.NodeTypeTrigger,
					Component: "github.onPush",
					Configuration: map[string]any{
						"repository": request.Repository,
						"refs":       conflictsPushRefsNodeValue(baseBranch),
					},
					Integration: request.Binding.integrationRef(),
					Position:    yaml.Position{X: 80, Y: 400},
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
					Position: yaml.Position{X: 360, Y: 140},
				},
				{
					ID:        prFeedbackListNodeID,
					Name:      "List Pull Requests",
					Type:      yaml.NodeTypeAction,
					Component: prFeedbackListComponent,
					Configuration: map[string]any{
						"repository": request.Repository,
					},
					Position: yaml.Position{X: 360, Y: 400},
				},
				{
					ID:        prFeedbackForEachNodeID,
					Name:      prFeedbackForEachNodeName,
					Type:      yaml.NodeTypeAction,
					Component: prFeedbackForEachComponent,
					Configuration: map[string]any{
						"arrayExpression": `{{ $["List Pull Requests"].data.pullRequests }}`,
					},
					Position: yaml.Position{X: 540, Y: 400},
				},
				{
					ID:            prFeedbackWaitMergeableNodeID,
					Name:          prFeedbackWaitMergeableNodeName,
					Type:          yaml.NodeTypeAction,
					Component:     prFeedbackWaitMergeableComponent,
					Configuration: prFeedbackWaitMergeableConfiguration(request),
					Integration:   request.Binding.integrationRef(),
					Position:      yaml.Position{X: 720, Y: 260},
				},
				{
					ID:        prFeedbackStartConflictRepairNodeID,
					Name:      "Start Conflict Repair",
					Type:      yaml.NodeTypeAction,
					Component: prFeedbackActivityComponent,
					Configuration: map[string]any{
						"pullRequestId": prFeedbackConflictsPullRequestIDExpression(),
						"revision":      prFeedbackConflictsHeadSHAExpression(),
						"access":        core.PullRequestActivityAccessExclusive,
						"description":   prFeedbackConflictsRepairDescriptionExpression(),
					},
					Position: yaml.Position{X: 900, Y: 260},
				},
				{
					ID:            prFeedbackConflictsRunnerNodeID,
					Name:          "Resolve Pull Request Conflicts",
					Type:          yaml.NodeTypeAction,
					Component:     request.Agent.component(),
					Configuration: prFeedbackConflictsRunnerConfiguration(request),
					Position:      yaml.Position{X: 1080, Y: 260},
				},
				{
					ID:        prFeedbackPauseFixesNodeID,
					Name:      "Pause Automatic Fixes",
					Type:      yaml.NodeTypeAction,
					Component: prFeedbackUpdateActivityComponent,
					Configuration: map[string]any{
						"description": prFeedbackConflictsLimitDescriptionExpression(request.MaximumAttempts),
					},
					Position: yaml.Position{X: 1080, Y: 420},
				},
				{
					ID:            prFeedbackAnnounceLimitNodeID,
					Name:          "Set Fixes Paused Note",
					Type:          yaml.NodeTypeAction,
					Component:     prFeedbackSetStatusNoteComponent,
					Configuration: prFeedbackConflictsLimitStatusNoteConfiguration(request.MaximumAttempts),
					Position:      yaml.Position{X: 1260, Y: 420},
				},
			},
		},
	}
}

func prFeedbackWaitMergeableConfiguration(request prFeedbackBuildRequest) map[string]any {
	return map[string]any{
		"repository": request.Repository,
		"number":     prFeedbackConflictsNumberExpression(),
	}
}

func prFeedbackConflictsRunnerConfiguration(request prFeedbackBuildRequest) map[string]any {
	configuration := map[string]any{
		"machineType":             prFeedbackMachineType,
		"executionTimeoutSeconds": prFeedbackTimeoutSeconds,
		"steps":                   prFeedbackConflictsRunnerSteps(),
		"environmentFrom":         prFeedbackEnvironmentFrom(request.Binding, nil),
		"environment": []any{
			map[string]any{
				"name":        "REPO",
				"value":       "{{ root().data.repository.full_name }}",
				"valueSource": "literal",
			},
			map[string]any{
				"name":        "PR_NUMBER",
				"value":       prFeedbackConflictsNumberExpression(),
				"valueSource": "literal",
			},
			map[string]any{
				"name":        "PR_HEAD",
				"value":       "{{ root().data.pull_request?.head?.ref ?? \"\" }}",
				"valueSource": "literal",
			},
			map[string]any{
				"name":        "PR_REVISION",
				"value":       prFeedbackConflictsHeadSHAExpression(),
				"valueSource": "literal",
			},
			map[string]any{
				"name":        "BASE_BRANCH",
				"value":       prFeedbackConflictsBaseRefExpression(),
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

func prFeedbackConflictsRunnerSteps() []any {
	steps := prFeedbackCheckoutAndDCOSteps()
	return append(steps,
		map[string]any{
			"name":             "Resolve Pull Request Conflicts",
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
				"git add -A",
				"if ! git diff --cached --quiet; then",
				`  git commit -s -m "fix: resolve merge conflicts on PR #${PR_NUMBER}"`,
				"  git push origin HEAD",
				"fi",
			}, "\n"),
		},
	)
}

func prFeedbackConflictsPrompt() string {
	return strings.Join([]string{
		"You resolve a merge conflict for a SuperPlane work order.",
		"The repository is already checked out on the pull request branch in the current working directory.",
		"",
		"Repository: {{ root().data.repository.full_name }}",
		"Base branch: ${BASE_BRANCH}",
		"",
		"Stay on this branch. Do not create a new branch or a new pull request.",
		"Merge or rebase the base branch into this branch.",
		"Resolve every conflict marker. Keep the intent of this pull request; do not discard its changes.",
		"Do not rewrite unrelated history.",
		"",
		"Verify the remote pull request head before you push.",
		"Stop without pushing when the remote head differs from this revision.",
		"Push at most one commit, or the result of the rebase.",
		"Do not report a work-order check. Do not add a work-order comment.",
	}, "\n")
}

func prFeedbackConflictsNumberExpression() string {
	return `{{ root().data.pull_request?.number ?? previous().data.item?.number }}`
}

func prFeedbackConflictsHeadSHAExpression() string {
	return `{{ previous().data.sha }}`
}

func prFeedbackConflictsBaseRefExpression() string {
	return `{{ $["` + prFeedbackWaitMergeableNodeName + `"]?.data.baseRef }}`
}

func prFeedbackConflictsPullRequestIDExpression() string {
	return `{{ $["Find Pull Request"]?.data.pullRequest.id ?? $["` + prFeedbackForEachNodeName + `"]?.data.item.id }}`
}

func prFeedbackConflictsWorkOrderIDExpression() string {
	return `{{ $["Find Pull Request"]?.data.workOrder.id ?? $["` + prFeedbackForEachNodeName + `"]?.data.item.workOrderId }}`
}

func prFeedbackConflictsRepairDescriptionExpression() string {
	return `Resolving conflicts on {{ previous().data.sha[:7] }}`
}

func prFeedbackConflictsLimitDescriptionExpression(maximumAttempts int) string {
	if maximumAttempts < 1 {
		maximumAttempts = prFeedbackDefaultMaximumAttempts
	}
	return "Automatic conflict fixes paused after " + attemptCountLabel(maximumAttempts)
}

func prFeedbackConflictsLimitStatusNoteConfiguration(maximumAttempts int) map[string]any {
	return map[string]any{
		"orderId":             prFeedbackConflictsWorkOrderIDExpression(),
		"noteKey":             prFeedbackStatusNoteKey,
		"headline":            "Automatic conflict fixes did not succeed",
		"body":                prFeedbackConflictsLimitStatusNoteBody(maximumAttempts),
		"ctaLabel":            prFeedbackConflictsReviewPRCtaLabelExpression(),
		"ctaUrl":              prFeedbackConflictsReviewPRCtaURLExpression(),
		"showOnlyWhenWaiting": true,
	}
}

func prFeedbackConflictsLimitStatusNoteBody(maximumAttempts int) string {
	if maximumAttempts < 1 {
		maximumAttempts = prFeedbackDefaultMaximumAttempts
	}
	return "SuperPlane paused automatic conflict fixes after " + attemptCountLabel(maximumAttempts) +
		". Review the pull request and resolve the remaining conflicts."
}

func prFeedbackConflictsReviewPRCtaLabelExpression() string {
	return `Review PR #{{ root().data.pull_request?.number ?? $["` + prFeedbackForEachNodeName + `"]?.data.item.number }}`
}

func prFeedbackConflictsReviewPRCtaURLExpression() string {
	return `{{ root().data.pull_request?.html_url ?? $["` + prFeedbackForEachNodeName + `"]?.data.item.url }}`
}

func conflictsBaseBranch(baseBranch string) string {
	trimmed := strings.TrimSpace(baseBranch)
	if trimmed == "" {
		return prFeedbackConflictsDefaultBaseBranch
	}
	return trimmed
}

func conflictsPushRefsNodeValue(baseBranch string) []any {
	return []any{
		map[string]any{
			"type":  configuration.PredicateTypeEquals,
			"value": "refs/heads/" + conflictsBaseBranch(baseBranch),
		},
	}
}

// conflictsBaseBranchFromPushNode reads the base branch back out of the
// onPush trigger's refs predicate list. It returns "" when the node is nil
// or the refs list has no equals predicate targeting a heads ref, which the
// caller treats as unhealthy.
func conflictsBaseBranchFromPushNode(node *models.Node) string {
	if node == nil {
		return ""
	}

	refs, ok := node.Configuration["refs"].([]any)
	if !ok {
		return ""
	}

	const prefix = "refs/heads/"
	for _, ref := range refs {
		predicate, ok := ref.(map[string]any)
		if !ok {
			continue
		}
		predicateType, _ := predicate["type"].(string)
		if predicateType != configuration.PredicateTypeEquals {
			continue
		}
		value, _ := predicate["value"].(string)
		if branch, found := strings.CutPrefix(value, prefix); found && branch != "" {
			return branch
		}
	}

	return ""
}
