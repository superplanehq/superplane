package pulls

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/integrations/prbody"
)

const (
	PullRequestCommentComponent   = "github.publishPullRequestComment"
	PullRequestCommentHook        = "publishPullRequestComment"
	PullRequestCommentPayloadType = "github.pullRequestCommentPublished"
)

// PublishPullRequestComment keeps one managed comment for each canvas, pull
// request, and comment key. The publication worker processes its hook.
type PublishPullRequestComment struct{}

type PullRequestCommentConfiguration struct {
	Repository           string `json:"repository" mapstructure:"repository"`
	PullNumber           string `json:"pullNumber" mapstructure:"pullNumber"`
	CommentKey           string `json:"commentKey" mapstructure:"commentKey"`
	Body                 string `json:"body" mapstructure:"body"`
	ExpectedHeadSHA      string `json:"expectedHeadSha,omitempty" mapstructure:"expectedHeadSha"`
	Enabled              *bool  `json:"enabled,omitempty" mapstructure:"enabled"`
	ActivationGeneration int64  `json:"activationGeneration,omitempty" mapstructure:"activationGeneration"`
}

func DecodePullRequestComment(value any) (PullRequestCommentConfiguration, error) {
	var config PullRequestCommentConfiguration
	if err := mapstructure.WeakDecode(value, &config); err != nil {
		return config, err
	}
	config.Repository = strings.TrimSpace(config.Repository)
	config.PullNumber = strings.TrimSpace(config.PullNumber)
	config.ExpectedHeadSHA = strings.TrimSpace(config.ExpectedHeadSHA)
	return config, nil
}

func (c PullRequestCommentConfiguration) IsEnabled() bool { return c.Enabled == nil || *c.Enabled }

func (c PullRequestCommentConfiguration) Number() (int64, error) {
	number, err := strconv.ParseInt(c.PullNumber, 10, 64)
	if err != nil || number <= 0 {
		return 0, fmt.Errorf("pull request number must be a positive integer")
	}
	return number, nil
}

func (c PullRequestCommentConfiguration) Validate() error {
	if c.Repository == "" {
		return fmt.Errorf("repository is required")
	}
	if c.PullNumber == "" {
		return fmt.Errorf("pull request number is required")
	}
	if !configuration.ExpressionPlaceholderRegex.MatchString(c.PullNumber) {
		if _, err := c.Number(); err != nil {
			return err
		}
	}
	if c.ActivationGeneration < 0 {
		return fmt.Errorf("activation generation must be positive")
	}
	if err := prbody.ValidateBodySectionKey(c.CommentKey); err != nil {
		return fmt.Errorf("comment key: %w", err)
	}
	return nil
}

// PublicationKey and Marker include the canvas, so each automation owns one comment.
func (c PullRequestCommentConfiguration) PublicationKey(canvasID uuid.UUID) string {
	return fmt.Sprintf("pull-request-comment/%s/%s/%s/%s", strings.ToLower(c.Repository), c.PullNumber, c.CommentKey, canvasID)
}

func (c PullRequestCommentConfiguration) Marker(canvasID uuid.UUID) string {
	return fmt.Sprintf("<!-- superplane:comment:%s:%s -->", c.CommentKey, canvasID)
}

func (*PublishPullRequestComment) Name() string  { return PullRequestCommentComponent }
func (*PublishPullRequestComment) Label() string { return "Publish PR Comment" }
func (*PublishPullRequestComment) Description() string {
	return "Create or update one managed comment on a GitHub pull request"
}
func (*PublishPullRequestComment) Icon() string  { return "github" }
func (*PublishPullRequestComment) Color() string { return "gray" }
func (*PublishPullRequestComment) Documentation() string {
	return `Create or update one managed comment on a pull request. Each canvas keeps one comment for each pull request and comment key.

Use a stable comment key with lowercase letters, numbers, and single hyphens. Start with a letter. SuperPlane adds a hidden marker with the key and the canvas ID. It edits the comment that has this marker and does not change other comments.

Writers are ordered by their persisted source run and attempt. The source is the broadcast source run, or the current run for other triggers. An older source or attempt cannot replace the comment, including on the same commit.

Each attempt reads the pull request, rechecks the order and the saved node, and then writes. Set **Expected Head SHA** to skip publication if the pull request commit changes. Closed pull requests are skipped. **Enabled** defaults to true. Increase **Activation Generation** when you enable the node again. Attempts that started with another generation do not write.

The saved comment ID is checked before an update. If the comment is missing or has no marker, SuperPlane searches the pull request for the marker before it creates a comment. A retry after an uncertain response updates the same comment.

Temporary failures retry through scheduled requests with the resolved body. Host rejections and exhausted retries fail the step. Disabling the node keeps published comments.`
}

func (*PublishPullRequestComment) Configuration() []configuration.Field {
	literalKey := false
	return []configuration.Field{
		{Name: "repository", Label: "Repository", Type: configuration.FieldTypeIntegrationResource, Required: true, TypeOptions: &configuration.TypeOptions{Resource: &configuration.ResourceTypeOptions{Type: "repository", UseNameAsValue: true}}},
		{Name: "pullNumber", Label: "Pull Request Number", Type: configuration.FieldTypeString, Required: true, Description: "Pull request number to comment on. Supports expressions."},
		{Name: "commentKey", Label: "Comment Key", Type: configuration.FieldTypeString, Required: true, Description: "Stable key of the managed comment. Expressions are not supported.", TypeOptions: &configuration.TypeOptions{String: &configuration.StringTypeOptions{AllowExpressions: &literalKey}}},
		{Name: "body", Label: "Body", Type: configuration.FieldTypeText, Required: true, Description: "Markdown comment body. Supports expressions."},
		{Name: "expectedHeadSha", Label: "Expected Head SHA", Type: configuration.FieldTypeString, Description: "Skip the update if the pull request commit changed."},
		{Name: "enabled", Label: "Enabled", Type: configuration.FieldTypeBool, Default: true, Description: "Publish this comment to the pull request."},
		{Name: "activationGeneration", Label: "Activation Generation", Type: configuration.FieldTypeNumber, Default: 1, Description: "Increase when you enable publication again. Older attempts do not write."},
	}
}

func (*PublishPullRequestComment) OutputChannels(any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (*PublishPullRequestComment) ExampleOutput() map[string]any {
	return map[string]any{"type": PullRequestCommentPayloadType, "timestamp": "2026-11-13T12:00:00Z", "data": map[string]any{"status": "published", "attempts": 1, "commentId": 123}}
}

func (*PublishPullRequestComment) ValidateNodeConfiguration(value map[string]any) error {
	config, err := DecodePullRequestComment(value)
	if err != nil {
		return err
	}
	return config.Validate()
}

func (*PublishPullRequestComment) Setup(ctx core.SetupContext) error {
	config, err := DecodePullRequestComment(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := config.Validate(); err != nil {
		return err
	}
	return common.EnsureRepoInMetadata(ctx.Metadata, ctx.Integration, ctx.HTTP, ctx.Configuration)
}

func (*PublishPullRequestComment) Execute(ctx core.ExecutionContext) error {
	config, err := DecodePullRequestComment(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := config.Validate(); err != nil {
		return err
	}
	if _, err := config.Number(); err != nil {
		return err
	}
	metadata, _ := ctx.Metadata.Get().(map[string]any)
	metadata = maps.Clone(metadata)
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["sourceRunId"], metadata["status"], metadata["attempts"] = ctx.RunID.String(), "queued", 0
	if config.IsEnabled() {
		canvasID, err := uuid.Parse(ctx.WorkflowID)
		if err != nil {
			return fmt.Errorf("comment publication requires a canvas: %w", err)
		}
		accepted, err := prbody.AcceptOrderedWrite(ctx, config.PublicationKey(canvasID))
		if err != nil {
			return err
		}
		if accepted == nil {
			metadata["status"], metadata["reason"] = string(prbody.StatusSkipped), prbody.SupersededReason
			if err := ctx.Metadata.Set(metadata); err != nil {
				return err
			}
			return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, PullRequestCommentPayloadType, []any{prbody.Skipped(prbody.SupersededReason)})
		}
		metadata["publication"] = accepted
	}
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}
	return ctx.Requests.ScheduleActionCall(PullRequestCommentHook, nil, time.Second)
}

func (*PublishPullRequestComment) Hooks() []core.Hook {
	return []core.Hook{{Name: PullRequestCommentHook, Type: core.HookTypeInternal}}
}

// The node request worker routes this hook to the publication worker, which
// runs provider calls outside database transactions.
func (*PublishPullRequestComment) HandleHook(ctx core.ActionHookContext) error {
	return fmt.Errorf("hook %s is processed by the publication worker", ctx.Name)
}

func (*PublishPullRequestComment) HandleWebhook(core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusNotFound, nil, nil
}
func (*PublishPullRequestComment) Cancel(core.ExecutionContext) error { return nil }
func (*PublishPullRequestComment) Cleanup(core.SetupContext) error    { return nil }

type ManagedComment struct {
	ID   int64
	Body string
}

type CommentClient interface {
	Read(ctx context.Context, repository string, number int64) (prbody.Description, error)
	// Get returns nil when the comment does not exist.
	Get(ctx context.Context, repository string, id int64) (*ManagedComment, error)
	List(ctx context.Context, repository string, number int64) ([]ManagedComment, error)
	Create(ctx context.Context, repository string, number int64, body string) (int64, error)
	Edit(ctx context.Context, repository string, id int64, body string) error
}

// CommentBoundary serializes the comment mutation and stores the managed comment ID.
type CommentBoundary interface {
	Begin() (savedID int64, deadline time.Time, err error)
	Finish(commentID int64) error
}

type CommentOutcome struct {
	prbody.Outcome
	CommentID int64 `json:"commentId,omitempty"`
}

// PublishComment claims the boundary before it looks up the managed comment, so
// concurrent attempts cannot create duplicates or overwrite newer content. It
// checks the pull request and eligibility before the boundary and again after
// the lookup, immediately before the create or edit.
func PublishComment(ctx context.Context, client CommentClient, config PullRequestCommentConfiguration, marker string, eligible func() (bool, error), boundary CommentBoundary) (CommentOutcome, error) {
	if err := config.Validate(); err != nil {
		return CommentOutcome{}, err
	}
	number, err := config.Number()
	if err != nil {
		return CommentOutcome{}, err
	}
	if !config.IsEnabled() {
		return CommentOutcome{Outcome: prbody.Skipped("Publication is disabled for this run.")}, nil
	}
	active, err := eligible()
	if err != nil || !active {
		return CommentOutcome{Outcome: prbody.Skipped("Publication is disabled, cancelled, or superseded.")}, err
	}
	publishable := func(ctx context.Context) (*prbody.Outcome, error) {
		return commentPublishable(ctx, client, config, number, eligible)
	}
	skipped, err := publishable(ctx)
	if err != nil || skipped != nil {
		return skippedComment(skipped, 0), err
	}
	savedID, deadline, err := boundary.Begin()
	if errors.Is(err, core.ErrPublicationSuperseded) {
		return CommentOutcome{Outcome: prbody.Skipped(prbody.SupersededReason)}, nil
	}
	if err != nil {
		return CommentOutcome{}, err
	}
	writeCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	outcome, err := upsertManagedComment(writeCtx, client, config.Repository, number, marker, savedID, marker+"\n"+config.Body, publishable)
	if finishErr := boundary.Finish(outcome.CommentID); finishErr != nil && err == nil {
		err = finishErr
	}
	return outcome, err
}

// commentPublishable reads the pull request and rechecks eligibility. It
// returns the skipped outcome when the comment must not be written.
func commentPublishable(ctx context.Context, client CommentClient, config PullRequestCommentConfiguration, number int64, eligible func() (bool, error)) (*prbody.Outcome, error) {
	latest, err := client.Read(ctx, config.Repository, number)
	if err != nil {
		return nil, err
	}
	if !latest.Open || (config.ExpectedHeadSHA != "" && latest.HeadSHA != config.ExpectedHeadSHA) {
		skipped := prbody.Skipped("Pull request is closed or its expected commit changed.")
		return &skipped, nil
	}
	active, err := eligible()
	if err != nil {
		return nil, err
	}
	if !active {
		skipped := prbody.Skipped("Publication was disabled, cancelled, or superseded.")
		return &skipped, nil
	}
	return nil, nil
}

func skippedComment(skipped *prbody.Outcome, commentID int64) CommentOutcome {
	if skipped == nil {
		return CommentOutcome{CommentID: commentID}
	}
	return CommentOutcome{Outcome: *skipped, CommentID: commentID}
}

func upsertManagedComment(ctx context.Context, client CommentClient, repository string, number int64, marker string, savedID int64, body string, publishable func(context.Context) (*prbody.Outcome, error)) (CommentOutcome, error) {
	existing, err := findManagedComment(ctx, client, repository, number, marker, savedID)
	if err != nil {
		return CommentOutcome{CommentID: savedID}, err
	}
	knownID := savedID
	if existing != nil {
		knownID = existing.ID
	}
	// The lookup can read many pages. Settings, the run, or the commit can change meanwhile.
	skipped, err := publishable(ctx)
	if err != nil || skipped != nil {
		return skippedComment(skipped, knownID), err
	}
	if existing == nil {
		id, err := client.Create(ctx, repository, number, body)
		if err != nil {
			return CommentOutcome{CommentID: savedID}, err
		}
		return CommentOutcome{Outcome: prbody.Outcome{Status: prbody.StatusPublished}, CommentID: id}, nil
	}
	if existing.Body == body {
		return CommentOutcome{Outcome: prbody.Outcome{Status: prbody.StatusUnchanged}, CommentID: existing.ID}, nil
	}
	if err := client.Edit(ctx, repository, existing.ID, body); err != nil {
		return CommentOutcome{CommentID: existing.ID}, err
	}
	return CommentOutcome{Outcome: prbody.Outcome{Status: prbody.StatusPublished}, CommentID: existing.ID}, nil
}

// A saved ID is used only while its comment keeps the ownership marker.
func findManagedComment(ctx context.Context, client CommentClient, repository string, number int64, marker string, savedID int64) (*ManagedComment, error) {
	if savedID > 0 {
		comment, err := client.Get(ctx, repository, savedID)
		if err != nil {
			return nil, err
		}
		if comment != nil && strings.HasPrefix(comment.Body, marker) {
			return comment, nil
		}
	}
	comments, err := client.List(ctx, repository, number)
	if err != nil {
		return nil, err
	}
	for i := range comments {
		if strings.HasPrefix(comments[i].Body, marker) {
			return &comments[i], nil
		}
	}
	return nil, nil
}

type githubCommentClient struct{ client *common.Client }

func NewCommentClient(integration core.IntegrationContext, http core.HTTPContext) (CommentClient, error) {
	client, err := common.NewClient(integration, http)
	if err != nil {
		return nil, err
	}
	return &githubCommentClient{client: client}, nil
}

func (c *githubCommentClient) Read(ctx context.Context, repository string, number int64) (prbody.Description, error) {
	return (&BodySectionClient{client: c.client}).Read(ctx, repository, number)
}

func (c *githubCommentClient) Get(ctx context.Context, repository string, id int64) (*ManagedComment, error) {
	comment, response, err := c.client.GetIssueComment(ctx, repository, id)
	if response != nil && response.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ManagedComment{ID: comment.GetID(), Body: comment.GetBody()}, nil
}

func (c *githubCommentClient) List(ctx context.Context, repository string, number int64) ([]ManagedComment, error) {
	comments, err := c.client.ListIssueComments(ctx, repository, int(number))
	if err != nil {
		return nil, err
	}
	managed := make([]ManagedComment, 0, len(comments))
	for _, comment := range comments {
		managed = append(managed, ManagedComment{ID: comment.GetID(), Body: comment.GetBody()})
	}
	return managed, nil
}

func (c *githubCommentClient) Create(ctx context.Context, repository string, number int64, body string) (int64, error) {
	comment, _, err := c.client.CreateIssueComment(ctx, repository, int(number), &github.IssueComment{Body: &body})
	if err != nil {
		return 0, err
	}
	return comment.GetID(), nil
}

func (c *githubCommentClient) Edit(ctx context.Context, repository string, id int64, body string) error {
	_, _, err := c.client.EditIssueComment(ctx, repository, id, &github.IssueComment{Body: &body})
	return err
}

// Leases bound every write. A write that has not finished before the lease
// expires is cancelled, so a successor cannot be overwritten.
const orderedCommentLease = time.Minute
const orderedCommentWriteMargin = 15 * time.Second

type orderedCommentState struct {
	CommentID int64 `json:"commentId,omitempty"`
}

type orderedCommentBoundary struct {
	store core.PublicationStore
	write core.OrderedWrite
	lease core.PublicationLease
}

// NewOrderedCommentBoundary applies source and attempt ordering at the comment write.
func NewOrderedCommentBoundary(store core.PublicationStore, write core.OrderedWrite) CommentBoundary {
	return &orderedCommentBoundary{store: store, write: write}
}

func (b *orderedCommentBoundary) Begin() (int64, time.Time, error) {
	lease, raw, err := core.BeginOrderedWrite(b.store, b.write, orderedCommentLease)
	if err != nil {
		return 0, time.Time{}, err
	}
	b.lease = lease
	var state orderedCommentState
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &state); err != nil {
			return 0, time.Time{}, err
		}
	}
	return state.CommentID, lease.ExpiresAt.Add(-orderedCommentWriteMargin), nil
}

func (b *orderedCommentBoundary) Finish(commentID int64) error {
	var state json.RawMessage
	if commentID > 0 {
		encoded, err := json.Marshal(orderedCommentState{CommentID: commentID})
		if err != nil {
			return err
		}
		state = encoded
	}
	return core.FinishOrderedWrite(b.store, b.write, b.lease, state)
}
