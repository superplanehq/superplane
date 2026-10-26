package mcp_test

import (
	"context"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/mcp"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestRunnerMCPRefreshDoesNotHoldDatabaseLocksDuringProviderCall(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	resource := connectOAuthResource(t, db, r, time.Now().Add(-time.Minute), "expired-access", "refresh-1")
	entered, release := make(chan struct{}), make(chan struct{})
	releaseProvider := sync.OnceFunc(func() { close(release) })
	client := tokenEndpointClient(t, func(w http.ResponseWriter, request *http.Request) {
		close(entered)
		select {
		case <-release:
		case <-request.Context().Done():
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"new-access","refresh_token":"refresh-2","expires_in":7200,"token_type":"Bearer"}`)
	})
	finished := make(chan error, 1)
	connectionPID := make(chan int, 1)
	go func() {
		finished <- db.Connection(func(connection *gorm.DB) error {
			var pid int
			if err := connection.Raw("SELECT pg_backend_pid()").Scan(&pid).Error; err != nil {
				return err
			}
			connectionPID <- pid
			_, err := mcp.MintFactoryAgentResourceAccessTokenForRunner(t.Context(), r.Encryptor, client, connection, resource, time.Hour)
			return err
		})
	}()
	defer releaseProvider()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("provider refresh did not start")
	}
	pid := <-connectionPID
	var advisoryLocks int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM pg_locks WHERE pid = ? AND locktype = 'advisory'", pid).Scan(&advisoryLocks).Error)
	require.Zero(t, advisoryLocks, "the provider call must run after transaction advisory locks are released")
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	changed := resource.Config.Data()
	changed.URL = "https://changed.example.com/mcp"
	// A separate connection must be able to change the resource while HTTP is
	// blocked. The delayed result must not be applied to its new recipient.
	require.NoError(t, db.WithContext(ctx).Model(&models.FactoryAgentResource{}).Where("id = ?", resource.ID).
		Update("config", datatypes.NewJSONType(changed)).Error)
	releaseProvider()
	select {
	case err := <-finished:
		require.ErrorIs(t, err, mcp.ErrFactoryAgentResourceChanged)
	case <-time.After(5 * time.Second):
		t.Fatal("changed resource did not fence the refresh result")
	}
	saved := reloadResource(t, db, resource)
	require.Equal(t, "refresh-1", decryptedSecret(t, r, db, saved, models.FactoryAgentResourceSecretRefreshToken))
}

func TestRunnerMCPRefreshCoordinatesConcurrentRunnerAndLegacyCalls(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	resource := connectOAuthResource(t, db, r, time.Now().Add(-time.Minute), "expired-access", "refresh-1")
	otherRunner := reloadResource(t, db, resource)
	legacyResource := reloadResource(t, db, resource)
	entered, release := make(chan struct{}), make(chan struct{})
	releaseProvider := sync.OnceFunc(func() { close(release) })
	var calls atomic.Int32
	client := tokenEndpointClient(t, func(w http.ResponseWriter, request *http.Request) {
		if calls.Add(1) != 1 {
			http.Error(w, "concurrent token rotation", http.StatusConflict)
			return
		}
		close(entered)
		select {
		case <-release:
		case <-request.Context().Done():
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"shared-access","refresh_token":"refresh-2","expires_in":7200,"token_type":"Bearer"}`)
	})
	type result struct {
		token string
		err   error
	}
	runnerResult, legacyResult := make(chan result, 1), make(chan result, 1)
	go func() {
		token, err := mcp.MintFactoryAgentResourceAccessTokenForRunner(t.Context(), r.Encryptor, client, db, resource, time.Hour)
		runnerResult <- result{token, err}
	}()
	defer releaseProvider()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("provider refresh did not start")
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	_, err := mcp.MintFactoryAgentResourceAccessTokenForRunner(ctx, r.Encryptor, client, db, otherRunner, time.Hour)
	cancel()
	require.ErrorIs(t, err, mcp.ErrFactoryAgentResourceRefreshBusy)
	legacyStarted := make(chan struct{})
	go func() {
		close(legacyStarted)
		token, err := mcp.MintFactoryAgentResourceAccessToken(t.Context(), r.Encryptor, client, db, legacyResource)
		legacyResult <- result{token, err}
	}()
	<-legacyStarted
	select {
	case early := <-legacyResult:
		t.Fatalf("legacy call did not wait for the refresh lease: %v", early.err)
	case <-time.After(100 * time.Millisecond):
	}
	releaseProvider()
	for _, response := range []<-chan result{runnerResult, legacyResult} {
		select {
		case got := <-response:
			require.NoError(t, got.err)
			require.Equal(t, "shared-access", got.token)
		case <-time.After(5 * time.Second):
			t.Fatal("refresh lease did not release its waiting caller")
		}
	}
	require.EqualValues(t, 1, calls.Load())
}

func TestRunnerMCPRefreshRejectsChangedSavedTokenEndpointBeforeHTTP(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	resource := connectOAuthResource(t, db, r, time.Now().Add(-time.Minute), "expired-access", "refresh-1")
	changed := resource.OAuthMetadata.Data()
	changed.TokenEndpoint = "https://other.example.com/token"
	require.NoError(t, db.Model(&models.FactoryAgentResource{}).Where("id = ?", resource.ID).
		Update("oauth_metadata", datatypes.NewJSONType(changed)).Error)
	var calls atomic.Int32
	client := tokenEndpointClient(t, func(w http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		http.Error(w, "a replacement endpoint must not receive the saved grant", http.StatusInternalServerError)
	})
	_, err := mcp.MintFactoryAgentResourceAccessTokenForRunner(t.Context(), r.Encryptor, client, db, resource, time.Hour)
	require.ErrorIs(t, err, mcp.ErrFactoryAgentResourceChanged)
	require.Zero(t, calls.Load())
}

func TestExpiredMCPRefreshLeaseFencesDelayedProviderResult(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	resource := connectOAuthResource(t, db, r, time.Now().Add(-time.Minute), "expired-access", "refresh-1")
	successor := reloadResource(t, db, resource)
	entered, release := make(chan struct{}), make(chan struct{})
	releaseProvider := sync.OnceFunc(func() { close(release) })
	defer releaseProvider()
	var calls atomic.Int32
	client := tokenEndpointClient(t, func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if calls.Add(1) == 1 {
			close(entered)
			select {
			case <-release:
			case <-request.Context().Done():
				return
			}
			_, _ = io.WriteString(w, `{"access_token":"stale-access","refresh_token":"stale-refresh","expires_in":7200,"token_type":"Bearer"}`)
			return
		}
		_, _ = io.WriteString(w, `{"access_token":"winner-access","refresh_token":"winner-refresh","expires_in":7200,"token_type":"Bearer"}`)
	})
	finished := make(chan error, 1)
	go func() {
		_, err := mcp.MintFactoryAgentResourceAccessTokenForRunner(t.Context(), r.Encryptor, client, db, resource, time.Hour)
		finished <- err
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("provider refresh did not start")
	}
	// Advance persisted expiry to represent an abandoned gateway attempt.
	// A successor can claim the same resource while its old HTTP result is late.
	require.NoError(t, db.Model(&models.FactoryAgentResourceRefresh{}).Where("resource_id = ?", resource.ID).
		Update("expires_at", time.Now().Add(-time.Second)).Error)
	token, err := mcp.MintFactoryAgentResourceAccessTokenForRunner(t.Context(), r.Encryptor, client, db, successor, time.Hour)
	require.NoError(t, err)
	require.Equal(t, "winner-access", token)
	releaseProvider()
	select {
	case err := <-finished:
		require.ErrorIs(t, err, mcp.ErrFactoryAgentResourceRefreshBusy)
	case <-time.After(5 * time.Second):
		t.Fatal("superseded refresh did not stop")
	}
	saved := reloadResource(t, db, successor)
	require.Equal(t, "winner-access", decryptedSecret(t, r, db, saved, models.FactoryAgentResourceSecretAccessToken))
	require.Equal(t, "winner-refresh", decryptedSecret(t, r, db, saved, models.FactoryAgentResourceSecretRefreshToken))
	require.EqualValues(t, 2, calls.Load())
}
