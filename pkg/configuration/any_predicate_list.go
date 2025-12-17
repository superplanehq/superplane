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
		matches, err := regexp.MatchString(p.Value, value)
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
