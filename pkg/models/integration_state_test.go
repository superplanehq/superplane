package models

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestIntegrationSetErrorStateBoundsUnicodeDescription(t *testing.T) {
	integration := &Integration{}

	integration.SetErrorState(strings.Repeat("界", IntegrationStateDescriptionMaxLength+1))

	assert.Equal(t, IntegrationStateError, integration.State)
	assert.Len(t, []rune(integration.StateDescription), IntegrationStateDescriptionMaxLength)
	assert.True(t, utf8.ValidString(integration.StateDescription))
	assert.True(t, strings.HasSuffix(integration.StateDescription, truncatedStateDescriptionSuffix))
}
