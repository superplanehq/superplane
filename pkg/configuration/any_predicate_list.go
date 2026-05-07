package configuration

import (
	"regexp"
	"sync"
)

const (
	PredicateTypeEquals    = "equals"
	PredicateTypeNotEquals = "notEquals"
	PredicateTypeMatches   = "matches"
)

var AllPredicateOperators = []FieldOption{
	{
		Label: "Equals",
		Value: PredicateTypeEquals,
	},
	{
		Label: "Not Equals",
		Value: PredicateTypeNotEquals,
	},
	{
		Label: "Matches",
		Value: PredicateTypeMatches,
	},
}

type Predicate struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func (p *Predicate) Matches(value string) bool {
	switch p.Type {
	case PredicateTypeEquals:
		return p.Value == value

	case PredicateTypeNotEquals:
		return p.Value != value

	case PredicateTypeMatches:
		re, err := compileMatchPattern(p.Value)
		if err != nil {
			return false
		}

		return re.MatchString(value)

	default:
		return false
	}
}

// compileMatchPattern returns a compiled regex for the given pattern, caching
// successes and failures so repeated calls — common when a single set of
// predicates is evaluated against many incoming values — don't recompile.
var matchPatternCache sync.Map // map[string]matchPatternEntry

type matchPatternEntry struct {
	re  *regexp.Regexp
	err error
}

func compileMatchPattern(pattern string) (*regexp.Regexp, error) {
	if v, ok := matchPatternCache.Load(pattern); ok {
		entry := v.(matchPatternEntry)
		return entry.re, entry.err
	}

	re, err := regexp.Compile(pattern)
	matchPatternCache.Store(pattern, matchPatternEntry{re: re, err: err})
	return re, err
}

type AnyPredicateListTypeOptions struct {
	Operators []FieldOption `json:"operators"`
}

func MatchesAnyPredicate(predicates []Predicate, value string) bool {
	for _, predicate := range predicates {
		if predicate.Matches(value) {
			return true
		}
	}

	return false
}
