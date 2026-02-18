package bitbucket

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetIssue__Setup(t *testing.T) {
	component := GetIssue{}

	t.Run("repository is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			HTTP:          &contexts.HTTPContext{},
			Integration:   testBitbucketIntegrationContext(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": "", "issueNumber": "42"},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("issue number is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			HTTP:          &contexts.HTTPContext{},
			Integration:   testBitbucketIntegrationContext(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": "hello", "issueNumber": ""},
		})

		require.ErrorContains(t, err, "issue number is required")
	})

	t.Run("repository metadata is set", func(t *testing.T) {
		nodeMetadata := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			HTTP:          testBitbucketRepositoryHTTPContext(),
			Integration:   testBitbucketIntegrationContext(),
			Metadata:      nodeMetadata,
			Configuration: map[string]any{"repository": "hello", "issueNumber": "42"},
		})
		require.NoError(t, err)

		metadata, ok := nodeMetadata.Get().(NodeMetadata)
		require.True(t, ok)
		require.NotNil(t, metadata.Repository)
		assert.Equal(t, "hello", metadata.Repository.Slug)
	})
}

func Test__GetIssue__Execute(t *testing.T) {
	component := GetIssue{}

	t.Run("fails when issue number is not a number", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository":  "hello",
				"issueNumber": "abc",
			},
			Integration: testBitbucketIntegrationContext(),
		})

		require.ErrorContains(t, err, "issue number is not a number")
	})

	t.Run("gets issue and emits payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":42,"title":"Issue title","state":"open"}`)),
				},
			},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    testBitbucketIntegrationContext(),
			NodeMetadata:   testBitbucketNodeMetadataContext(),
			ExecutionState: executionState,
			Configuration: map[string]any{
				"repository":  "hello",
				"issueNumber": "42",
			},
		})
		require.NoError(t, err)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
		assert.Equal(t, "/2.0/repositories/superplane/hello/issues/42", httpContext.Requests[0].URL.Path)
		assert.Equal(t, issuePayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)
	})
}

func Test__CreateIssue__Setup(t *testing.T) {
	component := CreateIssue{}

	t.Run("repository is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			HTTP:          &contexts.HTTPContext{},
			Integration:   testBitbucketIntegrationContext(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": "", "title": "test"},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("title is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			HTTP:          &contexts.HTTPContext{},
			Integration:   testBitbucketIntegrationContext(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": "hello", "title": ""},
		})

		require.ErrorContains(t, err, "title is required")
	})
}

func Test__CreateIssue__Execute(t *testing.T) {
	component := CreateIssue{}

	t.Run("creates issue and emits payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(`{"id":42,"title":"Issue title","state":"new"}`)),
				},
			},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    testBitbucketIntegrationContext(),
			NodeMetadata:   testBitbucketNodeMetadataContext(),
			ExecutionState: executionState,
			Configuration: map[string]any{
				"repository": "hello",
				"title":      "Issue title",
				"body":       "Issue body",
			},
		})
		require.NoError(t, err)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
		assert.Equal(t, "/2.0/repositories/superplane/hello/issues", httpContext.Requests[0].URL.Path)

		bodyBytes, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)

		requestBody := map[string]any{}
		require.NoError(t, json.Unmarshal(bodyBytes, &requestBody))
		assert.Equal(t, "Issue title", requestBody["title"])
		content, ok := requestBody["content"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Issue body", content["raw"])

		assert.Equal(t, issuePayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)
	})
}

func Test__UpdateIssue__Setup(t *testing.T) {
	component := UpdateIssue{}

	t.Run("repository is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			HTTP:          &contexts.HTTPContext{},
			Integration:   testBitbucketIntegrationContext(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": "", "issueNumber": 1},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("issue number is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			HTTP:          &contexts.HTTPContext{},
			Integration:   testBitbucketIntegrationContext(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": "hello", "issueNumber": 0},
		})

		require.ErrorContains(t, err, "issue number is required")
	})
}

func Test__UpdateIssue__Execute(t *testing.T) {
	component := UpdateIssue{}

	t.Run("requires at least one updated field", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			HTTP:         &contexts.HTTPContext{},
			Integration:  testBitbucketIntegrationContext(),
			NodeMetadata: testBitbucketNodeMetadataContext(),
			Configuration: map[string]any{
				"repository":  "hello",
				"issueNumber": 42,
			},
		})

		require.ErrorContains(t, err, "at least one field to update is required")
	})

	t.Run("updates issue and emits payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":42,"title":"Updated title","state":"resolved"}`)),
				},
			},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    testBitbucketIntegrationContext(),
			NodeMetadata:   testBitbucketNodeMetadataContext(),
			ExecutionState: executionState,
			Configuration: map[string]any{
				"repository":  "hello",
				"issueNumber": 42,
				"title":       "Updated title",
				"body":        "Updated body",
				"state":       "resolved",
			},
		})
		require.NoError(t, err)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodPut, httpContext.Requests[0].Method)
		assert.Equal(t, "/2.0/repositories/superplane/hello/issues/42", httpContext.Requests[0].URL.Path)

		bodyBytes, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)

		requestBody := map[string]any{}
		require.NoError(t, json.Unmarshal(bodyBytes, &requestBody))
		assert.Equal(t, "Updated title", requestBody["title"])
		assert.Equal(t, "resolved", requestBody["state"])
		content, ok := requestBody["content"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "Updated body", content["raw"])

		assert.Equal(t, issuePayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)
	})
}

func Test__CreateIssueComment__Setup(t *testing.T) {
	component := CreateIssueComment{}

	t.Run("repository is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			HTTP:          &contexts.HTTPContext{},
			Integration:   testBitbucketIntegrationContext(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": "", "issueNumber": "42", "body": "test"},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("issue number is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			HTTP:          &contexts.HTTPContext{},
			Integration:   testBitbucketIntegrationContext(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": "hello", "issueNumber": "", "body": "test"},
		})

		require.ErrorContains(t, err, "issue number is required")
	})

	t.Run("body is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			HTTP:          &contexts.HTTPContext{},
			Integration:   testBitbucketIntegrationContext(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": "hello", "issueNumber": "42", "body": ""},
		})

		require.ErrorContains(t, err, "body is required")
	})
}

func Test__CreateIssueComment__Execute(t *testing.T) {
	component := CreateIssueComment{}

	t.Run("fails when issue number is not a number", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration: testBitbucketIntegrationContext(),
			Configuration: map[string]any{
				"repository":  "hello",
				"issueNumber": "abc",
				"body":        "test comment",
			},
		})

		require.ErrorContains(t, err, "issue number is not a number")
	})

	t.Run("creates issue comment and emits payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(`{"id":1001,"content":{"raw":"test comment"}}`)),
				},
			},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpContext,
			Integration:    testBitbucketIntegrationContext(),
			NodeMetadata:   testBitbucketNodeMetadataContext(),
			ExecutionState: executionState,
			Configuration: map[string]any{
				"repository":  "hello",
				"issueNumber": "42",
				"body":        "test comment",
			},
		})
		require.NoError(t, err)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
		assert.Equal(t, "/2.0/repositories/superplane/hello/issues/42/comments", httpContext.Requests[0].URL.Path)

		bodyBytes, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)

		requestBody := map[string]any{}
		require.NoError(t, json.Unmarshal(bodyBytes, &requestBody))
		content, ok := requestBody["content"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "test comment", content["raw"])

		assert.Equal(t, issueCommentPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)
	})
}

func testBitbucketIntegrationContext() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		Configuration: map[string]any{
			"token": "token",
		},
		Metadata: Metadata{
			AuthType: AuthTypeWorkspaceAccessToken,
			Workspace: &WorkspaceMetadata{
				Slug: "superplane",
			},
		},
	}
}

func testBitbucketNodeMetadataContext() *contexts.MetadataContext {
	return &contexts.MetadataContext{
		Metadata: NodeMetadata{
			Repository: &RepositoryMetadata{
				UUID:     "{hello}",
				Name:     "hello",
				FullName: "superplane/hello",
				Slug:     "hello",
			},
		},
	}
}

func testBitbucketRepositoryHTTPContext() *contexts.HTTPContext {
	return &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`{"values":[{"uuid":"{hello}","name":"hello","full_name":"superplane/hello","slug":"hello"}]}`,
				)),
			},
		},
	}
}
