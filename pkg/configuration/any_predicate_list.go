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

// MatchesAnyPredicateInList evaluates predicates against a list of values.
//
// Semantics:
//   - equals/matches: existential (any value can satisfy)
//   - notEquals: universal (all values must differ from excluded value)
//
// This keeps notEquals meaningful for list-valued fields (e.g. changed paths),
// where a naive existential check would make it trivially true on multi-value inputs.
func MatchesAnyPredicateInList(predicates []Predicate, values []string) bool {
	if len(predicates) == 0 || len(values) == 0 {
		return false
	}

	positivePredicates := make([]Predicate, 0, len(predicates))
	for _, predicate := range predicates {
		if predicate.Type == PredicateTypeNotEquals {
			for _, value := range values {
				if predicate.Value == value {
					return false
				}
			}
			continue
		}

		positivePredicates = append(positivePredicates, predicate)
	}

	// If only notEquals predicates are configured and none excluded values were found,
	// the list matches.
	if len(positivePredicates) == 0 {
		return true
	}

	for _, value := range values {
		if MatchesAnyPredicate(positivePredicates, value) {
			return true
		}
	}

	return false
}
