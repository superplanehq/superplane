package models

import "time"

// RunnerRegistration is a short-lived, single-use bootstrap credential.
type RunnerRegistration struct {
	TokenHash  string     `gorm:"primaryKey;size:64"`
	FleetID    string     `gorm:"not null;index"`
	ExpiresAt  time.Time  `gorm:"not null;index"`
	ConsumedAt *time.Time `gorm:""`
	CreatedAt  time.Time  `gorm:"not null"`
}

// RunnerCredential is the credential issued to one registered runner.
type RunnerCredential struct {
	RunnerID        string    `gorm:"primaryKey"`
	FleetID         string    `gorm:"primaryKey;index"`
	AccessTokenHash string    `gorm:"not null;uniqueIndex;size:64"`
	CreatedAt       time.Time `gorm:"not null"`
}
