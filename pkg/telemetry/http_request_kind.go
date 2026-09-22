package telemetry

import "strings"

const (
	HTTPRequestKindAPI      = "api"
	HTTPRequestKindLongPoll = "long_poll"
	HTTPRequestKindWebhook  = "webhook"

	HTTPRequestKindAttribute = "superplane.request.kind"
)

// HTTPRequestKind classifies an HTTP route template for latency and error SLIs.
// Long polls and inbound webhooks are separate workloads from interactive API calls.
func HTTPRequestKind(route string) string {
	if IsLongPollHTTPRoute(route) {
		return HTTPRequestKindLongPoll
	}
	if IsWebhookHTTPRoute(route) {
		return HTTPRequestKindWebhook
	}
	return HTTPRequestKindAPI
}

func IsLongPollHTTPRoute(route string) bool {
	return strings.Contains(route, "/runner/planning-sessions/wait")
}

func IsWebhookHTTPRoute(route string) bool {
	if strings.Contains(route, "/webhooks/{webhookID}") {
		return true
	}
	if strings.Contains(route, "/github/app/webhook") {
		return true
	}
	if strings.Contains(route, "/sentry/app/webhook") {
		return true
	}
	return strings.Contains(route, "/api/v1/polar/webhooks")
}
