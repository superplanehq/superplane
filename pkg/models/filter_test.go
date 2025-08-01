package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test__Filters(t *testing.T) {
	t.Run("single expression filters -> true", func(t *testing.T) {
		filters := []Filter{
			{
				Type: FilterTypeData,
				Data: &DataFilter{Expression: `$.a == 1 && $.b == 2`},
			},
			{
				Type:   FilterTypeHeader,
				Header: &HeaderFilter{Expression: `headers.c == 3 && headers.d == 4`},
			},
		}

		event := &Event{Raw: []byte(`{"a": 1, "b": 2}`), Headers: []byte(`{"c": 3, "d": 4}`)}
		accept, err := ApplyFilters(filters, FilterOperatorAnd, event)
		require.NoError(t, err)
		require.True(t, accept)
	})

	t.Run("expression filter with case insensitive headers -> true", func(t *testing.T) {
		filters := []Filter{
			{
				Type: FilterTypeData,
				Data: &DataFilter{Expression: `$.a == 1 && $.b == 2`},
			},
			{
				Type:   FilterTypeHeader,
				Header: &HeaderFilter{Expression: `headers["Content-Type"] == "application/json" && headers["X-ExAmPlE-HeAdEr"] == "value"`},
			},
		}

		event := &Event{Raw: []byte(`{"a": 1, "b": 2}`), Headers: []byte(`{"ContEnT-tYpE": "application/json", "x-exAmplE-hEAdEr": "value"}`)}
		accept, err := ApplyFilters(filters, FilterOperatorAnd, event)
		require.NoError(t, err)
		require.True(t, accept)
	})

	t.Run("single expression filter -> false", func(t *testing.T) {
		filters := []Filter{
			{
				Type: FilterTypeData,
				Data: &DataFilter{Expression: `$.a == 1 && $.b == 2`},
			},
		}

		event := &Event{Raw: []byte(`{"a": 1, "b": 3}`)}
		accept, err := ApplyFilters(filters, FilterOperatorAnd, event)
		require.NoError(t, err)
		require.False(t, accept)
	})

	t.Run("expression filter with case insensitive headers -> false", func(t *testing.T) {
		filters := []Filter{
			{
				Type:   FilterTypeHeader,
				Header: &HeaderFilter{Expression: `headers["Content-Type"] == "text/plain" && headers["X-ExAmPlE-HeAdEr"] == "some-value"`},
			},
		}

		event := &Event{Raw: []byte(`{"a": 1, "b": 3}`), Headers: []byte(`{"ContEnT-tYpE": "application/json", "x-exAmplE-hEAdEr": "wrong-value"}`)}
		accept, err := ApplyFilters(filters, FilterOperatorAnd, event)
		require.NoError(t, err)
		require.False(t, accept)
	})

	t.Run("expression filter with dot syntax -> true", func(t *testing.T) {
		filters := []Filter{
			{
				Type: FilterTypeData,
				Data: &DataFilter{Expression: `$.a.b == 2`},
			},
		}

		event := &Event{Raw: []byte(`{"a": {"b": 2}}`)}
		accept, err := ApplyFilters(filters, FilterOperatorAnd, event)
		require.NoError(t, err)
		require.True(t, accept)
	})

	t.Run("expression filter with array syntax for array -> true", func(t *testing.T) {
		filters := []Filter{
			{
				Type: FilterTypeData,
				Data: &DataFilter{Expression: `1 in $.a`},
			},
		}

		event := &Event{Raw: []byte(`{"a": [1, 2, 3]}`)}
		accept, err := ApplyFilters(filters, FilterOperatorAnd, event)
		require.NoError(t, err)
		require.True(t, accept)
	})

	t.Run("expression filter with improper dot syntax -> error", func(t *testing.T) {
		filters := []Filter{
			{
				Type: FilterTypeData,
				Data: &DataFilter{Expression: `$.a.b == 2`},
			},
		}

		event := &Event{Raw: []byte(`{"a": 1, "b": 2}`)}
		_, err := ApplyFilters(filters, FilterOperatorAnd, event)
		require.ErrorContains(t, err, "error running expression")
	})

	t.Run("multiple expression filters with AND", func(t *testing.T) {
		filters := []Filter{
			{
				Type: FilterTypeData,
				Data: &DataFilter{Expression: `$.a == 1`},
			},
			{
				Type: FilterTypeData,
				Data: &DataFilter{Expression: `$.b == 3`},
			},
		}

		event := &Event{Raw: []byte(`{"a": 1, "b": 2}`)}
		accept, err := ApplyFilters(filters, FilterOperatorAnd, event)
		require.NoError(t, err)
		require.False(t, accept)
	})

	t.Run("multiple expression filters with OR", func(t *testing.T) {
		filters := []Filter{
			{
				Type: FilterTypeData,
				Data: &DataFilter{Expression: `$.a == 1`},
			},
			{
				Type: FilterTypeData,
				Data: &DataFilter{Expression: `$.b == 3`},
			},
		}

		event := &Event{Raw: []byte(`{"a": 1, "b": 2}`)}
		accept, err := ApplyFilters(filters, FilterOperatorOr, event)
		require.NoError(t, err)
		require.True(t, accept)
	})
}
