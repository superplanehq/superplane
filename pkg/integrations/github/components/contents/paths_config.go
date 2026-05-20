package contents

import (
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/pathfilter"
)

func decodeOnPushConfigurationForStruct(configuration any) (OnPushConfiguration, error) {
	cfg, ok := configuration.(map[string]any)
	if !ok {
		var out OnPushConfiguration
		err := mapstructure.Decode(configuration, &out)
		return out, err
	}

	stripped := make(map[string]any, len(cfg))
	for k, v := range cfg {
		if k == "paths" {
			continue
		}
		stripped[k] = v
	}

	var out OnPushConfiguration
	err := mapstructure.Decode(stripped, &out)
	return out, err
}

// onPushPathsFromConfiguration resolves stored "paths" for github.onPush.
//
// New format is a list of strings (GitHub Actions–style globs, optional "!..." excludes).
//
// Legacy any-predicate-list JSON is partially honored: "equals" values are reused as globs.
// Legacy "matches" (regex) values are skipped with a warning and must be reconfigured as globs.
func onPushPathsFromConfiguration(configuration any, decodedPaths []string, log *logrus.Entry) []string {
	cfg, ok := configuration.(map[string]any)
	if !ok || cfg == nil {
		return pathfilter.TrimNonEmptyStrings(decodedPaths)
	}

	raw, hasPaths := cfg["paths"]
	if hasPaths && raw != nil {
		out := coercePathsConfigurationValue(raw, log)
		if len(out) > 0 {
			return out
		}
		if hasLegacyPredicateShape(raw) {
			return nil
		}
	}

	return pathfilter.TrimNonEmptyStrings(decodedPaths)
}

func hasLegacyPredicateShape(raw any) bool {
	list, ok := raw.([]any)
	if !ok {
		return false
	}

	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}

		if _, ok := m["type"]; ok {
			return true
		}
	}

	return false
}

func coercePathsConfigurationValue(raw any, log *logrus.Entry) []string {
	switch v := raw.(type) {
	case []string:
		return pathfilter.TrimNonEmptyStrings(v)
	case []any:
		var out []string
		legacyCount := 0

		for _, item := range v {
			switch t := item.(type) {
			case string:
				s := strings.TrimSpace(t)
				if s != "" {
					out = append(out, s)
				}
			case map[string]any:
				legacyCount++

				typ, _ := t["type"].(string)
				val, _ := t["value"].(string)
				val = strings.TrimSpace(val)
				if val == "" {
					continue
				}

				switch typ {
				case configuration.PredicateTypeEquals:
					out = append(out, val)
				case configuration.PredicateTypeMatches:
					log.Warnf(
						"github.onPush paths: legacy \"matches\" predicate cannot be converted to globs; update this trigger (skipped value: %q)",
						val,
					)
				default:
					log.Warnf("github.onPush paths: unsupported legacy predicate type %q (skipped)", typ)
				}
			}
		}

		if legacyCount > 0 && len(out) == 0 {
			log.Warnf("github.onPush paths: legacy predicate configuration produced no usable glob patterns; path filter disabled until updated")
		}

		return out
	default:
		return nil
	}
}
