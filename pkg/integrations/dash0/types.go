package dash0

// UpsertSyntheticCheckConfiguration stores synthetic check upsert input.
type UpsertSyntheticCheckConfiguration struct {
	OriginOrID  string                `json:"originOrId" mapstructure:"originOrId"`
	Name        string                `json:"name" mapstructure:"name"`
	Enabled     bool                  `json:"enabled" mapstructure:"enabled"`
	PluginKind  string                `json:"pluginKind" mapstructure:"pluginKind"`
	Method      string                `json:"method" mapstructure:"method"`
	URL         string                `json:"url" mapstructure:"url"`
	Headers     []SyntheticCheckField `json:"headers" mapstructure:"headers"`
	RequestBody string                `json:"requestBody" mapstructure:"requestBody"`
	Spec        string                `json:"spec" mapstructure:"spec"`
}

// SyntheticCheckField stores one key/value pair for synthetic check request maps.
type SyntheticCheckField struct {
	Key   string `json:"key" mapstructure:"key"`
	Value string `json:"value" mapstructure:"value"`
}

// SyntheticCheck is a normalized synthetic check descriptor for resource listings.
type SyntheticCheck struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Origin string `json:"origin,omitempty"`
}
