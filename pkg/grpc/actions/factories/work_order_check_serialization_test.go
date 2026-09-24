package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/models"
)

func TestSerializeCheckScores_KeepsScoreFieldsOnly(t *testing.T) {
	scores := serializeCheckScores([]models.FactoryWorkOrderCheck{{
		Key:      "clarity",
		Name:     "Clarity score",
		Score:    4,
		MaxScore: 5,
		Summary:  "The plan is ready.",
		Analysis: "One decision is still open.",
	}})
	require.Len(t, scores, 1)

	got := scores[0]
	assert.Equal(t, "clarity", got.GetKey())
	assert.Equal(t, "Clarity score", got.GetName())
	assert.Equal(t, 4.0, got.GetScore())
	assert.Equal(t, 5.0, got.GetMaxScore())
}
