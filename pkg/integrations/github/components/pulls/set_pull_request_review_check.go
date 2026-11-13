package pulls

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
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

const ReviewCheckHook = "refreshPullRequestReviewCheck"
const reviewCheckWaitHook = "awaitReviewCheck"
const reviewCheckPrefix = "github.review-check/"

type SetPullRequestReviewCheck struct{}

type reviewCheckConfiguration struct {
	ActivationNodeID           string             `json:"activationNodeId,omitempty" mapstructure:"activationNodeId"`
	Enabled                    *bool              `json:"enabled,omitempty" mapstructure:"enabled"`
	Repository                 string             `json:"repository" mapstructure:"repository"`
	PullNumber                 any                `json:"pullNumber" mapstructure:"pullNumber"`
	AssessedSHA                string             `json:"assessedSha" mapstructure:"assessedSha"`
	Context                    string             `json:"context" mapstructure:"context"`
	ActivationGeneration       int64              `json:"activationGeneration" mapstructure:"activationGeneration"`
	SourceRunID                string             `json:"sourceRunId,omitempty" mapstructure:"sourceRunId"`
	AttemptRootEventID         string             `json:"attemptRootEventId,omitempty" mapstructure:"attemptRootEventId"`
	Phase                      string             `json:"phase" mapstructure:"phase"`
	State                      string             `json:"state,omitempty" mapstructure:"state"`
	Reviewers                  []ReviewerIdentity `json:"reviewers" mapstructure:"reviewers"`
	ReviewerRequestExecutionID string             `json:"reviewerRequestExecutionId,omitempty" mapstructure:"reviewerRequestExecutionId"`
	Reason                     string             `json:"reason" mapstructure:"reason"`
	ResultURL                  string             `json:"resultUrl,omitempty" mapstructure:"resultUrl"`
}

type reviewCheckState struct {
	Configuration reviewCheckConfiguration `json:"configuration"`
	SHA           string                   `json:"sha"`
	State         string                   `json:"state"`
	Description   string                   `json:"description"`
	Delivered     string                   `json:"delivered,omitempty"`
	Delivery      *reviewCheckDelivery     `json:"delivery,omitempty"`
	Error         string                   `json:"error,omitempty"`
	SetupFailures int                      `json:"setupFailures,omitempty"`
	Retired       []retiredCommitCheck     `json:"retired,omitempty"`
}

type retiredCommitCheck struct {
	SHA       string `json:"sha"`
	ResultURL string `json:"resultUrl"`
}

type reviewCheckDelivery struct {
	ID        uuid.UUID `json:"id"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func (*SetPullRequestReviewCheck) Name() string  { return "github.setPullRequestReviewCheck" }
func (*SetPullRequestReviewCheck) Label() string { return "Set Pull Request Review Check" }
func (*SetPullRequestReviewCheck) Description() string {
	return "Track required approvals for the assessed pull request commit"
}
func (*SetPullRequestReviewCheck) Icon() string  { return "github" }
func (*SetPullRequestReviewCheck) Color() string { return "gray" }
func (*SetPullRequestReviewCheck) OutputChannels(any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}
func (*SetPullRequestReviewCheck) Documentation() string {
	return `Publish a commit status and track required reviewers in any canvas. Pending selection finishes after durable registration. Other phases finish after delivery. Approval monitoring continues through integration hooks.

Use pending_selection before selection, required_reviewers after Ensure Pull Request Reviewers, no_review for an explicit decision, selection_error for failed selection, or disabled to release the check.

Supply the assessed SHA, status context, reason, and optional result URL. For required_reviewers, supply reviewer IDs and logins and reviewerRequestExecutionId from the recorded Ensure Pull Request Reviewers output. This component does not request reviewers again.

An optional Activation Node ID binds participation to that check node in the admitted canvas version. Its live enabled state and activation generation must still match. Saving false releases owned checks durably.

Source Run ID and Attempt Root Event ID default to trusted current execution records. Cross-canvas sources require broadcast provenance. Persisted creation times and IDs determine order. Older attempts cannot replace newer state, including on the same commit. Disabled activations remain fenced. Activation Generation is a positive saved version. After disabling, use a greater activation generation to enable the check again.

All required reviewers must approve the assessed current commit. Comments do not change approval. Requested changes fail the check. Dismissed replacement reviews cannot revive older approvals. New heads stay pending for a new decision. Closed pull requests stop polling; reopening resumes monitoring.

Provider failures publish error and retry. Delivery is durable and at least once. A crash after a GitHub write can repeat the same status. This component does not change branch rules or start agents.`
}
func (*SetPullRequestReviewCheck) Configuration() []configuration.Field {
	fields := (&ListPullRequestReviewCandidates{}).Configuration()
	reviewerField := reviewerConfigurationField()
	reviewerField.Required = false
	return append(fields,
		configuration.Field{Name: "enabled", Label: "Enabled", Type: configuration.FieldTypeBool, Default: true, Description: "Disabling this node releases its managed checks."},
		configuration.Field{Name: "activationNodeId", Label: "Activation node ID", Type: configuration.FieldTypeString, Description: "Optional check node whose saved enabled state and activation generation govern this action."},
		configuration.Field{Name: "assessedSha", Label: "Assessed SHA", Type: configuration.FieldTypeString, Required: true},
		configuration.Field{Name: "context", Label: "Status context", Type: configuration.FieldTypeString, Required: true, Description: "Stable status name for this requirement."},
		configuration.Field{Name: "activationGeneration", Label: "Activation generation", Type: configuration.FieldTypeNumber, Default: 1, Description: "Use a positive saved generation. After disabling, select a greater generation to enable again."},
		configuration.Field{Name: "sourceRunId", Label: "Source run ID", Type: configuration.FieldTypeString, Description: "Defaults to the broadcast source or current run."},
		configuration.Field{Name: "attemptRootEventId", Label: "Attempt root event ID", Type: configuration.FieldTypeString, Description: "Defaults to the current run root event."},
		configuration.Field{Name: "phase", Label: "Phase", Type: configuration.FieldTypeSelect, Required: true, TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{
			{Label: "Pending selection", Value: "pending_selection"}, {Label: "Required reviewers", Value: "required_reviewers"}, {Label: "No review", Value: "no_review"}, {Label: "Selection error", Value: "selection_error"}, {Label: "Disabled", Value: "disabled"},
		}}}},
		reviewerField,
		configuration.Field{Name: "reviewerRequestExecutionId", Label: "Reviewer request execution ID", Type: configuration.FieldTypeString, Description: "Execution ID from Ensure Pull Request Reviewers. Required when reviewers are required."},
		configuration.Field{Name: "reason", Label: "Reason", Type: configuration.FieldTypeString, Required: true},
		configuration.Field{Name: "resultUrl", Label: "Result URL", Type: configuration.FieldTypeString},
	)
}

func (c *SetPullRequestReviewCheck) Setup(ctx core.SetupContext) error {
	var config struct {
		Enabled          *bool  `mapstructure:"enabled"`
		ActivationNodeID string `mapstructure:"activationNodeId"`
	}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}
	if config.ActivationNodeID != "" && config.ActivationNodeID != ctx.NodeID {
		return nil
	}
	if config.Enabled != nil && !*config.Enabled {
		return c.Cleanup(ctx)
	}
	if err := (&ListPullRequestReviewCandidates{}).Setup(ctx); err != nil {
		return err
	}
	var candidatesConfig reviewCandidatesConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &candidatesConfig); err != nil {
		return err
	}
	webhook, err := common.WorkspaceWebhookConfiguration(ctx.Factory, common.WebhookConfiguration{Repository: candidatesConfig.Repository, EventTypes: []string{"pull_request", "pull_request_review"}})
	if err != nil {
		return err
	}
	return ctx.Integration.RequestWebhook(webhook)
}

func (*SetPullRequestReviewCheck) Execute(ctx core.ExecutionContext) error {
	var scopeConfig reviewCheckConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &scopeConfig); err != nil {
		return err
	}
	if scopeConfig.ActivationNodeID != "" {
		if ctx.Publications == nil {
			return fmt.Errorf("publication context is required")
		}
		order, err := ctx.Publications.ResolveOrder(uuid.Nil, uuid.Nil)
		if err != nil {
			return err
		}
		generation, eligible, err := reviewCheckActivation(ctx.Publications, uuid.MustParse(ctx.WorkflowID), scopeConfig.ActivationNodeID, order.AttemptRootEventID)
		if err != nil {
			return err
		}
		if !eligible {
			return emitReviewCheck(ctx.ExecutionState, "skipped", nil, nil)
		}
		raw, err := configurationWithActivation(ctx.Configuration, generation)
		if err != nil {
			return err
		}
		ctx.Configuration = raw
	} else if scopeConfig.Enabled != nil && !*scopeConfig.Enabled {
		return emitReviewCheck(ctx.ExecutionState, "skipped", nil, nil)
	}
	config, err := decodeReviewCheck(ctx.Configuration)
	if err != nil {
		return err
	}
	return executePullRequestCheck(ctx, config)
}

func executePullRequestCheck(ctx core.ExecutionContext, config reviewCheckConfiguration) error {
	if ctx.Publications == nil {
		return fmt.Errorf("publication context is required")
	}
	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return err
	}
	config.Repository = client.RepositoryFullName(config.Repository)
	source, err := optionalPublicationID(config.SourceRunID)
	if err != nil {
		return err
	}
	attempt, err := optionalPublicationID(config.AttemptRootEventID)
	if err != nil {
		return err
	}
	order, err := ctx.Publications.ResolveOrder(source, attempt)
	if err != nil {
		return err
	}
	config.SourceRunID, config.AttemptRootEventID = order.SourceRunID.String(), order.AttemptRootEventID.String()
	if config.Phase == "required_reviewers" {
		if err := validateRecordedReviewers(ctx.Publications, config); err != nil {
			return err
		}
	}
	if config.ActivationNodeID != "" {
		generation, eligible, err := reviewCheckActivation(ctx.Publications, uuid.MustParse(ctx.WorkflowID), config.ActivationNodeID, order.AttemptRootEventID)
		if err != nil {
			return err
		}
		if !eligible || generation != config.ActivationGeneration {
			return emitReviewCheck(ctx.ExecutionState, "superseded", nil, nil)
		}
	}
	key := config.key()
	var generation int64
	status := "applied"
	err = ctx.Publications.Update(key, func(record *core.OrderedPublication) error {
		comparison := order.Compare(record.Order)
		if !record.CanReplace(order, config.ActivationGeneration, config.Phase == "disabled") {
			status = "superseded"
			return core.ErrPublicationUnchanged
		}
		var previous reviewCheckState
		if record.Generation > 0 {
			previous, err = decodeReviewCheckState(record.Payload)
			if err != nil {
				return err
			}
		}
		if comparison == 0 && record.Generation > 0 && record.ActivationGeneration == config.ActivationGeneration {
			if reflect.DeepEqual(previous.Configuration, config) {
				generation = record.Generation
				if previous.Error != "" {
					previous.Error = ""
					previous.SetupFailures = 0
					record.Payload, err = json.Marshal(previous)
					if err != nil {
						return err
					}
					record.WakeAt = reviewCheckWake(time.Second)
					return nil
				}
				if previous.Delivered == "" || previous.Delivered != previous.fingerprint() {
					record.WakeAt = reviewCheckWake(time.Second)
					return nil
				}
				return core.ErrPublicationUnchanged
			}
			// One attempt can move from selection to one outcome, then disable.
			if previous.Configuration.Phase != "pending_selection" && previous.Configuration.State != "pending" && config.Phase != "disabled" {
				status = "superseded"
				return core.ErrPublicationUnchanged
			}
		}
		record.Generation++
		record.Order = order
		record.Disabled, record.ActivationGeneration = config.Phase == "disabled", config.ActivationGeneration
		record.OwnerCanvasID, record.OwnerNodeID = uuid.MustParse(ctx.WorkflowID), ctx.NodeID
		if config.ActivationNodeID != "" {
			record.OwnerNodeID = config.ActivationNodeID
		}
		// A replacement must wait for an already claimed status write to finish.
		retired := previous.Retired
		if config.State != "" && previous.Configuration.State == "pending" && previous.Configuration.AssessedSHA != config.AssessedSHA {
			retired = append(retired, retiredCommitCheck{SHA: previous.Configuration.AssessedSHA, ResultURL: previous.Configuration.ResultURL})
		}
		record.Payload, err = json.Marshal(reviewCheckState{Configuration: config, Delivery: previous.Delivery, Retired: retired})
		if err != nil {
			return err
		}
		record.Hook = ReviewCheckHook
		record.WakeAt = reviewCheckWake(time.Second)
		generation = record.Generation
		return nil
	})
	if err != nil {
		return err
	}
	if status == "superseded" {
		return emitReviewCheck(ctx.ExecutionState, status, nil, &reviewCheckState{Configuration: config})
	}
	if config.Phase == "pending_selection" {
		return emitReviewCheck(ctx.ExecutionState, "registered", nil, nil)
	}
	if err := ctx.Metadata.Set(map[string]any{"key": key, "generation": generation}); err != nil {
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
	var metadata struct {
		Key        string
		Generation int64
	}
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return err
	}
	if ctx.Publications == nil {
		return fmt.Errorf("publication context is required")
	}
	record, err := ctx.Publications.Get(metadata.Key)
	if err != nil {
		return err
	}
	if record == nil || record.Generation != metadata.Generation {
		var config reviewCheckConfiguration
		if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
			return err
		}
		return emitReviewCheck(ctx.ExecutionState, "superseded", nil, &reviewCheckState{Configuration: config})
	}
	state, err := decodeReviewCheckState(record.Payload)
	if err != nil {
		return err
	}
	if state.Error != "" {
		return ctx.ExecutionState.Fail("error", state.Error)
	}
	if state.Configuration.State != "" {
		owner, err := ctx.Publications.OwnerStatus(record.OwnerCanvasID, record.OwnerNodeID)
		if err != nil {
			return err
		}
		if !owner.Available {
			return ctx.ExecutionState.Fail("error", "The organization or integration is unavailable.")
		}
	}
	if state.Delivered == "" || state.Delivered != state.fingerprint() {
		return ctx.Requests.ScheduleActionCall(reviewCheckWaitHook, nil, time.Second)
	}
	return emitReviewCheck(ctx.ExecutionState, "applied", record, &state)
}

func (*SetPullRequestReviewCheck) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	code, err := common.VerifySignature(ctx)
	if err != nil {
		return code, nil, err
	}
	err = WakeReviewChecks(ctx.Integration, ctx.Headers.Get("X-GitHub-Event"), ctx.Body)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}
	return http.StatusOK, nil, nil
}
func (*SetPullRequestReviewCheck) Cancel(core.ExecutionContext) error { return nil }
func (*SetPullRequestReviewCheck) Cleanup(ctx core.SetupContext) error {
	if ctx.WorkflowID == "" || ctx.Integration == nil {
		return nil
	}
	store, err := reviewCheckStore(ctx.Integration)
	if err != nil {
		return err
	}
	return releaseReviewChecks(store, uuid.MustParse(ctx.WorkflowID), ctx.NodeID)
}
func (*SetPullRequestReviewCheck) ExampleOutput() map[string]any {
	return map[string]any{"type": "github.pullRequestReviewCheck", "timestamp": "2026-11-13T09:00:00Z", "data": map[string]any{"status": "applied", "state": "pending", "sha": "abc", "phase": "required_reviewers", "context": "team/review", "generation": 2, "activationGeneration": 1, "sourceRunId": "source-run-id", "attemptRootEventId": "attempt-root-event-id"}}
}

func decodeReviewCheck(raw any) (reviewCheckConfiguration, error) {
	c, err := decodePullRequestCheck(raw)
	if err != nil {
		return c, err
	}
	if c.State != "" || !slices.Contains([]string{"pending_selection", "required_reviewers", "no_review", "selection_error", "disabled"}, c.Phase) {
		return c, fmt.Errorf("invalid review check phase")
	}
	if c.Phase == "required_reviewers" {
		if err := validateDesiredReviewers(c.Reviewers); err != nil {
			return c, err
		}
		if _, err := uuid.Parse(c.ReviewerRequestExecutionID); err != nil {
			return c, fmt.Errorf("reviewer request execution ID is required")
		}
	}
	return c, nil
}

func decodePullRequestCheck(raw any) (reviewCheckConfiguration, error) {
	var c reviewCheckConfiguration
	if err := mapstructure.Decode(raw, &c); err != nil {
		return c, err
	}
	c.Repository, c.Context, c.AssessedSHA = strings.TrimSpace(c.Repository), strings.ToLower(strings.TrimSpace(c.Context)), strings.TrimSpace(c.AssessedSHA)
	number, err := parsePullNumber(c.PullNumber)
	if err != nil {
		return c, err
	}
	c.PullNumber = fmt.Sprint(number)
	if c.ActivationGeneration == 0 {
		c.ActivationGeneration = 1
	}
	if c.ActivationGeneration < 1 {
		return c, fmt.Errorf("activation generation must be positive")
	}
	if c.Repository == "" || c.Context == "" || c.AssessedSHA == "" || strings.TrimSpace(c.Reason) == "" {
		return c, fmt.Errorf("repository, assessed SHA, context, and reason are required")
	}
	return c, nil
}
func (c reviewCheckConfiguration) key() string {
	number, _ := parsePullNumber(c.PullNumber)
	return fmt.Sprintf("%s%s/%d/%s", reviewCheckPrefix, strings.ToLower(c.Repository), number, c.Context)
}
func optionalPublicationID(raw string) (uuid.UUID, error) {
	if raw == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(raw)
}
func reviewCheckWake(delay time.Duration) *time.Time { wake := time.Now().Add(delay); return &wake }
func decodeReviewCheckState(raw json.RawMessage) (reviewCheckState, error) {
	var state reviewCheckState
	err := json.Unmarshal(raw, &state)
	return state, err
}
func (s reviewCheckState) fingerprint() string {
	b, _ := json.Marshal([]string{s.SHA, s.State, s.Description, s.resultURL()})
	return string(b)
}

func (s reviewCheckState) resultURL() string {
	if len(s.Retired) > 0 {
		return s.Retired[0].ResultURL
	}
	return s.Configuration.ResultURL
}
func emitReviewCheck(execution core.ExecutionStateContext, status string, record *core.OrderedPublication, state *reviewCheckState) error {
	out := map[string]any{"status": status}
	if record != nil {
		out["generation"], out["sourceRunId"], out["attemptRootEventId"] = record.Generation, record.Order.SourceRunID, record.Order.AttemptRootEventID
		out["activationGeneration"] = record.ActivationGeneration
	}
	if state != nil {
		out["state"], out["sha"], out["phase"], out["context"] = state.State, state.SHA, state.Configuration.Phase, state.Configuration.Context
	}
	payloadType := "github.pullRequestReviewCheck"
	if state != nil && state.Configuration.State != "" {
		payloadType = "github.pullRequestCheck"
	}
	return execution.Emit("default", payloadType, []any{out})
}

func validateRecordedReviewers(publications core.PublicationContext, config reviewCheckConfiguration) error {
	execution, err := publications.ExecutionOutput(uuid.MustParse(config.ReviewerRequestExecutionID))
	if err != nil {
		return fmt.Errorf("read recorded reviewer requests: %w", err)
	}
	if execution.Component != "github.ensurePullRequestReviewers" {
		return fmt.Errorf("reviewer outcome must come from Ensure Pull Request Reviewers")
	}
	var ensured ensureReviewersConfiguration
	if err := mapstructure.Decode(execution.Configuration, &ensured); err != nil {
		return err
	}
	if !strings.Contains(ensured.Repository, "/") {
		owner, _, _ := strings.Cut(config.Repository, "/")
		ensured.Repository = owner + "/" + ensured.Repository
	}
	number, _ := parsePullNumber(config.PullNumber)
	ensuredNumber, err := parsePullNumber(ensured.PullNumber)
	if err != nil {
		return err
	}
	if !strings.EqualFold(ensured.Repository, config.Repository) || ensuredNumber != number {
		return fmt.Errorf("reviewer outcome belongs to another pull request")
	}
	var envelope struct {
		Type string
		Data ensureReviewersOutcome
	}
	if err := mapstructure.Decode(execution.Output, &envelope); err != nil {
		return err
	}
	if envelope.Type != "github.pullRequestReviewersEnsured" || !slices.Contains([]string{"requested", "already_covered"}, envelope.Data.Status) || envelope.Data.HeadSHA != config.AssessedSHA {
		return fmt.Errorf("reviewer outcome does not cover the assessed commit")
	}
	covered := append(slices.Clone(envelope.Data.Requested), envelope.Data.AlreadyCovered...)
	if len(covered) != len(config.Reviewers) {
		return fmt.Errorf("reviewers differ from the recorded outcome")
	}
	for _, reviewer := range config.Reviewers {
		if !slices.ContainsFunc(covered, func(other ReviewerIdentity) bool {
			return reviewer.ID == other.ID && strings.EqualFold(reviewer.Login, other.Login)
		}) {
			return fmt.Errorf("reviewer is missing from the recorded outcome: %s", reviewer.Login)
		}
	}
	return nil
}
