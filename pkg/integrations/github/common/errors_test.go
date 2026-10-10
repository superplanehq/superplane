package common

import (
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
)

func TestExplainGitHubError_Nil(t *testing.T) {
	assert.Nil(t, ExplainGitHubError(nil))
}

func TestExplainGitHubError_UnrelatedError(t *testing.T) {
	orig := errors.New("database connection refused")
	result := ExplainGitHubError(orig)
	assert.Equal(t, orig, result)
}

func TestExplainGitHubError_GitHubErrorResponse(t *testing.T) {
	t.Run("single message", func(t *testing.T) {
		ghErr := &github.ErrorResponse{
			Message: "Resource not found",
		}
		result := ExplainGitHubError(ghErr)
		require.Error(t, result)
		assert.Equal(t, "Resource not found", result.Error())

		var target *github.ErrorResponse
		assert.True(t, errors.As(result, &target))
		assert.Equal(t, ghErr, target)
	})

	t.Run("validation errors unpacked", func(t *testing.T) {
		ghErr := &github.ErrorResponse{
			Message: "Validation Failed",
			Errors: []github.Error{
				{Message: "A pull request already exists for testhq:feature."},
			},
		}
		result := ExplainGitHubError(ghErr)
		require.Error(t, result)
		assert.Equal(t, "Validation Failed: A pull request already exists for testhq:feature.", result.Error())
	})

	t.Run("empty message returns original error", func(t *testing.T) {
		ghErr := &github.ErrorResponse{}
		result := ExplainGitHubError(ghErr)
		assert.Equal(t, ghErr, result)
	})

	t.Run("wrapped error response", func(t *testing.T) {
		ghErr := &github.ErrorResponse{
			Message: "Pull request already merged",
		}
		wrapped := fmt.Errorf("upstream error: %w", ghErr)
		result := ExplainGitHubError(wrapped)
		require.Error(t, result)
		assert.Equal(t, "Pull request already merged", result.Error())
	})
}

func TestExplainGitHubError_HTTPError(t *testing.T) {
	t.Run("nil response surfaces error message", func(t *testing.T) {
		httpErr := &ghinstallation.HTTPError{
			InstallationID: 168860812,
			Message:        "could not connect to GitHub API",
		}
		result := ExplainGitHubError(httpErr)
		require.Error(t, result)
		assert.Contains(t, result.Error(), "could not connect to GitHub API")

		var target *ghinstallation.HTTPError
		assert.True(t, errors.As(result, &target))
		assert.Equal(t, int64(168860812), target.InstallationID)
	})

	t.Run("nil response with root cause", func(t *testing.T) {
		httpErr := &ghinstallation.HTTPError{
			InstallationID: 168860812,
			RootCause:      errors.New("i/o timeout"),
		}
		result := ExplainGitHubError(httpErr)
		require.Error(t, result)
		assert.Contains(t, result.Error(), "i/o timeout")
	})

	t.Run("403 suspended installation surfaces status, response body, message and hint", func(t *testing.T) {
		body := `{"message":"This installation has been suspended.","documentation_url":"https://docs.github.com/rest/apps/apps#create-an-installation-access-token-for-an-app"}`
		httpErr := &ghinstallation.HTTPError{
			InstallationID: 168860812,
			Message:        `received non 2xx response status "403 Forbidden" when fetching https://api.github.com/app/installations/168860812/access_tokens`,
			Response: &http.Response{
				StatusCode: http.StatusForbidden,
				Status:     "403 Forbidden",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(body)),
			},
		}

		result := ExplainGitHubError(httpErr)
		require.Error(t, result)

		errStr := result.Error()
		assert.Contains(t, errStr, "GitHub installation token mint failed")
		assert.Contains(t, errStr, "status 403 Forbidden")
		assert.Contains(t, errStr, "This installation has been suspended.")
		assert.Contains(t, errStr, "hint: the GitHub App installation is suspended")
		assert.Contains(t, errStr, fmt.Sprintf("response body: %s", body))

		// Check unwrap and status code helpers
		assert.Equal(t, http.StatusForbidden, StatusCode(result))
		assert.True(t, IsForbiddenError(result))

		var target *ghinstallation.HTTPError
		require.True(t, errors.As(result, &target))
		assert.Equal(t, int64(168860812), target.InstallationID)

		// Check response body is re-readable
		remainingBody, err := io.ReadAll(target.Response.Body)
		require.NoError(t, err)
		assert.Equal(t, body, string(remainingBody))
	})

	t.Run("403 IP allow list surfaces status, body, message and hint", func(t *testing.T) {
		body := `{"message":"You must be on the organization's IP allow list to perform this action.","documentation_url":"https://docs.github.com/rest/apps/apps"}`
		httpErr := &ghinstallation.HTTPError{
			InstallationID: 168860812,
			Message:        `received non 2xx response status "403 Forbidden"`,
			Response: &http.Response{
				StatusCode: http.StatusForbidden,
				Status:     "403 Forbidden",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(body)),
			},
		}

		result := ExplainGitHubError(httpErr)
		require.Error(t, result)

		errStr := result.Error()
		assert.Contains(t, errStr, "status 403 Forbidden")
		assert.Contains(t, errStr, "You must be on the organization's IP allow list to perform this action.")
		assert.Contains(t, errStr, "hint: organization IP allow list policy blocked token request")
		assert.Contains(t, errStr, body)
		assert.True(t, IsForbiddenError(result))
	})

	t.Run("404 installation not found surfaces status, body, message and hint", func(t *testing.T) {
		body := `{"message":"Not Found","documentation_url":"https://docs.github.com/rest/apps/apps"}`
		httpErr := &ghinstallation.HTTPError{
			InstallationID: 168860812,
			Message:        `received non 2xx response status "404 Not Found"`,
			Response: &http.Response{
				StatusCode: http.StatusNotFound,
				Status:     "404 Not Found",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(body)),
			},
		}

		result := ExplainGitHubError(httpErr)
		require.Error(t, result)

		errStr := result.Error()
		assert.Contains(t, errStr, "status 404 Not Found")
		assert.Contains(t, errStr, "Not Found")
		assert.Contains(t, errStr, "hint: the GitHub App installation was not found or has been uninstalled")
		assert.Contains(t, errStr, body)
		assert.True(t, IsNotFoundError(result))
	})

	t.Run("403 resource not accessible surfaces permission hint", func(t *testing.T) {
		body := `{"message":"Resource not accessible by integration"}`
		httpErr := &ghinstallation.HTTPError{
			InstallationID: 168860812,
			Response: &http.Response{
				StatusCode: http.StatusForbidden,
				Status:     "403 Forbidden",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(body)),
			},
		}

		result := ExplainGitHubError(httpErr)
		require.Error(t, result)
		assert.Contains(t, result.Error(), "hint: the GitHub App lacks required permissions for this repository")
	})

	t.Run("plain text response body", func(t *testing.T) {
		body := "502 Bad Gateway"
		httpErr := &ghinstallation.HTTPError{
			InstallationID: 168860812,
			Response: &http.Response{
				StatusCode: http.StatusBadGateway,
				Status:     "502 Bad Gateway",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(body)),
			},
		}

		result := ExplainGitHubError(httpErr)
		require.Error(t, result)
		assert.Contains(t, result.Error(), "status 502 Bad Gateway")
		assert.Contains(t, result.Error(), "502 Bad Gateway")
	})

	t.Run("empty response body falls back to library message", func(t *testing.T) {
		httpErr := &ghinstallation.HTTPError{
			InstallationID: 168860812,
			Message:        `received non 2xx response status "500 Internal Server Error"`,
			Response: &http.Response{
				StatusCode: http.StatusInternalServerError,
				Status:     "500 Internal Server Error",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("")),
			},
		}

		result := ExplainGitHubError(httpErr)
		require.Error(t, result)
		assert.Contains(t, result.Error(), "status 500 Internal Server Error")
		assert.Contains(t, result.Error(), `received non 2xx response status "500 Internal Server Error"`)
	})

	t.Run("wrapped in standard error chain", func(t *testing.T) {
		body := `{"message":"This installation has been suspended."}`
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

		result := ExplainGitHubError(wrapped)
		require.Error(t, result)
		assert.Contains(t, result.Error(), "This installation has been suspended.")
		assert.Contains(t, result.Error(), body)

		var target *ghinstallation.HTTPError
		assert.True(t, errors.As(result, &target))
		assert.Equal(t, int64(168860812), target.InstallationID)
	})
}
