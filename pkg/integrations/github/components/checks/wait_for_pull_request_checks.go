package checks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

const (
	WaitForPullRequestChecksName        = "github.waitForPullRequestChecks"
	waitChecksEvaluateHook              = "evaluate"
	waitChecksRefKV                     = "waitChecksRef"
	waitChecksPayloadType               = "github.pullRequestChecks"
	waitChecksPassedChannel             = "passed"
	waitChecksFailedChannel             = "failed"
	waitChecksTimedOutChannel           = "timedOut"
	waitChecksDefaultQuietPeriodSeconds = 60
	waitChecksDefaultTimeoutSeconds     = 3600
	waitChecksPollInterval              = 5 * time.Minute
	waitChecksWebhookDelay              = time.Second
)

type WaitForPullRequestChecks struct{}

type WaitForPullRequestChecksConfiguration struct {
	Repository         string   `json:"repository" mapstructure:"repository"`
	Ref                string   `json:"ref" mapstructure:"ref"`
	CheckNames         []string `json:"checkNames" mapstructure:"checkNames"`
	QuietPeriodSeconds *int     `json:"quietPeriodSeconds" mapstructure:"quietPeriodSeconds"`
	TimeoutSeconds     *int     `json:"timeoutSeconds" mapstructure:"timeoutSeconds"`
}

type WaitForPullRequestChecksMetadata struct {
	Repository     string              `json:"repository" mapstructure:"repository"`
	SHA            string              `json:"sha" mapstructure:"sha"`
	StartedAt      time.Time           `json:"startedAt" mapstructure:"startedAt"`
	LastChangeAt   time.Time           `json:"lastChangeAt" mapstructure:"lastChangeAt"`
	TimeoutAt      time.Time           `json:"timeoutAt" mapstructure:"timeoutAt"`
	CompletedAt    *time.Time          `json:"completedAt,omitempty" mapstructure:"completedAt"`
	Fingerprint    string              `json:"fingerprint" mapstructure:"fingerprint"`
	Outcome        string              `json:"outcome" mapstructure:"outcome"`
	Checks         []PullRequestCheck  `json:"checks" mapstructure:"checks"`
	SelectedChecks []PullRequestCheck  `json:"selectedChecks" mapstructure:"selectedChecks"`
	FailedChecks   []PullRequestCheck  `json:"failedChecks" mapstructure:"failedChecks"`
}

type WaitForPullRequestChecksOutput struct {
	Repository     string             `json:"repository"`
	SHA            string             `json:"sha"`
	Checks         []PullRequestCheck `json:"checks"`
	SelectedChecks []PullRequestCheck `json:"selectedChecks"`
	FailedChecks   []PullRequestCheck `json:"failedChecks"`
	StartedAt      time.Time          `json:"startedAt"`
	CompletedAt    time.Time          `json:"completedAt"`
}

func (c *WaitForPullRequestChecks) Name() string {
	return WaitForPullRequestChecksName
}

func (c *WaitForPullRequestChecks) Label() string {
	return "Wait For Pull Request Checks"
}

func (c *WaitForPullRequestChecks) Description() string {
	return "Wait until pull request checks on a commit become complete"
}

func (c *WaitForPullRequestChecks) Documentation() string {
	return `The Wait For Pull Request Checks component watches GitHub Checks and Commit Statuses for one commit.

It combines both GitHub status systems into one snapshot. It then waits until the selected checks become terminal.

GitHub does not report the expected total number of checks. When you leave the check name list empty, the component waits for a quiet period after the last change. That quiet period is the completeness signal. When you specify check names, the component finishes as soon as every named check is terminal.

## Use Cases

- **Factory quality gates**: Wait for pull request checks before an agent starts a fix
- **Release gates**: Continue only when selected checks pass
- **Failure summaries**: Collect every selected failed check in one payload

## Configuration

- **Repository**: Select the GitHub repository
- **Ref**: Full commit SHA to watch
- **Check Names** *(optional)*: Exact check or status names to require. An empty list waits for all observed checks.
- **Quiet Period Seconds**: Seconds to wait after the last check change when the name list is empty. Default: 60. Named checks skip this wait.
- **Timeout Seconds**: Maximum wait time. Default: 3600.

## Output Channels

- **Passed**: No selected check failed. For an empty name list, this is after the quiet period.
- **Failed**: A selected check has a failing conclusion
- **Timed Out**: The timeout expired, or a selected check never appeared

Each output includes the repository, SHA, all observed checks, selected checks, failed checks, and timestamps.

## Notes

- The component listens for check_run, check_suite, and status webhooks
- A five-minute poll runs only when a webhook does not arrive
- Unknown non-terminal conclusions stay pending`
}

func (c *WaitForPullRequestChecks) Icon() string {
	return "github"
}

func (c *WaitForPullRequestChecks) Color() string {
	return "gray"
}

func (c *WaitForPullRequestChecks) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: waitChecksPassedChannel, Label: "Passed"},
		{Name: waitChecksFailedChannel, Label: "Failed"},
		{Name: waitChecksTimedOutChannel, Label: "Timed out"},
	}
}

func (c *WaitForPullRequestChecks) Configuration() []configuration.Field {
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
			Name:        "checkNames",
			Label:       "Check Names",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Exact check or status names to require. Leave empty to wait for all checks. GitHub does not report the expected total number of checks.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Check name",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
		},
		{
			Name:        "quietPeriodSeconds",
			Label:       "Quiet Period Seconds",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     waitChecksDefaultQuietPeriodSeconds,
			Description: "Seconds to wait after the last check change when no check names are selected. Named checks skip this wait.",
		},
		{
			Name:        "timeoutSeconds",
			Label:       "Timeout Seconds",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     waitChecksDefaultTimeoutSeconds,
			Description: "Maximum seconds to wait for checks.",
		},
	}
}

func (c *WaitForPullRequestChecks) Setup(ctx core.SetupContext) error {
	if err := common.EnsureRepoInMetadata(ctx.Metadata, ctx.Integration, ctx.HTTP, ctx.Configuration); err != nil {
		return err
	}

	config, err := decodeWaitChecksConfig(ctx.Configuration)
	if err != nil {
		return err
	}

	return ctx.Integration.RequestWebhook(common.WebhookConfiguration{
		EventTypes: []string{"check_run", "check_suite", "status"},
		Repository: config.Repository,
	})
}

func (c *WaitForPullRequestChecks) Execute(ctx core.ExecutionContext) error {
	config, err := decodeWaitChecksConfig(ctx.Configuration)
	if err != nil {
		return err
	}

	now := time.Now()
	metadata := WaitForPullRequestChecksMetadata{
		Repository:   config.Repository,
		SHA:          config.Ref,
		StartedAt:    now,
		LastChangeAt: now,
		TimeoutAt:    now.Add(config.timeout()),
	}
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	return evaluateWaitForPullRequestChecks(waitChecksRuntime{
		Configuration:  config,
		HTTP:           ctx.HTTP,
		Metadata:       ctx.Metadata,
		ExecutionState: ctx.ExecutionState,
		Requests:       ctx.Requests,
		Integration:    ctx.Integration,
	}, now)
}

func (c *WaitForPullRequestChecks) Hooks() []core.Hook {
	return []core.Hook{
		{
			Name: waitChecksEvaluateHook,
			Type: core.HookTypeInternal,
		},
	}
}

func (c *WaitForPullRequestChecks) HandleHook(ctx core.ActionHookContext) error {
	if ctx.Name != waitChecksEvaluateHook {
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	config, err := decodeWaitChecksConfig(ctx.Configuration)
	if err != nil {
		return err
	}

	return evaluateWaitForPullRequestChecks(waitChecksRuntime{
		Configuration:  config,
		HTTP:           ctx.HTTP,
		Metadata:       ctx.Metadata,
		ExecutionState: ctx.ExecutionState,
		Requests:       ctx.Requests,
		Integration:    ctx.Integration,
	}, time.Now())
}

func (c *WaitForPullRequestChecks) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	statusCode, err := common.VerifySignature(ctx)
	if err != nil {
		return statusCode, nil, err
	}

	eventType := ctx.Headers.Get("X-GitHub-Event")
	if eventType == "" {
		return http.StatusBadRequest, nil, fmt.Errorf("missing X-GitHub-Event header")
	}
	if !isWaitChecksEvent(eventType) {
		return http.StatusOK, nil, nil
	}

	config, err := decodeWaitChecksConfig(ctx.Configuration)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	var payload map[string]any
	if err := json.Unmarshal(ctx.Body, &payload); err != nil {
		return http.StatusBadRequest, nil, fmt.Errorf("error parsing request body: %v", err)
	}

	repository, sha := waitChecksRefFromPayload(payload)
	if repository == "" || sha == "" {
		return http.StatusOK, nil, nil
	}
	if !repositoryMatches(config.Repository, repository) {
		return http.StatusOK, nil, nil
	}

	if ctx.FindExecutionByKV == nil {
		return http.StatusOK, nil, nil
	}

	executionCtx, err := ctx.FindExecutionByKV(waitChecksRefKV, waitChecksRefValue(config.Repository, sha))
	if err != nil {
		return http.StatusOK, nil, nil
	}
	if executionCtx.ExecutionState.IsFinished() {
		return http.StatusOK, nil, nil
	}

	if err := executionCtx.Requests.ScheduleActionCall(waitChecksEvaluateHook, map[string]any{}, waitChecksWebhookDelay); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return http.StatusOK, nil, nil
}

func (c *WaitForPullRequestChecks) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *WaitForPullRequestChecks) Cleanup(ctx core.SetupContext) error {
	return nil
}

type waitChecksRuntime struct {
	Configuration  WaitForPullRequestChecksConfiguration
	HTTP           core.HTTPContext
	Metadata       core.MetadataWriter
	ExecutionState core.ExecutionStateContext
	Requests       core.RequestContext
	Integration    core.IntegrationContext
}

func evaluateWaitForPullRequestChecks(ctx waitChecksRuntime, now time.Time) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	metadata, err := decodeWaitChecksMetadata(ctx.Metadata.Get())
	if err != nil {
		return fmt.Errorf("failed to decode metadata: %w", err)
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	checkRuns, combined, err := fetchPullRequestCheckSources(client, ctx.Configuration.Repository, ctx.Configuration.Ref)
	if err != nil {
		return err
	}

	checks := normalizePullRequestChecks(checkRuns, combined)
	sha := resolvedWaitChecksSHA(ctx.Configuration.Ref, checkRuns, combined)
	timedOut := !now.Before(metadata.TimeoutAt)
	evaluation := evaluatePullRequestChecks(checks, ctx.Configuration.CheckNames, timedOut)

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
	metadata.Checks = evaluation.Checks
	metadata.SelectedChecks = evaluation.SelectedChecks
	metadata.FailedChecks = evaluation.FailedChecks

	if err := ctx.ExecutionState.SetKV(waitChecksRefKV, waitChecksRefValue(ctx.Configuration.Repository, sha)); err != nil {
		return err
	}

	delay := nextEvaluateDelay(
		now,
		metadata.LastChangeAt,
		metadata.TimeoutAt,
		evaluation.AllTerminal,
		ctx.Configuration.quietPeriod(),
		waitChecksPollInterval,
	)
	if delay > 0 && evaluation.Outcome == waitChecksOutcomePending {
		if err := ctx.Metadata.Set(metadata); err != nil {
			return err
		}
		return ctx.Requests.ScheduleActionCall(waitChecksEvaluateHook, map[string]any{}, delay)
	}
	if delay > 0 && evaluation.AllTerminal && evaluation.Outcome != waitChecksOutcomeTimedOut {
		if err := ctx.Metadata.Set(metadata); err != nil {
			return err
		}
		return ctx.Requests.ScheduleActionCall(waitChecksEvaluateHook, map[string]any{}, delay)
	}

	completedAt := now
	metadata.CompletedAt = &completedAt
	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	channel := waitChecksPassedChannel
	switch evaluation.Outcome {
	case waitChecksOutcomeFailed:
		channel = waitChecksFailedChannel
	case waitChecksOutcomeTimedOut:
		channel = waitChecksTimedOutChannel
	}

	return ctx.ExecutionState.Emit(channel, waitChecksPayloadType, []any{
		WaitForPullRequestChecksOutput{
			Repository:     metadata.Repository,
			SHA:            metadata.SHA,
			Checks:         metadata.Checks,
			SelectedChecks: metadata.SelectedChecks,
			FailedChecks:   metadata.FailedChecks,
			StartedAt:      metadata.StartedAt,
			CompletedAt:    completedAt,
		},
	})
}

func fetchPullRequestCheckSources(client *common.Client, repository, ref string) (*github.ListCheckRunsResults, *github.CombinedStatus, error) {
	var (
		checkRuns *github.ListCheckRunsResults
		combined  *github.CombinedStatus
		checkErr  error
		statusErr error
		wg        sync.WaitGroup
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		checkRuns, checkErr = listAllCheckRunsForRef(client, ListCheckRunsForRefConfiguration{
			Repository: repository,
			Ref:        ref,
			Filter:     "latest",
		})
	}()
	go func() {
		defer wg.Done()
		combined, statusErr = listCombinedCommitStatus(client, repository, ref)
	}()
	wg.Wait()

	if checkErr != nil {
		return nil, nil, fmt.Errorf("failed to list check runs for ref: %w", checkErr)
	}
	if statusErr != nil {
		return nil, nil, fmt.Errorf("failed to get combined commit status: %w", statusErr)
	}
	return checkRuns, combined, nil
}

func listCombinedCommitStatus(client *common.Client, repository, ref string) (*github.CombinedStatus, error) {
	opts := &github.ListOptions{PerPage: 100}
	statuses := []*github.RepoStatus{}

	var combined *github.CombinedStatus
	for {
		page, response, err := client.GetCombinedStatus(context.Background(), repository, ref, opts)
		if err != nil {
			return nil, err
		}
		if combined == nil {
			combined = page
		}
		if page != nil {
			statuses = append(statuses, page.Statuses...)
		}
		if response == nil || response.NextPage == 0 {
			break
		}
		opts.Page = response.NextPage
	}

	if combined == nil {
		combined = &github.CombinedStatus{}
	}
	combined.Statuses = statuses
	return combined, nil
}

func decodeWaitChecksConfig(raw any) (WaitForPullRequestChecksConfiguration, error) {
	var config WaitForPullRequestChecksConfiguration
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
	if config.Repository == "" {
		return config, fmt.Errorf("repository is required")
	}
	if config.Ref == "" {
		return config, fmt.Errorf("ref is required")
	}
	return config, nil
}

func (c WaitForPullRequestChecksConfiguration) quietPeriodSeconds() int {
	if c.QuietPeriodSeconds == nil {
		return waitChecksDefaultQuietPeriodSeconds
	}
	if *c.QuietPeriodSeconds < 0 {
		return waitChecksDefaultQuietPeriodSeconds
	}
	return *c.QuietPeriodSeconds
}

func (c WaitForPullRequestChecksConfiguration) timeoutSeconds() int {
	if c.TimeoutSeconds == nil {
		return waitChecksDefaultTimeoutSeconds
	}
	if *c.TimeoutSeconds <= 0 {
		return waitChecksDefaultTimeoutSeconds
	}
	return *c.TimeoutSeconds
}

func (c WaitForPullRequestChecksConfiguration) hasSelectedCheckNames() bool {
	for _, name := range c.CheckNames {
		if strings.TrimSpace(name) != "" {
			return true
		}
	}
	return false
}

func (c WaitForPullRequestChecksConfiguration) quietPeriod() time.Duration {
	if c.hasSelectedCheckNames() {
		return 0
	}
	return time.Duration(c.quietPeriodSeconds()) * time.Second
}

func (c WaitForPullRequestChecksConfiguration) timeout() time.Duration {
	return time.Duration(c.timeoutSeconds()) * time.Second
}

func decodeWaitChecksMetadata(raw any) (WaitForPullRequestChecksMetadata, error) {
	var metadata WaitForPullRequestChecksMetadata
	if raw == nil {
		return metadata, fmt.Errorf("metadata is empty")
	}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &metadata,
		WeaklyTypedInput: true,
		DecodeHook:       decodeWaitChecksTimeHook,
	})
	if err != nil {
		return metadata, err
	}
	if err := decoder.Decode(raw); err != nil {
		return metadata, err
	}
	return metadata, nil
}

func decodeWaitChecksTimeHook(from, to reflect.Type, data any) (any, error) {
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

	parsed, err := parseWaitChecksTime(text)
	if err != nil {
		return nil, err
	}
	if to == reflect.TypeOf((*time.Time)(nil)) {
		return &parsed, nil
	}
	return parsed, nil
}

func parseWaitChecksTime(value string) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed, nil
	}
	return time.Parse(time.RFC3339, value)
}

func waitChecksRefValue(repository, sha string) string {
	return repository + "@" + sha
}

func isWaitChecksEvent(eventType string) bool {
	switch eventType {
	case "check_run", "check_suite", "status":
		return true
	default:
		return false
	}
}

func waitChecksRefFromPayload(payload map[string]any) (string, string) {
	repository := ""
	if repo, ok := payload["repository"].(map[string]any); ok {
		repository = firstNonEmpty(
			stringValue(repo["name"]),
			repositoryName(stringValue(repo["full_name"])),
		)
	}

	sha := stringValue(payload["sha"])
	if checkRun, ok := payload["check_run"].(map[string]any); ok {
		sha = firstNonEmpty(stringValue(checkRun["head_sha"]), sha)
	}
	if checkSuite, ok := payload["check_suite"].(map[string]any); ok {
		sha = firstNonEmpty(sha, stringValue(checkSuite["head_sha"]))
	}
	return repository, sha
}

func repositoryMatches(configured, incoming string) bool {
	configured = strings.TrimSpace(configured)
	incoming = strings.TrimSpace(incoming)
	if configured == "" || incoming == "" {
		return false
	}
	if strings.EqualFold(configured, incoming) {
		return true
	}
	return strings.EqualFold(repositoryName(configured), repositoryName(incoming))
}

func repositoryName(value string) string {
	value = strings.TrimSpace(value)
	if i := strings.LastIndex(value, "/"); i >= 0 && i+1 < len(value) {
		return value[i+1:]
	}
	return value
}

func stringValue(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func resolvedWaitChecksSHA(ref string, checkRuns *github.ListCheckRunsResults, combined *github.CombinedStatus) string {
	if combined != nil && strings.TrimSpace(combined.GetSHA()) != "" {
		return combined.GetSHA()
	}
	if checkRuns != nil {
		for _, run := range checkRuns.CheckRuns {
			if run != nil && strings.TrimSpace(run.GetHeadSHA()) != "" {
				return run.GetHeadSHA()
			}
		}
	}
	return ref
}
