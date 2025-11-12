package merge

//
// The execution metadata associated with a merge component
// holds information about the grouping of events.
//

type ExecutionMetadata struct {
	// GroupKey is a logical key used to correlate queue items into a single execution
	GroupKey string `json:"groupKey,omitempty" mapstructure:"groupKey"`

	// EventIDs collects upstream event ids that reached this merge
	EventIDs []string `json:"eventIDs,omitempty" mapstructure:"eventIDs"`

	// StopEarly indicates the merge was short-circuited based on a stop condition
	StopEarly bool `json:"stopEarly,omitempty" mapstructure:"stopEarly"`
}
