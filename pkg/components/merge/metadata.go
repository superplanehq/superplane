package merge

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

//
// The execution metadata associated with a merge component
// holds information about the grouping of events.
//

type ExecutionMetadata struct {
	// GroupKey is a logical key used to correlate queue items into a single execution
	GroupKey string `json:"merge_group,omitempty"`

	// EventIDs collects upstream event ids that reached this merge
	EventIDs []string `json:"events_ids,omitempty"`
}

func DecodeExecutionMetadata(data map[string]any) (*ExecutionMetadata, error) {
	var metadata ExecutionMetadata

	err := mapstructure.Decode(data, &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &metadata, nil
}
