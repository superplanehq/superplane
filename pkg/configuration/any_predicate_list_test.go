package configuration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__MatchesAnyPredicateInList(t *testing.T) {
	t.Run("empty predicates -> always false", func(t *testing.T) {
		assert.False(t, MatchesAnyPredicateInList([]Predicate{}, []string{"pkg/foo.go"}))
	})

	t.Run("empty values -> always false", func(t *testing.T) {
		assert.False(t, MatchesAnyPredicateInList(
			[]Predicate{{Type: PredicateTypeEquals, Value: "pkg/foo.go"}},
			[]string{},
		))
	})

	t.Run("equals predicate matches one value", func(t *testing.T) {
		assert.True(t, MatchesAnyPredicateInList(
			[]Predicate{{Type: PredicateTypeEquals, Value: "go.sum"}},
			[]string{"go.mod", "go.sum"},
		))
	})

	t.Run("equals predicate matches none", func(t *testing.T) {
		assert.False(t, MatchesAnyPredicateInList(
			[]Predicate{{Type: PredicateTypeEquals, Value: "go.sum"}},
			[]string{"go.mod", "README.md"},
		))
	})

	t.Run("matches predicate (regex) matches one value", func(t *testing.T) {
		assert.True(t, MatchesAnyPredicateInList(
			[]Predicate{{Type: PredicateTypeMatches, Value: "pkg/integrations/.*"}},
			[]string{"README.md", "pkg/integrations/github/on_push.go"},
		))
	})

	t.Run("matches predicate (regex) matches none", func(t *testing.T) {
		assert.False(t, MatchesAnyPredicateInList(
			[]Predicate{{Type: PredicateTypeMatches, Value: "pkg/integrations/.*"}},
			[]string{"README.md", "web_src/src/App.tsx"},
		))
	})

	t.Run("multiple predicates, first matches", func(t *testing.T) {
		assert.True(t, MatchesAnyPredicateInList(
			[]Predicate{
				{Type: PredicateTypeEquals, Value: "go.mod"},
				{Type: PredicateTypeMatches, Value: "pkg/.*"},
			},
			[]string{"go.mod"},
		))
	})

	t.Run("multiple predicates, second matches", func(t *testing.T) {
		assert.True(t, MatchesAnyPredicateInList(
			[]Predicate{
				{Type: PredicateTypeEquals, Value: "go.mod"},
				{Type: PredicateTypeMatches, Value: "pkg/.*"},
			},
			[]string{"pkg/models/canvas.go"},
		))
	})

	t.Run("multiple predicates, none match", func(t *testing.T) {
		assert.False(t, MatchesAnyPredicateInList(
			[]Predicate{
				{Type: PredicateTypeEquals, Value: "go.mod"},
				{Type: PredicateTypeMatches, Value: "pkg/.*"},
			},
			[]string{"README.md", "web_src/src/App.tsx"},
		))
	})

	t.Run("notEquals matches when excluded path is absent", func(t *testing.T) {
		assert.True(t, MatchesAnyPredicateInList(
			[]Predicate{{Type: PredicateTypeNotEquals, Value: "README.md"}},
			[]string{"go.mod", "pkg/main.go"},
		))
	})

	t.Run("notEquals does not match when excluded path is present", func(t *testing.T) {
		assert.False(t, MatchesAnyPredicateInList(
			[]Predicate{{Type: PredicateTypeNotEquals, Value: "README.md"}},
			[]string{"README.md", "go.mod"},
		))
	})

	t.Run("equals and notEquals are ANDed for list matching", func(t *testing.T) {
		assert.False(t, MatchesAnyPredicateInList(
			[]Predicate{
				{Type: PredicateTypeEquals, Value: "go.mod"},
				{Type: PredicateTypeNotEquals, Value: "README.md"},
			},
			[]string{"README.md", "go.mod"},
		))

		assert.True(t, MatchesAnyPredicateInList(
			[]Predicate{
				{Type: PredicateTypeEquals, Value: "go.mod"},
				{Type: PredicateTypeNotEquals, Value: "README.md"},
			},
			[]string{"go.mod", "pkg/main.go"},
		))
	})
}
