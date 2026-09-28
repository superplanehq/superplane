package github

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	gh "github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestFilterWritableRepositories(t *testing.T) {
	repositories := []common.Repository{
		{ID: 1, Name: "api"},
		{ID: 2, Name: "web"},
		{ID: 3, Name: "docs"},
	}
	permissions := map[string]string{"api": "admin", "web": "write", "docs": "read"}

	writable, err := filterWritableRepositories(
		context.Background(),
		"acme",
		"member",
		repositories,
		func(_ context.Context, owner, repository, username string) (*gh.RepositoryPermissionLevel, error) {
			assert.Equal(t, "acme", owner)
			assert.Equal(t, "member", username)
			return &gh.RepositoryPermissionLevel{Permission: gh.Ptr(permissions[repository])}, nil
		},
	)

	require.NoError(t, err)
	assert.Equal(t, []common.Repository{
		{ID: 1, Name: "acme/api"},
		{ID: 2, Name: "acme/web"},
	}, writable)
}

func TestFilterWritableRepositoriesFailsClosed(t *testing.T) {
	repositories := []common.Repository{{ID: 1, Name: "api"}}

	writable, err := filterWritableRepositories(
		context.Background(),
		"acme",
		"member",
		repositories,
		func(context.Context, string, string, string) (*gh.RepositoryPermissionLevel, error) {
			return nil, errors.New("GitHub unavailable")
		},
	)

	assert.Error(t, err)
	assert.Empty(t, writable)
}

func TestHostedDiscoveryErrorIsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{name: "no error", retryable: false},
		{name: "canceled", err: context.Canceled, retryable: true},
		{name: "deadline", err: context.DeadlineExceeded, retryable: true},
		{name: "network", err: &net.DNSError{IsTimeout: true}, retryable: true},
		{name: "primary rate limit", err: &gh.RateLimitError{}, retryable: true},
		{name: "secondary rate limit", err: &gh.AbuseRateLimitError{}, retryable: true},
		{
			name:      "request timeout",
			err:       &gh.ErrorResponse{Response: &http.Response{StatusCode: http.StatusRequestTimeout}},
			retryable: true,
		},
		{
			name:      "too many requests",
			err:       &gh.ErrorResponse{Response: &http.Response{StatusCode: http.StatusTooManyRequests}},
			retryable: true,
		},
		{
			name:      "server error",
			err:       &gh.ErrorResponse{Response: &http.Response{StatusCode: http.StatusBadGateway}},
			retryable: true,
		},
		{
			name: "revoked installation",
			err:  &gh.ErrorResponse{Response: &http.Response{StatusCode: http.StatusUnauthorized}},
		},
		{
			name: "permanent then transient errors",
			err: errors.Join(
				&gh.ErrorResponse{Response: &http.Response{StatusCode: http.StatusForbidden}},
				&gh.ErrorResponse{Response: &http.Response{StatusCode: http.StatusBadGateway}},
			),
			retryable: true,
		},
		{name: "permanent error", err: errors.New("revoked")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.retryable, hostedDiscoveryErrorIsRetryable(test.err))
		})
	}
}

func TestDiscoverAccessibleInstallationsUsesExplicitUnverifiedAccess(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		return []common.PendingInstallation{{ID: "11", AccountLogin: "acme", AccountType: "Organization"}}, nil
	}
	newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listInstallationRepos = func(context.Context, *gh.Client) ([]common.Repository, error) {
		return []common.Repository{{ID: 101, Name: "api"}, {ID: 102, Name: "web"}}, nil
	}

	installations, err := discoverAccessibleInstallations(
		context.Background(),
		&contexts.IntegrationContext{},
		common.HostedApp{ID: 99},
		hostedGitHubIdentity{Login: "devuser", AllowUnverifiedRepositories: true},
	)

	require.NoError(t, err)
	require.Len(t, installations, 1)
	assert.Equal(t, "acme", installations[0].AccountLogin)
	assert.Equal(t, []common.Repository{{ID: 101, Name: "acme/api"}, {ID: 102, Name: "acme/web"}}, installations[0].Repositories)
}

func TestDiscoverAccessibleInstallationsLimitsConcurrentChecks(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	installations := make([]common.PendingInstallation, 20)
	for i := range installations {
		installations[i] = common.PendingInstallation{ID: string(rune('a' + i)), AccountLogin: "acme"}
	}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		return installations, nil
	}
	newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	var active atomic.Int32
	var maximum atomic.Int32
	listInstallationRepos = func(context.Context, *gh.Client) ([]common.Repository, error) {
		current := active.Add(1)
		for {
			observed := maximum.Load()
			if current <= observed || maximum.CompareAndSwap(observed, current) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		active.Add(-1)
		return []common.Repository{{ID: 1, Name: "api"}}, nil
	}

	result, err := discoverAccessibleInstallations(
		context.Background(),
		&contexts.IntegrationContext{},
		common.HostedApp{ID: 99},
		hostedGitHubIdentity{Login: "devuser", AllowUnverifiedRepositories: true},
	)

	require.NoError(t, err)
	assert.Len(t, result, len(installations))
	assert.Greater(t, maximum.Load(), int32(1))
	assert.LessOrEqual(t, maximum.Load(), int32(hostedInstallationVerificationConcurrency))
}

func TestDiscoverAccessibleInstallationsReturnsOneCancellationError(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		return []common.PendingInstallation{{ID: "11"}, {ID: "22"}}, nil
	}
	var clientCalls atomic.Int32
	newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
		clientCalls.Add(1)
		return gh.NewClient(nil), nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := discoverAccessibleInstallations(
		ctx,
		&contexts.IntegrationContext{},
		common.HostedApp{ID: 99},
		hostedGitHubIdentity{Login: "devuser", AllowUnverifiedRepositories: true},
	)

	assert.Empty(t, result)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, int32(0), clientCalls.Load())
	assert.Equal(t, 1, strings.Count(err.Error(), context.Canceled.Error()))
}

func TestDiscoverAccessibleInstallationsBoundsFailureSummary(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	installations := make([]common.PendingInstallation, 10)
	for i := range installations {
		installations[i] = common.PendingInstallation{ID: string(rune('a' + i))}
	}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		return installations, nil
	}
	newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
		return nil, errors.New("unavailable")
	}

	_, err := discoverAccessibleInstallations(
		context.Background(),
		&contexts.IntegrationContext{},
		common.HostedApp{ID: 99},
		hostedGitHubIdentity{Login: "devuser", AllowUnverifiedRepositories: true},
	)

	require.Error(t, err)
	assert.LessOrEqual(t, strings.Count(err.Error(), "unavailable"), maxHostedDiscoveryErrors)
	assert.Contains(t, err.Error(), "additional installation checks failed")
}

func TestDiscoverAccessibleInstallationByIDRejectsReadOnlyRepository(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	getAppInstallation = func(context.Context, *gh.Client, int64) (*gh.Installation, error) {
		return &gh.Installation{
			ID:      gh.Ptr(int64(11)),
			Account: &gh.User{Login: gh.Ptr("acme"), Type: gh.Ptr("Organization")},
		}, nil
	}
	newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	resolveInstallationIdentity = func(context.Context, *gh.Client, int64) (*gh.User, error) {
		return &gh.User{ID: gh.Ptr(int64(7)), Login: gh.Ptr("member")}, nil
	}
	listInstallationRepos = func(context.Context, *gh.Client) ([]common.Repository, error) {
		return []common.Repository{{ID: 101, Name: "api"}}, nil
	}
	getRepositoryPermission = func(context.Context, *gh.Client, string, string, string) (*gh.RepositoryPermissionLevel, error) {
		return &gh.RepositoryPermissionLevel{Permission: gh.Ptr("read")}, nil
	}

	installations, err := discoverAccessibleInstallationByID(
		context.Background(),
		&contexts.IntegrationContext{},
		common.HostedApp{ID: 99},
		hostedGitHubIdentity{ID: 7, Login: "member"},
		"11",
		nil,
	)

	require.NoError(t, err)
	assert.Empty(t, installations)
}

func TestDiscoverAccessibleInstallationsResolvesIdentityOnce(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Cleanup(resetBindClientHooks)

	newAppJWTClient = func(core.IntegrationContext, int64) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	listAppInstallations = func(context.Context, *gh.Client) ([]common.PendingInstallation, error) {
		return []common.PendingInstallation{
			{ID: "11", AccountLogin: "acme"},
			{ID: "22", AccountLogin: "octo"},
		}, nil
	}
	newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	var identityCalls atomic.Int32
	resolveInstallationIdentity = func(context.Context, *gh.Client, int64) (*gh.User, error) {
		identityCalls.Add(1)
		return &gh.User{ID: gh.Ptr(int64(7)), Login: gh.Ptr("renamed-member")}, nil
	}
	listInstallationRepos = func(context.Context, *gh.Client) ([]common.Repository, error) {
		return []common.Repository{{ID: 101, Name: "api"}}, nil
	}
	getRepositoryPermission = func(_ context.Context, _ *gh.Client, _, _, username string) (*gh.RepositoryPermissionLevel, error) {
		assert.Equal(t, "renamed-member", username)
		return &gh.RepositoryPermissionLevel{Permission: gh.Ptr("write")}, nil
	}

	installations, err := discoverAccessibleInstallations(
		context.Background(),
		&contexts.IntegrationContext{},
		common.HostedApp{ID: 99},
		hostedGitHubIdentity{ID: 7, Login: "old-member"},
	)

	require.NoError(t, err)
	assert.Len(t, installations, 2)
	assert.Equal(t, int32(1), identityCalls.Load())
}

func TestDevelopmentGitHubDiscoveryRequiresExplicitLocalOptIn(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv(allowUnverifiedDevelopmentRepositoriesEnv, "")
	assert.False(t, useDevelopmentGitHubDiscovery())

	t.Setenv(allowUnverifiedDevelopmentRepositoriesEnv, "yes")
	assert.True(t, useDevelopmentGitHubDiscovery())

	t.Setenv("APP_ENV", "production")
	assert.False(t, useDevelopmentGitHubDiscovery())
}

func TestHasRepositoryWritePermission(t *testing.T) {
	assert.True(t, hasRepositoryWritePermission("admin"))
	assert.True(t, hasRepositoryWritePermission("write"))
	assert.False(t, hasRepositoryWritePermission("read"))
	assert.False(t, hasRepositoryWritePermission("none"))
}

func TestRetainInstalledRepositoriesDoesNotGrantNewRepositories(t *testing.T) {
	granted := []common.Repository{{ID: 1, Name: "acme/api"}, {ID: 2, Name: "acme/web"}}
	installed := []common.Repository{{ID: 2, Name: "web"}, {ID: 3, Name: "new"}}

	assert.Equal(t, []common.Repository{{ID: 2, Name: "acme/web"}}, retainInstalledRepositories(granted, installed))
}

func TestInstallationRepositoryEventRestoresSelectedRepository(t *testing.T) {
	t.Cleanup(resetBindClientHooks)
	selected := common.Repository{ID: 1, Name: "acme/api"}
	integration := &contexts.IntegrationContext{
		State: "ready",
		Metadata: common.Metadata{
			InstallationID:       "11",
			Owner:                "acme",
			Repositories:         []common.Repository{selected},
			SelectedRepositories: []common.Repository{selected},
			RepositoryScoped:     true,
			GitHubApp:            common.GitHubAppMetadata{ID: 99},
		},
	}
	newInstallationClient = func(core.IntegrationContext, int64, string) (*gh.Client, error) {
		return gh.NewClient(nil), nil
	}
	installed := []common.Repository{}
	listInstallationRepos = func(context.Context, *gh.Client) ([]common.Repository, error) {
		return installed, nil
	}
	ctx, recorder := hostedRequestContext(integration, "/api/v1/github/app/webhook", nil)
	event := &gh.InstallationRepositoriesEvent{}

	(&GitHub{}).handleInstallationRepositoriesEvent(ctx, event)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "error", integration.State)
	metadata := integration.Metadata.(common.Metadata)
	assert.Empty(t, metadata.Repositories)
	assert.Equal(t, []common.Repository{selected}, metadata.SelectedRepositories)

	installed = []common.Repository{{ID: 1, Name: "api"}}
	(&GitHub{}).handleInstallationRepositoriesEvent(ctx, event)

	assert.Equal(t, "ready", integration.State)
	metadata = integration.Metadata.(common.Metadata)
	assert.Equal(t, []common.Repository{selected}, metadata.Repositories)
	assert.Equal(t, []common.Repository{selected}, metadata.SelectedRepositories)
}
