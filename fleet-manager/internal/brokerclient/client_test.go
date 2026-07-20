package brokerclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/superplane/runner/shared/api"
)

func TestFleetTaskCounts_OK(t *testing.T) {
	var gotPath, gotAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.FleetTaskCountsResponse{Queued: 2, Claimed: 5})
	}))
	defer ts.Close()

	c := New(ts.URL, "tok")
	out, err := c.FleetTaskCounts(context.Background(), "fleet-a")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/fleets/fleet-a/task-counts" {
		t.Fatalf("path: %q", gotPath)
	}
	if gotAuth != "Bearer tok" {
		t.Fatalf("auth: %q", gotAuth)
	}
	if out.Queued != 2 || out.Claimed != 5 {
		t.Fatalf("out: %#v", out)
	}
}

func TestFleetTaskCounts_NonOK(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"fleet not found"}`))
	}))
	defer ts.Close()

	c := New(ts.URL, "tok")
	if _, err := c.FleetTaskCounts(context.Background(), "missing"); err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestFleetTaskCounts_BadInput(t *testing.T) {
	c := New("http://example", "tok")
	if _, err := c.FleetTaskCounts(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty fleet id")
	}
	empty := New("", "tok")
	if _, err := empty.FleetTaskCounts(context.Background(), "f"); err == nil {
		t.Fatal("expected error for empty base url")
	}
}

func TestDrainRunners_OK(t *testing.T) {
	var gotMethod, gotPath, gotAuth string
	var gotBody api.DrainRunnersRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.DrainRunnersResponse{
			Runners: []api.DrainRunnerStatus{
				{RunnerID: "i-idle", State: api.DrainRunnerStateDrained},
				{RunnerID: "i-busy", State: api.DrainRunnerStateBusy, ActiveTaskID: "task-1"},
			},
		})
	}))
	defer ts.Close()

	c := New(ts.URL, "tok")
	out, err := c.DrainRunners(context.Background(), api.DrainRunnersRequest{
		FleetID:   "fleet-a",
		RunnerIDs: []string{"i-idle", "i-busy"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/runners/drain" {
		t.Fatalf("method/path: %s %q", gotMethod, gotPath)
	}
	if gotAuth != "Bearer tok" {
		t.Fatalf("auth: %q", gotAuth)
	}
	if gotBody.FleetID != "fleet-a" || len(gotBody.RunnerIDs) != 2 {
		t.Fatalf("body: %#v", gotBody)
	}
	if len(out.Runners) != 2 {
		t.Fatalf("runners: %#v", out)
	}
	if out.Runners[0].RunnerID != "i-idle" || out.Runners[0].State != api.DrainRunnerStateDrained {
		t.Fatalf("first runner: %#v", out.Runners[0])
	}
	if out.Runners[1].RunnerID != "i-busy" || out.Runners[1].State != api.DrainRunnerStateBusy || out.Runners[1].ActiveTaskID != "task-1" {
		t.Fatalf("second runner: %#v", out.Runners[1])
	}
}

func TestDrainRunners_NonOK(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"runner drain unavailable"}`))
	}))
	defer ts.Close()

	c := New(ts.URL, "tok")
	if _, err := c.DrainRunners(context.Background(), api.DrainRunnersRequest{
		FleetID:   "fleet-a",
		RunnerIDs: []string{"i-idle"},
	}); err == nil {
		t.Fatal("expected error for non-2xx drain response")
	}
}

func TestDrainRunners_BadInput(t *testing.T) {
	c := New("http://example", "tok")
	if _, err := c.DrainRunners(context.Background(), api.DrainRunnersRequest{
		RunnerIDs: []string{"i-idle"},
	}); err == nil {
		t.Fatal("expected error for empty fleet id")
	}
	if _, err := c.DrainRunners(context.Background(), api.DrainRunnersRequest{
		FleetID: "fleet-a",
	}); err == nil {
		t.Fatal("expected error for empty runner ids")
	}
	empty := New("", "tok")
	if _, err := empty.DrainRunners(context.Background(), api.DrainRunnersRequest{
		FleetID:   "fleet-a",
		RunnerIDs: []string{"i-idle"},
	}); err == nil {
		t.Fatal("expected error for empty base url")
	}
}

func TestRegisterFleet_OK(t *testing.T) {
	var gotMethod, gotPath, gotAuth string
	var gotBody api.RegisterFleetRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(api.FleetResponse{
			ID:          gotBody.ID,
			Provisioner: gotBody.Provisioner,
			Arch:        gotBody.Arch,
			Size:        gotBody.Size,
			CreatedAt:   1710000000,
		})
	}))
	defer ts.Close()

	c := New(ts.URL, "tok")
	out, err := c.RegisterFleet(context.Background(), api.RegisterFleetRequest{
		ID:          "e1-tiny-amd64",
		Provisioner: "aws",
		Arch:        "amd64",
		Size:        "t3.micro",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/fleets" {
		t.Fatalf("method/path: %s %q", gotMethod, gotPath)
	}
	if gotAuth != "Bearer tok" {
		t.Fatalf("auth: %q", gotAuth)
	}
	if gotBody.ID != "e1-tiny-amd64" || gotBody.Provisioner != "aws" || gotBody.Arch != "amd64" || gotBody.Size != "t3.micro" {
		t.Fatalf("body: %#v", gotBody)
	}
	if out.ID != "e1-tiny-amd64" || out.Size != "t3.micro" || out.CreatedAt != 1710000000 {
		t.Fatalf("out: %#v", out)
	}
}

func TestRegisterFleet_NonOK(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"id required"}`))
	}))
	defer ts.Close()

	c := New(ts.URL, "tok")
	if _, err := c.RegisterFleet(context.Background(), api.RegisterFleetRequest{ID: "fleet-a"}); err == nil {
		t.Fatal("expected error for 400")
	}
}

func TestRegisterFleet_BadInput(t *testing.T) {
	c := New("http://example", "tok")
	if _, err := c.RegisterFleet(context.Background(), api.RegisterFleetRequest{}); err == nil {
		t.Fatal("expected error for empty fleet id")
	}
	empty := New("", "tok")
	if _, err := empty.RegisterFleet(context.Background(), api.RegisterFleetRequest{ID: "f"}); err == nil {
		t.Fatal("expected error for empty base url")
	}
}
