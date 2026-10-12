package licensing

import "slices"

// Feature is an Enterprise capability key granted by a license. Keys are
// immutable and compared exactly.
type Feature string

const (
	FeatureCustomRoles Feature = "custom_roles"
	FeatureGroups      Feature = "groups"

	// FeatureAuditLogs is a defined Enterprise entitlement without a product
	// implementation yet.
	FeatureAuditLogs Feature = "audit_logs"
)

var recognizedFeatures = []Feature{
	FeatureAuditLogs,
	FeatureCustomRoles,
	FeatureGroups,
}

// RecognizedFeatures returns every Enterprise feature that this SuperPlane
// version knows.
func RecognizedFeatures() []Feature {
	return slices.Clone(recognizedFeatures)
}

// IsRecognizedFeature reports whether this SuperPlane version knows the key.
// Unknown keys in a license never grant access.
func IsRecognizedFeature(key string) bool {
	return slices.Contains(recognizedFeatures, Feature(key))
}
