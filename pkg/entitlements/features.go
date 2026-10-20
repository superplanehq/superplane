package entitlements

// Definitions is the registry for plan features. Existing features start ungated.
var Definitions = []Feature{
	{Key: "byo_subscription", Label: "Bring your own subscription"},
	{Key: "byo_key", Label: "Bring your own API key"},
	{Key: "centralized_billing", Label: "Centralized billing"},
	{Key: "self_hosted_app", Label: "Self-hosted application"},
	{Key: "priority_support", Label: "Priority support"},
}

type Feature struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Gated bool   `json:"gated"`
}
