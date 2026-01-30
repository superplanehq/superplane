package rootly

// NodeMetadata contains metadata stored on trigger and component nodes
type NodeMetadata struct {
	Service *Service `json:"service,omitempty"`
}
