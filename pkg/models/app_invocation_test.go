package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppInvocation_BeforeCreate_AssignsID(t *testing.T) {
	invocation := &AppInvocation{}

	require.NoError(t, invocation.BeforeCreate(nil))
	assert.NotEqual(t, uuid.Nil, invocation.ID)
}

func TestAppInvocation_BeforeCreate_PreservesExistingID(t *testing.T) {
	existingID := uuid.New()
	invocation := &AppInvocation{ID: existingID}

	require.NoError(t, invocation.BeforeCreate(nil))
	assert.Equal(t, existingID, invocation.ID)
}
