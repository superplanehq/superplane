package factories

import (
	"strings"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
)

const (
	prFeedbackCommentTriggerNodeID         = "on-pr-comment"
	prFeedbackAcknowledgeCommentNodeID     = "acknowledge-pr-comment"
	prFeedbackAcknowledgeReviewNodeID      = "acknowledge-pr-review"
	prFeedbackAcknowledgeReviewReplyNodeID = "acknowledge-pr-review-reply"
	prFeedbackReviewTriggerNodeID          = "on-pr-review"
	prFeedbackReplyTriggerNodeID           = "on-pr-review-reply"
	prFeedbackFindNodeID                   = "find-pull-request"
	prFeedbackActivityNodeID               = "add-pr-activity"
	prFeedbackRunnerNodeID                 = "address-pr-feedback"
	prFeedbackReviewFindNodeID             = "find-pull-request-for-review"
	prFeedbackReviewActivityNodeID         = "add-pr-review-activity"
	prFeedbackReviewRunnerNodeID           = "address-pr-review-feedback"
	prFeedbackReplyFindNodeID              = "find-pull-request-for-review-reply"
	prFeedbackReplyActivityNodeID          = "add-pr-review-reply-activity"
	prFeedbackReplyRunnerNodeID            = "address-pr-review-reply-feedback"

	prFeedbackFindComponent     = "findPullRequest"
	prFeedbackActivityComponent = "addPullRequestActivity"

	prFeedbackDefaultName         = "Address PR feedback"
	prFeedbackDefaultDescription  = "Address pull request comments and reviews after a mention."
	prFeedbackDefaultMention      = "@superplaneagent"
	prFeedbackCommentScopeReplies = "replies"

	prFeedbackRunnerNodeName = "Address PR feedback"
	prFeedbackMachineType    = runner.MachineTypeE1LargeAMD64
	prFeedbackTimeoutSeconds = 3600

	prFeedbackDefaultMaximumAttempts = 3
	prFeedbackMaximumAttemptsMin     = 1
	prFeedbackMaximumAttemptsMax     = 10

	prFeedbackDiscussionTemplateID = "pr-feedback:discussion"
	prFeedbackChecksTemplateID     = "pr-feedback:checks"

	prFeedbackCommentFlowY = 80
	prFeedbackReviewFlowY  = 360
	prFeedbackReplyFlowY   = 640

	prFeedbackAcknowledgeX       = 360
	prFeedbackAcknowledgeOffsetY = 120

	// A node with no concurrency spec runs one execution at a time across
	// every pull request. Waiting for checks or addressing comments on one
	// PR would then block the others. The key partitions the queue by PR
	// number; max stays at the default of 1 so one PR still serializes.
	prFeedbackPRNumberSource = `root().data.pull_request?.number ?? root().data.issue?.number ?? root().data.pullrequest?.id`
	prFeedbackConcurrencyKey = "pr-{{ " + prFeedbackPRNumberSource + " }}"
)

var prFeedbackTriggerNodeIDs = []string{
	prFeedbackCommentTriggerNodeID,
	prFeedbackReviewTriggerNodeID,
	prFeedbackReplyTriggerNodeID,
}

type prFeedbackBuildRequest struct {
	Name                   string
	Repository             string
	Mention                string
	IgnoreBots             bool
	IncludeVisualEvidence  bool
	AllowedBots            []string
	CheckNames             []string
	MaximumAttempts        int
	RunnerIntegrationNames []string
	Binding                *intakeBinding
	Agent                  *intakeAgent
	// VCSProvider selects the Git host. Empty means GitHub.
	VCSProvider string
}

// prFeedbackVCSProvider normalizes the request host. Bitbucket canvases use
// Bitbucket triggers, waits, and checkout; everything else stays GitHub.
func prFeedbackVCSProvider(request prFeedbackBuildRequest) string {
	if strings.EqualFold(strings.TrimSpace(request.VCSProvider), models.ProviderBitbucket) {
		return models.ProviderBitbucket
	}
	return models.ProviderGitHub
}

func buildPRFeedbackCanvas(request prFeedbackBuildRequest) *yaml.Canvas {
	return buildDiscussionPRFeedbackCanvas(request)
}

func prFeedbackCanvasName(request prFeedbackBuildRequest, fallback string) string {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return fallback
	}
	return name
}

func buildDiscussionPRFeedbackCanvas(request prFeedbackBuildRequest) *yaml.Canvas {
	if prFeedbackVCSProvider(request) == models.ProviderBitbucket {
		return buildBitbucketDiscussionPRFeedbackCanvas(request)
	}
	name := prFeedbackCanvasName(request, prFeedbackDefaultName)
	mention := strings.TrimSpace(request.Mention)

	includeReviewSubmissions := false
	commentFlow := prFeedbackDiscussionFlowNodes(prFeedbackDiscussionFlowRequest{
		Trigger: yaml.Node{
			ID:            prFeedbackCommentTriggerNodeID,
			Name:          "On PR Comment",
			Type:          yaml.NodeTypeTrigger,
			Component:     "github.onPRComment",
			Configuration: prFeedbackTriggerConfiguration(request.Repository, mention, request.IgnoreBots, request.AllowedBots),
			Integration:   request.Binding.integrationRef(),
		},
		FindID:       prFeedbackFindNodeID,
		FindName:     "Find Pull Request",
		ActivityID:   prFeedbackActivityNodeID,
		ActivityName: "Add Comment Activity",
		RunnerID:     prFeedbackRunnerNodeID,
		Title:        prFeedbackCommentActivityTitleExpression(),
		Description:  prFeedbackCommentActivityDescriptionExpression(),
		Y:            prFeedbackCommentFlowY,
	}, request)
	commentFlow = commentFlow.withAcknowledge(prFeedbackAcknowledgeReactionNode(
		prFeedbackAcknowledgeCommentNodeID,
		"Acknowledge PR Comment",
		prFeedbackCommentAcknowledgeCommentIDExpression(),
		"issueComment",
		prFeedbackAcknowledgePosition(prFeedbackCommentFlowY),
		request.Binding,
	))
	reviewFlow := prFeedbackDiscussionFlowNodes(prFeedbackDiscussionFlowRequest{
		Trigger: yaml.Node{
			ID:            prFeedbackReviewTriggerNodeID,
			Name:          "On PR Review",
			Type:          yaml.NodeTypeTrigger,
			Component:     "github.onPRReview",
			Configuration: prFeedbackTriggerConfiguration(request.Repository, mention, request.IgnoreBots, request.AllowedBots),
			Integration:   request.Binding.integrationRef(),
		},
		FindID:       prFeedbackReviewFindNodeID,
		FindName:     "Find Pull Request For Review",
		ActivityID:   prFeedbackReviewActivityNodeID,
		ActivityName: "Add Review Activity",
		RunnerID:     prFeedbackReviewRunnerNodeID,
		Title:        prFeedbackReviewActivityTitleExpression(),
		Description:  prFeedbackReviewActivityDescriptionExpression(),
		Y:            prFeedbackReviewFlowY,
	}, request)
	reviewFlow = reviewFlow.withAcknowledge(prFeedbackAcknowledgeReactionNode(
		prFeedbackAcknowledgeReviewNodeID,
		"Acknowledge PR Review",
		prFeedbackReviewAcknowledgeCommentIDExpression(),
		"reviewComment",
		prFeedbackAcknowledgePosition(prFeedbackReviewFlowY),
		request.Binding,
	))
	replyFlow := prFeedbackDiscussionFlowNodes(prFeedbackDiscussionFlowRequest{
		Trigger: yaml.Node{
			ID:        prFeedbackReplyTriggerNodeID,
			Name:      "On PR Review Reply",
			Type:      yaml.NodeTypeTrigger,
			Component: "github.onPRReviewComment",
			Configuration: prFeedbackReplyTriggerConfiguration(
				request.Repository,
				mention,
				request.IgnoreBots,
				request.AllowedBots,
				includeReviewSubmissions,
			),
			Integration: request.Binding.integrationRef(),
		},
		FindID:       prFeedbackReplyFindNodeID,
		FindName:     "Find Pull Request For Review Reply",
		ActivityID:   prFeedbackReplyActivityNodeID,
		ActivityName: "Add Review Reply Activity",
		RunnerID:     prFeedbackReplyRunnerNodeID,
		Title:        prFeedbackReplyActivityTitleExpression(),
		Description:  prFeedbackCommentActivityDescriptionExpression(),
		Y:            prFeedbackReplyFlowY,
	}, request)
	replyFlow = replyFlow.withAcknowledge(prFeedbackAcknowledgeReactionNode(
		prFeedbackAcknowledgeReviewReplyNodeID,
		"Acknowledge PR Review Reply",
		prFeedbackCommentAcknowledgeCommentIDExpression(),
		"reviewComment",
		prFeedbackAcknowledgePosition(prFeedbackReplyFlowY),
		request.Binding,
	))

	return withPRFeedbackConcurrency(&yaml.Canvas{
		APIVersion: yaml.APIVersion,
		Kind:       yaml.KindCanvas,
		Metadata: &yaml.CanvasMetadata{
			Name:        name,
			Description: prFeedbackDefaultDescription,
		},
		Spec: &yaml.CanvasSpec{
			Edges: append(append(commentFlow.edges, reviewFlow.edges...), replyFlow.edges...),
			Nodes: append(append(commentFlow.nodes, reviewFlow.nodes...), replyFlow.nodes...),
		},
	})
}

// buildBitbucketDiscussionPRFeedbackCanvas listens for pull request
// comments only. Bitbucket exposes no review-submission triggers, so the
// review and reply flows stay GitHub-only until their own step.
func buildBitbucketDiscussionPRFeedbackCanvas(request prFeedbackBuildRequest) *yaml.Canvas {
	name := prFeedbackCanvasName(request, prFeedbackDefaultName)
	mention := strings.TrimSpace(request.Mention)

	triggerConfig := map[string]any{
		"contentFilter": mention,
	}
	if strings.TrimSpace(request.Repository) != "" {
		triggerConfig["repository"] = request.Repository
	}
	commentFlow := prFeedbackDiscussionFlowNodes(prFeedbackDiscussionFlowRequest{
		Trigger: yaml.Node{
			ID:            prFeedbackCommentTriggerNodeID,
			Name:          "On PR Comment",
			Type:          yaml.NodeTypeTrigger,
			Component:     "bitbucket.onPullRequestComment",
			Configuration: triggerConfig,
			Integration:   request.Binding.integrationRef(),
		},
		FindID:       prFeedbackFindNodeID,
		FindName:     "Find Pull Request",
		ActivityID:   prFeedbackActivityNodeID,
		ActivityName: "Add Comment Activity",
		RunnerID:     prFeedbackRunnerNodeID,
		Title:        prFeedbackCommentActivityTitleExpression(),
		Description:  prFeedbackCommentActivityDescriptionExpression(),
		Y:            prFeedbackCommentFlowY,
	}, request)
	commentFlow = commentFlow.withAcknowledge(yaml.Node{
		ID:        prFeedbackAcknowledgeCommentNodeID,
		Name:      "Acknowledge PR Comment",
		Type:      yaml.NodeTypeAction,
		Component: "bitbucket.createPullRequestComment",
		Configuration: map[string]any{
			"repository":      "{{ root().data.repository.full_name }}",
			"pullNumber":      "{{ root().data.pullrequest.id }}",
			"parentCommentId": "{{ root().data.comment.id }}",
			"body":            "SuperPlane is looking into this.",
		},
		Integration: request.Binding.integrationRef(),
		Position:    prFeedbackAcknowledgePosition(prFeedbackCommentFlowY),
	})

	return withPRFeedbackConcurrency(&yaml.Canvas{
		APIVersion: yaml.APIVersion,
		Kind:       yaml.KindCanvas,
		Metadata: &yaml.CanvasMetadata{
			Name:        name,
			Description: prFeedbackDefaultDescription,
		},
		Spec: &yaml.CanvasSpec{
			Edges: commentFlow.edges,
			Nodes: commentFlow.nodes,
		},
	})
}

// withPRFeedbackConcurrency lets each pull request run at the same time.
// Triggers have no queue, so they keep the default.
func withPRFeedbackConcurrency(canvas *yaml.Canvas) *yaml.Canvas {
	if canvas == nil || canvas.Spec == nil {
		return canvas
	}
	for i := range canvas.Spec.Nodes {
		if canvas.Spec.Nodes[i].Type != yaml.NodeTypeAction {
			continue
		}
		canvas.Spec.Nodes[i].Concurrency = prFeedbackConcurrency()
	}
	return canvas
}

func prFeedbackConcurrency() *yaml.ConcurrencySpec {
	return &yaml.ConcurrencySpec{Key: prFeedbackConcurrencyKey}
}

func prFeedbackModelConcurrency() *models.ConcurrencySpec {
	return &models.ConcurrencySpec{Key: prFeedbackConcurrencyKey}
}

// ensurePRFeedbackConcurrency stamps the per-PR key on action nodes that
// still use the global default of one execution at a time. A node that
// already has a spec keeps it, so a custom limit is not overwritten.
func ensurePRFeedbackConcurrency(nodes []models.Node) []models.Node {
	for i := range nodes {
		if nodes[i].Type == models.NodeTypeTrigger {
			continue
		}
		if nodes[i].Concurrency != nil {
			continue
		}
		nodes[i].Concurrency = prFeedbackModelConcurrency()
	}
	return nodes
}

type prFeedbackDiscussionFlowRequest struct {
	Trigger      yaml.Node
	FindID       string
	FindName     string
	ActivityID   string
	ActivityName string
	RunnerID     string
	Title        string
	Description  string
	Y            int
}

type prFeedbackDiscussionFlowSpec struct {
	triggerID string
	nodes     []yaml.Node
	edges     []yaml.Edge
}

func (flow prFeedbackDiscussionFlowSpec) withAcknowledge(node yaml.Node) prFeedbackDiscussionFlowSpec {
	flow.nodes = append(flow.nodes, node)
	flow.edges = append(flow.edges, yaml.Edge{
		Channel:  "default",
		SourceID: flow.triggerID,
		TargetID: node.ID,
	})
	return flow
}

func prFeedbackAcknowledgeReactionNode(
	id string,
	name string,
	commentID string,
	target string,
	position yaml.Position,
	binding *intakeBinding,
) yaml.Node {
	return yaml.Node{
		ID:        id,
		Name:      name,
		Type:      yaml.NodeTypeAction,
		Component: "github.addReaction",
		Configuration: map[string]any{
			"repository": "{{ root().data.repository.full_name }}",
			"commentId":  commentID,
			"content":    "eyes",
			"target":     target,
		},
		Integration: binding.integrationRef(),
		Position:    position,
	}
}

func prFeedbackAcknowledgePosition(flowY int) yaml.Position {
	return yaml.Position{X: prFeedbackAcknowledgeX, Y: flowY - prFeedbackAcknowledgeOffsetY}
}

func prFeedbackDiscussionFlowNodes(
	flow prFeedbackDiscussionFlowRequest,
	request prFeedbackBuildRequest,
) prFeedbackDiscussionFlowSpec {
	flow.Trigger.Position = yaml.Position{X: 80, Y: flow.Y}
	return prFeedbackDiscussionFlowSpec{
		triggerID: flow.Trigger.ID,
		nodes: []yaml.Node{
			flow.Trigger,
			{
				ID:        flow.FindID,
				Name:      flow.FindName,
				Type:      yaml.NodeTypeAction,
				Component: prFeedbackFindComponent,
				Configuration: map[string]any{
					"provider":   prFeedbackVCSProvider(request),
					"repository": "{{ root().data.repository.full_name }}",
					"number":     prFeedbackPRNumberExpression(),
					"url":        prFeedbackPRURLExpression(),
				},
				Position: yaml.Position{X: 360, Y: flow.Y},
			},
			{
				ID:        flow.ActivityID,
				Name:      flow.ActivityName,
				Type:      yaml.NodeTypeAction,
				Component: prFeedbackActivityComponent,
				Configuration: map[string]any{
					"pullRequestId": `{{ $["` + flow.FindName + `"].data.pullRequest.id }}`,
					"title":         flow.Title,
					"description":   flow.Description,
					"access":        "exclusive",
				},
				Position: yaml.Position{X: 640, Y: flow.Y},
			},
			{
				ID:            flow.RunnerID,
				Name:          prFeedbackRunnerNodeName,
				Type:          yaml.NodeTypeAction,
				Component:     request.Agent.component(),
				Configuration: prFeedbackRunnerConfiguration(request),
				Position:      yaml.Position{X: 920, Y: flow.Y},
			},
		},
		edges: []yaml.Edge{
			{Channel: "default", SourceID: flow.Trigger.ID, TargetID: flow.FindID},
			{Channel: "found", SourceID: flow.FindID, TargetID: flow.ActivityID},
			{Channel: "default", SourceID: flow.ActivityID, TargetID: flow.RunnerID},
		},
	}
}

func prFeedbackReplyTriggerConfiguration(repository, mention string, ignoreBots bool, allowedBots []string, includeReviewSubmissions bool) map[string]any {
	configuration := prFeedbackTriggerConfiguration(repository, mention, ignoreBots, allowedBots)
	configuration["includeReviewSubmissions"] = includeReviewSubmissions
	configuration["commentScope"] = prFeedbackCommentScopeReplies
	return configuration
}

func prFeedbackTriggerConfiguration(repository, mention string, ignoreBots bool, allowedBots []string) map[string]any {
	configuration := map[string]any{
		"contentFilter": mention,
		"ignoreBots":    ignoreBots,
	}
	if strings.TrimSpace(repository) != "" {
		configuration["repository"] = repository
	}
	if len(allowedBots) > 0 {
		configuration["allowedBots"] = allowedBotsNodeValue(allowedBots)
	}
	return configuration
}

func prFeedbackCommentAcknowledgeCommentIDExpression() string {
	return "{{ root().data.comment.id }}"
}

func prFeedbackReviewAcknowledgeCommentIDExpression() string {
	return `{{ first(root().data.review_comments ?? [])?.id ?? "" }}`
}

func prFeedbackCommentActivityDescriptionExpression() string {
	return `{{ root().data.comment.body }}`
}

func prFeedbackReviewActivityDescriptionExpression() string {
	reviewBody := `(root().data.review?.body ?? "")`
	reviewComments := `root().data.review_comments ?? []`
	reviewCommentSections := `join(map(` + reviewComments + `, "· [" + .path + "](" + .html_url + ")\n" + .body), "\n\n")`
	return `{{ ` + reviewBody +
		` + (` + reviewBody + ` != "" && len(` + reviewComments + `) > 0 ? "\n\n" : "")` +
		` + ` + reviewCommentSections + ` }}`
}

func prFeedbackCommentActivityTitleExpression() string {
	return `{{ "[@" + root().data.comment.user.login + "](" + root().data.comment.user.html_url` +
		` + ") left a [comment](" + root().data.comment.html_url + ")" }}`
}

func prFeedbackReplyActivityTitleExpression() string {
	return `{{ "[@" + root().data.comment.user.login + "](" + root().data.comment.user.html_url` +
		` + ") left a [comment](" + root().data.comment.html_url + ")"` +
		" + \" in `\" + root().data.comment.path + \"`\" }}"
}

func prFeedbackReviewActivityTitleExpression() string {
	return `{{ "[@" + root().data.review.user.login + "](" + root().data.review.user.html_url` +
		` + ") left a [review](" + root().data.review.html_url + ")" }}`
}

func prFeedbackDiscussionActivityExpressions(nodeID string) (string, string, bool) {
	switch nodeID {
	case prFeedbackActivityNodeID:
		return prFeedbackCommentActivityTitleExpression(), prFeedbackCommentActivityDescriptionExpression(), true
	case prFeedbackReviewActivityNodeID:
		return prFeedbackReviewActivityTitleExpression(), prFeedbackReviewActivityDescriptionExpression(), true
	case prFeedbackReplyActivityNodeID:
		return prFeedbackReplyActivityTitleExpression(), prFeedbackCommentActivityDescriptionExpression(), true
	default:
		return "", "", false
	}
}

func prFeedbackPRURLExpression() string {
	return "{{ root().data.pull_request?.html_url ?? root().data.issue?.pull_request?.html_url ?? root().data.pullrequest?.links?.html?.href }}"
}

func prFeedbackPRNumberExpression() string {
	return "{{ " + prFeedbackPRNumberSource + " }}"
}

func prFeedbackPRHeadExpression() string {
	return "{{ root().data.pull_request?.head?.ref ?? root().data.pullrequest?.source?.branch?.name ?? \"\" }}"
}

func prFeedbackCoauthorsExpression() string {
	return `{{ task() == nil ? "" : join(map(filter(task().assignees, {#.email != ""}), "Co-authored-by: " + #.name + " <" + #.email + ">"), "\n") }}`
}

func prFeedbackRunnerConfiguration(request prFeedbackBuildRequest) map[string]any {
	configuration := map[string]any{
		"machineType":             prFeedbackMachineType,
		"executionTimeoutSeconds": prFeedbackTimeoutSeconds,
		"includeVisualEvidence":   request.IncludeVisualEvidence,
		"steps":                   prFeedbackRunnerStepsFor(request),
		"environmentFrom":         prFeedbackEnvironmentFrom(request.Binding, request.RunnerIntegrationNames),
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
	request.Agent.applyLLMProvider(configuration)

	return configuration
}

func prFeedbackGitHubEnvironmentFrom(binding *intakeBinding) []any {
	return prFeedbackEnvironmentFrom(binding, nil)
}

func prFeedbackEnvironmentFrom(binding *intakeBinding, extraIntegrationNames []string) []any {
	name := intakeGitHubAppName
	if binding != nil && binding.Integration != nil && strings.TrimSpace(binding.Integration.Name) != "" {
		name = binding.Integration.Name
	}

	entries := []any{
		map[string]any{
			"source": "integration",
			"integration": map[string]any{
				"name": name,
			},
		},
	}
	seen := map[string]bool{strings.ToLower(name): true}
	for _, extra := range extraIntegrationNames {
		trimmed := strings.TrimSpace(extra)
		if trimmed == "" || seen[strings.ToLower(trimmed)] {
			continue
		}
		seen[strings.ToLower(trimmed)] = true
		entries = append(entries, map[string]any{
			"source": "integration",
			"integration": map[string]any{
				"name": trimmed,
			},
		})
	}
	return entries
}

func prFeedbackRunnerSteps() []any {
	return prFeedbackRunnerStepsFor(prFeedbackBuildRequest{})
}

func prFeedbackRunnerStepsFor(request prFeedbackBuildRequest) []any {
	if prFeedbackVCSProvider(request) == models.ProviderBitbucket {
		return prFeedbackBitbucketRunnerSteps(request)
	}
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

// prFeedbackBitbucketRunnerSteps checks out the pull request source
// repository. A same-named branch on the destination is not a push target.
func prFeedbackBitbucketRunnerSteps(request prFeedbackBuildRequest) []any {
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
			"name":    "Checkout Pull Request",
			"type":    "bash",
			"command": bitbucketCheckoutCommand(),
		},
		map[string]any{
			"name":             "Set Up DCO Signing",
			"type":             "bash",
			"workingDirectory": "repo",
			"command":          runner.FactoryRepoCommitSetup(),
		},
		map[string]any{
			"name":             "Address PR feedback",
			"type":             "prompt",
			"workingDirectory": "repo",
			"prompt":           prFeedbackPromptFor(request),
		},
		map[string]any{
			"name":             "Commit and Push",
			"type":             "bash",
			"workingDirectory": "repo",
			"command":          bitbucketCommitPushCommand("fix: address PR #${PR_NUMBER} feedback"),
		},
	}
}

func prFeedbackPrompt() string {
	return prFeedbackPromptFor(prFeedbackBuildRequest{})
}

func prFeedbackPromptFor(request prFeedbackBuildRequest) string {
	if prFeedbackVCSProvider(request) == models.ProviderBitbucket {
		return strings.Join([]string{
			"You address current pull request feedback for a SuperPlane work order.",
			"The repository is already checked out in the current working directory.",
			"Stay on this branch. Push commits to this branch. Do not create a new branch.",
			"",
			"Repository: {{ root().data.repository.full_name }}",
			"Pull request: #{{ root().data.pull_request?.number ?? root().data.issue?.number ?? root().data.pullrequest?.id }}",
			"",
			"Use the Bitbucket token in BITBUCKET_TOKEN.",
			"When BITBUCKET_EMAIL is set, send HTTP basic authentication with that email and BITBUCKET_TOKEN.",
			"Read pull request comments that mention the workspace agent.",
			"Use inline file and line context on code-anchored comments.",
			"Ignore replies that SuperPlane Agent already wrote.",
			"",
			"For each request:",
			"- Check that the change is valid and safe.",
			"- Apply valid requests.",
			"- Explain disagreements in the completion comment. Do not make unsafe changes.",
			"",
			"Do not report a work-order check or add a work-order comment.",
			"Post one pull request comment after you address all feedback.",
			"Summarize the changes in this comment.",
			"If you upload visual evidence, include it in the same comment.",
			"Do not post a separate visual evidence comment.",
			"Stop after current feedback is addressed.",
			"Keep the change focused. Add tests where they are needed.",
			runner.FactoryCommitIdentityPrompt,
		}, "\n")
	}
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
		"Also read comments from bots that the automation allows, even without a mention.",
		"Ignore comments from all other bots.",
		"Ignore replies that SuperPlane Agent already wrote.",
		"",
		"For each request:",
		"- Check that the change is valid and safe.",
		"- Apply valid requests.",
		"- Explain disagreements in the completion comment. Do not make unsafe changes.",
		"",
		"Do not resolve GitHub review threads. The reviewer controls resolution.",
		"Do not report a work-order check or add a work-order comment.",
		"Post one pull request comment after you address all feedback.",
		"Summarize the changes in this comment.",
		"If you upload visual evidence, include it in the same comment.",
		"Do not post a separate visual evidence comment.",
		"Stop after current feedback is addressed.",
		"Keep the change focused. Add tests where they are needed.",
		runner.FactoryCommitIdentityPrompt,
	}, "\n")
}
