package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/superplanehq/superplane/pkg/fleets/adminclient"
	fleetconfig "github.com/superplanehq/superplane/pkg/fleets/config"
)

func TestPrintVersion(t *testing.T) {
	previousVersion := Version
	Version = "v1.2.3-rc.1"
	t.Cleanup(func() { Version = previousVersion })

	var output bytes.Buffer
	if !printVersion([]string{"--version"}, &output) {
		t.Fatal("expected --version to be handled")
	}
	if output.String() != "fleet-manager v1.2.3-rc.1\n" {
		t.Fatalf("version output = %q", output.String())
	}
}

func TestSyncConfiguredFleetsCreatesAndUpdatesFleets(t *testing.T) {
	requests := make([]string, 0, 7)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.Header.Get("Authorization") != "Bearer personal-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		switch r.Method + " " + r.URL.Path {
		case "GET /admin/api/installation/fleets/new":
			http.NotFound(w, r)
		case "POST /admin/api/installation/fleets":
			_, _ = w.Write([]byte(`{"fleet":{"id":"new"}}`))
		case "GET /admin/api/installation/fleets/existing":
			_, _ = w.Write([]byte(`{"fleet":{"id":"existing"}}`))
		case "PATCH /admin/api/installation/fleets/existing":
			_, _ = w.Write([]byte(`{"fleet":{"id":"existing"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := adminclient.New(server.URL, "personal-token", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	fleets := []fleetconfig.Fleet{
		{ID: "new", RunnerVersion: "v1.2.3", Spec: fleetconfig.FleetSpec{OperatingSystem: "linux"}},
		{ID: "existing", RunnerVersion: "v1.2.3", Spec: fleetconfig.FleetSpec{OperatingSystem: "linux"}},
	}
	if err := syncConfiguredFleets(context.Background(), client, fleets); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"GET /admin/api/installation/fleets/new",
		"POST /admin/api/installation/fleets",
		"GET /admin/api/installation/fleets/existing",
		"PATCH /admin/api/installation/fleets/existing",
	}
	if len(requests) != len(want) {
		t.Fatalf("requests = %#v, want %#v", requests, want)
	}
	for index := range want {
		if requests[index] != want[index] {
			t.Fatalf("requests = %#v, want %#v", requests, want)
		}
	}
}

func TestSyncConfiguredFleetsUpdatesAfterCreateConflict(t *testing.T) {
	requests := make([]string, 0, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method)
		switch r.Method {
		case http.MethodGet:
			http.NotFound(w, r)
		case http.MethodPost:
			http.Error(w, "already exists", http.StatusConflict)
		case http.MethodPatch:
			_, _ = w.Write([]byte(`{"fleet":{"id":"raced"}}`))
		}
	}))
	defer server.Close()

	client, err := adminclient.New(server.URL, "personal-token", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	err = syncConfiguredFleets(context.Background(), client, []fleetconfig.Fleet{{
		ID: "raced", RunnerVersion: "v1.2.3", Spec: fleetconfig.FleetSpec{OperatingSystem: "linux"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 3 || requests[2] != http.MethodPatch {
		t.Fatalf("requests = %#v", requests)
	}
}

func TestSyncConfiguredFleetsReturnsAuthenticationFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()

	client, err := adminclient.New(server.URL, "personal-token", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	err = syncConfiguredFleets(context.Background(), client, []fleetconfig.Fleet{{
		ID: "denied", RunnerVersion: "v1.2.3",
	}})
	if !adminclient.IsStatus(err, http.StatusUnauthorized) {
		t.Fatalf("error = %v", err)
	}
}

func TestNewEC2ClientUsesFleetRegion(t *testing.T) {
	base := aws.Config{Region: "us-east-1"}
	east := newEC2Client(base, "us-east-1")
	west := newEC2Client(base, "us-west-2")

	if east == west {
		t.Fatal("expected each fleet to receive its own EC2 client")
	}
	if east.Options().Region != "us-east-1" {
		t.Fatalf("east client region = %q", east.Options().Region)
	}
	if west.Options().Region != "us-west-2" {
		t.Fatalf("west client region = %q", west.Options().Region)
	}
	if base.Region != "us-east-1" {
		t.Fatalf("base AWS config region changed to %q", base.Region)
	}
}
