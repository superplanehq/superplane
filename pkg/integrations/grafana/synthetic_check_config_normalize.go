package grafana

import "github.com/mitchellh/mapstructure"

var (
	syntheticCheckRequestFlatKeys = []string{
		"target",
		"method",
		"headers",
		"body",
		"noFollowRedirects",
		"basicAuth",
		"bearerToken",
	}
	syntheticCheckScheduleFlatKeys = []string{
		"enabled",
		"frequency",
		"timeout",
		"probes",
	}
	syntheticCheckValidationFlatKeys = []string{
		"failIfSSL",
		"failIfNotSSL",
		"validStatusCodes",
		"failIfBodyMatchesRegexp",
		"failIfBodyNotMatchesRegexp",
		"failIfHeaderMatchesRegexp",
	}
)

// normalizeSyntheticCheckConfigMap groups legacy flat configuration keys into request, schedule, and
// validation objects so the UI matches newer nested storage. Existing nested configs are unchanged.
func normalizeSyntheticCheckConfigMap(m map[string]any) {
	if m == nil {
		return
	}

	legacyFrequencyMilliseconds, hasLegacyFrequencyMilliseconds := legacySyntheticFrequencyMilliseconds(m)
	liftFlatKeysIntoSection(m, "request", syntheticCheckRequestFlatKeys)
	liftFlatKeysIntoSection(m, "schedule", syntheticCheckScheduleFlatKeys)
	liftFlatKeysIntoSection(m, "validation", syntheticCheckValidationFlatKeys)
	if hasLegacyFrequencyMilliseconds {
		section := toStringMap(m["schedule"])
		if section == nil {
			section = map[string]any{}
		}
		section["frequency"] = syntheticFrequencySecondsFromMilliseconds(legacyFrequencyMilliseconds)
		section["frequencyMilliseconds"] = legacyFrequencyMilliseconds
		m["schedule"] = section
	}
}

func legacySyntheticFrequencyMilliseconds(m map[string]any) (int64, bool) {
	if _, hasSchedule := m["schedule"]; hasSchedule {
		return 0, false
	}

	value, ok := m["frequency"]
	if !ok {
		return 0, false
	}

	frequency := castToInt64(value)
	return frequency, frequency > 0
}

func liftFlatKeysIntoSection(m map[string]any, sectionKey string, keys []string) {
	if _, exists := m[sectionKey]; exists {
		return
	}

	section := map[string]any{}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			section[k] = v
			delete(m, k)
		}
	}
	if len(section) > 0 {
		m[sectionKey] = section
	}
}

// flattenSyntheticCheckConfigMap merges nested request, schedule, and validation maps back into a
// single map for decoding into SyntheticCheckSpecBase (flat struct tags).
func flattenSyntheticCheckConfigMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}

	out := make(map[string]any, len(m)+32)
	for k, v := range m {
		if k == "request" || k == "schedule" || k == "validation" {
			continue
		}
		out[k] = v
	}

	mergeStringMapInto(out, m["request"])
	mergeStringMapInto(out, m["schedule"])
	mergeStringMapInto(out, m["validation"])

	return out
}

func mergeStringMapInto(out map[string]any, v any) {
	section := toStringMap(v)
	if len(section) == 0 {
		return
	}
	for k, val := range section {
		out[k] = val
	}
}

func toStringMap(v any) map[string]any {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	var out map[string]any
	_ = mapstructure.Decode(v, &out)
	return out
}
