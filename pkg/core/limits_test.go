package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaxForEachItems(t *testing.T) {
	t.Run("defaults to 100", func(t *testing.T) {
		t.Setenv(maxForEachItemsEnvVar, "")
		assert.Equal(t, 100, MaxForEachItems())
	})

	t.Run("reads SUPERPLANE_FOREACH_MAX_ITEMS", func(t *testing.T) {
		t.Setenv(maxForEachItemsEnvVar, "25")
		assert.Equal(t, 25, MaxForEachItems())
	})

	t.Run("ignores invalid env values", func(t *testing.T) {
		t.Setenv(maxForEachItemsEnvVar, "not-a-number")
		assert.Equal(t, 100, MaxForEachItems())
	})

	t.Run("caps override at MaxEmitCount", func(t *testing.T) {
		t.Setenv(maxForEachItemsEnvVar, "999")
		assert.Equal(t, MaxEmitCount, MaxForEachItems())
	})
}
