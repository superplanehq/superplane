package claude

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ReviewAgentPayloadType  = "claude.reviewAgent.finished"
	ReviewAgentDefaultModel = "claude-sonnet-4-6"
	ReviewAgentMaxTokens    = 4096
)

type ReviewAgent struct{}

type ReviewAgentSpec struct {
	Model       string `json:"model" mapstructure:"model"`
	Repository  string `json:"repository" mapstructure:"repository"`
	PrNumber    string `json:"prNumber" mapstructure:"prNumber"`
	Plan        string `json:"plan" mapstructure:"plan"`
	GitHubToken string `json:"githubToken" mapstructure:"githubToken"`
}

type ReviewAgentOutputPayload struct {
	Status   string `json:"status"`
	Decision string `json:"decision"`
	Review   string `json:"review"`
	ErrorMsg string `json:"error,omitempty"`
}

func (r *ReviewAgent) Name() string { return "claude.reviewAgent" }

func (r *ReviewAgent) Label() string { return "Review Agent" }

func (r *ReviewAgent) Description() string {
	return "Fetches a PR diff, asks Claude to review it, and posts APPROVE or REQUEST_CHANGES back to GitHub"
}

func (r *ReviewAgent) Documentation() string {
	return `The Review Agent fetches a pull request diff from GitHub, sends it to Claude for review, and posts the result directly back to the PR using a GitHub token.

## How it works
1. Fetches the changed files and diff from the PR via GitHub API
2. Sends the diff + implementation plan to Claude for review
3. Parses Claude response for APPROVE or REQUEST_CHANGES decision
4. Posts the review back to the PR via GitHub API

## Configuration
- **Repository**: GitHub repository in owner/repo format
- **PR Number**: Pull request number (from claude.codeAgent output)
- **Plan**: Implementation plan for context (from claude.codeAgent output)
- **GitHub Token**: Personal access token with repo scope
- **Model**: Claude model to use

## Output
- **decision**: APPROVE or REQUEST_CHANGES
- **review**: Full review text posted to the PR
- **status**: succeeded or failed`
}

func (r *ReviewAgent) Icon() string  { return "cpu" }
func (r *ReviewAgent) Color() string { return "green" }

func (r *ReviewAgent) ExampleOutput() map[string]any {
	return map[string]any{
		"type": ReviewAgentPayloadType,
		"data": map[string]any{
			"status":   "succeeded",
			"decision": "APPROVE",
			"review":   "APPROVE\n\nThe implementation looks clean and follows best practices.",
		},
	}
}

func (r *ReviewAgent) OutputChannels(config any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (r *ReviewAgent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "GitHub repository in owner/repo format",
			Placeholder: "acme/my-app",
		},
		{
			Name:        "prNumber",
			Label:       "PR Number",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "Pull request number — use {{ $['claude.codeAgent'].data.prNumber }}",
		},
		{
			Name:        "plan",
			Label:       "Implementation Plan",
			Type:        configuration.FieldTypeExpression,
			Required:    false,
			Description: "Plan from claude.codeAgent for review context",
		},
		{
			Name:        "githubToken",
			Label:       "GitHub Token",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "GitHub personal access token with repo scope",
		},
		{
			Name:        "model",
			Label:       "Model",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Default:     ReviewAgentDefaultModel,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: "model"},
			},
		},
	}
}

func (r *ReviewAgent) Setup(ctx core.SetupContext) error {
	spec := ReviewAgentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}
	if spec.Repository == "" {
		return fmt.Errorf("repository is required")
	}
	if spec.GitHubToken == "" {
		return fmt.Errorf("githubToken is required")
	}
	return nil
}

func (r *ReviewAgent) Execute(ctx core.ExecutionContext) error {
	spec := ReviewAgentSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.Model == "" {
		spec.Model = ReviewAgentDefaultModel
	}

	parts := strings.SplitN(spec.Repository, "/", 2)
	if len(parts) != 2 {
		return emitReviewFailure(ctx, "repository must be in owner/repo format")
	}
	owner, repo := parts[0], parts[1]

	prNumber, err := strconv.Atoi(spec.PrNumber)
	if err != nil {
		return emitReviewFailure(ctx, fmt.Sprintf("invalid prNumber %q: %v", spec.PrNumber, err))
	}

	ghClient := NewGitHubClient(spec.GitHubToken, ctx.HTTP)

	// 1. Fetch PR files
	ctx.Logger.Infof("[ReviewAgent] Fetching PR #%d files from %s", prNumber, spec.Repository)
	files, err := ghClient.GetPRFiles(owner, repo, prNumber)
	if err != nil {
		return emitReviewFailure(ctx, fmt.Sprintf("failed to fetch PR files: %v", err))
	}

	// 2. Build review prompt
	claudeClient, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return emitReviewFailure(ctx, fmt.Sprintf("failed to create Claude client: %v", err))
	}

	prompt := buildReviewPrompt(files, spec.Plan)
	ctx.Logger.Infof("[ReviewAgent] Sending %d files to Claude for review", len(files))

	response, err := claudeClient.CreateMessage(CreateMessageRequest{
		Model:     spec.Model,
		MaxTokens: ReviewAgentMaxTokens,
		System:    reviewAgentSystemPrompt(),
		Messages:  []Message{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return emitReviewFailure(ctx, fmt.Sprintf("Claude API error: %v", err))
	}

	reviewText := extractMessageText(response)
	decision := parseReviewDecision(reviewText)

	// 3. Post review to GitHub
	ctx.Logger.Infof("[ReviewAgent] Posting %s review to PR #%d", decision, prNumber)
	// GitHub doesn't allow approving your own PR — fall back to COMMENT
	eventToPost := decision
	if decision == "APPROVE" {
		eventToPost = "COMMENT"
	}
	if err := ghClient.SubmitPRReview(owner, repo, prNumber, eventToPost, reviewText); err != nil {
		return emitReviewFailure(ctx, fmt.Sprintf("failed to post review: %v", err))
	}

	ctx.Logger.Infof("[ReviewAgent] Review posted successfully — decision: %s", decision)

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		ReviewAgentPayloadType,
		[]any{ReviewAgentOutputPayload{
			Status:   "succeeded",
			Decision: decision,
			Review:   reviewText,
		}},
	)
}

func emitReviewFailure(ctx core.ExecutionContext, errMsg string) error {
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		ReviewAgentPayloadType,
		[]any{ReviewAgentOutputPayload{
			Status:   "failed",
			ErrorMsg: errMsg,
		}},
	)
}

func reviewAgentSystemPrompt() string {
	return `You are an expert code reviewer. Review the pull request diff provided and respond with either APPROVE or REQUEST_CHANGES as the first word, followed by your detailed review.

Be concise but thorough. Focus on: correctness, security issues, code quality, and alignment with the implementation plan.`
}

func buildReviewPrompt(files []PRFile, plan string) string {
	var sb strings.Builder

	if plan != "" {
		sb.WriteString("## Implementation Plan\n")
		sb.WriteString(plan)
		sb.WriteString("\n\n")
	}

	sb.WriteString("## Changed Files\n\n")
	for _, f := range files {
		sb.WriteString(fmt.Sprintf("### %s (%s)\n", f.Filename, f.Status))
		if f.Patch != "" {
			sb.WriteString("```diff\n")
			sb.WriteString(f.Patch)
			sb.WriteString("\n```\n\n")
		}
	}

	sb.WriteString("Review this pull request. Start your response with APPROVE or REQUEST_CHANGES.")
	return sb.String()
}

func parseReviewDecision(text string) string {
	upper := strings.ToUpper(strings.TrimSpace(text))
	if strings.HasPrefix(upper, "APPROVE") {
		return "APPROVE"
	}
	if strings.HasPrefix(upper, "REQUEST_CHANGES") {
		return "REQUEST_CHANGES"
	}
	return "COMMENT"
}

func (r *ReviewAgent) Cancel(ctx core.ExecutionContext) error      { return nil }
func (r *ReviewAgent) Cleanup(ctx core.SetupContext) error          { return nil }
func (r *ReviewAgent) Actions() []core.Action                       { return []core.Action{} }
func (r *ReviewAgent) HandleAction(ctx core.ActionContext) error    { return nil }

func (r *ReviewAgent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (r *ReviewAgent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}
