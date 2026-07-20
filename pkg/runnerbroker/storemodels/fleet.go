package models

import "time"

// Fleet is a runner pool registered on the task-broker for routing.
type Fleet struct {
	ID          string    `gorm:"primaryKey"`
	Provisioner string    `gorm:"not null"`
	Arch        string    `gorm:"not null"`
	Size        string    `gorm:"not null"`
	CreatedAt   time.Time `gorm:"not null"`
}

func (Fleet) TableName() string {
	return "runner_broker_fleets"
}
