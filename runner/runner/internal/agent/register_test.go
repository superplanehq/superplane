package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/superplane/runner/shared/api"
)

func TestRegisterRunnerExchangesBootstrapToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/runners/register" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer bootstrap-token" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		var req api.RegisterRunnerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.RunnerID != "i-012345" || req.FleetID != "fleet-a" {
			t.Errorf("request = %#v", req)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(api.RegisterRunnerResponse{AccessToken: "runner-token"})
	}))
	defer ts.Close()

	token, err := RegisterRunner(
		context.Background(), ts.Client(), ts.URL, "bootstrap-token", "i-012345", "fleet-a",
	)
	if err != nil {
		t.Fatal(err)
	}
	if token != "runner-token" {
		t.Fatalf("token = %q", token)
	}
}
