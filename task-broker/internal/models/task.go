package models

import "time"

// Task is a queued work item persisted in Postgres.
type Task struct {
	ID                           string     `gorm:"primaryKey"`
	FleetID                      string     `gorm:"not null;index:idx_tasks_fleet_status_created,priority:1"`
	RunMode                      string     `gorm:"not null;default:command_list"`
	ScriptJSON                   string     `gorm:""`
	MessageChainJSON             string     `gorm:""`
	CommandJSON                  string     `gorm:"not null"`
	CommandsJSON                 string     `gorm:""`
	SetupCommandsJSON            string     `gorm:""`
	EnvironmentJSON              string     `gorm:""`
	LabelsJSON                   string     `gorm:""`
	FilesJSON                    string     `gorm:""`
	WebhookURL                   string     `gorm:"not null"`
	Status                       string     `gorm:"not null;index:idx_tasks_fleet_status_created,priority:2"`
	CreatedAt                    time.Time  `gorm:"not null;index:idx_tasks_fleet_status_created,priority:3"`
	ClaimedAt                    *time.Time `gorm:""`
	LeaseUntil                   *time.Time `gorm:""`
	RunnerID                     string     `gorm:""`
	ExecutionMode                string     `gorm:"not null;default:host"`
	DockerImage                  string     `gorm:""`
	ExecutionTimeoutSeconds      *int       `gorm:""`
	ExitCode                     *int       `gorm:""`
	Output                       string     `gorm:""`
	ResultJSON                   string     `gorm:""`
	ErrorMessage                 string     `gorm:""`
	InfraRetryCount              int        `gorm:"not null;default:0"`
	CancelRequested              bool       `gorm:"not null;default:false"`
	RunnerTerminationRequestedAt *time.Time `gorm:""`
	// DispatchRequestedAt is set when the broker actively starts an execution
	// unit for this task (e.g. Lambda invoke). Fleets that pull work on their
	// own (EC2) leave this nil forever. Used only by the dispatch sweeper to
	// find tasks whose dispatch attempt is missing or stale.
	DispatchRequestedAt *time.Time `gorm:""`
}
