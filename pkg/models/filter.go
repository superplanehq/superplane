package models

import "fmt"

const (
	FilterTypeData    = "data"
	FilterTypeHeader  = "header"
	FilterOperatorAnd = "and"
	FilterOperatorOr  = "or"
)

type Filter struct {
	Type   string
	Data   *DataFilter
	Header *HeaderFilter
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
	switch f.Type {
	case FilterTypeData:
		return event.EvaluateBoolExpression(f.Data.Expression, FilterTypeData)
	case FilterTypeHeader:
		return event.EvaluateBoolExpression(f.Header.Expression, FilterTypeHeader)
	default:
		return false, fmt.Errorf("invalid filter type: %s", f.Type)
	}
}

func (f *Filter) Evaluate(event *Event) (bool, error) {
	switch f.Type {
	case FilterTypeData:
		return f.EvaluateExpression(event)
	case FilterTypeHeader:
		return f.EvaluateExpression(event)

	default:
		return false, fmt.Errorf("invalid filter type: %s", f.Type)
	}
}

type DataFilter struct {
	Expression string
}

type HeaderFilter struct {
	Expression string
}
