package sentry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__Sentry__Name(t *testing.T) {
	sentry := &Sentry{}
	assert.Equal(t, "sentry", sentry.Name())
}

func Test__Sentry__Label(t *testing.T) {
	sentry := &Sentry{}
	assert.Equal(t, "Sentry", sentry.Label())
}

func Test__Sentry__Components(t *testing.T) {
	sentry := &Sentry{}
	components := sentry.Components()
	assert.Len(t, components, 1)
	assert.Equal(t, "sentry.updateIssue", components[0].Name())
}

func Test__Sentry__Triggers(t *testing.T) {
	sentry := &Sentry{}
	triggers := sentry.Triggers()
	assert.Len(t, triggers, 1)
	assert.Equal(t, "sentry.onIssueEvent", triggers[0].Name())
}

func Test__Sentry__Configuration(t *testing.T) {
	sentry := &Sentry{}
	config := sentry.Configuration()
	assert.Len(t, config, 3)

	// Check organization field
	assert.Equal(t, "organization", config[0].Name)
	assert.True(t, config[0].Required)

	// Check authToken field
	assert.Equal(t, "authToken", config[1].Name)
	assert.True(t, config[1].Required)
	assert.True(t, config[1].Sensitive)

	// Check clientSecret field
	assert.Equal(t, "clientSecret", config[2].Name)
	assert.False(t, config[2].Required)
	assert.True(t, config[2].Sensitive)
}
