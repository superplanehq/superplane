package harness

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const OnPipelineCompletedPayloadType = "harness.pipeline.completed"
const (
	OnPipelineCompletedPollAction               = "poll"
	OnPipelineCompletedPollInterval             = 1 * time.Minute
	OnPipelineCompletedPollPageSize             = 100
	OnPipelineCompletedPollMaxPages             = 100
	OnPipelineCompletedMaxPollErrors            = 5
	OnPipelineCompletedMaxTimestamplessIDs      = 64
	OnPipelineCompletedCheckpointLockShardCount = 64
	// When pipeline-scoped webhook delivery is configured, defer very recent
	// poll items to avoid duplicate emits from webhook/poll races.
	OnPipelineCompletedPollRaceWindow = 2 * time.Minute
)

var errOnPipelineCompletedUnsupportedPipelineIdentifierFilter = errors.New(
	"pipelineIdentifier filter is not supported by this Harness account/endpoint",
)
var errOnPipelineCompletedDeferredForRaceWindow = errors.New(
	"poll execution deferred due to race window",
)
var onPipelineCompletedCheckpointLocks [OnPipelineCompletedCheckpointLockShardCount]sync.Mutex

type OnPipelineCompleted struct{}

type OnPipelineCompletedConfiguration struct {
	OrgID              string   `json:"orgId" mapstructure:"orgId"`
	ProjectID          string   `json:"projectId" mapstructure:"projectId"`
	PipelineIdentifier string   `json:"pipelineIdentifier" mapstructure:"pipelineIdentifier"`
	Statuses           []string `json:"statuses" mapstructure:"statuses"`
}

type OnPipelineCompletedMetadata struct {
	PipelineIdentifier                 string `json:"pipelineIdentifier,omitempty" mapstructure:"pipelineIdentifier"`
	LastExecutionID                    string `json:"lastExecutionId,omitempty" mapstructure:"lastExecutionId"`
	LastTimestamplessExecutionIDs      string `json:"lastTimestamplessExecutionIds,omitempty" mapstructure:"lastTimestamplessExecutionIds"`
	LastTimestamplessExecutionID       string `json:"lastTimestamplessExecutionId,omitempty" mapstructure:"lastTimestamplessExecutionId"`
	LastExecutionEnded                 int64  `json:"lastExecutionEnded,omitempty" mapstructure:"lastExecutionEnded"`
	PollErrorCount                     int    `json:"pollErrorCount,omitempty" mapstructure:"pollErrorCount"`
	DisableServerPipelineIDFilterInAPI bool   `json:"disableServerPipelineIdFilterInApi,omitempty" mapstructure:"disableServerPipelineIdFilterInApi"`
}

var onPipelineCompletedStatusOptions = []configuration.FieldOption{
	{Label: "Succeeded", Value: "succeeded"},
	{Label: "Failed", Value: "failed"},
	{Label: "Aborted", Value: "aborted"},
	{Label: "Expired", Value: "expired"},
}

var onPipelineCompletedAllowedStatuses = []string{"succeeded", "failed", "aborted", "expired"}

var parseEpochMillisecondsLayouts = []string{
	time.RFC3339,
	"Mon Jan 2 15:04:05 MST 2006",
	time.RFC1123,
	time.RFC1123Z,
}

func (t *OnPipelineCompleted) Name() string {
	return "harness.onPipelineCompleted"
}

func (t *OnPipelineCompleted) Label() string {
	return "On Pipeline Completed"
}

func (t *OnPipelineCompleted) Description() string {
	return "Listen to Harness pipeline completion events"
}

func (t *OnPipelineCompleted) Documentation() string {
	return `The On Pipeline Completed trigger starts a workflow when a Harness pipeline execution finishes.

## Use Cases

- **Failure notifications**: Send Slack alerts when critical pipelines fail
- **Release automation**: Trigger post-deploy checks when a deployment pipeline succeeds
- **Incident workflows**: Create tickets for aborted/expired pipeline runs

## Configuration

- **Org**: Harness organization identifier
- **Project**: Harness project identifier
- **Pipeline Identifier**: Optional pipeline identifier filter. Leave empty to accept all pipeline completions.
- **Statuses**: Completion statuses that should trigger the workflow.

## Webhook Setup

SuperPlane automatically provisions Harness pipeline ` + "`notificationRules`" + ` when **Pipeline** is selected.

If no pipeline is selected, or webhook delivery is unavailable in your Harness account, SuperPlane falls back to polling recent executions.`
}

func (t *OnPipelineCompleted) Icon() string {
	return "workflow"
}

func (t *OnPipelineCompleted) Color() string {
	return "gray"
}

func (t *OnPipelineCompleted) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "orgId",
			Label:       "Organization",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select Harness organization",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeOrg,
				},
			},
		},
		{
			Name:        "projectId",
			Label:       "Project",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Select Harness project",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeProject,
					Parameters: []configuration.ParameterRef{
						{
							Name: "orgId",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "orgId",
							},
						},
					},
				},
			},
		},
		{
			Name:        "pipelineIdentifier",
			Label:       "Pipeline",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Description: "Optional pipeline filter",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypePipeline,
					Parameters: []configuration.ParameterRef{
						{
							Name: "orgId",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "orgId",
							},
						},
						{
							Name: "projectId",
							ValueFrom: &configuration.ParameterValueFrom{
								Field: "projectId",
							},
						},
					},
				},
			},
		},
		{
			Name:        "statuses",
			Label:       "Statuses",
			Type:        configuration.FieldTypeMultiSelect,
			Required:    true,
			Default:     []string{"succeeded", "failed"},
			Description: "Pipeline completion statuses to listen for",
			TypeOptions: &configuration.TypeOptions{
				MultiSelect: &configuration.MultiSelectTypeOptions{Options: onPipelineCompletedStatusOptions},
			},
		},
	}
}

func (t *OnPipelineCompleted) Setup(ctx core.TriggerContext) error {
	config, err := decodeOnPipelineCompletedConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}

	if ctx.Integration == nil {
		return fmt.Errorf("missing integration context")
	}
	if ctx.Metadata == nil {
		return fmt.Errorf("missing metadata context")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	if err := validateHarnessScopeSelection(client, config.OrgID, config.ProjectID); err != nil {
		return err
	}

	if err := validateHarnessPipelineSelection(client, config.OrgID, config.ProjectID, config.PipelineIdentifier); err != nil {
		return err
	}

	webhookConfig := WebhookConfiguration{
		PipelineIdentifier: config.PipelineIdentifier,
		OrgID:              config.OrgID,
		ProjectID:          config.ProjectID,
		EventTypes:         defaultWebhookEventTypes,
	}
	webhookReconciled := true
	if err := ctx.Integration.RequestWebhook(webhookConfig); err != nil {
		webhookReconciled = false
		if ctx.Logger != nil {
			ctx.Logger.Warnf("failed to reconcile Harness webhook configuration: %v", err)
		}
	}

	wantsWebhookDelivery := strings.TrimSpace(config.PipelineIdentifier) != ""
	webhookReady := false
	if wantsWebhookDelivery && webhookReconciled && ctx.Webhook != nil {
		resolvedURL, setupErr := ctx.Webhook.Setup()
		if setupErr != nil {
			if ctx.Logger != nil {
				ctx.Logger.Warnf("failed to setup Harness webhook URL, using polling fallback: %v", setupErr)
			}
		} else if strings.TrimSpace(resolvedURL) != "" {
			webhookReady = true
		}
	} else if wantsWebhookDelivery && ctx.Logger != nil {
		if !webhookReconciled {
			ctx.Logger.Warnf("Harness webhook reconciliation failed, using polling fallback")
		} else {
			ctx.Logger.Warnf("Harness webhook context is unavailable, using polling fallback")
		}
	}

	currentMetadata, err := decodeOnPipelineCompletedMetadata(ctx.Metadata.Get())
	if err != nil {
		currentMetadata = OnPipelineCompletedMetadata{}
	}
	if currentMetadata.LastExecutionEnded == 0 {
		currentMetadata.LastExecutionEnded = time.Now().UnixMilli()
	}
	currentMetadata.PipelineIdentifier = config.PipelineIdentifier

	if err := ctx.Metadata.Set(currentMetadata); err != nil {
		return err
	}

	if wantsWebhookDelivery && !webhookReady && ctx.Logger != nil {
		ctx.Logger.Warnf("Harness webhook is unavailable, running in polling mode")
	}

	// Always schedule polling as safety fallback even when webhook delivery
	// appears ready. Some Harness accounts can silently miss webhook delivery
	// due to account-level flags/configuration.
	if ctx.Requests != nil {
		if err := scheduleOnPipelineCompletedPoll(ctx.Requests); err != nil {
			return fmt.Errorf("failed to schedule polling action: %w", err)
		}
	}

	return nil
}

func (t *OnPipelineCompleted) Actions() []core.Action {
	return []core.Action{
		{
			Name:           OnPipelineCompletedPollAction,
			Description:    "Poll Harness executions as fallback for webhook delivery",
			UserAccessible: false,
		},
	}
}

func (t *OnPipelineCompleted) HandleAction(ctx core.TriggerActionContext) (map[string]any, error) {
	switch ctx.Name {
	case OnPipelineCompletedPollAction:
		return nil, t.poll(ctx)
	default:
		return nil, fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (t *OnPipelineCompleted) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	if err := authorizeWebhook(ctx); err != nil {
		return http.StatusForbidden, err
	}

	config, err := decodeOnPipelineCompletedConfiguration(ctx.Configuration)
	if err != nil {
		return http.StatusBadRequest, err
	}

	payload := map[string]any{}
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	event := extractPipelineWebhookEvent(payload)
	if !isPipelineCompletedEventType(event.EventType) {
		return http.StatusOK, nil
	}

	if !isTerminalStatus(event.Status) {
		return http.StatusOK, nil
	}

	webhookExecution := executionSummaryFromWebhookPayload(event, payload)

	if config.PipelineIdentifier != "" {
		if event.PipelineIdentifier == "" || config.PipelineIdentifier != event.PipelineIdentifier {
			return http.StatusOK, nil
		}
	}

	checkpointMu := onPipelineCompletedCheckpointMutex(config)
	checkpointMu.Lock()
	defer checkpointMu.Unlock()

	if ctx.Metadata != nil {
		metadata, decodeErr := decodeOnPipelineCompletedMetadata(ctx.Metadata.Get())
		if decodeErr != nil {
			return http.StatusInternalServerError, decodeErr
		}
		if !isNewerExecution(metadata, webhookExecution) {
			return http.StatusOK, nil
		}
	}

	if len(config.Statuses) > 0 && !statusSelected(config.Statuses, event.Status) {
		if updateErr := t.updateCheckpointFromExecutionUnlocked(ctx.Metadata, webhookExecution); updateErr != nil {
			return http.StatusInternalServerError, updateErr
		}
		return http.StatusOK, nil
	}

	emittedPayload := map[string]any{
		"executionId":        event.ExecutionID,
		"pipelineIdentifier": event.PipelineIdentifier,
		"status":             canonicalStatus(event.Status),
		"eventType":          event.EventType,
		"raw":                payload,
	}

	if err := ctx.Events.Emit(OnPipelineCompletedPayloadType, emittedPayload); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to emit event: %w", err)
	}

	if updateErr := t.updateCheckpointFromExecutionUnlocked(ctx.Metadata, webhookExecution); updateErr != nil {
		return http.StatusInternalServerError, updateErr
	}

	return http.StatusOK, nil
}

func (t *OnPipelineCompleted) poll(ctx core.TriggerActionContext) error {
	config, err := decodeOnPipelineCompletedConfiguration(ctx.Configuration)
	if err != nil {
		return err
	}
	if ctx.Metadata == nil {
		return fmt.Errorf("missing metadata context")
	}

	metadata, err := decodeOnPipelineCompletedMetadata(ctx.Metadata.Get())
	if err != nil {
		return fmt.Errorf("failed to decode trigger metadata: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	client = client.withScope(config.OrgID, config.ProjectID)

	pipelineIdentifierForAPI := config.PipelineIdentifier
	if metadata.DisableServerPipelineIDFilterInAPI {
		pipelineIdentifierForAPI = ""
	}

	executions, err := t.collectExecutionsSinceCheckpoint(client, metadata, pipelineIdentifierForAPI)
	if err != nil && pipelineIdentifierForAPI != "" && errors.Is(err, errOnPipelineCompletedUnsupportedPipelineIdentifierFilter) {
		updatedMetadata, setErr := t.updatePollingMetadata(ctx.Metadata, config, func(current *OnPipelineCompletedMetadata) {
			current.DisableServerPipelineIDFilterInAPI = true
		})
		if setErr != nil {
			return setErr
		}
		metadata = updatedMetadata
		if ctx.Logger != nil {
			ctx.Logger.Warnf("Harness execution summary API does not support pipelineIdentifier filter, falling back to unfiltered polling")
		}

		executions, err = t.collectExecutionsSinceCheckpoint(client, metadata, "")
	}

	if err != nil {
		updatedMetadata, setErr := t.updatePollingMetadata(ctx.Metadata, config, func(current *OnPipelineCompletedMetadata) {
			current.PollErrorCount++
		})
		if setErr != nil {
			return setErr
		}
		metadata = updatedMetadata

		if ctx.Logger != nil {
			ctx.Logger.Warnf("failed to poll Harness pipeline executions (attempt %d): %v", metadata.PollErrorCount, err)
		}
		if metadata.PollErrorCount >= OnPipelineCompletedMaxPollErrors && ctx.Logger != nil {
			ctx.Logger.Warnf("Harness polling has reached %d consecutive failures; continuing with backoff polling", metadata.PollErrorCount)
		}

		return scheduleOnPipelineCompletedPoll(ctx.Requests)
	}

	if metadata.PollErrorCount > 0 {
		updatedMetadata, err := t.updatePollingMetadata(ctx.Metadata, config, func(current *OnPipelineCompletedMetadata) {
			current.PollErrorCount = 0
		})
		if err != nil {
			return err
		}
		metadata = updatedMetadata
	}

	for _, execution := range executions {
		err := t.processPolledExecution(ctx, config, execution)
		if errors.Is(err, errOnPipelineCompletedDeferredForRaceWindow) {
			break
		}
		if err != nil {
			return err
		}
	}

	return scheduleOnPipelineCompletedPoll(ctx.Requests)
}

func (t *OnPipelineCompleted) collectExecutionsSinceCheckpoint(
	client *Client,
	metadata OnPipelineCompletedMetadata,
	pipelineIdentifier string,
) ([]ExecutionSummary, error) {
	executions := make([]ExecutionSummary, 0, OnPipelineCompletedPollPageSize)

	for page := 0; page < OnPipelineCompletedPollMaxPages; page++ {
		pageExecutions, err := client.ListExecutionSummariesPage(page, OnPipelineCompletedPollPageSize, pipelineIdentifier)
		if err != nil {
			if strings.TrimSpace(pipelineIdentifier) != "" && IsExecutionSummaryPipelineIdentifierFilterUnsupported(err) {
				return nil, fmt.Errorf("%w: %v", errOnPipelineCompletedUnsupportedPipelineIdentifierFilter, err)
			}
			return nil, err
		}

		if len(pageExecutions) == 0 {
			break
		}

		// Harness responses are expected newest-first, but sort defensively.
		slices.SortFunc(pageExecutions, func(a, b ExecutionSummary) int {
			return -compareExecutionOrdering(a, b)
		})

		for _, execution := range pageExecutions {
			if !isNewerExecution(metadata, execution) {
				executionID := strings.TrimSpace(execution.ExecutionID)
				executionEnded := executionOrderingTimestamp(execution)
				// Allow checkpoint timestamp refresh for executions already
				// checkpointed by ID even when the "newer" predicate returns false.
				if executionEnded > metadata.LastExecutionEnded &&
					executionID == strings.TrimSpace(metadata.LastExecutionID) {
					executions = append(executions, execution)
				}
				slices.SortFunc(executions, compareExecutionOrdering)
				return executions, nil
			}
			executions = append(executions, execution)
		}

		if len(pageExecutions) < OnPipelineCompletedPollPageSize {
			break
		}
	}

	// Process oldest first to preserve event ordering.
	slices.SortFunc(executions, compareExecutionOrdering)

	return executions, nil
}

func (t *OnPipelineCompleted) processPolledExecution(
	ctx core.TriggerActionContext,
	config OnPipelineCompletedConfiguration,
	execution ExecutionSummary,
) error {
	checkpointMu := onPipelineCompletedCheckpointMutex(config)
	checkpointMu.Lock()
	defer checkpointMu.Unlock()

	metadata, err := decodeOnPipelineCompletedMetadata(ctx.Metadata.Get())
	if err != nil {
		return err
	}
	if !isNewerExecution(metadata, execution) {
		// Same execution may first arrive without timestamps (webhook), then
		// later via polling with endTs. Refresh checkpoint timestamp dimension
		// without re-emitting the event.
		executionID := strings.TrimSpace(execution.ExecutionID)
		if executionID == strings.TrimSpace(metadata.LastExecutionID) ||
			metadataHasTimestamplessExecutionID(metadata, executionID) {
			return t.updateCheckpointFromExecutionUnlocked(ctx.Metadata, execution)
		}
		return nil
	}

	status := canonicalStatus(execution.Status)
	isTerminal := isCanonicalTerminalStatus(status)
	shouldEmit := isTerminal
	if metadataHasTimestamplessExecutionID(metadata, execution.ExecutionID) {
		// Same execution was already emitted from a timestampless webhook.
		// Poll can refresh checkpoint timestamp, but must not emit again.
		shouldEmit = false
	}

	// Never advance checkpoints for non-terminal executions.
	// Their ordering can be based on start time and may jump ahead of
	// unrelated executions that have not completed yet.
	if !isTerminal {
		return nil
	}

	pipelineScoped := strings.TrimSpace(config.PipelineIdentifier) != ""
	if pipelineScoped && strings.TrimSpace(execution.PipelineIdentifier) != config.PipelineIdentifier {
		shouldEmit = false
	}
	if len(config.Statuses) > 0 && !slices.Contains(config.Statuses, status) {
		shouldEmit = false
	}

	// For pipeline-scoped mode, avoid polling terminal executions that just
	// finished. Webhook delivery should win for near-real-time events.
	if shouldEmit && pipelineScoped && isWithinPollRaceWindow(execution) {
		return errOnPipelineCompletedDeferredForRaceWindow
	}

	if shouldEmit {
		emittedPayload := map[string]any{
			"executionId":        execution.ExecutionID,
			"pipelineIdentifier": execution.PipelineIdentifier,
			"status":             status,
			"eventType":          "PipelineEnd",
			"raw":                map[string]any{},
		}

		if err := ctx.Events.Emit(OnPipelineCompletedPayloadType, emittedPayload); err != nil {
			return fmt.Errorf("failed to emit polled event: %w", err)
		}
	}

	return t.updateCheckpointFromExecutionUnlocked(ctx.Metadata, execution)
}

func (t *OnPipelineCompleted) updateCheckpointFromExecutionUnlocked(
	metadataCtx core.MetadataContext,
	execution ExecutionSummary,
) error {
	if metadataCtx == nil {
		return nil
	}

	metadata, err := decodeOnPipelineCompletedMetadata(metadataCtx.Get())
	if err != nil {
		return err
	}

	updated := updateCheckpoint(metadata, execution)
	if updated.LastExecutionEnded == metadata.LastExecutionEnded &&
		strings.TrimSpace(updated.LastExecutionID) == strings.TrimSpace(metadata.LastExecutionID) &&
		strings.TrimSpace(updated.LastTimestamplessExecutionIDs) == strings.TrimSpace(metadata.LastTimestamplessExecutionIDs) &&
		strings.TrimSpace(updated.LastTimestamplessExecutionID) == strings.TrimSpace(metadata.LastTimestamplessExecutionID) {
		return nil
	}

	return metadataCtx.Set(updated)
}

func (t *OnPipelineCompleted) updatePollingMetadata(
	metadataCtx core.MetadataContext,
	config OnPipelineCompletedConfiguration,
	mutate func(*OnPipelineCompletedMetadata),
) (OnPipelineCompletedMetadata, error) {
	if metadataCtx == nil {
		return OnPipelineCompletedMetadata{}, fmt.Errorf("missing metadata context")
	}

	checkpointMu := onPipelineCompletedCheckpointMutex(config)
	checkpointMu.Lock()
	defer checkpointMu.Unlock()

	current, err := decodeOnPipelineCompletedMetadata(metadataCtx.Get())
	if err != nil {
		return OnPipelineCompletedMetadata{}, err
	}

	updated := current
	if mutate != nil {
		mutate(&updated)
	}

	if updated == current {
		return updated, nil
	}

	if err := metadataCtx.Set(updated); err != nil {
		return OnPipelineCompletedMetadata{}, err
	}

	return updated, nil
}

func onPipelineCompletedCheckpointMutex(config OnPipelineCompletedConfiguration) *sync.Mutex {
	lockKey := onPipelineCompletedCheckpointLockKey(config)
	shard := onPipelineCompletedCheckpointLockShardIndex(lockKey)
	return &onPipelineCompletedCheckpointLocks[shard]
}

func onPipelineCompletedCheckpointLockKey(config OnPipelineCompletedConfiguration) string {
	statuses := make([]string, 0, len(config.Statuses))
	for _, status := range config.Statuses {
		normalized := normalizeStatus(status)
		if normalized == "" {
			continue
		}
		statuses = append(statuses, normalized)
	}

	slices.Sort(statuses)
	statuses = slices.Compact(statuses)

	return strings.Join([]string{
		strings.TrimSpace(config.OrgID),
		strings.TrimSpace(config.ProjectID),
		strings.TrimSpace(config.PipelineIdentifier),
		strings.Join(statuses, ","),
	}, "|")
}

func onPipelineCompletedCheckpointLockShardIndex(lockKey string) int {
	if lockKey == "" {
		return 0
	}

	hash := uint64(14695981039346656037)
	for i := 0; i < len(lockKey); i++ {
		hash ^= uint64(lockKey[i])
		hash *= 1099511628211
	}

	return int(hash % uint64(OnPipelineCompletedCheckpointLockShardCount))
}

func executionSummaryFromWebhookPayload(event pipelineWebhookEvent, payload map[string]any) ExecutionSummary {
	return ExecutionSummary{
		ExecutionID:        event.ExecutionID,
		PipelineIdentifier: event.PipelineIdentifier,
		Status:             event.Status,
		PlanExecutionURL: firstNonEmpty(
			findStringByPaths(payload,
				[]string{"eventData", "executionUrl"},
				[]string{"eventData", "planExecutionUrl"},
				[]string{"data", "executionUrl"},
				[]string{"data", "planExecutionUrl"},
				[]string{"executionUrl"},
				[]string{"planExecutionUrl"},
			),
		),
		StartedAt: firstNonEmpty(
			findStringByPaths(payload,
				[]string{"eventData", "startTs"},
				[]string{"data", "startTs"},
				[]string{"startTs"},
			),
			findStringByPaths(payload,
				[]string{"eventData", "startTime"},
				[]string{"data", "startTime"},
				[]string{"startTime"},
				[]string{"timestamp"},
			),
		),
		EndedAt: firstNonEmpty(
			findStringByPaths(payload,
				[]string{"eventData", "endTs"},
				[]string{"data", "endTs"},
				[]string{"endTs"},
			),
			findStringByPaths(payload,
				[]string{"eventData", "endedAt"},
				[]string{"data", "endedAt"},
				[]string{"endedAt"},
			),
		),
	}
}

func updateCheckpoint(metadata OnPipelineCompletedMetadata, execution ExecutionSummary) OnPipelineCompletedMetadata {
	executionID := strings.TrimSpace(execution.ExecutionID)
	if executionID == "" {
		return metadata
	}
	currentExecutionID := strings.TrimSpace(metadata.LastExecutionID)

	ended := executionOrderingTimestamp(execution)
	if ended == 0 {
		// Keep timestamped checkpoint dimensions untouched.
		// Store timestampless execution IDs only for dedupe.
		return metadataWithTimestamplessExecutionID(metadata, executionID)
	}

	if executionID == currentExecutionID && ended > metadata.LastExecutionEnded {
		metadata.LastExecutionEnded = ended
		return metadata
	}

	if ended > metadata.LastExecutionEnded {
		metadata.LastExecutionEnded = ended
		metadata.LastExecutionID = executionID
		return metadata
	}

	if ended == metadata.LastExecutionEnded &&
		strings.Compare(executionID, currentExecutionID) > 0 {
		metadata.LastExecutionID = executionID
	}

	return metadata
}

func isNewerExecution(metadata OnPipelineCompletedMetadata, execution ExecutionSummary) bool {
	executionID := strings.TrimSpace(execution.ExecutionID)
	if executionID == "" {
		return false
	}
	currentExecutionID := strings.TrimSpace(metadata.LastExecutionID)
	if executionID == currentExecutionID {
		return false
	}

	ended := executionOrderingTimestamp(execution)
	if ended == 0 {
		return !metadataHasTimestamplessExecutionID(metadata, executionID)
	}

	if metadataHasTimestamplessExecutionID(metadata, executionID) {
		// Same execution first seen without timestamp should be revisited once
		// when a timestamp becomes available to refresh checkpoint dimensions.
		// Emission dedupe is handled in processPolledExecution.
		return ended > metadata.LastExecutionEnded
	}

	if ended > metadata.LastExecutionEnded {
		return true
	}

	if ended < metadata.LastExecutionEnded {
		return false
	}

	return strings.Compare(executionID, currentExecutionID) > 0
}

func executionOrderingTimestamp(execution ExecutionSummary) int64 {
	ended := parseEpochMilliseconds(execution.EndedAt)
	if ended > 0 {
		return ended
	}

	started := parseEpochMilliseconds(execution.StartedAt)
	if started > 0 {
		return started
	}

	return 0
}

func parseEpochMilliseconds(value string) int64 {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}

	number, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		for _, layout := range parseEpochMillisecondsLayouts {
			if parsedTime, parseErr := time.Parse(layout, trimmed); parseErr == nil {
				return parsedTime.UnixMilli()
			}
		}
		return 0
	}

	// Harness webhook payloads may send seconds while execution summaries commonly use milliseconds.
	if number > 0 && number < 1_000_000_000_000 {
		return number * 1000
	}

	return number
}

func (t *OnPipelineCompleted) Cleanup(ctx core.TriggerContext) error {
	return nil
}

func decodeOnPipelineCompletedConfiguration(value any) (OnPipelineCompletedConfiguration, error) {
	config := OnPipelineCompletedConfiguration{}
	if err := mapstructure.Decode(value, &config); err != nil {
		return OnPipelineCompletedConfiguration{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	config.OrgID = strings.TrimSpace(config.OrgID)
	if config.OrgID == "" {
		return OnPipelineCompletedConfiguration{}, fmt.Errorf("orgId is required")
	}

	config.ProjectID = strings.TrimSpace(config.ProjectID)
	if config.ProjectID == "" {
		return OnPipelineCompletedConfiguration{}, fmt.Errorf("projectId is required")
	}

	config.PipelineIdentifier = strings.TrimSpace(config.PipelineIdentifier)
	config.Statuses = normalizeSelectedStatuses(config.Statuses)
	if len(config.Statuses) == 0 {
		config.Statuses = []string{"succeeded", "failed"}
	}

	return config, nil
}

func normalizeSelectedStatuses(statuses []string) []string {
	selected := make([]string, 0, len(statuses))
	for _, status := range statuses {
		normalized := normalizeStatus(status)
		if normalized == "" {
			continue
		}

		if !slices.Contains(onPipelineCompletedAllowedStatuses, normalized) {
			continue
		}

		if slices.Contains(selected, normalized) {
			continue
		}

		selected = append(selected, normalized)
	}

	return selected
}

func statusSelected(selected []string, currentStatus string) bool {
	normalized := canonicalStatus(currentStatus)
	if normalized == "" {
		return false
	}

	return slices.Contains(selected, normalized)
}

func authorizeWebhook(ctx core.WebhookRequestContext) error {
	secret, err := readWebhookSecret(ctx)
	if err != nil {
		return err
	}

	candidateTokens := []string{}
	authorizationHeader := strings.TrimSpace(ctx.Headers.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(authorizationHeader), "bearer ") {
		candidateTokens = append(candidateTokens, strings.TrimSpace(authorizationHeader[7:]))
	}

	candidateTokens = append(candidateTokens,
		strings.TrimSpace(ctx.Headers.Get("X-Harness-Webhook-Token")),
		strings.TrimSpace(ctx.Headers.Get("X-Api-Key")),
	)

	for _, token := range candidateTokens {
		if token == "" {
			continue
		}

		if subtle.ConstantTimeCompare([]byte(token), []byte(secret)) == 1 {
			return nil
		}
	}

	return fmt.Errorf("invalid webhook authorization")
}

func readWebhookSecret(ctx core.WebhookRequestContext) (string, error) {
	if ctx.Webhook == nil {
		return "", fmt.Errorf("webhook context is required")
	}

	secretBytes, err := ctx.Webhook.GetSecret()
	if err != nil {
		return "", fmt.Errorf("failed to read webhook secret: %w", err)
	}

	secret := strings.TrimSpace(string(secretBytes))
	if secret == "" {
		return "", fmt.Errorf("webhook secret is not configured")
	}

	return secret, nil
}

func metadataTimestamplessExecutionIDs(metadata OnPipelineCompletedMetadata) []string {
	ids := make([]string, 0, OnPipelineCompletedMaxTimestamplessIDs)

	appendID := func(value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		if slices.Contains(ids, trimmed) {
			return
		}
		ids = append(ids, trimmed)
	}

	for _, token := range strings.Split(strings.TrimSpace(metadata.LastTimestamplessExecutionIDs), ",") {
		appendID(token)
	}
	appendID(metadata.LastTimestamplessExecutionID)

	if len(ids) > OnPipelineCompletedMaxTimestamplessIDs {
		ids = ids[len(ids)-OnPipelineCompletedMaxTimestamplessIDs:]
	}

	return ids
}

func metadataHasTimestamplessExecutionID(metadata OnPipelineCompletedMetadata, executionID string) bool {
	executionID = strings.TrimSpace(executionID)
	if executionID == "" {
		return false
	}

	ids := metadataTimestamplessExecutionIDs(metadata)
	return slices.Contains(ids, executionID)
}

func metadataWithTimestamplessExecutionID(
	metadata OnPipelineCompletedMetadata,
	executionID string,
) OnPipelineCompletedMetadata {
	executionID = strings.TrimSpace(executionID)
	if executionID == "" {
		return metadata
	}

	ids := metadataTimestamplessExecutionIDs(metadata)
	if !slices.Contains(ids, executionID) {
		ids = append(ids, executionID)
		if len(ids) > OnPipelineCompletedMaxTimestamplessIDs {
			ids = ids[len(ids)-OnPipelineCompletedMaxTimestamplessIDs:]
		}
	}

	metadata.LastTimestamplessExecutionID = executionID
	metadata.LastTimestamplessExecutionIDs = strings.Join(ids, ",")

	return metadata
}

func decodeOnPipelineCompletedMetadata(value any) (OnPipelineCompletedMetadata, error) {
	metadata := OnPipelineCompletedMetadata{}
	if err := mapstructure.Decode(value, &metadata); err != nil {
		return OnPipelineCompletedMetadata{}, fmt.Errorf("failed to decode metadata: %w", err)
	}

	ids := metadataTimestamplessExecutionIDs(metadata)
	metadata.LastTimestamplessExecutionIDs = strings.Join(ids, ",")
	if len(ids) > 0 {
		metadata.LastTimestamplessExecutionID = ids[len(ids)-1]
	}

	return metadata, nil
}

func compareExecutionOrdering(a, b ExecutionSummary) int {
	aTimestamp := executionOrderingTimestamp(a)
	bTimestamp := executionOrderingTimestamp(b)
	switch {
	case aTimestamp > bTimestamp:
		return 1
	case aTimestamp < bTimestamp:
		return -1
	default:
		return strings.Compare(strings.TrimSpace(a.ExecutionID), strings.TrimSpace(b.ExecutionID))
	}
}

func isWithinPollRaceWindow(execution ExecutionSummary) bool {
	ended := executionOrderingTimestamp(execution)
	if ended <= 0 {
		return false
	}

	now := time.Now().UnixMilli()
	threshold := int64(OnPipelineCompletedPollRaceWindow / time.Millisecond)
	return now-ended < threshold
}

func scheduleOnPipelineCompletedPoll(requests core.RequestContext) error {
	if requests == nil {
		return nil
	}

	return requests.ScheduleActionCall(
		OnPipelineCompletedPollAction,
		map[string]any{},
		OnPipelineCompletedPollInterval,
	)
}
