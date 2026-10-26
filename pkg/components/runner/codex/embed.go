package codex

import _ "embed"

//go:embed run.js
var runScript string

//go:embed codex_subscription.js
var subscriptionScript string

func SubscriptionScript() string {
	return subscriptionScript
}
