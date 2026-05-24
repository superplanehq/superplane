package models

import "time"

// Fleet is a runner pool registered on the task-broker for routing.
type Fleet struct {
	ID        string    `gorm:"primaryKey"`
	Labels    []string  `gorm:"serializer:json;not null"`
	CreatedAt time.Time `gorm:"not null"`
}
