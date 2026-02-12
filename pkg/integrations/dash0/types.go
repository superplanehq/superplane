package dash0

// GetCheckDetailsConfiguration stores action settings for check detail retrieval.
type GetCheckDetailsConfiguration struct {
	CheckID        string `json:"checkId" mapstructure:"checkId"`
	IncludeHistory bool   `json:"includeHistory" mapstructure:"includeHistory"`
}
