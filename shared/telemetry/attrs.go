package telemetry

import "go.opentelemetry.io/otel/attribute"

// FleetAttr labels metrics that relate to a runner pool.
func FleetAttr(fleetID string) attribute.KeyValue {
	return attribute.String("fleet_id", fleetID)
}
