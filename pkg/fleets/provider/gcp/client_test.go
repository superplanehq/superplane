package gcpprovider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

func TestInstancesFromPageFailsOnUnreachableScopes(t *testing.T) {
	_, err := instancesFromPage(&compute.InstanceAggregatedList{
		Items: map[string]compute.InstancesScopedList{
			"zones/us-central1-a": {Instances: []*compute.Instance{{Name: "runner-a"}}},
			"zones/us-central1-b": {Warning: &compute.InstancesScopedListWarning{Code: warningCodeUnreachable}},
		},
		Unreachables: []string{"zones/us-central1-c"},
	})
	if err == nil || !strings.Contains(err.Error(), "zones/us-central1-b, zones/us-central1-c") {
		t.Fatalf("error = %v", err)
	}
}

func TestInstancesFromPageIgnoresEmptyScopes(t *testing.T) {
	instances, err := instancesFromPage(&compute.InstanceAggregatedList{
		Items: map[string]compute.InstancesScopedList{
			"zones/us-central1-a": {Instances: []*compute.Instance{{Name: "runner-a"}}},
			"zones/us-central1-b": {Warning: &compute.InstancesScopedListWarning{Code: "NO_RESULTS_ON_PAGE"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(instances) != 1 || instances[0].Name != "runner-a" {
		t.Fatalf("instances = %#v", instances)
	}
}

func TestInsertInstanceRetriesTemporaryWaitErrors(t *testing.T) {
	var waits atomic.Int32
	client := newHTTPCompute(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/wait") && waits.Add(1) == 1 {
			http.Error(w, `{"error":{"code":503,"message":"unavailable"}}`, http.StatusServiceUnavailable)
			return
		}
		status := "RUNNING"
		if strings.HasSuffix(r.URL.Path, "/wait") {
			status = operationStatusDone
		}
		writeOperation(t, w, status)
	})

	err := client.InsertInstance(context.Background(), "my-project", "us-central1-a", &compute.Instance{Name: "runner"})
	if err != nil {
		t.Fatal(err)
	}
	if waits.Load() != 2 {
		t.Fatalf("wait calls = %d", waits.Load())
	}
}

func TestInsertInstanceReportsUnknownResultWhenWaitFails(t *testing.T) {
	client := newHTTPCompute(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/wait") {
			http.Error(w, `{"error":{"code":403,"message":"denied"}}`, http.StatusForbidden)
			return
		}
		writeOperation(t, w, "RUNNING")
	})

	err := client.InsertInstance(context.Background(), "my-project", "us-central1-a", &compute.Instance{Name: "runner"})
	if !errors.Is(err, errOperationResultUnknown) {
		t.Fatalf("error = %v", err)
	}
}

func newHTTPCompute(t *testing.T, handler http.HandlerFunc) *sdkCompute {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	service, err := compute.NewService(
		context.Background(),
		option.WithEndpoint(server.URL+"/"),
		option.WithoutAuthentication(),
	)
	if err != nil {
		t.Fatal(err)
	}
	return &sdkCompute{service: service}
}

func writeOperation(t *testing.T, w http.ResponseWriter, status string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(compute.Operation{Name: "operation-1", Status: status}); err != nil {
		t.Fatal(err)
	}
}
