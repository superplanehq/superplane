package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/superplane/runner/shared/api"
)

func TestResolveAccessTokenPersistsAndReusesFile(t *testing.T) {
	var registerCalls int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		registerCalls++
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(api.RegisterRunnerResponse{AccessToken: "persisted-token"})
	}))
	defer ts.Close()

	path := filepath.Join(t.TempDir(), "access_token")
	token, err := ResolveAccessToken(context.Background(), ts.Client(), ResolveAccessTokenOptions{
		BaseURL:           ts.URL,
		RegistrationToken: "registration-jwt",
		RunnerID:          "i-1",
		FleetID:           "fleet-a",
		AccessTokenPath:   path,
	})
	if err != nil {
		t.Fatal(err)
	}
	if token != "persisted-token" {
		t.Fatalf("token = %q", token)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(raw); got != "persisted-token\n" {
		t.Fatalf("file = %q", got)
	}

	token, err = ResolveAccessToken(context.Background(), ts.Client(), ResolveAccessTokenOptions{
		BaseURL:           ts.URL,
		RegistrationToken: "registration-jwt",
		RunnerID:          "i-1",
		FleetID:           "fleet-a",
		AccessTokenPath:   path,
	})
	if err != nil {
		t.Fatal(err)
	}
	if token != "persisted-token" {
		t.Fatalf("reused token = %q", token)
	}
	if registerCalls != 1 {
		t.Fatalf("register calls = %d, want 1", registerCalls)
	}
}

func TestResolveAccessTokenPrefersEnvOverFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access_token")
	if err := os.WriteFile(path, []byte("from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	token, err := ResolveAccessToken(context.Background(), nil, ResolveAccessTokenOptions{
		AccessToken:     "from-env",
		AccessTokenPath: path,
	})
	if err != nil {
		t.Fatal(err)
	}
	if token != "from-env" {
		t.Fatalf("token = %q", token)
	}
}
