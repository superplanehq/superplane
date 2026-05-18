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

// Matches implements the default predicate semantics used by MatchesAnyPredicate.
// For PredicateTypeMatches, regexp.MatchString is used (substring search), which
// existing trigger configurations may rely on.
func (p *Predicate) Matches(value string) bool {
	return p.eval(value, false)
}

// eval evaluates the predicate against value. When anchorMatches is true,
// PredicateTypeMatches is applied as a full-string regex (\A(?:pattern)\z) so
// path-shaped values are not matched as substrings (e.g. pkg/.* vs vendor/pkg/x).
func (p *Predicate) eval(value string, anchorMatches bool) bool {
	switch p.Type {
	case PredicateTypeEquals:
		return p.Value == value

	case PredicateTypeNotEquals:
		return p.Value != value

	case PredicateTypeMatches:
		pattern := p.Value
		if anchorMatches {
			pattern = `\A(?:` + p.Value + `)\z`
		}
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
		if predicate.eval(value, false) {
			return true
		}
	}

	return false
}

// MatchesAnyPredicateInList returns true if any entry in values satisfies any
// configured predicate. Equals and notEquals behave like MatchesAnyPredicate;
// matches uses full-string regex for each value so path filters align with
// common expectations (substring matches would false-positive on paths).
func MatchesAnyPredicateInList(predicates []Predicate, values []string) bool {
	if len(predicates) == 0 || len(values) == 0 {
		return false
	}

	for _, value := range values {
		for _, predicate := range predicates {
			if predicate.eval(value, true) {
				return true
			}
		}
	}

	return false
}
