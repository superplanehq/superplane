package configuration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__Predicate_Matches(t *testing.T) {
	t.Run("equals", func(t *testing.T) {
		p := Predicate{Type: PredicateTypeEquals, Value: "main"}
		assert.True(t, p.Matches("main"))
		assert.False(t, p.Matches("dev"))
	})

	t.Run("not equals", func(t *testing.T) {
		p := Predicate{Type: PredicateTypeNotEquals, Value: "main"}
		assert.False(t, p.Matches("main"))
		assert.True(t, p.Matches("dev"))
	})

	t.Run("matches", func(t *testing.T) {
		p := Predicate{Type: PredicateTypeMatches, Value: `^v\d+\.\d+$`}
		assert.True(t, p.Matches("v1.2"))
		assert.False(t, p.Matches("release"))
	})

	t.Run("matches with invalid pattern returns false", func(t *testing.T) {
		p := Predicate{Type: PredicateTypeMatches, Value: `(unclosed`}
		assert.False(t, p.Matches("anything"))
		// Invalid pattern stays cached as an error; subsequent calls also return false.
		assert.False(t, p.Matches("anything"))
	})

	t.Run("unknown predicate type returns false", func(t *testing.T) {
		p := Predicate{Type: "unsupported", Value: "main"}
		assert.False(t, p.Matches("main"))
	})
}

func Test__MatchesAnyPredicate(t *testing.T) {
	predicates := []Predicate{
		{Type: PredicateTypeEquals, Value: "main"},
		{Type: PredicateTypeMatches, Value: `^release/.*$`},
	}

	assert.True(t, MatchesAnyPredicate(predicates, "main"))
	assert.True(t, MatchesAnyPredicate(predicates, "release/2026-q2"))
	assert.False(t, MatchesAnyPredicate(predicates, "feature/foo"))
	assert.False(t, MatchesAnyPredicate(nil, "main"))
}

// BenchmarkMatchesAnyPredicate exercises the typical webhook-handler pattern
// where the same set of predicates is evaluated against many incoming values.
// Without the regex cache, each call recompiles every "matches" predicate.
func BenchmarkMatchesAnyPredicate(b *testing.B) {
	predicates := []Predicate{
		{Type: PredicateTypeEquals, Value: "main"},
		{Type: PredicateTypeMatches, Value: `^release/.*$`},
		{Type: PredicateTypeMatches, Value: `^hotfix/\d+$`},
		{Type: PredicateTypeMatches, Value: `^feature/[a-z0-9-]+$`},
	}
	values := []string{"main", "release/2026-q2", "feature/payments", "develop", "hotfix/42"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, v := range values {
			_ = MatchesAnyPredicate(predicates, v)
		}
	}
}
