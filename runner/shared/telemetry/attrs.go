package telemetry

import "go.opentelemetry.io/otel/attribute"

// FleetAttr labels metrics that relate to a runner pool.
func FleetAttr(fleetID string) attribute.KeyValue {
	return attribute.String("fleet_id", fleetID)
}

// OutcomeAttr labels task completion outcome (succeeded, failed, canceled).
func OutcomeAttr(outcome string) attribute.KeyValue {
	return attribute.String("outcome", outcome)
}

// PhaseAttr labels multi-phase metrics such as instance spinup duration.
func PhaseAttr(phase string) attribute.KeyValue {
	return attribute.String("phase", phase)
}
