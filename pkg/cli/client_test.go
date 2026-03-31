package cli

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMethodSafeRedirectPolicyAllowsSameMethodRedirect(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(target.Close)

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+r.URL.Path, http.StatusMovedPermanently)
	}))
	t.Cleanup(redirector.Close)

	client := &http.Client{CheckRedirect: methodSafeRedirectPolicy()}

	resp, err := client.Get(redirector.URL + "/test")
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestMethodSafeRedirectPolicyBlocksPostToGetRedirect(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(target.Close)

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+r.URL.Path, http.StatusMovedPermanently)
	}))
	t.Cleanup(redirector.Close)

	client := &http.Client{CheckRedirect: methodSafeRedirectPolicy()}

	req, err := http.NewRequest(http.MethodPost, redirector.URL+"/api/v1/canvases", strings.NewReader(`{}`))
	require.NoError(t, err)

	_, err = client.Do(req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "refusing to follow redirect that changes method")
	require.Contains(t, err.Error(), "try https://")
}

func TestMethodSafeRedirectPolicyAllowsMethodPreservingRedirect(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(target.Close)

	// 307 preserves the original method
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+r.URL.Path, http.StatusTemporaryRedirect)
	}))
	t.Cleanup(redirector.Close)

	client := &http.Client{CheckRedirect: methodSafeRedirectPolicy()}

	req, err := http.NewRequest(http.MethodPost, redirector.URL+"/api/v1/canvases", strings.NewReader(`{}`))
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
