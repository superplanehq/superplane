package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func Test__EventSource__Accept(t *testing.T) {

	t.Run("no event types defined -> accept all events", func(t *testing.T) {
		event := EventSource{}
		e := Event{Type: "push", Raw: []byte(`{}`)}
		accepted, err := event.Accept(&e)
		require.NoError(t, err)
		assert.True(t, accepted)
		e = Event{Type: "pull_request", Raw: []byte(`{}`)}
		accepted, err = event.Accept(&e)
		require.NoError(t, err)
		assert.True(t, accepted)
	})

	t.Run("event type is accepted", func(t *testing.T) {
		event := EventSource{
			EventTypes: datatypes.JSONSlice[EventType]{
				{
					Type: "push",
				},
			},
		}

		e := Event{Type: "push", Raw: []byte(`{}`)}
		accepted, err := event.Accept(&e)
		require.NoError(t, err)
		assert.True(t, accepted)
	})

	t.Run("event type is not accepted", func(t *testing.T) {
		event := EventSource{
			EventTypes: datatypes.JSONSlice[EventType]{
				{
					Type: "push",
				},
			},
		}

		e := Event{Type: "pull_request", Raw: []byte(`{}`)}
		accepted, err := event.Accept(&e)
		require.NoError(t, err)
		assert.False(t, accepted)
	})

	t.Run("event type with filters", func(t *testing.T) {
		event := EventSource{
			EventTypes: datatypes.JSONSlice[EventType]{
				{
					Type:           "push",
					FilterOperator: FilterOperatorAnd,
					Filters: []Filter{
						{
							Type:       FilterTypeExpression,
							Expression: &ExpressionFilter{Expression: `$.ref == 'refs/heads/main'`},
						},
					},
				},
			},
		}

		//
		// Event that passes the filter
		//
		e := Event{Type: "push", Raw: []byte(`{"ref": "refs/heads/main"}`)}
		accepted, err := event.Accept(&e)
		require.NoError(t, err)
		assert.True(t, accepted)

		//
		// Event that does not pass the filter
		//
		e = Event{Type: "push", Raw: []byte(`{"ref": "refs/heads/feature-1"}`)}
		accepted, err = event.Accept(&e)
		require.NoError(t, err)
		assert.False(t, accepted)
	})
}
