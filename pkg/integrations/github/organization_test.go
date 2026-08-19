package github

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeHTTPContext struct {
	statusCode int
	err        error
	requestURL string
}

func (f *fakeHTTPContext) Do(request *http.Request) (*http.Response, error) {
	f.requestURL = request.URL.String()
	if f.err != nil {
		return nil, f.err
	}

	return &http.Response{
		StatusCode: f.statusCode,
		Body:       io.NopCloser(strings.NewReader("{}")),
	}, nil
}

func Test__ValidateOrganizationName(t *testing.T) {
	t.Run("accepts a plain organization login", func(t *testing.T) {
		require.NoError(t, validateOrganizationName("superplanehq"))
	})

	t.Run("accepts hyphens between characters", func(t *testing.T) {
		require.NoError(t, validateOrganizationName("super-plane-hq"))
		require.NoError(t, validateOrganizationName("a1-2b"))
	})

	t.Run("rejects a URL", func(t *testing.T) {
		err := validateOrganizationName("https://github.com/superplanehq")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "enter only the organization name")
	})

	t.Run("rejects a repository path", func(t *testing.T) {
		err := validateOrganizationName("superplanehq/superplane")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "enter only the organization name")
	})

	t.Run("rejects a name that is too long", func(t *testing.T) {
		err := validateOrganizationName(strings.Repeat("a", maxOrganizationNameLength+1))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "39 characters or less")
	})

	t.Run("rejects leading, trailing, and repeated hyphens", func(t *testing.T) {
		for _, name := range []string{"-superplanehq", "superplanehq-", "super--plane"} {
			err := validateOrganizationName(name)
			require.Error(t, err, name)
			assert.Contains(t, err.Error(), "single hyphens")
		}
	})

	t.Run("rejects spaces and other characters", func(t *testing.T) {
		for _, name := range []string{"super plane", "super_plane", "super.plane", ""} {
			require.Error(t, validateOrganizationName(name), name)
		}
	})
}

func Test__CheckOrganizationExists(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	t.Run("accepts an organization that GitHub knows", func(t *testing.T) {
		httpCtx := &fakeHTTPContext{statusCode: http.StatusOK}

		require.NoError(t, checkOrganizationExists(httpCtx, logger, "superplanehq"))
		assert.Equal(t, "https://api.github.com/orgs/superplanehq", httpCtx.requestURL)
	})

	t.Run("rejects an organization that GitHub does not know", func(t *testing.T) {
		httpCtx := &fakeHTTPContext{statusCode: http.StatusNotFound}

		err := checkOrganizationExists(httpCtx, logger, "no-such-org")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no GitHub organization named no-such-org exists")
		assert.Contains(t, err.Error(), "owner of the organization")
	})

	t.Run("continues when GitHub cannot be reached", func(t *testing.T) {
		httpCtx := &fakeHTTPContext{err: fmt.Errorf("connection refused")}

		require.NoError(t, checkOrganizationExists(httpCtx, logger, "superplanehq"))
	})

	t.Run("continues when GitHub answers with a rate limit", func(t *testing.T) {
		httpCtx := &fakeHTTPContext{statusCode: http.StatusForbidden}

		require.NoError(t, checkOrganizationExists(httpCtx, logger, "superplanehq"))
	})
}
