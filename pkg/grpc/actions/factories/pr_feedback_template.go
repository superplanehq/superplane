package factories

import (
	"strings"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/yaml"
)

const (
	prFeedbackCommentTriggerNodeID = "on-pr-comment"
	prFeedbackReviewTriggerNodeID  = "on-pr-review"
	prFeedbackReplyTriggerNodeID   = "on-pr-review-reply"
	prFeedbackFindNodeID           = "find-pull-request"
	prFeedbackActivityNodeID       = "add-pr-activity"
	prFeedbackRunnerNodeID         = "address-pr-feedback"

	prFeedbackFindComponent     = "findPullRequest"
	prFeedbackActivityComponent = "addPullRequestActivity"

	prFeedbackDefaultName         = "Address PR feedback"
	prFeedbackDefaultDescription  = "Address pull request comments and reviews after a mention."
	prFeedbackDefaultMention      = "@superplaneagent"
	prFeedbackCommentScopeReplies = "replies"

	prFeedbackRunnerNodeName = "Address PR feedback"
	prFeedbackMachineType    = runner.MachineTypeE1LargeAMD64
	prFeedbackTimeoutSeconds = 3600
)

var prFeedbackTriggerNodeIDs = []string{
	prFeedbackCommentTriggerNodeID,
	prFeedbackReviewTriggerNodeID,
	prFeedbackReplyTriggerNodeID,
}

type prFeedbackBuildRequest struct {
	Name       string
	Repository string
	Mention    string
	IgnoreBots bool
	Binding    *intakeBinding
	Agent      *intakeAgent
}

func buildPRFeedbackCanvas(request prFeedbackBuildRequest) *yaml.Canvas {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		name = prFeedbackDefaultName
	}
	mention := strings.TrimSpace(request.Mention)
	if mention == "" {
		mention = prFeedbackDefaultMention
	}

	includeReviewSubmissions := false
	return &yaml.Canvas{
		APIVersion: yaml.APIVersion,
		Kind:       yaml.KindCanvas,
		Metadata: &yaml.CanvasMetadata{
			Name:        name,
			Description: prFeedbackDefaultDescription,
		},
		Spec: &yaml.CanvasSpec{
			Edges: []yaml.Edge{
				{Channel: "default", SourceID: prFeedbackCommentTriggerNodeID, TargetID: prFeedbackFindNodeID},
				{Channel: "default", SourceID: prFeedbackReviewTriggerNodeID, TargetID: prFeedbackFindNodeID},
				{Channel: "default", SourceID: prFeedbackReplyTriggerNodeID, TargetID: prFeedbackFindNodeID},
				{Channel: "found", SourceID: prFeedbackFindNodeID, TargetID: prFeedbackActivityNodeID},
				{Channel: "default", SourceID: prFeedbackActivityNodeID, TargetID: prFeedbackRunnerNodeID},
			},
			Nodes: []yaml.Node{
				{
					ID:            prFeedbackCommentTriggerNodeID,
					Name:          "On PR Comment",
					Type:          yaml.NodeTypeTrigger,
					Component:     "github.onPRComment",
					Configuration: prFeedbackTriggerConfiguration(request.Repository, mention, request.IgnoreBots),
					Integration:   request.Binding.integrationRef(),
					Position:      yaml.Position{X: 80, Y: 80},
				},
				{
					ID:            prFeedbackReviewTriggerNodeID,
					Name:          "On PR Review",
					Type:          yaml.NodeTypeTrigger,
					Component:     "github.onPRReview",
					Configuration: prFeedbackTriggerConfiguration(request.Repository, mention, request.IgnoreBots),
					Integration:   request.Binding.integrationRef(),
					Position:      yaml.Position{X: 80, Y: 260},
				},
				{
					ID:            prFeedbackReplyTriggerNodeID,
					Name:          "On PR Review Reply",
					Type:          yaml.NodeTypeTrigger,
					Component:     "github.onPRReviewComment",
					Configuration: prFeedbackReplyTriggerConfiguration(request.Repository, mention, request.IgnoreBots, includeReviewSubmissions),
					Integration:   request.Binding.integrationRef(),
					Position:      yaml.Position{X: 80, Y: 440},
				},
				{
					ID:        prFeedbackFindNodeID,
					Name:      "Find Pull Request",
					Type:      yaml.NodeTypeAction,
					Component: prFeedbackFindComponent,
					Configuration: map[string]any{
						"provider":   "github",
						"repository": "{{ root().data.repository.full_name }}",
						"number":     prFeedbackPRNumberExpression(),
						"url":        prFeedbackPRURLExpression(),
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
						"description":   prFeedbackActivityDescriptionExpression(),
					},
					Position: yaml.Position{X: 500, Y: 260},
				},
				{
					ID:            prFeedbackRunnerNodeID,
					Name:          prFeedbackRunnerNodeName,
					Type:          yaml.NodeTypeAction,
					Component:     request.Agent.component(),
					Configuration: prFeedbackRunnerConfiguration(request),
					Concurrency:   prFeedbackRunnerConcurrency(),
					Position:      yaml.Position{X: 640, Y: 260},
				},
			},
		},
	}
}

func prFeedbackReplyTriggerConfiguration(repository, mention string, ignoreBots, includeReviewSubmissions bool) map[string]any {
	configuration := prFeedbackTriggerConfiguration(repository, mention, ignoreBots)
	configuration["includeReviewSubmissions"] = includeReviewSubmissions
	configuration["commentScope"] = prFeedbackCommentScopeReplies
	return configuration
}

func prFeedbackTriggerConfiguration(repository, mention string, ignoreBots bool) map[string]any {
	configuration := map[string]any{
		"contentFilter": mention,
		"ignoreBots":    ignoreBots,
	}
	if strings.TrimSpace(repository) != "" {
		configuration["repository"] = repository
	}
	return configuration
}

func prFeedbackActivityDescriptionExpression() string {
	return "{{ root().data.comment?.body ?? root().data.review?.body ?? join(map(root().data.review_comments ?? [], .body), \"\\n\\n\") ?? \"\" }}"
}

func prFeedbackPRURLExpression() string {
	return "{{ root().data.pull_request?.html_url ?? root().data.issue?.pull_request?.html_url }}"
}

func prFeedbackPRNumberExpression() string {
	return "{{ root().data.pull_request?.number ?? root().data.issue?.number }}"
}

func prFeedbackPRHeadExpression() string {
	return "{{ root().data.pull_request?.head?.ref ?? \"\" }}"
}

func prFeedbackCoauthorsExpression() string {
	return `{{ order() == nil ? "" : join(map(filter(order().assignees, {#.email != ""}), "Co-authored-by: " + #.name + " <" + #.email + ">"), "\n") }}`
}

func prFeedbackRunnerConcurrency() *yaml.ConcurrencySpec {
	max := 1
	return &yaml.ConcurrencySpec{
		Key: "github-{{ root().data.repository.id }}-pr-{{ root().data.pull_request?.number ?? root().data.issue?.number }}",
		Max: &max,
	}
}

func prFeedbackRunnerConfiguration(request prFeedbackBuildRequest) map[string]any {
	configuration := map[string]any{
		"machineType":             prFeedbackMachineType,
		"executionTimeoutSeconds": prFeedbackTimeoutSeconds,
		"steps":                   prFeedbackRunnerSteps(),
		"environmentFrom":         prFeedbackGitHubEnvironmentFrom(request.Binding),
		"environment": []any{
			map[string]any{
				"name":        "REPO",
				"value":       "{{ root().data.repository.full_name }}",
				"valueSource": "literal",
			},
			map[string]any{
				"name":        "PR_NUMBER",
				"value":       prFeedbackPRNumberExpression(),
				"valueSource": "literal",
			},
			map[string]any{
				"name":        "PR_HEAD",
				"value":       prFeedbackPRHeadExpression(),
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

func prFeedbackGitHubEnvironmentFrom(binding *intakeBinding) []any {
	name := intakeGitHubAppName
	if binding != nil && binding.Integration != nil && strings.TrimSpace(binding.Integration.Name) != "" {
		name = binding.Integration.Name
	}

	return []any{
		map[string]any{
			"source": "integration",
			"integration": map[string]any{
				"name": name,
			},
		},
	}
}

func prFeedbackRunnerSteps() []any {
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
				"rm -rf repo",
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
			"name":             "Address PR feedback",
			"type":             "prompt",
			"workingDirectory": "repo",
			"prompt":           prFeedbackPrompt(),
		},
		map[string]any{
			"name":             "Commit and Push",
			"type":             "bash",
			"workingDirectory": "repo",
			"command": strings.Join([]string{
				"set -euo pipefail",
				"git add -A",
				"if ! git diff --cached --quiet; then",
				`  git commit -s -m "fix: address PR #${PR_NUMBER} feedback"`,
				"  git push origin HEAD",
				"fi",
			}, "\n"),
		},
	}
}

func prFeedbackPrompt() string {
	return strings.Join([]string{
		"You address current pull request feedback for a SuperPlane work order.",
		"The repository is already checked out in the current working directory.",
		"Stay on this branch. Push commits to this branch. Do not create a new branch.",
		"",
		"Repository: {{ root().data.repository.full_name }}",
		"Pull request: #{{ root().data.pull_request?.number ?? root().data.issue?.number }}",
		"",
		"Use the GitHub token in GITHUB_TOKEN.",
		"Read unresolved review threads.",
		"Read pull request conversation comments that mention @superplaneagent.",
		"Ignore comments from bots.",
		"Ignore replies that SuperPlane Agent already wrote.",
		"",
		"For each request:",
		"- Check that the change is valid and safe.",
		"- Apply valid requests.",
		"- Explain disagreements in a pull request comment. Do not make unsafe changes.",
		"",
		"Do not resolve GitHub review threads. The reviewer controls resolution.",
		"Do not report a work-order check or add a work-order comment.",
		"Stop after current feedback is addressed.",
		"Keep the change focused. Add tests where they are needed.",
	}, "\n")
}
