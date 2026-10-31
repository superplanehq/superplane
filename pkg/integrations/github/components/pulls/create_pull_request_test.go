package pulls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

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

	t.Run("fails when token minting fails with response body", func(t *testing.T) {
		body := `{"message":"This installation has been suspended."}`
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusForbidden, body),
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForLegacySetupFlow(testRSAPEM(t)),
			HTTP:           httpCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"head":       "feature",
				"base":       "main",
				"title":      "My PR",
			},
		})

		require.ErrorContains(t, err, "failed to create pull request")
		require.ErrorContains(t, err, "status 403 Forbidden")
		require.ErrorContains(t, err, "This installation has been suspended.")
		require.ErrorContains(t, err, "hint: the GitHub App installation is suspended")
		require.ErrorContains(t, err, body)
	})
}

func testRSAPEM(t *testing.T) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

func Test__ExplainGitHubError(t *testing.T) {
	t.Run("unwraps github.ErrorResponse", func(t *testing.T) {
		err := &github.ErrorResponse{
			Message: "Validation Failed",
			Errors: []github.Error{
				{Message: "A pull request already exists for testhq:feature."},
			},
		}
		explained := explainGitHubError(err)
		require.Error(t, explained)
		assert.Equal(t, "Validation Failed: A pull request already exists for testhq:feature.", explained.Error())
	})

	t.Run("unwraps ghinstallation.HTTPError and surfaces status and body", func(t *testing.T) {
		body := `{"message":"This installation has been suspended.","documentation_url":"https://docs.github.com"}`
		httpErr := &ghinstallation.HTTPError{
			InstallationID: 168860812,
			Response: &http.Response{
				StatusCode: http.StatusForbidden,
				Status:     "403 Forbidden",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(body)),
			},
		}
		wrapped := fmt.Errorf("could not refresh installation id 168860812's token: %w", httpErr)
		explained := explainGitHubError(wrapped)
		require.Error(t, explained)
		assert.Contains(t, explained.Error(), "status 403 Forbidden")
		assert.Contains(t, explained.Error(), "This installation has been suspended.")
		assert.Contains(t, explained.Error(), "hint: the GitHub App installation is suspended")
		assert.Contains(t, explained.Error(), body)

		var target *ghinstallation.HTTPError
		assert.True(t, errors.As(explained, &target))
	})

	t.Run("preserves unrelated error", func(t *testing.T) {
		err := errors.New("context deadline exceeded")
		assert.Equal(t, err, explainGitHubError(err))
	})
}

