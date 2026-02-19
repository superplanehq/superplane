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
}

// SyntheticCheckField stores one key/value pair for synthetic check request maps.
type SyntheticCheckField struct {
	Key   string `json:"key" mapstructure:"key"`
	Value string `json:"value" mapstructure:"value"`
}
