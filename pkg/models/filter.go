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
