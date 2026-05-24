package models

import "time"

// Fleet is a downstream fleet-manager the broker proxies to.
type Fleet struct {
	ID        string    `gorm:"primaryKey"`
	BaseURL   string    `gorm:"not null"`
	AuthToken string    `gorm:""`
	Labels    []string  `gorm:"serializer:json;not null"`
	CreatedAt time.Time `gorm:"not null"`
}

// BrokerTask correlates broker-facing ids with a fleet-managed task id and caller webhook.
type BrokerTask struct {
	ID               string    `gorm:"primaryKey"`
	FleetID          string    `gorm:"not null;index:idx_broker_tasks_fleet_task"`
	FleetTaskID      string    `gorm:"index:idx_broker_tasks_fleet_task"`
	CallerWebhookURL string    `gorm:"not null"`
	CreatedAt        time.Time `gorm:"not null"`
}
