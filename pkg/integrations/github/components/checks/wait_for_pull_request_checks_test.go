package checks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func Test__WaitForPullRequestChecks__Setup(t *testing.T) {
	component := &WaitForPullRequestChecks{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			mocks.GitHubResponse(http.StatusOK, `{
				"id": 123456,
				"name": "hello",
				"html_url": "https://github.com/testhq/hello"
			}`),
		},
	}
	integrationCtx := mocks.IntegrationContextForNewSetupFlow()
	metadata := &contexts.MetadataContext{}

	err := component.Setup(core.SetupContext{
		Integration:   integrationCtx,
		HTTP:          httpCtx,
		Metadata:      metadata,
		Configuration: map[string]any{"repository": "hello", "ref": "abc123"},
	})

	require.NoError(t, err)
	require.Len(t, integrationCtx.WebhookRequests, 1)
	webhookRequest := integrationCtx.WebhookRequests[0].(common.WebhookConfiguration)
	assert.ElementsMatch(t, []string{"check_run", "check_suite", "status"}, webhookRequest.EventTypes)
	assert.Equal(t, "hello", webhookRequest.Repository)
}

func Test__WaitForPullRequestChecks__Execute(t *testing.T) {
	component := &WaitForPullRequestChecks{}
	sha := "d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44"
	zero := 0

	t.Run("emits passed when all checks are terminal and quiet period is zero", func(t *testing.T) {
		httpCtx := newSnapshotHTTP(passingCheckRunsBody(sha), passingStatusesBody(sha))
		executionState := &contexts.ExecutionStateContext{}
		requests := &contexts.RequestContext{}
		metadata := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: WaitForPullRequestChecksConfiguration{
				Repository:         "hello",
				Ref:                sha,
				QuietPeriodSeconds: &zero,
			},
			HTTP:           httpCtx,
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: executionState,
			Requests:       requests,
			Metadata:       metadata,
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, waitChecksPassedChannel, executionState.Channel)
		assert.Equal(t, waitChecksPayloadType, executionState.Type)
		assert.Equal(t, "hello@"+sha, executionState.KVs[waitChecksRefKV])
		assert.Empty(t, requests.Action)
		assert.True(t, requestedCheckSnapshot(httpCtx))
	})

	t.Run("emits passed as soon as selected names are terminal", func(t *testing.T) {
		httpCtx := newSnapshotHTTP(passingCheckRunsBody(sha), passingStatusesBody(sha))
		executionState := &contexts.ExecutionStateContext{}
		requests := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: WaitForPullRequestChecksConfiguration{
				Repository: "hello",
				Ref:        sha,
				CheckNames: []string{"DCO"},
			},
			HTTP:           httpCtx,
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: executionState,
			Requests:       requests,
			Metadata:       &contexts.MetadataContext{},
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		assert.Equal(t, waitChecksPassedChannel, executionState.Channel)
		assert.Empty(t, requests.Action)
	})

	t.Run("waits the quiet period when no check names are selected", func(t *testing.T) {
		httpCtx := newSnapshotHTTP(passingCheckRunsBody(sha), passingStatusesBody(sha))
		executionState := &contexts.ExecutionStateContext{}
		requests := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: WaitForPullRequestChecksConfiguration{
				Repository: "hello",
				Ref:        sha,
			},
			HTTP:           httpCtx,
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: executionState,
			Requests:       requests,
			Metadata:       &contexts.MetadataContext{},
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		assert.False(t, executionState.Finished)
		assert.Equal(t, waitChecksEvaluateHook, requests.Action)
		assert.Equal(t, time.Duration(waitChecksDefaultQuietPeriodSeconds)*time.Second, requests.Duration)
	})

	t.Run("emits failed when a selected check failed", func(t *testing.T) {
		httpCtx := newSnapshotHTTP(
			fmt.Sprintf(`{
				"total_count": 1,
				"check_runs": [{
					"id": 1,
					"name": "build",
					"head_sha": %q,
					"status": "completed",
					"conclusion": "failure",
					"details_url": "https://example.com/build",
					"app": {"slug": "github-actions"}
				}]
			}`, sha),
			fmt.Sprintf(`{
				"sha": %q,
				"state": "failure",
				"total_count": 0,
				"statuses": []
			}`, sha),
		)
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: WaitForPullRequestChecksConfiguration{
				Repository:         "hello",
				Ref:                sha,
				CheckNames:         []string{"build"},
				QuietPeriodSeconds: &zero,
			},
			HTTP:           httpCtx,
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: executionState,
			Requests:       &contexts.RequestContext{},
			Metadata:       &contexts.MetadataContext{},
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		assert.Equal(t, waitChecksFailedChannel, executionState.Channel)
	})

	t.Run("schedules evaluate while checks are pending", func(t *testing.T) {
		httpCtx := newSnapshotHTTP(
			fmt.Sprintf(`{
				"total_count": 1,
				"check_runs": [{
					"id": 1,
					"name": "build",
					"head_sha": %q,
					"status": "in_progress",
					"app": {"slug": "github-actions"}
				}]
			}`, sha),
			fmt.Sprintf(`{
				"sha": %q,
				"state": "pending",
				"total_count": 0,
				"statuses": []
			}`, sha),
		)
		executionState := &contexts.ExecutionStateContext{}
		requests := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: WaitForPullRequestChecksConfiguration{
				Repository: "hello",
				Ref:        sha,
			},
			HTTP:           httpCtx,
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: executionState,
			Requests:       requests,
			Metadata:       &contexts.MetadataContext{},
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		assert.False(t, executionState.Finished)
		assert.Equal(t, waitChecksEvaluateHook, requests.Action)
		assert.Equal(t, waitChecksPollInterval, requests.Duration)
		assert.Equal(t, "hello@"+sha, executionState.KVs[waitChecksRefKV])
	})
}

func Test__WaitForPullRequestChecks__HandleWebhook(t *testing.T) {
	component := &WaitForPullRequestChecks{}
	sha := "d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44"
	httpCtx := &contexts.HTTPContext{}

	t.Run("ignores other event types", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-GitHub-Event", "push")

		code, _, err := component.HandleWebhook(signedWaitChecksRequest(
			[]byte(`{"repository":{"name":"hello"},"sha":"abc"}`),
			headers,
			WaitForPullRequestChecksConfiguration{Repository: "hello", Ref: sha},
			nil,
			httpCtx,
		))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("schedules evaluate for a matching check run", func(t *testing.T) {
		requests := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{}
		headers := http.Header{}
		headers.Set("X-GitHub-Event", "check_run")

		code, _, err := component.HandleWebhook(signedWaitChecksRequest(
			[]byte(fmt.Sprintf(`{
				"repository": {"name": "hello", "full_name": "testhq/hello"},
				"check_run": {"head_sha": %q, "name": "build"}
			}`, sha)),
			headers,
			WaitForPullRequestChecksConfiguration{Repository: "hello", Ref: sha},
			func(key, value string) (*core.ExecutionContext, error) {
				assert.Equal(t, waitChecksRefKV, key)
				assert.Equal(t, "hello@"+sha, value)
				return &core.ExecutionContext{
					Requests:       requests,
					ExecutionState: executionState,
				}, nil
			},
			httpCtx,
		))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, waitChecksEvaluateHook, requests.Action)
		assert.Equal(t, waitChecksWebhookDelay, requests.Duration)
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("rejects an invalid signature", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-GitHub-Event", "status")
		headers.Set("X-Hub-Signature-256", "sha256=deadbeef")

		code, _, err := component.HandleWebhook(core.WebhookRequestContext{
			Body:          []byte(`{"sha":"abc"}`),
			Headers:       headers,
			Logger:        logrus.NewEntry(logrus.New()),
			Configuration: WaitForPullRequestChecksConfiguration{Repository: "hello", Ref: sha},
			Webhook:       &contexts.NodeWebhookContext{Secret: "test-secret"},
			HTTP:          httpCtx,
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.Error(t, err)
		assert.Empty(t, httpCtx.Requests)
	})
}

func Test__WaitForPullRequestChecks__HandleHook(t *testing.T) {
	component := &WaitForPullRequestChecks{}
	sha := "d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44"
	zero := 0

	t.Run("skips work when the execution already finished", func(t *testing.T) {
		httpCtx := newSnapshotHTTP("", "")
		err := component.HandleHook(core.ActionHookContext{
			Name:           waitChecksEvaluateHook,
			Configuration:  WaitForPullRequestChecksConfiguration{Repository: "hello", Ref: sha},
			HTTP:           httpCtx,
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: &contexts.ExecutionStateContext{Finished: true},
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
			Logger:         logrus.NewEntry(logrus.New()),
		})
		require.NoError(t, err)
		assert.Empty(t, httpCtx.requests)
	})

	t.Run("emits passed after a later evaluate", func(t *testing.T) {
		httpCtx := newSnapshotHTTP(passingCheckRunsBody(sha), passingStatusesBody(sha))
		executionState := &contexts.ExecutionStateContext{}
		startedAt := time.Now().Add(-time.Minute)

		err := component.HandleHook(core.ActionHookContext{
			Name: waitChecksEvaluateHook,
			Configuration: WaitForPullRequestChecksConfiguration{
				Repository:         "hello",
				Ref:                sha,
				QuietPeriodSeconds: &zero,
			},
			HTTP:           httpCtx,
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: executionState,
			Metadata: &contexts.MetadataContext{
				Metadata: WaitForPullRequestChecksMetadata{
					Repository:   "hello",
					SHA:          sha,
					StartedAt:    startedAt,
					LastChangeAt: startedAt,
					TimeoutAt:    startedAt.Add(time.Hour),
				},
			},
			Requests: &contexts.RequestContext{},
			Logger:   logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		assert.Equal(t, waitChecksPassedChannel, executionState.Channel)
	})

	t.Run("decodes timestamps after JSON persist", func(t *testing.T) {
		httpCtx := newSnapshotHTTP(passingCheckRunsBody(sha), passingStatusesBody(sha))
		executionState := &contexts.ExecutionStateContext{}
		startedAt := time.Now().Add(-time.Minute).UTC().Truncate(time.Second)

		err := component.HandleHook(core.ActionHookContext{
			Name: waitChecksEvaluateHook,
			Configuration: WaitForPullRequestChecksConfiguration{
				Repository:         "hello",
				Ref:                sha,
				QuietPeriodSeconds: &zero,
			},
			HTTP:           httpCtx,
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: executionState,
			Metadata: &contexts.MetadataContext{
				Metadata: persistWaitChecksMetadata(t, WaitForPullRequestChecksMetadata{
					Repository:   "hello",
					SHA:          sha,
					StartedAt:    startedAt,
					LastChangeAt: startedAt,
					TimeoutAt:    startedAt.Add(time.Hour),
				}),
			},
			Requests: &contexts.RequestContext{},
			Logger:   logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		assert.Equal(t, waitChecksPassedChannel, executionState.Channel)
	})
}

func Test__DecodeWaitChecksMetadata(t *testing.T) {
	startedAt := time.Now().UTC().Truncate(time.Millisecond)
	completedAt := startedAt.Add(2 * time.Minute)
	raw := persistWaitChecksMetadata(t, WaitForPullRequestChecksMetadata{
		Repository:   "hello",
		SHA:          "abc123",
		StartedAt:    startedAt,
		LastChangeAt: startedAt.Add(time.Second),
		TimeoutAt:    startedAt.Add(time.Hour),
		CompletedAt:  &completedAt,
		Fingerprint:  "fp",
		Outcome:      waitChecksPassedChannel,
	})

	startedAtRaw, ok := raw["startedAt"].(string)
	require.True(t, ok, "production persist stores startedAt as a JSON string")
	require.NotEmpty(t, startedAtRaw)

	metadata, err := decodeWaitChecksMetadata(raw)
	require.NoError(t, err)
	assert.Equal(t, "hello", metadata.Repository)
	assert.True(t, metadata.StartedAt.Equal(startedAt), "startedAt=%s want=%s", metadata.StartedAt, startedAt)
	assert.True(t, metadata.LastChangeAt.Equal(startedAt.Add(time.Second)))
	assert.True(t, metadata.TimeoutAt.Equal(startedAt.Add(time.Hour)))
	require.NotNil(t, metadata.CompletedAt)
	assert.True(t, metadata.CompletedAt.Equal(completedAt))
	assert.Equal(t, waitChecksPassedChannel, metadata.Outcome)
}

func signedWaitChecksRequest(
	body []byte,
	headers http.Header,
	config WaitForPullRequestChecksConfiguration,
	find func(key, value string) (*core.ExecutionContext, error),
	httpCtx core.HTTPContext,
) core.WebhookRequestContext {
	secret := "test-secret"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	headers.Set("X-Hub-Signature-256", "sha256="+fmt.Sprintf("%x", mac.Sum(nil)))

	return core.WebhookRequestContext{
		Body:              body,
		Headers:           headers,
		Logger:            logrus.NewEntry(logrus.New()),
		Configuration:     config,
		Webhook:           &contexts.NodeWebhookContext{Secret: secret},
		FindExecutionByKV: find,
		HTTP:              httpCtx,
	}
}

func passingCheckRunsBody(sha string) string {
	return fmt.Sprintf(`{
		"total_count": 1,
		"check_runs": [{
			"id": 1,
			"name": "DCO",
			"head_sha": %q,
			"status": "completed",
			"conclusion": "success",
			"details_url": "https://example.com/dco",
			"app": {"slug": "dco"}
		}]
	}`, sha)
}

func passingStatusesBody(sha string) string {
	return fmt.Sprintf(`{
		"sha": %q,
		"state": "success",
		"total_count": 1,
		"statuses": [{
			"context": "ci/semaphore",
			"state": "success",
			"target_url": "https://example.com/ci"
		}]
	}`, sha)
}

func persistWaitChecksMetadata(t *testing.T, metadata WaitForPullRequestChecksMetadata) map[string]any {
	t.Helper()
	body, err := json.Marshal(metadata)
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(body, &raw))
	return raw
}

func requestedCheckSnapshot(httpCtx *snapshotHTTP) bool {
	var sawChecks, sawStatus bool
	for _, request := range httpCtx.requests {
		path := request.URL.Path
		if strings.Contains(path, "/check-runs") {
			sawChecks = true
		}
		if strings.HasSuffix(path, "/status") {
			sawStatus = true
		}
	}
	return sawChecks && sawStatus
}

type snapshotHTTP struct {
	mu            sync.Mutex
	checkRunsBody string
	statusBody    string
	requests      []*http.Request
}

func newSnapshotHTTP(checkRunsBody, statusBody string) *snapshotHTTP {
	return &snapshotHTTP{checkRunsBody: checkRunsBody, statusBody: statusBody}
}

func (h *snapshotHTTP) Do(request *http.Request) (*http.Response, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.requests = append(h.requests, request)

	body := h.statusBody
	if strings.Contains(request.URL.Path, "/check-runs") {
		body = h.checkRunsBody
	}
	if body == "" {
		return nil, fmt.Errorf("no response mocked for %s", request.URL.Path)
	}
	return mocks.GitHubResponse(http.StatusOK, body), nil
}
