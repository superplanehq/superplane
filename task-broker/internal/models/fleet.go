package models

import "time"

// Fleet is a downstream fleet-manager the broker proxies to.
type Fleet struct {
	ID        string
	BaseURL   string
	AuthToken string
	Labels    []string
	CreatedAt time.Time
}

// BrokerTask correlates broker-facing ids with a fleet-managed task id and caller webhook.
type BrokerTask struct {
	ID               string
	FleetID          string
	FleetTaskID      string
	CallerWebhookURL string
	CreatedAt        time.Time
}
