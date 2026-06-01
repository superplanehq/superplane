package fleetmanager

import (
	"context"
	"errors"
	"testing"

	"github.com/superplane/runner/fleet-manager/internal/ec2provision"
)

type fakeLister struct {
	fleetID string
	rows    []ec2provision.ManagedRunnerSummary
	err     error
	calls   int
}

func (f *fakeLister) ListManagedRunners(_ context.Context) ([]ec2provision.ManagedRunnerSummary, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.rows, nil
}

func (f *fakeLister) FleetID() string { return f.fleetID }

func TestAggregateManagedRunners_ConcatenatesAcrossPools(t *testing.T) {
	a := &fakeLister{
		fleetID: "aws-amd64",
		rows: []ec2provision.ManagedRunnerSummary{
			{FleetID: "aws-amd64", InstanceID: "i-1", State: "running"},
			{FleetID: "aws-amd64", InstanceID: "i-2", State: "pending"},
		},
	}
	b := &fakeLister{
		fleetID: "aws-arm64",
		rows: []ec2provision.ManagedRunnerSummary{
			{FleetID: "aws-arm64", InstanceID: "i-3", State: "running"},
		},
	}

	out := aggregateManagedRunners(context.Background(), nil, []managedRunnerLister{a, b})

	if len(out) != 3 {
		t.Fatalf("expected 3 rows, got %d (%v)", len(out), out)
	}
	if a.calls != 1 || b.calls != 1 {
		t.Errorf("each launcher called once: a=%d b=%d", a.calls, b.calls)
	}
	if out[0].FleetID != "aws-amd64" || out[1].FleetID != "aws-amd64" || out[2].FleetID != "aws-arm64" {
		t.Errorf("fleet ids in output rows = [%s %s %s], want [aws-amd64 aws-amd64 aws-arm64]",
			out[0].FleetID, out[1].FleetID, out[2].FleetID)
	}
	if out[0].InstanceID != "i-1" || out[1].InstanceID != "i-2" || out[2].InstanceID != "i-3" {
		t.Errorf("order not preserved: %v", out)
	}
}

func TestAggregateManagedRunners_OnePoolErrorSkippedOthersIncluded(t *testing.T) {
	// One pool's EC2 hiccup must not poison the whole admin listing — the surviving
	// pool's rows still come back.
	a := &fakeLister{fleetID: "broken", err: errors.New("describe failed")}
	b := &fakeLister{
		fleetID: "healthy",
		rows: []ec2provision.ManagedRunnerSummary{
			{FleetID: "healthy", InstanceID: "i-9", State: "running"},
		},
	}

	out := aggregateManagedRunners(context.Background(), nil, []managedRunnerLister{a, b})

	if len(out) != 1 {
		t.Fatalf("expected 1 row from healthy pool, got %d (%v)", len(out), out)
	}
	if out[0].FleetID != "healthy" || out[0].InstanceID != "i-9" {
		t.Errorf("got %+v, want healthy/i-9", out[0])
	}
}

func TestAggregateManagedRunners_NoLaunchersReturnsEmptySlice(t *testing.T) {
	out := aggregateManagedRunners(context.Background(), nil, nil)
	if out == nil {
		t.Errorf("expected non-nil empty slice (so JSON encodes as [])")
	}
	if len(out) != 0 {
		t.Errorf("expected length 0, got %d", len(out))
	}
}
