package issues

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func Test__AddIssueReaction__Setup(t *testing.T) {
	component := AddIssueReaction{}

	validConfig := func(overrides map[string]any) map[string]any {
		config := map[string]any{
			"repository":  "hello",
			"issueNumber": "42",
			"content":     "eyes",
		}
		for key, value := range overrides {
			config[key] = value
		}
		return config
	}

	t.Run("repository is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{"repository": ""}),
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("issue number is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{"issueNumber": ""}),
		})

		require.ErrorContains(t, err, "issue number is required")
	})

	t.Run("reaction content is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{"content": ""}),
		})

		require.ErrorContains(t, err, "reaction content is required")
	})

	t.Run("reaction content must be supported", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{"content": "thumbs-up"}),
		})

		require.ErrorContains(t, err, "invalid reaction content")
	})

	for _, field := range []string{"repository", "issueNumber", "content"} {
		t.Run(field+" rejects whitespace-only values", func(t *testing.T) {
			err := component.Setup(core.SetupContext{
				Integration:   &contexts.IntegrationContext{},
				Metadata:      &contexts.MetadataContext{},
				Configuration: validConfig(map[string]any{field: "   "}),
			})

			require.ErrorContains(t, err, "required")
		})
	}
}

func Test__AddIssueReaction__Execute(t *testing.T) {
	component := AddIssueReaction{}

	t.Run("issue number must be numeric", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository":  "hello",
				"issueNumber": "abc",
				"content":     "eyes",
			},
		})

		require.ErrorContains(t, err, "issue number is not a number")
	})

	for _, issueNumber := range []string{"0", "-1"} {
		t.Run("issue number must be positive "+issueNumber, func(t *testing.T) {
			err := component.Execute(core.ExecutionContext{
				Integration:    mocks.IntegrationContextForNewSetupFlow(),
				ExecutionState: &contexts.ExecutionStateContext{},
				Configuration: map[string]any{
					"repository":  "hello",
					"issueNumber": issueNumber,
					"content":     "eyes",
				},
			})

			require.ErrorContains(t, err, "issue number must be positive")
		})
	}

	t.Run("reaction content is validated after expressions are resolved", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}
		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository":  "hello",
				"issueNumber": "42",
				"content":     "thumbs-up",
			},
		})

		require.ErrorContains(t, err, "invalid reaction content")
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("permanent provider errors are not retried", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusUnprocessableEntity, `{"message":"invalid reaction"}`),
			},
		}
		requests := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			Requests:       requests,
			Metadata:       &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository":  "hello",
				"issueNumber": "42",
				"content":     "eyes",
			},
		})

		require.ErrorContains(t, err, "failed to create issue reaction")
		assert.Empty(t, requests.Action)
	})

	t.Run("transient provider errors schedule a durable retry", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusServiceUnavailable, `{"message":"temporarily unavailable"}`),
			},
		}
		requests := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			Requests:       requests,
			Metadata:       &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository":  "hello",
				"issueNumber": "42",
				"content":     "eyes",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, common.ReactionRetryHookName, requests.Action)
	})

	for _, status := range []int{http.StatusCreated, http.StatusOK} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			httpCtx := &contexts.HTTPContext{
				Responses: []*http.Response{
					mocks.GitHubResponse(status, `{"id": 99, "content": "eyes"}`),
				},
			}
			executionState := &contexts.ExecutionStateContext{}

			err := component.Execute(core.ExecutionContext{
				Integration:    mocks.IntegrationContextForNewSetupFlow(),
				HTTP:           httpCtx,
				Metadata:       &contexts.MetadataContext{},
				Requests:       &contexts.RequestContext{},
				ExecutionState: executionState,
				Configuration: map[string]any{
					"repository":  "hello",
					"issueNumber": "42",
					"content":     "eyes",
				},
			})

			require.NoError(t, err)
			require.Len(t, httpCtx.Requests, 1)
			assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
			assert.Equal(t, "/repos/testhq/hello/issues/42/reactions", httpCtx.Requests[0].URL.Path)

			body, err := httpCtx.Requests[0].GetBody()
			require.NoError(t, err)
			var requestBody map[string]string
			require.NoError(t, json.NewDecoder(body).Decode(&requestBody))
			assert.Equal(t, map[string]string{"content": "eyes"}, requestBody)

			assert.True(t, executionState.Passed)
			assert.Equal(t, "github.reaction", executionState.Type)
		})
	}
}

func Test__AddIssueReaction__HandleHook(t *testing.T) {
	component := AddIssueReaction{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			mocks.GitHubResponse(http.StatusServiceUnavailable, `{"message":"temporarily unavailable"}`),
			mocks.GitHubResponse(http.StatusCreated, `{"id":99,"content":"eyes"}`),
		},
	}
	metadata := &contexts.MetadataContext{}
	state := &contexts.ExecutionStateContext{}
	configuration := map[string]any{
		"repository":  "hello",
		"issueNumber": "42",
		"content":     "eyes",
	}

	require.NoError(t, component.Execute(core.ExecutionContext{
		Integration:    mocks.IntegrationContextForNewSetupFlow(),
		HTTP:           httpCtx,
		Metadata:       metadata,
		Requests:       &contexts.RequestContext{},
		ExecutionState: state,
		Configuration:  configuration,
	}))

	err := component.HandleHook(core.ActionHookContext{
		Name:           common.ReactionRetryHookName,
		Integration:    mocks.IntegrationContextForNewSetupFlow(),
		HTTP:           httpCtx,
		Metadata:       metadata,
		Requests:       &contexts.RequestContext{},
		ExecutionState: state,
		Configuration:  configuration,
	})

	require.NoError(t, err)
	assert.True(t, state.Passed)
	assert.Equal(t, "github.reaction", state.Type)
}

func Test__AddIssueReaction__HandleHookFinishesOnPreflightError(t *testing.T) {
	validConfiguration := map[string]any{
		"repository":  "hello",
		"issueNumber": "42",
		"content":     "eyes",
	}

	for _, tt := range []struct {
		name          string
		configuration any
		invalidClient bool
		message       string
	}{
		{name: "configuration decode", configuration: "not a map", message: "failed to decode configuration"},
		{name: "issue number", configuration: map[string]any{"repository": "hello", "issueNumber": "abc", "content": "eyes"}, message: "issue number is not a number"},
		{name: "reaction content", configuration: map[string]any{"repository": "hello", "issueNumber": "42", "content": "invalid"}, message: "invalid reaction content"},
		{name: "GitHub client", configuration: validConfiguration, invalidClient: true, message: "failed to initialize GitHub client"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			integration := mocks.IntegrationContextForNewSetupFlow()
			if tt.invalidClient {
				integration.CurrentSecrets = nil
			}
			state := &contexts.ExecutionStateContext{}

			err := (&AddIssueReaction{}).HandleHook(core.ActionHookContext{
				Name:           common.ReactionRetryHookName,
				Integration:    integration,
				HTTP:           &contexts.HTTPContext{},
				Metadata:       &contexts.MetadataContext{},
				Requests:       &contexts.RequestContext{},
				ExecutionState: state,
				Configuration:  tt.configuration,
			})

			require.NoError(t, err)
			assert.True(t, state.Finished)
			assert.False(t, state.Passed)
			assert.Contains(t, state.FailureMessage, tt.message)
		})
	}
}
