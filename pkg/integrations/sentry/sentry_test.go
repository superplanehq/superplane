package sentry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__Sentry__Name(t *testing.T) {
	s := &Sentry{}
	assert.Equal(t, "sentry", s.Name())
}

func Test__Sentry__Label(t *testing.T) {
	s := &Sentry{}
	assert.Equal(t, "Sentry", s.Label())
}

func Test__Sentry__Components(t *testing.T) {
	s := &Sentry{}
	components := s.Components()
	assert.Len(t, components, 1)
	assert.Equal(t, "sentry.updateIssue", components[0].Name())
}

func Test__Sentry__Triggers(t *testing.T) {
	s := &Sentry{}
	triggers := s.Triggers()
	assert.Len(t, triggers, 1)
	assert.Equal(t, "sentry.onIssueEvent", triggers[0].Name())
}
