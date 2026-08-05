package common

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__ValidateReactionContent(t *testing.T) {
	for _, content := range []string{"+1", "-1", "laugh", "confused", "heart", "hooray", "rocket", "eyes"} {
		t.Run(content, func(t *testing.T) {
			require.NoError(t, ValidateReactionContent(content))
		})
	}

	require.ErrorContains(t, ValidateReactionContent("thumbs-up"), "invalid reaction content")
}

func Test__ExecuteReaction__RetriesTransientFailures(t *testing.T) {
	tests := []struct {
		name     string
		response *github.Response
	}{
		{name: "network error"},
		{name: "request timeout", response: reactionResponse(http.StatusRequestTimeout, nil)},
		{name: "too many requests", response: reactionResponse(http.StatusTooManyRequests, nil)},
		{name: "internal server error", response: reactionResponse(http.StatusInternalServerError, nil)},
		{name: "bad gateway", response: reactionResponse(http.StatusBadGateway, nil)},
		{name: "service unavailable", response: reactionResponse(http.StatusServiceUnavailable, nil)},
		{name: "gateway timeout", response: reactionResponse(http.StatusGatewayTimeout, nil)},
		{
			name: "primary rate limit",
			response: reactionResponse(http.StatusForbidden, http.Header{
				"X-Ratelimit-Remaining": []string{"0"},
			}),
		},
		{
			name: "secondary rate limit",
			response: reactionResponse(http.StatusForbidden, http.Header{
				"Retry-After": []string{"17"},
			}),
		},
		{
			name: "long provider retry window",
			response: reactionResponse(http.StatusTooManyRequests, http.Header{
				"Retry-After": []string{"900"},
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := &contexts.MetadataContext{}
			requests := &contexts.RequestContext{}
			state := &contexts.ExecutionStateContext{}

			err := ExecuteReaction(core.ExecutionContext{
				Metadata:       metadata,
				Requests:       requests,
				ExecutionState: state,
			}, func() (*github.Reaction, *github.Response, error) {
				return nil, tt.response, errors.New("provider unavailable")
			})

			require.NoError(t, err)
			assert.Equal(t, ReactionRetryHookName, requests.Action)
			assert.GreaterOrEqual(t, requests.Duration, time.Second)
			assert.False(t, state.Finished)

			if tt.name == "secondary rate limit" {
				assert.Equal(t, 17*time.Second, requests.Duration)
			}
			if tt.name == "long provider retry window" {
				assert.Equal(t, 15*time.Minute, requests.Duration)
			}
		})
	}
}

func Test__ExecuteReaction__DoesNotRetryPermanentFailures(t *testing.T) {
	for _, status := range []int{
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusUnprocessableEntity,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			requests := &contexts.RequestContext{}
			err := ExecuteReaction(core.ExecutionContext{
				Metadata:       &contexts.MetadataContext{},
				Requests:       requests,
				ExecutionState: &contexts.ExecutionStateContext{},
			}, func() (*github.Reaction, *github.Response, error) {
				return nil, reactionResponse(status, nil), errors.New("permanent provider error")
			})

			require.ErrorContains(t, err, "permanent provider error")
			assert.Empty(t, requests.Action)
		})
	}
}

func Test__RetryReaction__EventuallySucceeds(t *testing.T) {
	metadata := &contexts.MetadataContext{}
	initialRequests := &contexts.RequestContext{}
	state := &contexts.ExecutionStateContext{}

	require.NoError(t, ExecuteReaction(core.ExecutionContext{
		Metadata:       metadata,
		Requests:       initialRequests,
		ExecutionState: state,
	}, func() (*github.Reaction, *github.Response, error) {
		return nil, reactionResponse(http.StatusServiceUnavailable, nil), errors.New("temporary outage")
	}))

	err := RetryReaction(core.ActionHookContext{
		Name:           ReactionRetryHookName,
		Metadata:       metadata,
		Requests:       &contexts.RequestContext{},
		ExecutionState: state,
	}, func() (*github.Reaction, *github.Response, error) {
		return &github.Reaction{ID: github.Ptr(int64(99)), Content: github.Ptr("eyes")}, reactionResponse(http.StatusCreated, nil), nil
	})

	require.NoError(t, err)
	assert.True(t, state.Passed)
	assert.Equal(t, "github.reaction", state.Type)
	assert.Len(t, state.Payloads, 1)
}

func Test__RetryReaction__StopsAfterMaximumAttempts(t *testing.T) {
	metadata := &contexts.MetadataContext{}
	state := &contexts.ExecutionStateContext{}
	operation := func() (*github.Reaction, *github.Response, error) {
		return nil, reactionResponse(http.StatusServiceUnavailable, nil), errors.New("temporary outage")
	}

	require.NoError(t, ExecuteReaction(core.ExecutionContext{
		Metadata:       metadata,
		Requests:       &contexts.RequestContext{},
		ExecutionState: state,
	}, operation))

	require.NoError(t, RetryReaction(core.ActionHookContext{
		Name:           ReactionRetryHookName,
		Metadata:       metadata,
		Requests:       &contexts.RequestContext{},
		ExecutionState: state,
	}, operation))

	err := RetryReaction(core.ActionHookContext{
		Name:           ReactionRetryHookName,
		Metadata:       metadata,
		Requests:       &contexts.RequestContext{},
		ExecutionState: state,
	}, operation)

	require.NoError(t, err)
	assert.True(t, state.Finished)
	assert.False(t, state.Passed)
	assert.Contains(t, state.FailureMessage, "failed after 3 attempts")
}

func Test__RetryReaction__FinishesOnPermanentProviderFailure(t *testing.T) {
	metadata := &contexts.MetadataContext{}
	state := &contexts.ExecutionStateContext{}

	require.NoError(t, ExecuteReaction(core.ExecutionContext{
		Metadata:       metadata,
		Requests:       &contexts.RequestContext{},
		ExecutionState: state,
	}, func() (*github.Reaction, *github.Response, error) {
		return nil, reactionResponse(http.StatusServiceUnavailable, nil), errors.New("temporary outage")
	}))

	err := RetryReaction(core.ActionHookContext{
		Name:           ReactionRetryHookName,
		Metadata:       metadata,
		Requests:       &contexts.RequestContext{},
		ExecutionState: state,
	}, func() (*github.Reaction, *github.Response, error) {
		return nil, reactionResponse(http.StatusForbidden, nil), errors.New("permission denied")
	})

	require.NoError(t, err)
	assert.True(t, state.Finished)
	assert.False(t, state.Passed)
	assert.Contains(t, state.FailureMessage, "permission denied")
}

func Test__RetryReaction__FinishesOnInvalidRetryMetadata(t *testing.T) {
	for _, tt := range []struct {
		name     string
		metadata any
		message  string
	}{
		{name: "decode failure", metadata: "not a map", message: "failed to decode reaction retry metadata"},
		{name: "invalid attempt", metadata: map[string]any{"attempts": 0}, message: "invalid reaction retry attempt"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			state := &contexts.ExecutionStateContext{}

			err := RetryReaction(core.ActionHookContext{
				Name:           ReactionRetryHookName,
				Metadata:       &contexts.MetadataContext{Metadata: tt.metadata},
				ExecutionState: state,
			}, func() (*github.Reaction, *github.Response, error) {
				t.Fatal("operation must not run with invalid retry metadata")
				return nil, nil, nil
			})

			require.NoError(t, err)
			assert.True(t, state.Finished)
			assert.False(t, state.Passed)
			assert.Contains(t, state.FailureMessage, tt.message)
		})
	}
}

func Test__ExecuteReaction__ReportsSchedulingFailure(t *testing.T) {
	err := ExecuteReaction(core.ExecutionContext{
		Metadata:       &contexts.MetadataContext{},
		Requests:       failingReactionRequestContext{},
		ExecutionState: &contexts.ExecutionStateContext{},
	}, func() (*github.Reaction, *github.Response, error) {
		return nil, reactionResponse(http.StatusServiceUnavailable, nil), errors.New("temporary outage")
	})

	require.ErrorContains(t, err, "failed to schedule reaction retry")
}

func reactionResponse(status int, headers http.Header) *github.Response {
	if headers == nil {
		headers = make(http.Header)
	}

	return &github.Response{Response: &http.Response{StatusCode: status, Header: headers}}
}

type failingReactionRequestContext struct{}

func (failingReactionRequestContext) ScheduleActionCall(string, map[string]any, time.Duration) error {
	return errors.New("scheduler unavailable")
}
