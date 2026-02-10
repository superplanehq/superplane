package dash0

// UpsertSyntheticCheckConfiguration stores synthetic check upsert input.
type UpsertSyntheticCheckConfiguration struct {
	OriginOrID string `json:"originOrId" mapstructure:"originOrId"`
	Spec       string `json:"spec" mapstructure:"spec"`
}

// SyntheticCheck is a normalized synthetic check descriptor for resource listings.
type SyntheticCheck struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Origin string `json:"origin,omitempty"`
}
