package models

import "fmt"

const (
	FilterTypeExpression = "expression"
	FilterOperatorAnd    = "and"
	FilterOperatorOr     = "or"
)

type Filter struct {
	Type       string            `json:"type"`
	Expression *ExpressionFilter `json:"expression,omitempty"`
}

type ExpressionFilter struct {
	Expression string `json:"expression"`
}

func ApplyFilters(filters []Filter, operator string, event *Event) (bool, error) {
	if len(filters) == 0 {
		return true, nil
	}

	switch operator {
	case FilterOperatorOr:
		return applyOrFilter(filters, event)

	case FilterOperatorAnd:
		return applyAndFilter(filters, event)

	default:
		return false, fmt.Errorf("invalid filter operator: %s", operator)
	}
}

func applyAndFilter(filters []Filter, event *Event) (bool, error) {
	for _, filter := range filters {
		ok, err := filter.Evaluate(event)
		if err != nil {
			return false, fmt.Errorf("error evaluating filter: %v", err)
		}

		if !ok {
			return false, nil
		}
	}

	return true, nil
}

func applyOrFilter(filters []Filter, event *Event) (bool, error) {
	for _, filter := range filters {
		ok, err := filter.Evaluate(event)
		if err != nil {
			return false, fmt.Errorf("error evaluating filter: %v", err)
		}

		if ok {
			return true, nil
		}
	}

	return false, nil
}

func (f *Filter) EvaluateExpression(event *Event) (bool, error) {
	return event.EvaluateBoolExpression(f.Expression.Expression, FilterTypeExpression)
}

func (f *Filter) Evaluate(event *Event) (bool, error) {
	if f.Type != FilterTypeExpression {
		return false, fmt.Errorf("invalid filter type: %s", f.Type)
	}
	return f.EvaluateExpression(event)
}
