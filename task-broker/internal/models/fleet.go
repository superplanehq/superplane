package models

import "time"

// Fleet is a runner pool registered on the task-broker for routing.
type Fleet struct {
	ID          string    `gorm:"primaryKey"`
	Provisioner string    `gorm:"not null"`
	Arch        string    `gorm:"not null"`
	Size        string    `gorm:"not null"`
	CreatedAt   time.Time `gorm:"not null"`

	// LambdaFunctionName is the AWS Lambda function invoked to dispatch tasks
	// on this fleet. Required when Provisioner is "aws-lambda".
	LambdaFunctionName string `gorm:""`
	// MaxExecutionTimeoutSeconds caps execution_timeout_seconds accepted for
	// tasks on this fleet (e.g. Lambda's 15-minute hard limit). Nil means
	// no fleet-specific cap beyond the request-wide maximum.
	MaxExecutionTimeoutSeconds *int `gorm:""`
	// SupportsDocker is false for fleets that cannot run docker execution_mode
	// tasks (e.g. Lambda has no Docker daemon). Nil means unknown/unrestricted.
	SupportsDocker *bool `gorm:""`
}
