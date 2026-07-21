package broker

import (
	"testing"

	"github.com/superplanehq/superplane/pkg/taskbroker/shared/api"
)

func TestRunnerDrainHub_DrainIdleRunner(t *testing.T) {
	h := NewRunnerDrainHub()
	unregister := h.Register("i-idle", "fleet-a", nil)
	defer unregister()

	statuses := h.Drain("fleet-a", []string{"i-idle"})

	if len(statuses) != 1 || statuses[0].RunnerID != "i-idle" || statuses[0].State != api.DrainRunnerStateDrained {
		t.Fatalf("statuses: %#v", statuses)
	}
	if !h.IsDraining("i-idle") {
		t.Fatal("expected runner to be marked draining")
	}
	if h.TryStartClaim("i-idle") {
		t.Fatal("drained runner should not start a new claim")
	}
}

func TestRunnerDrainHub_DrainActiveRunnerIsBusy(t *testing.T) {
	h := NewRunnerDrainHub()
	unregister := h.Register("i-active", "fleet-a", nil)
	defer unregister()
	if !h.TryStartClaim("i-active") {
		t.Fatal("expected claim reservation")
	}
	h.FinishClaim("i-active", "task-1")

	statuses := h.Drain("fleet-a", []string{"i-active"})

	if len(statuses) != 1 || statuses[0].RunnerID != "i-active" || statuses[0].State != api.DrainRunnerStateBusy {
		t.Fatalf("statuses: %#v", statuses)
	}
	if statuses[0].ActiveTaskID != "task-1" {
		t.Fatalf("active task id: %q", statuses[0].ActiveTaskID)
	}
	if !h.IsDraining("i-active") {
		t.Fatal("expected active runner to be marked draining")
	}
	h.CompleteTask("i-active", "task-1")
	statuses = h.Drain("fleet-a", []string{"i-active"})
	if len(statuses) != 1 || statuses[0].State != api.DrainRunnerStateDrained {
		t.Fatalf("statuses after complete: %#v", statuses)
	}
}

func TestRunnerDrainHub_DrainUnknownRunnerIsAlreadyDrained(t *testing.T) {
	h := NewRunnerDrainHub()

	statuses := h.Drain("fleet-a", []string{"", "i-gone", "i-gone"})

	if len(statuses) != 1 || statuses[0].RunnerID != "i-gone" || statuses[0].State != api.DrainRunnerStateDrained {
		t.Fatalf("statuses: %#v", statuses)
	}
	if h.TryStartClaim("i-gone") {
		t.Fatal("unknown drained runner should stay draining on reconnect")
	}
}

func TestRunnerDrainHub_DrainWrongFleetIsBusy(t *testing.T) {
	h := NewRunnerDrainHub()
	unregister := h.Register("i-other", "fleet-b", nil)
	defer unregister()

	statuses := h.Drain("fleet-a", []string{"i-other"})

	if len(statuses) != 1 || statuses[0].RunnerID != "i-other" || statuses[0].State != api.DrainRunnerStateBusy {
		t.Fatalf("statuses: %#v", statuses)
	}
}

func TestRunnerDrainHub_DrainClaimingRunnerIsBusy(t *testing.T) {
	h := NewRunnerDrainHub()
	unregister := h.Register("i-claiming", "fleet-a", nil)
	defer unregister()
	if !h.TryStartClaim("i-claiming") {
		t.Fatal("expected claim reservation")
	}

	statuses := h.Drain("fleet-a", []string{"i-claiming"})

	if len(statuses) != 1 || statuses[0].RunnerID != "i-claiming" || statuses[0].State != api.DrainRunnerStateBusy {
		t.Fatalf("statuses: %#v", statuses)
	}
	if statuses[0].ActiveTaskID != "" {
		t.Fatalf("claiming sentinel should not be exposed, got %q", statuses[0].ActiveTaskID)
	}
	h.FinishClaim("i-claiming", "")
	statuses = h.Drain("fleet-a", []string{"i-claiming"})
	if len(statuses) != 1 || statuses[0].State != api.DrainRunnerStateDrained {
		t.Fatalf("statuses after empty claim: %#v", statuses)
	}
}
