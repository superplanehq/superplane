package factories

import (
	"slices"
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

	prFeedbackDefaultMaximumAttempts    = 3
	prFeedbackMaximumAttemptsMin        = 1
	prFeedbackMaximumAttemptsMax        = 10
	prFeedbackDefaultMergeConflictHours = 24
	prFeedbackMergeConflictHoursMin     = 1
	prFeedbackMergeConflictHoursMax     = 168

	bitbucketPRCommentAcknowledgement = `Sure, looking into this. Follow along: [{{ $["Find Pull Request"].data.workOrder.key }}]({{ $["Find Pull Request"].data.workOrder.url }})`

	prFeedbackDiscussionTemplateID = "pr-feedback:discussion"
	prFeedbackChecksTemplateID     = "pr-feedback:checks"

	prFeedbackCommentFlowY = 80
	prFeedbackReviewFlowY  = 360
	prFeedbackReplyFlowY   = 640

	prFeedbackAcknowledgeX = 640
	prFeedbackActivityX    = 920
	prFeedbackRunnerX      = 1200

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
	Name                  string
	Repository            string
	Mention               string
	IgnoreBots            bool
	AddressAllBots        bool
	IncludeVisualEvidence bool
	AllowedBots           []string
	CheckNames            []string
	MaximumAttempts       int
	// FixMergeConflicts is nil when the caller did not set it. Nil means on.
	FixMergeConflicts      *bool
	MergeConflictHours     int
	RunnerIntegrationNames []string
	Binding                *intakeBinding
	Agent                  *intakeAgent
	AgentSettings          map[string]*factoryAgentSettings
	TriggerSettings        map[string]*factoryTriggerSettings
	// VCSProvider selects the Git host. Empty means GitHub.
	VCSProvider string
	// OmitCommitIdentity drops the git user and DCO steps. The factory git
	// wrapper already sets the agent identity and the sign-off hook.
	OmitCommitIdentity bool
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
			Configuration: prFeedbackTriggerConfiguration(request.Repository, mention, request.IgnoreBots, request.AllowedBots, request.AddressAllBots),
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
			Configuration: prFeedbackTriggerConfiguration(request.Repository, mention, request.IgnoreBots, request.AllowedBots, request.AddressAllBots),
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
				request.AddressAllBots,
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
		"allowedBots":   allowedBotsNodeValue(request.AllowedBots),
		"ignoreBots":    request.IgnoreBots,
	}
	if request.AddressAllBots {
		triggerConfig["addressAllBots"] = true
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
		Title:        prFeedbackBitbucketCommentActivityTitleExpression(),
		Description:  prFeedbackCommentActivityDescriptionExpression(),
		Y:            prFeedbackCommentFlowY,
	}, request)
	commentFlow = commentFlow.withAcknowledgeFrom(prFeedbackFindNodeID, "found", yaml.Node{
		ID:        prFeedbackAcknowledgeCommentNodeID,
		Name:      "Acknowledge PR Comment",
		Type:      yaml.NodeTypeAction,
		Component: "bitbucket.createPullRequestComment",
		Configuration: map[string]any{
			"repository":      "{{ root().data.repository.full_name }}",
			"pullNumber":      "{{ root().data.pullrequest.id }}",
			"parentCommentId": "{{ root().data.comment.id }}",
			"body":            bitbucketPRCommentAcknowledgement,
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
		if canvas.Spec.Nodes[i].Concurrency == nil {
			canvas.Spec.Nodes[i].Concurrency = prFeedbackConcurrency()
		}
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
	findID string
	nodes  []yaml.Node
	edges  []yaml.Edge
}

// withAcknowledge reacts only after Find Pull Request emits found.
// The reaction stays on that path, so the canvas remains one line.
// A mention on a pull request that is not a factory task does not reach it.
func (flow prFeedbackDiscussionFlowSpec) withAcknowledge(node yaml.Node) prFeedbackDiscussionFlowSpec {
	return flow.withAcknowledgeFrom(flow.findID, "found", node)
}

func (flow prFeedbackDiscussionFlowSpec) withAcknowledgeFrom(sourceID, channel string, node yaml.Node) prFeedbackDiscussionFlowSpec {
	flow.nodes = append(flow.nodes, node)
	for i := range flow.edges {
		if flow.edges[i].SourceID != sourceID || flow.edges[i].Channel != channel {
			continue
		}
		nextID := flow.edges[i].TargetID
		flow.edges[i].TargetID = node.ID
		flow.edges = slices.Insert(flow.edges, i+1, yaml.Edge{
			Channel:  "default",
			SourceID: node.ID,
			TargetID: nextID,
		})
		return flow
	}
	flow.edges = append(flow.edges, yaml.Edge{
		Channel:  channel,
		SourceID: sourceID,
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
	return yaml.Position{X: prFeedbackAcknowledgeX, Y: flowY}
}

func prFeedbackDiscussionFlowNodes(
	flow prFeedbackDiscussionFlowRequest,
	request prFeedbackBuildRequest,
) prFeedbackDiscussionFlowSpec {
	flow.Trigger = request.TriggerSettings[flow.Trigger.ID].configure(flow.Trigger)
	flow.Trigger.Position = yaml.Position{X: 80, Y: flow.Y}
	return prFeedbackDiscussionFlowSpec{
		findID: flow.FindID,
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
				Position: yaml.Position{X: prFeedbackActivityX, Y: flow.Y},
			},
			request.AgentSettings[flow.RunnerID].configure(yaml.Node{
				ID:            flow.RunnerID,
				Name:          prFeedbackRunnerNodeName,
				Type:          yaml.NodeTypeAction,
				Component:     request.Agent.component(),
				Configuration: prFeedbackRunnerConfiguration(request),
				Position:      yaml.Position{X: prFeedbackRunnerX, Y: flow.Y},
			}),
		},
		edges: []yaml.Edge{
			{Channel: "default", SourceID: flow.Trigger.ID, TargetID: flow.FindID},
			{Channel: "found", SourceID: flow.FindID, TargetID: flow.ActivityID},
			{Channel: "default", SourceID: flow.ActivityID, TargetID: flow.RunnerID},
		},
	}
}

func prFeedbackReplyTriggerConfiguration(repository, mention string, ignoreBots bool, allowedBots []string, addressAllBots bool, includeReviewSubmissions bool) map[string]any {
	configuration := prFeedbackTriggerConfiguration(repository, mention, ignoreBots, allowedBots, addressAllBots)
	configuration["includeReviewSubmissions"] = includeReviewSubmissions
	configuration["commentScope"] = prFeedbackCommentScopeReplies
	return configuration
}

func prFeedbackTriggerConfiguration(repository, mention string, ignoreBots bool, allowedBots []string, addressAllBots bool) map[string]any {
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
	if addressAllBots {
		configuration["addressAllBots"] = true
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

func prFeedbackBitbucketCommentActivityTitleExpression() string {
	return `{{ "[@" + (root().data.comment.author.login ?? root().data.comment.author.nickname) + "](" + root().data.comment.author.html_url` +
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

func prFeedbackDiscussionActivityExpressions(nodeID, provider string) (string, string, bool) {
	switch nodeID {
	case prFeedbackActivityNodeID:
		title := prFeedbackCommentActivityTitleExpression()
		if provider == models.ProviderBitbucket {
			title = prFeedbackBitbucketCommentActivityTitleExpression()
		}
		return title, prFeedbackCommentActivityDescriptionExpression(), true
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

func prFeedbackCommitIdentitySteps() []any {
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
			"name":             "Set Up DCO Signing",
			"type":             "bash",
			"workingDirectory": "repo",
			"command":          runner.FactoryRepoCommitSetup(),
		},
	}
}

func prFeedbackRunnerStepsFor(request prFeedbackBuildRequest) []any {
	steps := []any{}
	if !request.OmitCommitIdentity {
		steps = append(steps, prFeedbackCommitIdentitySteps()...)
	}
	if prFeedbackVCSProvider(request) == models.ProviderBitbucket {
		return append(steps, prFeedbackBitbucketWorkSteps(request)...)
	}
	return append(steps,
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
			"command": strings.Join([]string{
				"set -euo pipefail",
				"git add -A",
				"if ! git diff --cached --quiet; then",
				`  git commit -s -m "fix: address PR #${PR_NUMBER} feedback"`,
				"  git push origin HEAD",
				"fi",
			}, "\n"),
		},
	)
}

// prFeedbackBitbucketWorkSteps pushes to the pull request source
// branch. The runner checks that branch out before these steps. A
// same-named branch on the destination is not a push target.
func prFeedbackBitbucketWorkSteps(request prFeedbackBuildRequest) []any {
	return []any{
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
			strings.Join(prFeedbackBitbucketReadInstructions(request), "\n"),
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
		strings.Join(prFeedbackGitHubReadInstructions(request), "\n"),
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

func prFeedbackGitHubReadInstructions(request prFeedbackBuildRequest) []string {
	if request.AddressAllBots {
		return []string{
			"Read unresolved review threads.",
			"Read pull request conversation comments that mention " + prFeedbackPromptMention(request) + ".",
			"Read every bot comment, including comments with no mention.",
			"Ignore comments SuperPlane already wrote.",
		}
	}
	return []string{
		"Read unresolved review threads.",
		"Read pull request conversation comments that mention @superplaneagent.",
		"Also read comments from bots that the automation allows, even without a mention.",
		"Ignore comments from all other bots.",
		"Ignore replies that SuperPlane Agent already wrote.",
	}
}

func prFeedbackBitbucketReadInstructions(request prFeedbackBuildRequest) []string {
	if request.AddressAllBots {
		return []string{
			"Read pull request comments that mention " + prFeedbackPromptMention(request) + ".",
			"Read every bot comment, including comments with no mention.",
			"Ignore comments SuperPlane already wrote.",
			"Use inline file and line context on code-anchored comments.",
		}
	}
	return []string{
		"Read pull request comments that mention the workspace agent.",
		"Use inline file and line context on code-anchored comments.",
		"Ignore replies that SuperPlane Agent already wrote.",
	}
}

func prFeedbackPromptMention(request prFeedbackBuildRequest) string {
	mention := strings.TrimSpace(request.Mention)
	if mention == "" {
		return prFeedbackDefaultMention
	}
	return mention
}

func refreshDiscussionRunnerPrompt(configuration map[string]any, request prFeedbackBuildRequest) {
	steps, ok := configuration["steps"].([]any)
	if !ok {
		return
	}
	prompt := prFeedbackPromptFor(request)
	for _, item := range steps {
		step, ok := item.(map[string]any)
		if !ok || step["name"] != "Address PR feedback" || step["type"] != "prompt" {
			continue
		}
		step["prompt"] = prompt
	}
}
