package ec2provision

import (
	"context"
	"testing"

	"github.com/superplane/runner/shared/api"
)

func TestTerminateUnhealthyRunnersDefersBusyRunners(t *testing.T) {
	fake := &fakeBrokerClient{drain: api.DrainRunnersResponse{
		Runners: []api.DrainRunnerStatus{
			{RunnerID: "i-busy", State: api.DrainRunnerStateBusy, ActiveTaskID: "task-1"},
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
