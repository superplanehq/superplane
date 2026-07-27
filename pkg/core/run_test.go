package core

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__RunFinishedCallback__RoundTrip(t *testing.T) {
	runID := uuid.New()
	appID := uuid.New()
	errMessage := "boom"

	callback := NewRunFinishedCallback(NewRun(runID, appID, RunResultFailed, &errMessage))

	params, err := callback.ToParameters()
	require.NoError(t, err)

	decoded, err := DecodeRunFinishedCallback(params)
	require.NoError(t, err)

	assert.Equal(t, runID, decoded.Run.ID)
	assert.Equal(t, appID, decoded.Run.AppID)
	assert.Equal(t, RunResultFailed, decoded.Run.Result)
	require.NotNil(t, decoded.Run.Error)
	assert.Equal(t, errMessage, *decoded.Run.Error)
}
