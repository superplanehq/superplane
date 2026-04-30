package configuration

import "regexp"

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

// EqualsAndMatchesPredicateOperators is for any-predicate-list fields where
// notEquals is intentionally unsupported (see field TypeOptions).
var EqualsAndMatchesPredicateOperators = []FieldOption{
	{
		Label: "Equals",
		Value: PredicateTypeEquals,
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
		// Match the full string, not a substring (regexp.MatchString searches anywhere).
		pattern := `\A(?:` + p.Value + `)\z`
		matches, err := regexp.MatchString(pattern, value)
		if err != nil {
			return false
		}

		return matches

	default:
		return false
	}
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

// MatchesAnyPredicateInList returns true if any entry in values satisfies any
// configured predicate (same semantics as MatchesAnyPredicate, evaluated per value).
func MatchesAnyPredicateInList(predicates []Predicate, values []string) bool {
	if len(predicates) == 0 || len(values) == 0 {
		return false
	}

	for _, value := range values {
		if MatchesAnyPredicate(predicates, value) {
			return true
		}
	}

	return false
}
