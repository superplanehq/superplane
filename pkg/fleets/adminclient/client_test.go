package adminclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateRunnerUsesFleetPathAndAdminBearer(t *testing.T) {
	var received CreateRunnerRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/admin/api/installation/fleets/linux-amd64/runners" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer personal-token" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"runner":{"id":"runner-1","fleetId":"linux-amd64","state":"pending","runnerVersion":"1.2.3","ephemeral":true},
			"registrationToken":"registration-token",
			"registrationExpiresAt":"2026-09-27T15:00:00Z",
			"runnerApiUrl":"https://superplane.example/runner/v1",
			"displayName":"runner-runner-1"
		}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "personal-token", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.CreateRunner(context.Background(), "linux-amd64", CreateRunnerRequest{
		IdempotencyKey: "fleet-manager/linux-amd64/generic/request-1",
		Ephemeral:      true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if received.IdempotencyKey != "fleet-manager/linux-amd64/generic/request-1" ||
		!received.Ephemeral {
		t.Fatalf("request = %#v", received)
	}
	if response.Runner.ID != "runner-1" || response.Runner.RunnerVersion != "1.2.3" {
		t.Fatalf("response = %#v", response)
	}
}

func TestCapacityAndListQueriesUseProtoJSONNames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/admin/api/installation/fleets/fleet-a/capacity":
			if r.URL.Query().Get("generation") != "gen-1" || r.URL.Query().Get("waitSeconds") != "30" {
				t.Fatalf("capacity query = %q", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{
				"runnableTasks":"2",
				"pendingRunners":"1",
				"idleRunners":"0",
				"busyRunners":"3",
				"terminatedRunners":"4",
				"generation":"gen-2"
			}`))
		case "/admin/api/installation/fleets/fleet-a/runners":
			states := r.URL.Query()["states"]
			if len(states) != 2 || states[0] != "pending" || states[1] != "idle" {
				t.Fatalf("states = %#v", states)
			}
			if r.URL.Query().Get("limit") != "1000" {
				t.Fatalf("limit = %q", r.URL.Query().Get("limit"))
			}
			_, _ = w.Write([]byte(`{"runners":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := New(server.URL, "personal-token", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	capacity, err := client.GetFleetCapacity(context.Background(), "fleet-a", "gen-1", 30)
	if err != nil {
		t.Fatal(err)
	}
	if capacity.RunnableTasks != 2 || capacity.BusyRunners != 3 {
		t.Fatalf("capacity = %#v", capacity)
	}
	if _, err := client.ListRunners(
		context.Background(),
		"fleet-a",
		[]string{RunnerStatePending, RunnerStateIdle},
		1000,
	); err != nil {
		t.Fatal(err)
	}
}

func TestHTTPErrorIncludesStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "runner is busy", http.StatusConflict)
	}))
	defer server.Close()

	client, err := New(server.URL, "personal-token", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.DeleteRunner(context.Background(), "fleet-a", "runner-1")
	if !IsStatus(err, http.StatusConflict) {
		t.Fatalf("error = %v", err)
	}
}

func TestDescribeRunnerUsesFleetAndRunnerPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q", r.Method)
		}
		if r.URL.Path != "/admin/api/installation/fleets/fleet-a/runners/runner-1" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"runner":{"id":"runner-1","fleetId":"fleet-a","state":"terminated","runnerVersion":"1.2.3"}
		}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "personal-token", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	runner, err := client.DescribeRunner(context.Background(), "fleet-a", "runner-1")
	if err != nil {
		t.Fatal(err)
	}
	if runner.ID != "runner-1" || runner.State != RunnerStateTerminated {
		t.Fatalf("runner = %#v", runner)
	}
}

func TestCreateRunnerRetriesServerFailureWithSameIdempotencyKey(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var request CreateRunnerRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.IdempotencyKey != "stable-key" {
			t.Fatalf("idempotency key = %q", request.IdempotencyKey)
		}
		if calls == 1 {
			http.Error(w, "temporary failure", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{
			"runner":{"id":"runner-1","fleetId":"fleet-a","state":"pending","runnerVersion":"1.0.0"},
			"registrationToken":"token",
			"registrationExpiresAt":"2026-09-27T15:00:00Z",
			"runnerApiUrl":"https://superplane.example/runner/v1",
			"displayName":"runner-runner-1"
		}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "personal-token", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.CreateRunner(context.Background(), "fleet-a", CreateRunnerRequest{
		IdempotencyKey: "stable-key",
		Ephemeral:      true,
	}); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}
