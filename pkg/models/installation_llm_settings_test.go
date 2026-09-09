package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test__WelcomeGrantTTL(t *testing.T) {
	assert.Equal(t, models.DefaultWelcomeGrantTTL, (*models.InstallationLLMSettings)(nil).WelcomeGrantTTL())
	assert.Equal(t, models.DefaultWelcomeGrantTTL, (&models.InstallationLLMSettings{}).WelcomeGrantTTL())
	assert.Equal(t, 7*24*time.Hour, (&models.InstallationLLMSettings{WelcomeGrantTTLDays: 7}).WelcomeGrantTTL())
}
