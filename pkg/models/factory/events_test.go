package factory

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutomationRef_PreservesZeroStepIndex(t *testing.T) {
	stepIndex := 0
	ref := AutomationRef{
		LineName:  "Plan",
		StepIndex: &stepIndex,
		StepName:  "intake",
	}

	data, err := json.Marshal(ref)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"stepIndex":0`)

	var round AutomationRef
	require.NoError(t, json.Unmarshal(data, &round))
	require.NotNil(t, round.StepIndex)
	assert.Equal(t, 0, *round.StepIndex)
}

func TestAutomationRef_OmitsUnsetStepIndex(t *testing.T) {
	ref := AutomationRef{NodeName: "gate"}

	data, err := json.Marshal(ref)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "stepIndex")

	var round AutomationRef
	require.NoError(t, json.Unmarshal(data, &round))
	assert.Nil(t, round.StepIndex)
}
