package broker

import (
	"testing"

	brokermodels "github.com/superplane/runner/task-broker/internal/models"
)

func TestResolveExecutionTimeoutSecondsNoFleetCap(t *testing.T) {
	if v, msg := resolveExecutionTimeoutSeconds(nil, nil); v != nil || msg != "" {
		t.Fatalf("v=%v msg=%q", v, msg)
	}
	requested := 120
	v, msg := resolveExecutionTimeoutSeconds(&brokermodels.Fleet{}, &requested)
	if msg != "" || v == nil || *v != 120 {
		t.Fatalf("v=%v msg=%q", v, msg)
	}
}

func TestResolveExecutionTimeoutSecondsRejectsAboveCap(t *testing.T) {
	capSeconds := 900
	requested := 1000
	fleet := &brokermodels.Fleet{ID: "lambda-fleet", MaxExecutionTimeoutSeconds: &capSeconds}
	v, msg := resolveExecutionTimeoutSeconds(fleet, &requested)
	if v != nil || msg == "" {
		t.Fatalf("expected rejection, got v=%v msg=%q", v, msg)
	}
}

func TestResolveExecutionTimeoutSecondsAllowsAtOrBelowCap(t *testing.T) {
	capSeconds := 900
	requested := 900
	fleet := &brokermodels.Fleet{MaxExecutionTimeoutSeconds: &capSeconds}
	v, msg := resolveExecutionTimeoutSeconds(fleet, &requested)
	if msg != "" || v == nil || *v != 900 {
		t.Fatalf("v=%v msg=%q", v, msg)
	}
}

func TestResolveExecutionTimeoutSecondsClampsOmittedRequest(t *testing.T) {
	capSeconds := 900
	fleet := &brokermodels.Fleet{MaxExecutionTimeoutSeconds: &capSeconds}
	v, msg := resolveExecutionTimeoutSeconds(fleet, nil)
	if msg != "" || v == nil || *v != 900 {
		t.Fatalf("expected clamp to fleet cap, got v=%v msg=%q", v, msg)
	}
}

func TestResolveExecutionTimeoutSecondsOmittedBelowCapUsesDefault(t *testing.T) {
	capSeconds := 7200
	fleet := &brokermodels.Fleet{MaxExecutionTimeoutSeconds: &capSeconds}
	v, msg := resolveExecutionTimeoutSeconds(fleet, nil)
	if msg != "" || v == nil || *v != 3600 {
		t.Fatalf("expected default (below cap), got v=%v msg=%q", v, msg)
	}
}
