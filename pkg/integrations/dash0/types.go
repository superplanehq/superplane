package dash0

// CheckRuleKeyValue stores one key/value pair for check rule labels and annotations.
type CheckRuleKeyValue struct {
	Key   string `json:"key" mapstructure:"key"`
	Value string `json:"value" mapstructure:"value"`
}

// UpsertCheckRuleConfiguration stores check rule upsert input.
type UpsertCheckRuleConfiguration struct {
	OriginOrID    string              `json:"originOrId" mapstructure:"originOrId"`
	Name          string              `json:"name" mapstructure:"name"`
	Expression    string              `json:"expression" mapstructure:"expression"`
	For           string              `json:"for" mapstructure:"for"`
	Interval      string              `json:"interval" mapstructure:"interval"`
	KeepFiringFor string              `json:"keepFiringFor" mapstructure:"keepFiringFor"`
	Labels        []CheckRuleKeyValue `json:"labels" mapstructure:"labels"`
	Annotations   []CheckRuleKeyValue `json:"annotations" mapstructure:"annotations"`
}
