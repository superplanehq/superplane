package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
)

func TestOriginLabelsForTask(t *testing.T) {
	t.Run("returns nil when no origin fields are set", func(t *testing.T) {
		assert.Nil(t, OriginLabelsForTask(core.ExecutionContext{}))
	})

	t.Run("includes only non-empty origin fields", func(t *testing.T) {
		labels := OriginLabelsForTask(core.ExecutionContext{
			WorkflowID:     "canvas-1",
			OrganizationID: "org-1",
			CanvasName:     "release-train",
			NodeName:       "Run commands",
		})

		assert.Equal(t, map[string]string{
			"canvas_id":       "canvas-1",
			"organization_id": "org-1",
			"canvas_name":     "release-train",
			"node_name":       "Run commands",
		}, labels)
	})

	t.Run("omits blank fields", func(t *testing.T) {
		labels := OriginLabelsForTask(core.ExecutionContext{
			WorkflowID: "canvas-1",
			NodeName:   "Run commands",
		})

		assert.Equal(t, map[string]string{
			"canvas_id": "canvas-1",
			"node_name": "Run commands",
		}, labels)
	})
}
