package pulls

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

// fakeFactoryContext lets these tests drive `core.FactoryContext` without a
// database. Only ResolveWorkOrderAssigneeAccounts and SetWorkOrderStatusNote
// are wired up; the rest of the interface returns zero values.
type fakeFactoryContext struct {
	resolveAssigneeAccountsCalls  int
	resolveAssigneeAccountsParams core.ResolveWorkOrderAssigneeAccountsParams
	resolveAssigneeAccountsResult *core.WorkOrderAssigneeAccounts
	resolveAssigneeAccountsErr    error

	setStatusNoteCalls  int
	setStatusNoteParams core.SetWorkOrderStatusNoteParams

	clearStatusNoteCalls  int
	clearStatusNoteParams core.ClearWorkOrderStatusNoteParams
}

func (f *fakeFactoryContext) CreateWorkOrder(_ core.WorkOrderParams) (*core.WorkOrder, error) {
	return nil, nil
}

func (f *fakeFactoryContext) FindWorkOrder(_ core.FindWorkOrderParams) (*core.WorkOrder, error) {
	return nil, nil
}

func (f *fakeFactoryContext) UpdateWorkOrderStatus(_ core.UpdateWorkOrderStatusParams) (*core.WorkOrder, bool, error) {
	return nil, false, nil
}

func (f *fakeFactoryContext) AddWorkOrderComment(_ core.AddWorkOrderCommentParams) error {
	return nil
}

func (f *fakeFactoryContext) AddWorkOrderArtifact(_ core.AddWorkOrderArtifactParams) (*core.WorkOrderArtifact, error) {
	return nil, nil
}

func (f *fakeFactoryContext) ReportWorkOrderCheck(_ core.ReportWorkOrderCheckParams) (*core.WorkOrderCheck, error) {
	return nil, nil
}

func (f *fakeFactoryContext) SetWorkOrderStatusNote(params core.SetWorkOrderStatusNoteParams) (*core.WorkOrderStatusNote, error) {
	f.setStatusNoteCalls++
	f.setStatusNoteParams = params
	return nil, nil
}

func (f *fakeFactoryContext) ClearWorkOrderStatusNote(params core.ClearWorkOrderStatusNoteParams) error {
	f.clearStatusNoteCalls++
	f.clearStatusNoteParams = params
	return nil
}

func (f *fakeFactoryContext) AddPullRequest(_ core.AddPullRequestParams) (*core.PullRequest, error) {
	return nil, nil
}

func (f *fakeFactoryContext) UpdatePullRequest(_ core.UpdatePullRequestParams) (*core.PullRequest, error) {
	return nil, nil
}

func (f *fakeFactoryContext) FindPullRequest(_ core.FindPullRequestParams) (*core.PullRequestMatch, error) {
	return nil, nil
}

func (f *fakeFactoryContext) AddPullRequestActivity(_ core.AddPullRequestActivityParams) (*core.PullRequestActivityResult, error) {
	return nil, nil
}

func (f *fakeFactoryContext) UpdatePullRequestActivity(_ core.UpdatePullRequestActivityParams) (*core.PullRequestActivityResult, error) {
	return nil, nil
}

func (f *fakeFactoryContext) ResolveWorkOrderAssigneeAccounts(params core.ResolveWorkOrderAssigneeAccountsParams) (*core.WorkOrderAssigneeAccounts, error) {
	f.resolveAssigneeAccountsCalls++
	f.resolveAssigneeAccountsParams = params
	if f.resolveAssigneeAccountsErr != nil {
		return nil, f.resolveAssigneeAccountsErr
	}
	if f.resolveAssigneeAccountsResult != nil {
		return f.resolveAssigneeAccountsResult, nil
	}
	return &core.WorkOrderAssigneeAccounts{}, nil
}

func createdPullRequestResponse() *http.Response {
	return mocks.GitHubResponse(http.StatusCreated, `{
		"number": 42,
		"title": "My PR",
		"html_url": "https://github.com/testhq/hello/pull/42"
	}`)
}

func Test__CreatePullRequest__Setup(t *testing.T) {
	component := CreatePullRequest{}

	validConfig := func(overrides map[string]any) map[string]any {
		config := map[string]any{
			"repository": "hello",
			"head":       "feature",
			"base":       "main",
			"title":      "My PR",
		}
		for k, v := range overrides {
			config[k] = v
		}
		return config
	}

	t.Run("repository is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{"repository": ""}),
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("head branch is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{"head": ""}),
		})

		require.ErrorContains(t, err, "head branch is required")
	})

	t.Run("base branch is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{"base": ""}),
		})

		require.ErrorContains(t, err, "base branch is required")
	})

	t.Run("title is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{"title": ""}),
		})

		require.ErrorContains(t, err, "title is required")
	})

	t.Run("head and base must differ when both are literals", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{"head": "main", "base": "main"}),
		})

		require.ErrorContains(t, err, "head and base branches must be different")
	})

	t.Run("head and base equality check is skipped when either is an expression", func(t *testing.T) {
		integrationCtx := mocks.IntegrationContextForNewSetupFlow()
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{
					"id": 123456,
					"name": "hello",
					"html_url": "https://github.com/testhq/hello"
				}`),
			},
		}

		require.NoError(t, component.Setup(core.SetupContext{
			Integration: integrationCtx,
			HTTP:        httpCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: validConfig(map[string]any{
				"head": `{{$["github.onPush"].data.ref}}`,
				"base": `{{$["github.onPush"].data.ref}}`,
			}),
		}))
	})
}

func Test__CreatePullRequest__Execute(t *testing.T) {
	component := CreatePullRequest{}

	t.Run("fails when configuration decode fails", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  "not a map",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("repository is required", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "",
				"head":       "feature",
				"base":       "main",
				"title":      "My PR",
			},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("head branch is required", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"head":       "",
				"base":       "main",
				"title":      "My PR",
			},
		})

		require.ErrorContains(t, err, "head branch is required")
	})

	t.Run("base branch is required", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"head":       "feature",
				"base":       "",
				"title":      "My PR",
			},
		})

		require.ErrorContains(t, err, "base branch is required")
	})

	t.Run("title is required", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"head":       "feature",
				"base":       "main",
				"title":      "",
			},
		})

		require.ErrorContains(t, err, "title is required")
	})

	t.Run("head and base must differ", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"head":       "main",
				"base":       "main",
				"title":      "My PR",
			},
		})

		require.ErrorContains(t, err, "head and base branches must be different")
	})
}

func Test__CreatePullRequest__Execute_AssignsWorkOrderAssignees(t *testing.T) {
	component := CreatePullRequest{}

	baseConfig := func(overrides map[string]any) map[string]any {
		config := map[string]any{
			"repository": "hello",
			"head":       "feature",
			"base":       "main",
			"title":      "My PR",
			"orderId":    "wo-1",
		}
		for k, v := range overrides {
			config[k] = v
		}
		return config
	}

	t.Run("does nothing when orderId is not set", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{}
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{createdPullRequestResponse()}}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Factory:        factoryCtx,
			Configuration:  baseConfig(map[string]any{"orderId": ""}),
		})

		require.NoError(t, err)
		assert.Equal(t, 0, factoryCtx.resolveAssigneeAccountsCalls)
		assert.Len(t, httpCtx.Requests, 1, "no assignment call should be made")
	})

	t.Run("assigns resolved GitHub logins to the created pull request", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{
			resolveAssigneeAccountsResult: &core.WorkOrderAssigneeAccounts{Logins: []string{"octocat"}},
		}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				createdPullRequestResponse(),
				mocks.GitHubResponse(http.StatusOK, `{"number": 42, "assignees": [{"login": "octocat"}]}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Factory:        factoryCtx,
			Configuration:  baseConfig(nil),
		})

		require.NoError(t, err)
		assert.Equal(t, "wo-1", factoryCtx.resolveAssigneeAccountsParams.OrderID)
		assert.Equal(t, "github", factoryCtx.resolveAssigneeAccountsParams.Provider)
		assert.Equal(t, 0, factoryCtx.setStatusNoteCalls, "no notice expected when everyone is linked")
		require.Equal(t, 1, factoryCtx.clearStatusNoteCalls, "a stale unlinked notice should be cleared when everyone is linked")
		assert.Equal(t, "wo-1", factoryCtx.clearStatusNoteParams.OrderID)
		assert.Equal(t, assigneeLinkNoticeKey, factoryCtx.clearStatusNoteParams.NoteKey)
		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, "/repos/testhq/hello/issues/42/assignees", httpCtx.Requests[1].URL.Path)
	})

	t.Run("sets a status note when an assignee has no linked GitHub account", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{
			resolveAssigneeAccountsResult: &core.WorkOrderAssigneeAccounts{Unlinked: 1},
		}
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{createdPullRequestResponse()}}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Factory:        factoryCtx,
			Configuration:  baseConfig(nil),
		})

		require.NoError(t, err)
		require.Equal(t, 1, factoryCtx.setStatusNoteCalls)
		assert.Equal(t, 0, factoryCtx.clearStatusNoteCalls, "the notice must not be cleared while an assignee is unlinked")
		assert.Equal(t, "wo-1", factoryCtx.setStatusNoteParams.OrderID)
		assert.Equal(t, assigneeLinkNoticeKey, factoryCtx.setStatusNoteParams.NoteKey)
		assert.NotEmpty(t, factoryCtx.setStatusNoteParams.Headline)
		assert.Len(t, httpCtx.Requests, 1, "no assignment call should be made without a resolved login")
	})

	t.Run("skips an assignee GitHub rejects and still succeeds", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{
			resolveAssigneeAccountsResult: &core.WorkOrderAssigneeAccounts{Logins: []string{"octocat", "hubot"}},
		}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				createdPullRequestResponse(),
				mocks.GitHubResponse(http.StatusUnprocessableEntity, `{"message": "Validation Failed"}`),
				mocks.GitHubResponse(http.StatusOK, `{"number": 42, "assignees": [{"login": "hubot"}]}`),
			},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: executionState,
			Factory:        factoryCtx,
			Configuration:  baseConfig(nil),
		})

		require.NoError(t, err, "a rejected assignee must not fail pull request creation")
		assert.True(t, executionState.Passed)
		require.Len(t, httpCtx.Requests, 3)
	})

	t.Run("does not resolve assignees when the run has no factory context", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{createdPullRequestResponse()}}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  baseConfig(nil),
		})

		require.NoError(t, err)
		assert.Len(t, httpCtx.Requests, 1)
	})

	t.Run("resolution failure does not fail pull request creation", func(t *testing.T) {
		factoryCtx := &fakeFactoryContext{resolveAssigneeAccountsErr: assert.AnError}
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{createdPullRequestResponse()}}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Factory:        factoryCtx,
			Configuration:  baseConfig(nil),
		})

		require.NoError(t, err)
	})
}
