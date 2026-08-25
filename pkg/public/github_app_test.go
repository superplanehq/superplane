package public

import (
	"net/http"
	"net/http/httptest"
	"testing"

	gh "github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__HandleGitHubAppSetup_missingState(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/github/app/setup", nil)
	rec := httptest.NewRecorder()

	server.HandleGitHubAppSetup(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func Test__githubInstallationID(t *testing.T) {
	id := int64(42)
	got, ok := githubInstallationID(&gh.InstallationEvent{
		Installation: &gh.Installation{ID: &id},
	})
	require.True(t, ok)
	assert.Equal(t, "42", got)

	_, ok = githubInstallationID(&gh.PushEvent{})
	assert.False(t, ok)
}
