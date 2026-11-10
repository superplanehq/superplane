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
		Integration: integrationCtx,
		HTTP:        httpCtx,
		Metadata:    metadata,
		Configuration: map[string]any{
			"repository": "hello",
			"ref":        "abc123",
			"checkNames": []string{"DCO"},
		},
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

	t.Run("emits passed when selected checks are terminal", func(t *testing.T) {
		httpCtx := newSnapshotHTTP(passingCheckRunsBody(sha), passingStatusesBody(sha))
		executionState := &contexts.ExecutionStateContext{}
		requests := &contexts.RequestContext{}
		metadata := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: WaitForPullRequestChecksConfiguration{
				Repository: "hello",
				Ref:        sha,
				CheckNames: []string{"DCO", "ci/semaphore"},
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

	t.Run("rejects empty check names", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: WaitForPullRequestChecksConfiguration{
				Repository: "hello",
				Ref:        sha,
			},
			HTTP:           newSnapshotHTTP("", ""),
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Requests:       &contexts.RequestContext{},
			Metadata:       &contexts.MetadataContext{},
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "checkNames is required")
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
				Repository: "hello",
				Ref:        sha,
				CheckNames: []string{"build"},
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

	t.Run("keys the wait by the lowercase full repository name", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: WaitForPullRequestChecksConfiguration{
				Repository: "TestHQ/World",
				Ref:        sha,
				CheckNames: []string{"DCO"},
			},
			HTTP:           newSnapshotHTTP(passingCheckRunsBody(sha), passingStatusesBody(sha)),
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: executionState,
			Requests:       &contexts.RequestContext{},
			Metadata:       &contexts.MetadataContext{},
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		assert.Equal(t, "testhq/world@"+sha, executionState.KVs[waitChecksRefKV])
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
				CheckNames: []string{"build"},
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

func Test__WaitForPullRequestChecks__GitHubErrors(t *testing.T) {
	sha := "abc123"
	for _, endpoint := range []string{"check-runs", "status"} {
		for _, statusCode := range []int{
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
		} {
			t.Run(fmt.Sprintf("%s/%d", endpoint, statusCode), func(t *testing.T) {
				component := &WaitForPullRequestChecks{}
				config := WaitForPullRequestChecksConfiguration{
					Repository: "TestHQ/hello",
					Ref:        sha,
					CheckNames: []string{"DCO", "ci/semaphore"},
				}
				basePath := "/repos/TestHQ/hello/commits/" + sha + "/"
				httpCtx := &contexts.HTTPContext{ResponsesByPath: map[string][]*http.Response{
					basePath + "check-runs": {
						mocks.GitHubResponse(http.StatusOK, passingCheckRunsBody(sha)),
						mocks.GitHubResponse(http.StatusOK, passingCheckRunsBody(sha)),
					},
					basePath + "status": {
						mocks.GitHubResponse(http.StatusOK, passingStatusesBody(sha)),
						mocks.GitHubResponse(http.StatusOK, passingStatusesBody(sha)),
					},
				}}
				httpCtx.ResponsesByPath[basePath+endpoint][0] = mocks.GitHubResponse(statusCode, `{"message":"GitHub API error"}`)
				executionState := &contexts.ExecutionStateContext{}
				requests := &contexts.RequestContext{}
				metadata := &contexts.MetadataContext{}
				integration := mocks.IntegrationContextForNewSetupFlow()

				err := component.Execute(core.ExecutionContext{
					Configuration:  config,
					HTTP:           httpCtx,
					Integration:    integration,
					ExecutionState: executionState,
					Requests:       requests,
					Metadata:       metadata,
				})
				if statusCode < 500 {
					require.Error(t, err)
					assert.Empty(t, requests.Action)
					assert.False(t, executionState.Finished)
					return
				}

				require.NoError(t, err)
				assert.False(t, executionState.Finished)
				assert.Equal(t, waitChecksEvaluateHook, requests.Action)
				assert.Equal(t, 30*time.Second, requests.Duration)
				assert.Equal(t, "testhq/hello@"+sha, executionState.KVs[waitChecksRefKV])
				beforeRetry := metadata.Metadata.(WaitForPullRequestChecksMetadata)

				err = component.HandleHook(core.ActionHookContext{
					Name:           requests.Action,
					Configuration:  config,
					HTTP:           httpCtx,
					Integration:    integration,
					ExecutionState: executionState,
					Requests:       &contexts.RequestContext{},
					Metadata:       metadata,
				})
				require.NoError(t, err)
				assert.Equal(t, waitChecksPassedChannel, executionState.Channel)
				afterRetry := metadata.Metadata.(WaitForPullRequestChecksMetadata)
				assert.Equal(t, beforeRetry.StartedAt, afterRetry.StartedAt)
				assert.Equal(t, beforeRetry.TimeoutAt, afterRetry.TimeoutAt)
			})
		}
	}
}

func Test__WaitForPullRequestChecks__PollingDeadline(t *testing.T) {
	for _, test := range []struct {
		name      string
		remaining time.Duration
		delay     time.Duration
		pending   bool
	}{
		{name: "retries a later poll", remaining: time.Minute, delay: 30 * time.Second},
		{name: "caps a retry at the deadline", remaining: 10 * time.Second, delay: 10 * time.Second},
		{name: "rounds up a retry near the deadline", remaining: 500 * time.Millisecond, delay: time.Second},
		{name: "rounds up a pending poll near the deadline", remaining: 500 * time.Millisecond, delay: time.Second, pending: true},
		{name: "stops at the deadline", remaining: -time.Second},
	} {
		t.Run(test.name, func(t *testing.T) {
			now := time.Now()
			snapshot := WaitForPullRequestChecksMetadata{
				Repository:   "testhq/hello",
				SHA:          "abc123",
				StartedAt:    now.Add(-time.Minute),
				LastChangeAt: now.Add(-time.Minute),
				TimeoutAt:    now.Add(test.remaining),
				Outcome:      waitChecksOutcomePending,
				Checks:       []PullRequestCheck{{Name: "build", Status: checkStatusPending}},
				SelectedChecks: []PullRequestCheck{{
					Name: "build", Status: checkStatusPending,
				}},
			}
			metadata := &contexts.MetadataContext{Metadata: snapshot}
			executionState := &contexts.ExecutionStateContext{}
			requests := &contexts.RequestContext{}
			checkRuns := mocks.GitHubResponse(http.StatusServiceUnavailable, `{"message":"unavailable"}`)
			if test.pending {
				checkRuns = mocks.GitHubResponse(http.StatusOK, passingCheckRunsBody("abc123"))
			}
			httpCtx := &contexts.HTTPContext{ResponsesByPath: map[string][]*http.Response{
				"/repos/testhq/hello/commits/abc123/check-runs": {checkRuns},
				"/repos/testhq/hello/commits/abc123/status": {
					mocks.GitHubResponse(http.StatusOK, passingStatusesBody("abc123")),
				},
			}}

			err := evaluateWaitForPullRequestChecks(waitChecksRuntime{
				Configuration: WaitForPullRequestChecksConfiguration{
					Repository: snapshot.Repository,
					Ref:        snapshot.SHA,
					CheckNames: []string{"build"},
				},
				HTTP:           httpCtx,
				Integration:    mocks.IntegrationContextForNewSetupFlow(),
				ExecutionState: executionState,
				Requests:       requests,
				Metadata:       metadata,
			}, now)
			require.NoError(t, err)
			if test.delay > 0 {
				assert.False(t, executionState.Finished)
				assert.Equal(t, waitChecksEvaluateHook, requests.Action)
				assert.Equal(t, test.delay, requests.Duration)
				if !test.pending {
					assert.Equal(t, snapshot, metadata.Metadata)
				}
				return
			}

			assert.Equal(t, waitChecksTimedOutChannel, executionState.Channel)
			assert.Empty(t, requests.Action)
			completed := metadata.Metadata.(WaitForPullRequestChecksMetadata)
			assert.NotNil(t, completed.CompletedAt)
			assert.Equal(t, waitChecksOutcomeTimedOut, completed.Outcome)
			assert.Equal(t, snapshot.TimeoutAt, completed.TimeoutAt)
			require.Len(t, executionState.Payloads, 1)
			payload := executionState.Payloads[0].(map[string]any)["data"].(WaitForPullRequestChecksOutput)
			assert.Equal(t, snapshot.Checks, payload.Checks)
			assert.Equal(t, snapshot.SelectedChecks, payload.SelectedChecks)
		})
	}
}

func Test__WaitForPullRequestChecks__HandleWebhook(t *testing.T) {
	component := &WaitForPullRequestChecks{}
	sha := "d6f3c8a2e8b7f0a9c0a1f67f0c5d7b2a1d9e3f44"
	httpCtx := &contexts.HTTPContext{}
	config := WaitForPullRequestChecksConfiguration{
		Repository: "hello",
		Ref:        sha,
		CheckNames: []string{"build"},
	}

	t.Run("ignores other event types", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-GitHub-Event", "push")

		code, _, err := component.HandleWebhook(signedWaitChecksRequest(
			[]byte(`{"repository":{"name":"hello"},"sha":"abc"}`),
			headers,
			config,
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
			config,
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

	t.Run("workspace scope finds the wait in any workspace repository", func(t *testing.T) {
		requests := &contexts.RequestContext{}
		headers := http.Header{}
		headers.Set("X-GitHub-Event", "check_run")
		workspaceConfig := WaitForPullRequestChecksConfiguration{
			Repository: "{{ root().data.repository.full_name }}",
			Ref:        sha,
			CheckNames: []string{"build"},
		}

		code, _, err := component.HandleWebhook(signedWaitChecksRequest(
			[]byte(fmt.Sprintf(`{
				"repository": {"name": "world", "full_name": "TestHQ/World"},
				"check_run": {"head_sha": %q, "name": "build"}
			}`, sha)),
			headers,
			workspaceConfig,
			func(key, value string) (*core.ExecutionContext, error) {
				assert.Equal(t, "testhq/world@"+sha, value)
				return &core.ExecutionContext{
					Requests:       requests,
					ExecutionState: &contexts.ExecutionStateContext{},
				}, nil
			},
			httpCtx,
		))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, waitChecksEvaluateHook, requests.Action)
	})

	t.Run("does not schedule evaluate when the execution is cancelling", func(t *testing.T) {
		requests := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{Cancelling: true}
		headers := http.Header{}
		headers.Set("X-GitHub-Event", "check_run")

		code, _, err := component.HandleWebhook(signedWaitChecksRequest(
			[]byte(fmt.Sprintf(`{
				"repository": {"name": "hello", "full_name": "testhq/hello"},
				"check_run": {"head_sha": %q, "name": "build"}
			}`, sha)),
			headers,
			config,
			func(key, value string) (*core.ExecutionContext, error) {
				return &core.ExecutionContext{
					Requests:       requests,
					ExecutionState: executionState,
				}, nil
			},
			httpCtx,
		))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Empty(t, requests.Action)
	})

	t.Run("rejects an invalid signature", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-GitHub-Event", "status")
		headers.Set("X-Hub-Signature-256", "sha256=deadbeef")

		code, _, err := component.HandleWebhook(core.WebhookRequestContext{
			Body:          []byte(`{"sha":"abc"}`),
			Headers:       headers,
			Logger:        logrus.NewEntry(logrus.New()),
			Configuration: config,
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
	config := WaitForPullRequestChecksConfiguration{
		Repository: "hello",
		Ref:        sha,
		CheckNames: []string{"DCO", "ci/semaphore"},
	}

	t.Run("skips work when the execution already finished", func(t *testing.T) {
		httpCtx := newSnapshotHTTP("", "")
		err := component.HandleHook(core.ActionHookContext{
			Name:           waitChecksEvaluateHook,
			Configuration:  config,
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

	t.Run("skips work when the execution is cancelling", func(t *testing.T) {
		httpCtx := newSnapshotHTTP(passingCheckRunsBody(sha), passingStatusesBody(sha))
		executionState := &contexts.ExecutionStateContext{Cancelling: true}
		requests := &contexts.RequestContext{}

		err := component.HandleHook(core.ActionHookContext{
			Name:           waitChecksEvaluateHook,
			Configuration:  config,
			HTTP:           httpCtx,
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: executionState,
			Metadata:       &contexts.MetadataContext{},
			Requests:       requests,
			Logger:         logrus.NewEntry(logrus.New()),
		})
		require.NoError(t, err)
		assert.Empty(t, httpCtx.requests)
		assert.Empty(t, requests.Action)
		assert.Empty(t, executionState.Channel)
	})

	t.Run("emits passed after a later evaluate", func(t *testing.T) {
		httpCtx := newSnapshotHTTP(passingCheckRunsBody(sha), passingStatusesBody(sha))
		executionState := &contexts.ExecutionStateContext{}
		startedAt := time.Now().Add(-time.Minute)

		err := component.HandleHook(core.ActionHookContext{
			Name:           waitChecksEvaluateHook,
			Configuration:  config,
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
			Name:           waitChecksEvaluateHook,
			Configuration:  config,
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
