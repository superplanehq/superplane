package canvases

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test__SerializeCanvasEvent__ConvertsJSONNumbersForProto(t *testing.T) {
	now := time.Now()
	runID := uuid.New()
	event := models.CanvasEvent{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
		NodeID:     "trigger",
		Channel:    "default",
		RunID:      runID,
		Data: models.NewJSONValue(map[string]any{
			"type": "webhook",
			"data": map[string]any{
				"plain":     json.Number("14000000"),
				"unsafeInt": json.Number("9007199254740993"),
				"large":     json.Number("12345678901234567890"),
				"small":     json.Number("0.0000001"),
				"normal":    json.Number("42"),
				"name":      "deploy",
				"active":    true,
				"labels":    []any{"prod"},
				"nested":    map[string]any{"missing": nil},
			},
		}),
		CreatedAt: &now,
	}

	serialized, err := SerializeCanvasEvent(event)
	require.NoError(t, err)
	assert.Equal(t, runID.String(), serialized.RunId)

	payload, ok := serialized.Data.AsMap()["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(14000000), payload["plain"])
	assert.Equal(t, "9007199254740993", payload["unsafeInt"])
	assert.Equal(t, "12345678901234567890", payload["large"])
	assert.Equal(t, float64(0.0000001), payload["small"])
	assert.Equal(t, float64(42), payload["normal"])
	assert.Equal(t, "deploy", payload["name"])
	assert.Equal(t, true, payload["active"])
	assert.Equal(t, []any{"prod"}, payload["labels"])

	nested, ok := payload["nested"].(map[string]any)
	require.True(t, ok)
	assert.Nil(t, nested["missing"])
}
