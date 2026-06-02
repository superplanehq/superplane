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
		ID:          "aws-standard-amd64",
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
	if gotBody.ID != "aws-standard-amd64" || gotBody.Provisioner != "aws" || gotBody.Arch != "amd64" || gotBody.Size != "t3.micro" {
		t.Fatalf("body: %#v", gotBody)
	}
	if out.ID != "aws-standard-amd64" || out.Size != "t3.micro" || out.CreatedAt != 1710000000 {
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
