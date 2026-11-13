package bitbucket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const reviewStatusKey = "superplane/review-request"
const reviewCheckPrefix = "bitbucket.review-check/"
const reviewCheckHook = "refreshPullRequestReviewCheck"
const reviewCheckWaitHook = "awaitReviewCheck"

type SetPullRequestReviewCheck struct{ pullRequestAction }

type reviewCheckConfiguration struct {
	Repository                 string             `json:"repository" mapstructure:"repository"`
	RepositoryUUID             string             `json:"repositoryUUID" mapstructure:"-"`
	PullRequestID              string             `json:"pullRequestId" mapstructure:"pullRequestId"`
	AssessedSHA                string             `json:"assessedSha" mapstructure:"assessedSha"`
	Phase                      string             `json:"phase" mapstructure:"phase"`
	Reviewers                  []ReviewerIdentity `json:"reviewers" mapstructure:"reviewers"`
	ReviewerRequestExecutionID string             `json:"reviewerRequestExecutionId" mapstructure:"reviewerRequestExecutionId"`
	Reason                     string             `json:"reason" mapstructure:"reason"`
	ResultURL                  string             `json:"resultUrl" mapstructure:"resultUrl"`
	SourceRunID                string             `json:"sourceRunId" mapstructure:"sourceRunId"`
	AttemptRootEventID         string             `json:"attemptRootEventId" mapstructure:"attemptRootEventId"`
	Enabled                    *bool              `json:"enabled,omitempty" mapstructure:"enabled"`
	ActivationGeneration       int64              `json:"activationGeneration" mapstructure:"activationGeneration"`
	ActivationNodeID           string             `json:"activationNodeId" mapstructure:"activationNodeId"`
}

type reviewCheckState struct {
	Configuration reviewCheckConfiguration `json:"configuration"`
	SHA           string                   `json:"sha"`
	Refname       string                   `json:"refname"`
	State         string                   `json:"state"`
	Reason        string                   `json:"reason"`
	Description   string                   `json:"description"`
	Reviewers     []ReviewerDecision       `json:"reviewers,omitempty"`
	Delivered     string                   `json:"delivered,omitempty"`
	Delivery      *core.PublicationLease   `json:"delivery,omitempty"`
	Failures      int                      `json:"failures,omitempty"`
	Error         string                   `json:"error,omitempty"`
	Invalidated   bool                     `json:"invalidated,omitempty"`
}

func (*SetPullRequestReviewCheck) Name() string  { return "bitbucket.setPullRequestReviewCheck" }
func (*SetPullRequestReviewCheck) Label() string { return "Set Pull Request Review Check" }
func (*SetPullRequestReviewCheck) Description() string {
	return "Track required approvals for the assessed pull request commit"
}
func (*SetPullRequestReviewCheck) OutputChannels(any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}
func (*SetPullRequestReviewCheck) Documentation() string {
	return `Publish superplane/review-request on a Bitbucket pull request from any canvas. Supply repository, pullRequestId, assessedSha, phase, and reason. The result URL defaults to the current run.

Use pending_selection before selection, required_reviewers after Ensure Pull Request Reviewers, no_review for an explicit decision, selection_error for failed selection, or disabled to release the check. Required reviewers use UUIDs and separate displayName values. Supply reviewerRequestExecutionId from the recorded reviewer action.

Pending selection finishes after durable registration. Other phases finish after delivery. Monitoring continues after the selection run finishes. Signed review events and the complete activity log must prove approval of the assessed current commit. Existing approvals without reliable commit evidence stay unresolved. Comments do not revoke approvals. Changes requests fail the check. A new head requires a new decision.

Source and attempt order come from trusted execution records. Activation generation defaults to 1. Use a greater saved generation after disabling. An optional activationNodeId uses that node's admitted and live settings. Disabling or removing the check releases its statuses. Publication retries are bounded and interrupted writes recover through existing integration hooks. This action does not change repository rules or enable Review request.`
}

func (*SetPullRequestReviewCheck) Configuration() []configuration.Field {
	return []configuration.Field{
		repositoryField(),
		{Name: "pullRequestId", Label: "Pull request ID", Type: configuration.FieldTypeString, Required: true},
		{Name: "assessedSha", Label: "Assessed SHA", Type: configuration.FieldTypeString, Required: true},
		{Name: "phase", Label: "Phase", Type: configuration.FieldTypeString, Required: true, Description: "Enter pending_selection, required_reviewers, no_review, selection_error, or disabled."},
		{Name: "reviewers", Label: "Required reviewers", Type: configuration.FieldTypeList, Description: "Reviewers identified by Bitbucket UUID.", TypeOptions: &configuration.TypeOptions{List: &configuration.ListTypeOptions{ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeObject, Schema: []configuration.Field{
			{Name: "uuid", Label: "UUID", Type: configuration.FieldTypeString, Required: true},
			{Name: "displayName", Label: "Display name", Type: configuration.FieldTypeString},
		}}}}},
		{Name: "reviewerRequestExecutionId", Label: "Reviewer request execution ID", Type: configuration.FieldTypeString},
		{Name: "reason", Label: "Reason", Type: configuration.FieldTypeString, Required: true},
		{Name: "resultUrl", Label: "Run URL", Type: configuration.FieldTypeString},
		{Name: "enabled", Label: "Enabled", Type: configuration.FieldTypeBool, Default: true, Description: "Disabling this node releases its managed checks."},
		{Name: "activationGeneration", Label: "Activation generation", Type: configuration.FieldTypeNumber, Default: 1},
		{Name: "activationNodeId", Label: "Activation node ID", Type: configuration.FieldTypeString},
		{Name: "sourceRunId", Label: "Source run ID", Type: configuration.FieldTypeString},
		{Name: "attemptRootEventId", Label: "Attempt root event ID", Type: configuration.FieldTypeString},
	}
}

func (c *SetPullRequestReviewCheck) Setup(ctx core.SetupContext) error {
	// Dynamic fields resolve at execution. Setup reads only saved activation.
	var activation struct {
		Enabled    *bool  `mapstructure:"enabled"`
		Node       string `mapstructure:"activationNodeId"`
		Repository string `mapstructure:"repository"`
	}
	if err := mapstructure.Decode(ctx.Configuration, &activation); err != nil {
		return err
	}
	if activation.Node != "" && activation.Node != ctx.NodeID {
		return nil
	}
	if activation.Enabled != nil && !*activation.Enabled {
		return c.Cleanup(ctx)
	}
	events := reviewCheckEvents()
	var metadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return err
	}
	if metadata.AuthType == AuthTypeForgeApp {
		events = []string{"repo:push", pullRequestEventCreated, pullRequestEventUpdated, pullRequestEventFulfilled, pullRequestEventRejected, forgeReviewerStatusEvent}
	}
	return requestRepositoryWebhook(ctx.HTTP, ctx.Metadata, ctx.Integration, ctx.Factory, activation.Repository, events)
}

func (*SetPullRequestReviewCheck) Execute(ctx core.ExecutionContext) error {
	if ctx.Publications == nil {
		return fmt.Errorf("publication context is required")
	}
	var config reviewCheckConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}
	canvas, err := uuid.Parse(ctx.WorkflowID)
	if err != nil {
		return err
	}
	if config.ActivationNodeID == "" {
		config.ActivationNodeID = ctx.NodeID
	}
	source, err := optionalReviewID(config.SourceRunID)
	if err != nil {
		return err
	}
	attempt, err := optionalReviewID(config.AttemptRootEventID)
	if err != nil {
		return err
	}
	order, err := ctx.Publications.ResolveOrder(source, attempt)
	if err != nil {
		return err
	}
	generation, eligible, err := reviewCheckActivation(ctx.Publications, canvas, config.ActivationNodeID, order.AttemptRootEventID)
	if err != nil {
		return err
	}
	if !eligible {
		return emitReviewCheck(ctx.ExecutionState, "skipped", nil, nil)
	}
	config.Repository = strings.TrimSpace(config.Repository)
	config.AssessedSHA = strings.ToLower(strings.TrimSpace(config.AssessedSHA))
	number, err := parsePullRequestID(config.PullRequestID)
	if err != nil {
		return err
	}
	config.PullRequestID = fmt.Sprint(number)
	config.ActivationGeneration = generation
	config.SourceRunID, config.AttemptRootEventID = order.SourceRunID.String(), order.AttemptRootEventID.String()
	if err := validateReviewCheck(config); err != nil {
		return err
	}
	if err := requireRepositoryInWorkspace(ctx.Integration, config.Repository); err != nil {
		return err
	}
	client, err := newIntegrationClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	repository, err := client.GetRepository(config.Repository)
	if err != nil {
		return err
	}
	if _, err := uuid.Parse(repository.UUID); err != nil {
		return fmt.Errorf("repository UUID is missing or invalid")
	}
	config.RepositoryUUID = bitbucketUUID(repository.UUID)
	if config.ResultURL == "" {
		config.ResultURL = fmt.Sprintf("%s/%s/apps/%s?run=%s", strings.TrimRight(ctx.BaseURL, "/"), ctx.OrganizationID, ctx.WorkflowID, ctx.RunID)
	}
	if config.Phase == "required_reviewers" {
		if err := validateRecordedReviewers(ctx.Publications, config); err != nil {
			return err
		}
	}
	accepted := int64(0)
	err = ctx.Publications.Update(config.key(), func(record *core.OrderedPublication) error {
		if !record.CanReplace(order, generation, config.Phase == "disabled") {
			return core.ErrPublicationUnchanged
		}
		var previous reviewCheckState
		if record.Generation > 0 {
			if err := json.Unmarshal(record.Payload, &previous); err != nil {
				return err
			}
			if order.Compare(record.Order) == 0 && record.ActivationGeneration == generation {
				if reflect.DeepEqual(previous.Configuration, config) {
					accepted = record.Generation
					previous.Error, previous.Failures = "", 0
					record.WakeAt = reviewWake(time.Second)
					record.Payload, err = json.Marshal(previous)
					return err
				}
				if config.Phase != "disabled" && previous.Configuration.Phase != "pending_selection" {
					return core.ErrPublicationUnchanged
				}
			}
		}
		next := reviewCheckState{Configuration: config, Delivery: previous.Delivery, Invalidated: previous.Invalidated && order.Compare(record.Order) == 0}
		if config.Phase == "disabled" {
			next.SHA, next.Refname = previous.SHA, previous.Refname
		}
		record.Generation++
		record.Order, record.ActivationGeneration, record.Disabled = order, generation, config.Phase == "disabled"
		record.OwnerCanvasID, record.OwnerNodeID = canvas, config.ActivationNodeID
		record.Hook, record.WakeAt = reviewCheckHook, reviewWake(time.Second)
		record.Payload, err = json.Marshal(next)
		accepted = record.Generation
		return err
	})
	if err != nil {
		return err
	}
	if accepted == 0 {
		return emitReviewCheck(ctx.ExecutionState, "superseded", nil, nil)
	}
	if config.Phase == "pending_selection" {
		return emitReviewCheck(ctx.ExecutionState, "registered", nil, nil)
	}
	if err := ctx.Metadata.Set(map[string]any{"key": config.key(), "generation": accepted}); err != nil {
		return err
	}
	return ctx.Requests.ScheduleActionCall(reviewCheckWaitHook, nil, time.Second)
}

func (*SetPullRequestReviewCheck) Hooks() []core.Hook {
	return []core.Hook{{Name: reviewCheckWaitHook, Type: core.HookTypeInternal}}
}
func (*SetPullRequestReviewCheck) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name != reviewCheckWaitHook {
		return fmt.Errorf("unknown hook: %s", ctx.Name)
	}
	if ctx.ExecutionState.IsFinished() || ctx.ExecutionState.IsCancelling() {
		return nil
	}
	if ctx.Publications == nil {
		return fmt.Errorf("publication context is required")
	}
	var metadata struct {
		Key        string
		Generation int64
	}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return err
	}
	record, err := ctx.Publications.Get(metadata.Key)
	if err != nil {
		return err
	}
	if record == nil || record.Generation != metadata.Generation {
		return emitReviewCheck(ctx.ExecutionState, "superseded", nil, nil)
	}
	var state reviewCheckState
	if err := json.Unmarshal(record.Payload, &state); err != nil {
		return err
	}
	if state.Error != "" {
		return ctx.ExecutionState.Fail("error", state.Error)
	}
	if state.Delivered == "" || state.Delivered != state.fingerprint() {
		return ctx.Requests.ScheduleActionCall(reviewCheckWaitHook, nil, time.Second)
	}
	return emitReviewCheck(ctx.ExecutionState, "applied", record, &state)
}

func (*SetPullRequestReviewCheck) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	code, err := verifyBitbucketSignature(ctx)
	if err != nil {
		return code, nil, err
	}
	store, err := bitbucketReviewStore(ctx.Integration)
	if err == nil {
		err = receiveReviewEvent(store, ctx.Headers.Get("X-Event-Key"), ctx.Body)
	}
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}
	return http.StatusOK, nil, nil
}

func (*SetPullRequestReviewCheck) Cleanup(ctx core.SetupContext) error {
	if ctx.WorkflowID == "" || ctx.Integration == nil {
		return nil
	}
	store, err := bitbucketReviewStore(ctx.Integration)
	if err != nil {
		return err
	}
	records, err := store.List(reviewCheckPrefix)
	if err != nil {
		return err
	}
	for _, record := range records {
		if record.OwnerCanvasID.String() != ctx.WorkflowID || record.OwnerNodeID != ctx.NodeID {
			continue
		}
		if err := store.Update(record.Key, func(current *core.OrderedPublication) error {
			if current.OwnerCanvasID.String() != ctx.WorkflowID || current.OwnerNodeID != ctx.NodeID {
				return core.ErrPublicationUnchanged
			}
			return disableReviewCheck(current)
		}); err != nil {
			return err
		}
	}
	return nil
}

func validateReviewCheck(config reviewCheckConfiguration) error {
	if !IsValidFullCommitSHA(config.AssessedSHA) || strings.TrimSpace(config.Reason) == "" {
		return fmt.Errorf("assessed full SHA and reason are required")
	}
	if !slices.Contains([]string{"pending_selection", "required_reviewers", "no_review", "selection_error", "disabled"}, config.Phase) {
		return fmt.Errorf("invalid review check phase")
	}
	if config.Phase != "required_reviewers" {
		return nil
	}
	if len(config.Reviewers) == 0 {
		return fmt.Errorf("required reviewers must not be empty")
	}
	seen := map[uuid.UUID]bool{}
	for _, reviewer := range config.Reviewers {
		id, err := uuid.Parse(reviewer.UUID)
		if err != nil || id == uuid.Nil || seen[id] {
			return fmt.Errorf("reviewers must have distinct Bitbucket UUIDs")
		}
		seen[id] = true
	}
	return nil
}

func validateRecordedReviewers(publications core.PublicationContext, config reviewCheckConfiguration) error {
	id, err := uuid.Parse(config.ReviewerRequestExecutionID)
	if err != nil {
		return fmt.Errorf("recorded reviewer request execution is required")
	}
	execution, err := publications.ExecutionOutput(id)
	if err != nil {
		return err
	}
	var ensured struct {
		Repository    string
		PullRequestID string `mapstructure:"pullRequestId"`
	}
	if err := mapstructure.Decode(execution.Configuration, &ensured); err != nil {
		return err
	}
	number, err := parsePullRequestID(ensured.PullRequestID)
	if err != nil {
		return err
	}
	ensured.PullRequestID = fmt.Sprint(number)
	var output struct {
		Type string
		Data struct {
			Status         string
			HeadSHA        string `mapstructure:"headSha"`
			Requested      []ReviewerIdentity
			AlreadyCovered []ReviewerIdentity `mapstructure:"alreadyCovered"`
		}
	}
	if err := mapstructure.Decode(execution.Output, &output); err != nil {
		return err
	}
	if execution.Component != "bitbucket.ensurePullRequestReviewers" || !strings.EqualFold(ensured.Repository, config.Repository) || ensured.PullRequestID != config.PullRequestID || output.Type != "bitbucket.pullRequestReviewersEnsured" || !slices.Contains([]string{"requested", "already_covered"}, output.Data.Status) || output.Data.HeadSHA != config.AssessedSHA {
		return fmt.Errorf("reviewer outcome does not cover this pull request and assessed commit")
	}
	covered := append(output.Data.Requested, output.Data.AlreadyCovered...)
	if len(covered) != len(config.Reviewers) {
		return fmt.Errorf("reviewers differ from the recorded outcome")
	}
	for _, reviewer := range config.Reviewers {
		if !slices.ContainsFunc(covered, func(other ReviewerIdentity) bool { return bitbucketUUID(other.UUID) == bitbucketUUID(reviewer.UUID) }) {
			return fmt.Errorf("reviewer is missing from the recorded outcome")
		}
	}
	return nil
}

func (c reviewCheckConfiguration) key() string {
	return fmt.Sprintf("%s%s/%s", reviewCheckPrefix, c.RepositoryUUID, c.PullRequestID)
}

// UUID paths cannot drift to a different repository after a slug is reused.
func (c reviewCheckConfiguration) apiRepository() string {
	workspace, _, _ := strings.Cut(c.Repository, "/")
	return workspace + "/{" + c.RepositoryUUID + "}"
}
func optionalReviewID(value string) (uuid.UUID, error) {
	if value == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(value)
}
func reviewWake(delay time.Duration) *time.Time { wake := time.Now().Add(delay); return &wake }
func (s reviewCheckState) fingerprint() string {
	raw, _ := json.Marshal([]string{s.SHA, s.Refname, s.State, s.Description, s.Configuration.ResultURL})
	return string(raw)
}
func emitReviewCheck(execution core.ExecutionStateContext, status string, record *core.OrderedPublication, state *reviewCheckState) error {
	out := map[string]any{"status": status, "key": reviewStatusKey}
	if record != nil {
		out["generation"], out["activationGeneration"] = record.Generation, record.ActivationGeneration
		out["sourceRunId"], out["attemptRootEventId"] = record.Order.SourceRunID, record.Order.AttemptRootEventID
	}
	if state != nil {
		out["state"], out["sha"], out["reason"], out["reviewers"] = state.State, state.SHA, state.Reason, state.Reviewers
	}
	return execution.Emit("default", "bitbucket.pullRequestReviewCheck", []any{out})
}
func (*SetPullRequestReviewCheck) ExampleOutput() map[string]any {
	return map[string]any{"type": "bitbucket.pullRequestReviewCheck", "timestamp": "2026-11-13T09:00:00Z", "data": map[string]any{"status": "applied", "key": reviewStatusKey, "state": "INPROGRESS", "reason": "approval_commit_unverified", "sha": strings.Repeat("a", 40)}}
}
