package bitbucketapp

import (
	"context"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyCacheRejectsExpiredKeysWhenRefreshFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	cached := &rsa.PublicKey{}
	cache := &keyCache{
		url:       server.URL,
		keys:      map[string]*rsa.PublicKey{"kid-1": cached},
		fetchedAt: time.Now().Add(-2 * jwksTTL),
	}

	key, err := cache.key(context.Background(), "kid-1")
	require.Error(t, err)
	assert.Nil(t, key)
}

func TestKeyCacheKeepsFreshKeysWhenRefreshWouldFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	cached := &rsa.PublicKey{}
	cache := &keyCache{
		url:       server.URL,
		keys:      map[string]*rsa.PublicKey{"kid-1": cached},
		fetchedAt: time.Now(),
	}

	key, err := cache.key(context.Background(), "kid-1")
	require.NoError(t, err)
	assert.Same(t, cached, key)
}
