package ec2provision

import (
	"context"
	"testing"

	"github.com/superplane/runner/shared/api"
)

func TestTerminateUnhealthyRunnersDefersInProgressClaim(t *testing.T) {
	fake := &fakeBrokerClient{drain: api.DrainRunnersResponse{
		Runners: []api.DrainRunnerStatus{
			{RunnerID: "i-busy", State: api.DrainRunnerStateBusy},
		},
	}}
	l := &Launcher{
		Config:       Config{RunnerFleetID: "fleet-a"},
		BrokerClient: fake,
	}

	got, err := l.terminateUnhealthyRunners(context.Background(), []string{"i-busy"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("terminated ids: got %#v want none", got)
	}
	if fake.drainCalls != 1 {
		t.Fatalf("drain calls = %d, want 1", fake.drainCalls)
	}
	if fake.drainRequest.FleetID != "fleet-a" || len(fake.drainRequest.RunnerIDs) != 1 {
		t.Fatalf("drain request: %#v", fake.drainRequest)
	}
}

func TestRecordHealthFailureRequiresConsecutiveFailures(t *testing.T) {
	l := &Launcher{Config: Config{
		RunnerFleetID:                "fleet-a",
		RunnerHealthFailureThreshold: 3,
	}}

	if l.recordHealthFailure("i-runner", "timeout") {
		t.Fatal("first failure should not terminate")
	}
	if l.recordHealthFailure("i-runner", "timeout") {
		t.Fatal("second failure should not terminate")
	}
	if !l.recordHealthFailure("i-runner", "timeout") {
		t.Fatal("third failure should terminate")
	}
}

func TestRecordHealthSuccessResetsConsecutiveFailures(t *testing.T) {
	l := &Launcher{Config: Config{
		RunnerFleetID:                "fleet-a",
		RunnerHealthFailureThreshold: 2,
	}}

	if l.recordHealthFailure("i-runner", "timeout") {
		t.Fatal("first failure should not terminate")
	}
	l.recordHealthSuccess("i-runner")
	if l.recordHealthFailure("i-runner", "timeout") {
		t.Fatal("failure after recovery should restart the count")
	}
}
