package bitbucket

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	WaitForBuildsName = "bitbucket.waitForBuilds"

	waitBuildsEvaluateHook = "evaluate"
	waitBuildsRefKV        = "waitBuildsRef"

	waitBuildsPayloadType     = "bitbucket.builds"
	waitBuildsPassedChannel   = "passed"
	waitBuildsFailedChannel   = "failed"
	waitBuildsTimedOutChannel = "timedOut"

	waitBuildsDefaultTimeoutSeconds = 3600
	waitBuildsPollInterval          = 5 * time.Minute
	waitBuildsWebhookDelay          = time.Second

	waitBuildsStatusCreated = "repo:commit_status_created"
	waitBuildsStatusUpdated = "repo:commit_status_updated"

	waitBuildsOutcomePassed   = "passed"
	waitBuildsOutcomeFailed   = "failed"
	waitBuildsOutcomeTimedOut = "timedOut"
	waitBuildsOutcomePending  = "pending"

	waitBuildsStatusPending   = "pending"
	waitBuildsStatusCompleted = "completed"
)

// waitBuildsFailingStates are terminal states that fail a required build.
// STOPPED never produced a passing signal, so it fails instead of passing.
// Unknown and empty states stay pending until the timeout.
var waitBuildsFailingStates = map[string]bool{
	CommitStatusFailed:  true,
	CommitStatusStopped: true,
}

type WaitForBuilds struct{}

type WaitForBuildsConfiguration struct {
	Repository     string   `json:"repository" mapstructure:"repository"`
	Ref            string   `json:"ref" mapstructure:"ref"`
	BuildKeys      []string `json:"buildKeys" mapstructure:"buildKeys"`
	TimeoutSeconds *int     `json:"timeoutSeconds" mapstructure:"timeoutSeconds"`
}

type WaitForBuildsMetadata struct {
	Repository     string        `json:"repository" mapstructure:"repository"`
	SHA            string        `json:"sha" mapstructure:"sha"`
	StartedAt      time.Time     `json:"startedAt" mapstructure:"startedAt"`
	LastChangeAt   time.Time     `json:"lastChangeAt" mapstructure:"lastChangeAt"`
	TimeoutAt      time.Time     `json:"timeoutAt" mapstructure:"timeoutAt"`
	CompletedAt    *time.Time    `json:"completedAt,omitempty" mapstructure:"completedAt,omitempty"`
	Fingerprint    string        `json:"fingerprint" mapstructure:"fingerprint"`
	Outcome        string        `json:"outcome" mapstructure:"outcome"`
	Builds         []BuildStatus `json:"builds" mapstructure:"builds"`
	SelectedBuilds []BuildStatus `json:"selectedBuilds" mapstructure:"selectedBuilds"`
	FailedBuilds   []BuildStatus `json:"failedBuilds" mapstructure:"failedBuilds"`
}

type WaitForBuildsOutput struct {
	Repository     string        `json:"repository"`
	SHA            string        `json:"sha"`
	Builds         []BuildStatus `json:"builds"`
	SelectedBuilds []BuildStatus `json:"selectedBuilds"`
	FailedBuilds   []BuildStatus `json:"failedBuilds"`
	StartedAt      time.Time     `json:"startedAt"`
	CompletedAt    time.Time     `json:"completedAt"`
}

// BuildStatus is one observed Bitbucket build, normalized like a check.
type BuildStatus struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Conclusion  string `json:"conclusion,omitempty"`
	Description string `json:"description,omitempty"`
	DetailsURL  string `json:"detailsUrl,omitempty"`
}

type waitBuildsEvaluation struct {
	Outcome         string
	Fingerprint     string
	Builds          []BuildStatus
	SelectedBuilds  []BuildStatus
	FailedBuilds    []BuildStatus
	MissingSelected []string
}

func (c *WaitForBuilds) Name() string {
	return WaitForBuildsName
}

func (c *WaitForBuilds) Label() string {
	return "Wait for Builds"
}

func (c *WaitForBuilds) Description() string {
	return "Wait for Bitbucket builds to finish on a commit"
}

func (c *WaitForBuilds) Documentation() string {
	return `The Wait for Builds action waits until required Bitbucket builds finish on a commit.

## Use Cases

- **Required builds**: Block a workflow until selected builds pass
- **CI gating**: Continue only when external CI reports success

## Configuration

- **Repository**: Select the Bitbucket repository to monitor
- **Ref**: Full commit SHA to watch. Use the pull request head SHA.
- **Build Keys**: Exact build keys to require. A key is the commit status key from Bitbucket.
- **Timeout Seconds**: Maximum seconds to wait for builds.

## Output Channels

- **Passed**: No selected build failed
- **Failed**: A selected build failed or stopped
- **Timed Out**: The timeout expired, or a selected build never appeared

Each output includes the repository, SHA, all observed builds, selected builds, failed builds, and timestamps.

## Notes

- The action listens for commit status webhooks
- A five-minute poll runs only when a webhook does not arrive
- Missing builds stay pending until the timeout`
}

func (c *WaitForBuilds) Icon() string {
	return "bitbucket"
}

func (c *WaitForBuilds) Color() string {
	return "blue"
}

func (c *WaitForBuilds) ExampleOutput() map[string]any {
	return map[string]any{
		"type":      waitBuildsPayloadType,
		"timestamp": "2026-04-22T10:05:00Z",
		"data": map[string]any{
			"repository": "acme/widgets",
			"sha":        "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			"builds": []any{
				map[string]any{"key": "build-a", "name": "Build A", "status": "completed", "conclusion": "success"},
			},
			"selectedBuilds": []any{
				map[string]any{"key": "build-a", "name": "Build A", "status": "completed", "conclusion": "success"},
			},
			"failedBuilds": []any{},
		},
	}
}

func (c *WaitForBuilds) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: waitBuildsPassedChannel, Label: "Passed"},
		{Name: waitBuildsFailedChannel, Label: "Failed"},
		{Name: waitBuildsTimedOutChannel, Label: "Timed out"},
	}
}

func (c *WaitForBuilds) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "ref",
			Label:       "Ref",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "Full commit SHA",
			Description: "Full commit SHA to watch. Use the pull request head SHA.",
		},
		{
			Name:        "buildKeys",
			Label:       "Build Keys",
			Type:        configuration.FieldTypeList,
			Required:    true,
			Description: "Exact build keys to require.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Build key",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "timeoutSeconds",
			Label:       "Timeout Seconds",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     waitBuildsDefaultTimeoutSeconds,
			Description: "Maximum seconds to wait for builds.",
		},
	}
}

func (c *WaitForBuilds) Setup(ctx core.SetupContext) error {
	config, err := decodeWaitBuildsConfig(ctx.Configuration)
	if err != nil {
		return err
	}

	repo, err := ensureRepoInMetadata(ctx.HTTP, ctx.Metadata, ctx.Integration, config.Repository)
	if err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(WebhookConfiguration{
		EventTypes:     []string{waitBuildsStatusCreated, waitBuildsStatusUpdated},
		RepositorySlug: repo.Slug,
	})
}

func (c *WaitForBuilds) Execute(ctx core.ExecutionContext) error {
	config, err := decodeWaitBuildsConfig(ctx.Configuration)
	if err != nil {
		return err
	}

	now := time.Now()
	metadata := WaitForBuildsMetadata{
		Repository:   config.Repository,
		SHA:          config.Ref,
		StartedAt:    now,
		LastChangeAt: now,
		TimeoutAt:    now.Add(config.timeout()),
	}
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	return evaluateWaitForBuilds(waitBuildsRuntime{
		Configuration:  config,
		HTTP:           ctx.HTTP,
		Metadata:       ctx.Metadata,
		ExecutionState: ctx.ExecutionState,
		Requests:       ctx.Requests,
		Integration:    ctx.Integration,
		Logger:         ctx.Logger,
	}, now)
}

func (c *WaitForBuilds) Hooks() []core.Hook {
	return []core.Hook{
		{
			Name: waitBuildsEvaluateHook,
			Type: core.HookTypeInternal,
		},
	}
}

func waitBuildsStopped(state core.ExecutionStateContext) bool {
	return state.IsFinished() || state.IsCancelling()
}

func (c *WaitForBuilds) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name != waitBuildsEvaluateHook {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
	if waitBuildsStopped(ctx.ExecutionState) {
		return nil
	}

	config, err := decodeWaitBuildsConfig(ctx.Configuration)
	if err != nil {
		return err
	}

	return evaluateWaitForBuilds(waitBuildsRuntime{
		Configuration:  config,
		HTTP:           ctx.HTTP,
		Metadata:       ctx.Metadata,
		ExecutionState: ctx.ExecutionState,
		Requests:       ctx.Requests,
		Integration:    ctx.Integration,
		Logger:         ctx.Logger,
	}, time.Now())
}

func (c *WaitForBuilds) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	if code, err := verifyBitbucketSignature(ctx); err != nil {
		return code, nil, err
	}

	eventKey := ctx.Headers.Get("X-Event-Key")
	if eventKey == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("missing X-Event-Key header")
	}
	if eventKey != waitBuildsStatusCreated && eventKey != waitBuildsStatusUpdated {
		return http.StatusOK, nil, nil
	}

	config, err := decodeWaitBuildsConfig(ctx.Configuration)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	var payload map[string]any
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %v", err)
	}

	repository, sha := waitBuildsRefFromPayload(payload)
	if repository == "" || sha == "" {
		return http.StatusOK, nil, nil
	}
	if !strings.EqualFold(strings.TrimSpace(repository), strings.TrimSpace(config.Repository)) {
		return http.StatusOK, nil, nil
	}

	if ctx.FindActiveExecutionByKV != nil {
		executionCtx, err := ctx.FindActiveExecutionByKV(waitBuildsRefKV, waitBuildsRefValue(config.Repository, sha))
		if err != nil || executionCtx == nil {
			return http.StatusOK, nil, nil
		}
		if waitBuildsStopped(executionCtx.ExecutionState) {
			return http.StatusOK, nil, nil
		}
		if err := executionCtx.Requests.ScheduleActionCall(waitBuildsEvaluateHook, map[string]any{}, waitBuildsWebhookDelay); err != nil {
			return http.StatusInternalServerError, nil, err
		}
		return http.StatusOK, nil, nil
	}

	if ctx.FindExecutionByKV == nil {
		return http.StatusOK, nil, nil
	}

	executionCtx, err := ctx.FindExecutionByKV(waitBuildsRefKV, waitBuildsRefValue(config.Repository, sha))
	if err != nil || executionCtx == nil {
		return http.StatusOK, nil, nil
	}
	if waitBuildsStopped(executionCtx.ExecutionState) {
		return http.StatusOK, nil, nil
	}

	if err := executionCtx.Requests.ScheduleActionCall(waitBuildsEvaluateHook, map[string]any{}, waitBuildsWebhookDelay); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return http.StatusOK, nil, nil
}

func (c *WaitForBuilds) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *WaitForBuilds) Cleanup(ctx core.SetupContext) error {
	return nil
}

type waitBuildsRuntime struct {
	Configuration  WaitForBuildsConfiguration
	HTTP           core.HTTPContext
	Metadata       core.MetadataWriter
	ExecutionState core.ExecutionStateContext
	Requests       core.RequestContext
	Integration    core.IntegrationContext
	Logger         *log.Entry
}

func evaluateWaitForBuilds(ctx waitBuildsRuntime, now time.Time) error {
	if waitBuildsStopped(ctx.ExecutionState) {
		return nil
	}

	metadata, err := decodeWaitBuildsMetadata(ctx.Metadata.Get())
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	var integrationMetadata Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &integrationMetadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	client, err := NewClient(integrationMetadata.AuthType, ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize Bitbucket client: %w", err)
	}

	statuses, err := client.ListCommitStatuses(ctx.Configuration.Repository, ctx.Configuration.Ref)
	if err != nil {
		return err
	}

	builds := normalizeBuildStatuses(statuses)
	sha := strings.TrimSpace(ctx.Configuration.Ref)
	timedOut := !now.Before(metadata.TimeoutAt)
	evaluation := evaluateBuildStatuses(builds, ctx.Configuration.BuildKeys, timedOut)

	if metadata.Fingerprint != "" && metadata.Fingerprint != evaluation.Fingerprint {
		metadata.LastChangeAt = now
	}
	if metadata.Fingerprint == "" {
		metadata.LastChangeAt = now
	}

	metadata.Repository = ctx.Configuration.Repository
	metadata.SHA = sha
	metadata.Fingerprint = evaluation.Fingerprint
	metadata.Outcome = evaluation.Outcome
	metadata.Builds = evaluation.Builds
	metadata.SelectedBuilds = evaluation.SelectedBuilds
	metadata.FailedBuilds = evaluation.FailedBuilds

	if err := ctx.ExecutionState.SetKV(waitBuildsRefKV, waitBuildsRefValue(ctx.Configuration.Repository, sha)); err != nil {
		return err
	}

	delay := nextWaitBuildsDelay(now, metadata.TimeoutAt, waitBuildsPollInterval)
	if delay > 0 && evaluation.Outcome == waitBuildsOutcomePending {
		if err := ctx.Metadata.Set(metadata); err != nil {
			return err
		}
		return ctx.Requests.ScheduleActionCall(waitBuildsEvaluateHook, map[string]any{}, delay)
	}

	completedAt := now
	metadata.CompletedAt = &completedAt
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	channel := waitBuildsPassedChannel
	switch evaluation.Outcome {
	case waitBuildsOutcomeFailed:
		channel = waitBuildsFailedChannel
	case waitBuildsOutcomeTimedOut:
		channel = waitBuildsTimedOutChannel
	}

	return ctx.ExecutionState.Emit(channel, waitBuildsPayloadType, []any{
		WaitForBuildsOutput{
			Repository:     metadata.Repository,
			SHA:            metadata.SHA,
			Builds:         metadata.Builds,
			SelectedBuilds: metadata.SelectedBuilds,
			FailedBuilds:   metadata.FailedBuilds,
			StartedAt:      metadata.StartedAt,
			CompletedAt:    completedAt,
		},
	})
}

// normalizeBuildStatuses projects commit statuses into check-like builds.
// Unknown and empty states stay pending until the timeout.
func normalizeBuildStatuses(statuses []CommitStatus) []BuildStatus {
	latest := map[string]BuildStatus{}
	for _, status := range statuses {
		key := strings.TrimSpace(status.Key)
		if key == "" {
			key = strings.TrimSpace(status.Name)
		}
		if key == "" {
			continue
		}
		state := strings.ToUpper(strings.TrimSpace(status.State))
		build := BuildStatus{
			Key:         key,
			Name:        strings.TrimSpace(status.Name),
			Status:      waitBuildsStatusPending,
			Description: strings.TrimSpace(status.Description),
			DetailsURL:  strings.TrimSpace(status.URL),
		}
		switch state {
		case CommitStatusSuccessful:
			build.Status = waitBuildsStatusCompleted
			build.Conclusion = "success"
		case CommitStatusFailed, CommitStatusStopped:
			build.Status = waitBuildsStatusCompleted
			build.Conclusion = strings.ToLower(state)
		}
		latest[strings.ToLower(key)] = build
	}

	builds := make([]BuildStatus, 0, len(latest))
	for _, build := range latest {
		builds = append(builds, build)
	}
	sort.Slice(builds, func(i, j int) bool {
		return builds[i].Key < builds[j].Key
	})
	return builds
}

func evaluateBuildStatuses(builds []BuildStatus, selectedKeys []string, timedOut bool) waitBuildsEvaluation {
	selected := selectedBuilds(builds, selectedKeys)
	missing := missingSelectedBuilds(builds, selectedKeys)
	selected = append(selected, pendingSelectedBuilds(missing)...)
	failed := failedBuilds(selected)

	evaluation := waitBuildsEvaluation{
		Builds:          builds,
		SelectedBuilds:  selected,
		FailedBuilds:    failed,
		MissingSelected: missing,
		Fingerprint:     buildFingerprint(builds),
	}

	if hasPendingBuild(selected) || len(missing) > 0 {
		if timedOut {
			evaluation.Outcome = waitBuildsOutcomeTimedOut
			return evaluation
		}
		evaluation.Outcome = waitBuildsOutcomePending
		return evaluation
	}

	if timedOut {
		evaluation.Outcome = waitBuildsOutcomeTimedOut
		return evaluation
	}
	if len(failed) > 0 {
		evaluation.Outcome = waitBuildsOutcomeFailed
		return evaluation
	}
	evaluation.Outcome = waitBuildsOutcomePassed
	return evaluation
}

func selectedBuilds(builds []BuildStatus, selectedKeys []string) []BuildStatus {
	wanted := map[string]bool{}
	for _, key := range selectedKeys {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		wanted[strings.ToLower(trimmed)] = true
	}
	if len(wanted) == 0 {
		return nil
	}

	selected := make([]BuildStatus, 0, len(builds))
	for _, build := range builds {
		if wanted[strings.ToLower(build.Key)] {
			selected = append(selected, build)
		}
	}
	return selected
}

func missingSelectedBuilds(builds []BuildStatus, selectedKeys []string) []string {
	if len(selectedKeys) == 0 {
		return nil
	}

	seen := map[string]bool{}
	for _, build := range builds {
		seen[strings.ToLower(build.Key)] = true
	}

	var missing []string
	for _, key := range selectedKeys {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		if !seen[strings.ToLower(trimmed)] {
			missing = append(missing, trimmed)
		}
	}
	return missing
}

func pendingSelectedBuilds(missing []string) []BuildStatus {
	if len(missing) == 0 {
		return nil
	}
	pending := make([]BuildStatus, 0, len(missing))
	for _, key := range missing {
		pending = append(pending, BuildStatus{
			Key:    key,
			Name:   key,
			Status: waitBuildsStatusPending,
		})
	}
	return pending
}

func failedBuilds(builds []BuildStatus) []BuildStatus {
	var failed []BuildStatus
	for _, build := range builds {
		if build.Status != waitBuildsStatusCompleted {
			continue
		}
		if waitBuildsFailingStates[strings.ToUpper(build.Conclusion)] {
			failed = append(failed, build)
		}
	}
	return failed
}

func hasPendingBuild(builds []BuildStatus) bool {
	for _, build := range builds {
		if build.Status != waitBuildsStatusCompleted {
			return true
		}
	}
	return false
}

// NormalizeBuildStatuses exposes build normalization for factory mergeability.
func NormalizeBuildStatuses(statuses []CommitStatus) []BuildStatus {
	return normalizeBuildStatuses(statuses)
}

// BuildStatusCompleted is the terminal passing-adjacent build status.
const BuildStatusCompleted = waitBuildsStatusCompleted

// BuildFailed reports whether a normalized build failed its required signal.
func BuildFailed(build BuildStatus) bool {
	return build.Status == waitBuildsStatusCompleted && waitBuildsFailingStates[strings.ToUpper(build.Conclusion)]
}

func buildFingerprint(builds []BuildStatus) string {
	payload, err := json.Marshal(builds)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func nextWaitBuildsDelay(now, timeoutAt time.Time, pollInterval time.Duration) time.Duration {
	if !now.Before(timeoutAt) {
		return 0
	}
	timeoutRemain := timeoutAt.Sub(now)
	if pollInterval < timeoutRemain {
		return pollInterval
	}
	return timeoutRemain
}

// waitBuildsRefFromPayload extracts the repository and commit SHA from a
// commit status event. Missing identity is ignored, never matched.
func waitBuildsRefFromPayload(payload map[string]any) (string, string) {
	repository, _ := payload["repository"].(map[string]any)
	fullName, _ := repository["full_name"].(string)
	status, _ := payload["commit_status"].(map[string]any)
	commit, _ := status["commit"].(map[string]any)
	sha, _ := commit["hash"].(string)
	if fullName == "" {
		if nested, ok := status["repository"].(map[string]any); ok {
			fullName, _ = nested["full_name"].(string)
		}
	}
	return fullName, sha
}

func waitBuildsRefValue(repository, sha string) string {
	return strings.ToLower(strings.TrimSpace(repository)) + "@" + strings.ToLower(strings.TrimSpace(sha))
}

func decodeWaitBuildsConfig(raw any) (WaitForBuildsConfiguration, error) {
	var config WaitForBuildsConfiguration
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &config,
		WeaklyTypedInput: true,
		TagName:          "mapstructure",
	})
	if err != nil {
		return config, fmt.Errorf("failed to decode configuration: %w", err)
	}
	if err := decoder.Decode(raw); err != nil {
		return config, fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(config.Repository) == "" {
		return config, fmt.Errorf("repository is required")
	}
	if strings.TrimSpace(config.Ref) == "" {
		return config, fmt.Errorf("ref is required")
	}
	config.BuildKeys = normalizeWaitBuildKeys(config.BuildKeys)
	if len(config.BuildKeys) == 0 {
		return config, fmt.Errorf("buildKeys is required")
	}
	return config, nil
}

func normalizeWaitBuildKeys(keys []string) []string {
	normalized := make([]string, 0, len(keys))
	seen := map[string]bool{}
	for _, key := range keys {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if seen[lower] {
			continue
		}
		seen[lower] = true
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func decodeWaitBuildsMetadata(raw any) (WaitForBuildsMetadata, error) {
	var metadata WaitForBuildsMetadata
	if raw == nil {
		return metadata, fmt.Errorf("metadata is empty")
	}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &metadata,
		WeaklyTypedInput: true,
		DecodeHook:       decodeWaitBuildsTimeHook,
	})
	if err != nil {
		return metadata, err
	}
	if err := decoder.Decode(raw); err != nil {
		return metadata, err
	}
	return metadata, nil
}

func (c WaitForBuildsConfiguration) timeout() time.Duration {
	if c.TimeoutSeconds != nil && *c.TimeoutSeconds > 0 {
		return time.Duration(*c.TimeoutSeconds) * time.Second
	}
	return waitBuildsDefaultTimeoutSeconds * time.Second
}

func decodeWaitBuildsTimeHook(from, to reflect.Type, data any) (any, error) {
	if from == nil || to == nil {
		return data, nil
	}
	if from.Kind() != reflect.String {
		return data, nil
	}
	if to != reflect.TypeOf(time.Time{}) && to != reflect.TypeOf((*time.Time)(nil)) {
		return data, nil
	}
	text, ok := data.(string)
	if !ok || text == "" {
		return data, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, text)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, text)
	}
	if err != nil {
		return nil, err
	}
	if to == reflect.TypeOf((*time.Time)(nil)) {
		return &parsed, nil
	}
	return parsed, nil
}
