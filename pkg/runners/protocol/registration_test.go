package protocol

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterExchangesOpaqueRegistrationToken(t *testing.T) {
	for _, test := range []struct {
		name  string
		token string
	}{
		{name: "generic runner", token: "generic-registration-token"},
		{name: "task-specific runner", token: "task-bound-registration-token"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/runner/v1/register" {
					t.Fatalf("path = %q", r.URL.Path)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer "+test.token {
					t.Fatalf("authorization = %q", got)
				}
				var request registerRequest
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Fatalf("decode request: %v", err)
				}
				if request.Version != "0.1.0" {
					t.Fatalf("version = %q", request.Version)
				}
				writeJSON(t, w, http.StatusOK, Registration{
					RunnerID:    "runner-1",
					FleetID:     "linux-amd64",
					AccessToken: "runner-access-token",
					Ephemeral:   true,
				})
			}))
			defer server.Close()

			registration, err := Register(
				t.Context(),
				server.Client(),
				server.URL,
				test.token,
				"0.1.0",
			)
			if err != nil {
				t.Fatalf("Register: %v", err)
			}
			if registration.RunnerID != "runner-1" ||
				registration.FleetID != "linux-amd64" ||
				registration.AccessToken != "runner-access-token" ||
				!registration.Ephemeral {
				t.Fatalf("registration = %#v", registration)
			}
		})
	}
}

func TestRegisterIncludesResponseBodyInStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "runner version does not match", http.StatusConflict)
	}))
	defer server.Close()

	_, err := Register(t.Context(), server.Client(), server.URL, "token", "wrong")
	if err == nil {
		t.Fatal("expected registration error")
	}
	if got := err.Error(); got != "register runner: status 409: runner version does not match" {
		t.Fatalf("error = %q", got)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}
