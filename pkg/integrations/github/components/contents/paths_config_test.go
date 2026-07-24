package contents

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/configuration"
)

func Test__CoercePathsConfigurationValue(t *testing.T) {
	log := logrus.NewEntry(logrus.New())

	t.Run("[]any of glob strings -> trimmed, non-empty patterns", func(t *testing.T) {
		raw := []any{"src/**", " docs/**.md ", ""}

		out := coercePathsConfigurationValue(raw, log)

		assert.Equal(t, []string{"src/**", "docs/**.md"}, out)
	})

	t.Run("[]string of globs -> trimmed, non-empty patterns", func(t *testing.T) {
		raw := []string{"src/**", " docs/**.md ", ""}

		out := coercePathsConfigurationValue(raw, log)

		assert.Equal(t, []string{"src/**", "docs/**.md"}, out)
	})

	t.Run("legacy equals predicates -> reused as globs", func(t *testing.T) {
		raw := []any{
			map[string]any{"type": configuration.PredicateTypeEquals, "value": " src/** "},
		}

		out := coercePathsConfigurationValue(raw, log)

		assert.Equal(t, []string{"src/**"}, out)
	})

	t.Run("legacy matches predicates -> skipped", func(t *testing.T) {
		raw := []any{
			map[string]any{"type": configuration.PredicateTypeMatches, "value": "^src/.*$"},
		}

		out := coercePathsConfigurationValue(raw, log)

		assert.Empty(t, out)
	})

	t.Run("unsupported value type -> nil", func(t *testing.T) {
		out := coercePathsConfigurationValue("src/**", log)

		assert.Nil(t, out)
	})
}
