package models

import "time"

// UsedRegistrationJTI records a registration JWT's jti after first successful use.
// Presence of a row means the token has been consumed and must not register again.
type UsedRegistrationJTI struct {
	JTI        string    `gorm:"primaryKey;size:64"`
	FleetID    string    `gorm:"not null;index"`
	ConsumedAt time.Time `gorm:"not null"`
}

// RunnerCredential is the credential issued to one registered runner.
type RunnerCredential struct {
	RunnerID        string    `gorm:"primaryKey"`
	FleetID         string    `gorm:"primaryKey;index"`
	AccessTokenHash string    `gorm:"not null;uniqueIndex;size:64"`
	CreatedAt       time.Time `gorm:"not null"`
}
