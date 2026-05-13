package fleetmanager

import "testing"

func TestRunnerCancelHub_PushCancelWithoutRegister(t *testing.T) {
	h := NewRunnerCancelHub()
	if h.PushCancel("runner-1", "task-1") {
		t.Fatal("expected false when not registered")
	}
}

func TestRunnerCancelHub_nilSafe(t *testing.T) {
	var h *RunnerCancelHub
	unreg := h.Register("r", "t", nil, nil)
	unreg()
	if h.PushCancel("r", "t") {
		t.Fatal("expected false on nil hub")
	}
}
