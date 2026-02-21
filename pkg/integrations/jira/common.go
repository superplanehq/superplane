package jira

// NodeMetadata stores metadata on trigger/component nodes.
type NodeMetadata struct {
	Project *Project `json:"project,omitempty"`
}
