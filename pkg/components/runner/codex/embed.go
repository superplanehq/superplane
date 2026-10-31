package codex

import _ "embed"

//go:embed run.js
var runScript string

//go:embed rollout_observer.js
var rolloutObserverScript string

//go:embed codex_subscription.js
var subscriptionScript string

func SubscriptionScript() string {
	return subscriptionScript
}
