package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/extensions"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	RunnerStatePending = "pending"
	RunnerStateIdle    = "idle"
	RunnerStateBusy    = "busy"

	RunnerJobTypeInvokeExtension = "invoke-extension"

	RunnerJobStatePending  = "pending"
	RunnerJobStateAssigned = "assigned"
	RunnerJobStateFinished = "finished"

	RunnerJobResultPassed = "passed"
	RunnerJobResultFailed = "failed"
)

type RunnerPool struct {
	ID             uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	OrganizationID uuid.UUID
	Name           string
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
}

func CreateRunnerPool(organizationID uuid.UUID, name string) (*RunnerPool, error) {
	now := time.Now()
	pool := &RunnerPool{
		OrganizationID: organizationID,
		Name:           name,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	err := database.Conn().Create(pool).Error
	if err != nil {
		return nil, err
	}

	return pool, nil
}

func FindPoolForOrganization(organizationID uuid.UUID) (*RunnerPool, error) {
	var pool RunnerPool
	err := database.Conn().
		Where("organization_id = ?", organizationID).
		First(&pool).
		Error

	if err != nil {
		return nil, err
	}

	return &pool, nil
}

func (p *RunnerPool) AddRunner(runnerID uuid.UUID) error {
	now := time.Now()
	runner := &Runner{
		ID:             runnerID,
		PoolID:         p.ID,
		OrganizationID: p.OrganizationID,
		State:          RunnerStateIdle,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	err := database.Conn().Create(runner).Error
	if err != nil {
		return err
	}

	return nil
}

func (p *RunnerPool) FindRunner(id uuid.UUID) (*Runner, error) {
	var runner Runner
	err := database.Conn().
		Where("organization_id = ?", p.OrganizationID).
		Where("pool_id = ?", p.ID).
		Where("id = ?", id).
		First(&runner).
		Error

	if err != nil {
		return nil, err
	}

	return &runner, nil
}

type Runner struct {
	ID             uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	PoolID         uuid.UUID
	OrganizationID uuid.UUID
	State          string
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
}

func OccupyRunner(tx *gorm.DB, organizationID uuid.UUID, job *RunnerJob) (*Runner, error) {
	now := time.Now()

	var runner Runner
	err := tx.Transaction(func(tx2 *gorm.DB) error {
		err := tx2.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("organization_id = ?", organizationID).
			Where("state = ?", RunnerStateIdle).
			First(&runner).
			Error

		if err != nil {
			return err
		}

		runner.State = RunnerStateBusy
		runner.UpdatedAt = &now
		err = tx2.Save(&runner).Error
		if err != nil {
			return err
		}

		job.RunnerID = &runner.ID
		job.State = RunnerJobStateAssigned
		job.UpdatedAt = &now
		err = tx2.Save(job).Error
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &runner, nil
}

func FindRunnerInTransaction(tx *gorm.DB, organizationID uuid.UUID, runnerID *uuid.UUID) (*Runner, error) {
	var runner Runner
	err := tx.
		Where("organization_id = ?", organizationID).
		Where("id = ?", runnerID).
		First(&runner).
		Error

	if err != nil {
		return nil, err
	}

	return &runner, nil
}

func (r *Runner) UpdateState(state string) error {
	now := time.Now()
	r.State = state
	r.UpdatedAt = &now
	return database.Conn().Save(r).Error
}

type RunnerJob struct {
	ID             uuid.UUID `gorm:"primaryKey;default:uuid_generate_v4()"`
	OrganizationID uuid.UUID
	Type           string
	Spec           datatypes.JSONType[*RunnerJobSpec]
	State          string
	Result         string
	ResultReason   string
	RunnerID       *uuid.UUID
	CreatedAt      *time.Time
	UpdatedAt      *time.Time

	//
	// Reference to the block that this job is associated with.
	// For example, if this is a InvokeExtension job,
	// for a component.Execute() operation, referenceID points to the CanvasNodeExecution record.
	//
	// TODO: there might be a better way to do this.
	//
	ReferenceID uuid.UUID
}

type RunnerJobSpec struct {
	InvokeExtension *InvokeExtensionJobSpec `json:"invokeExtension,omitempty"`
}

type InvokeExtensionJobSpec struct {
	OrganizationID string                       `json:"organizationId"`
	Target         *extensions.InvocationTarget `json:"target"`
	Extension      *ExtensionRef                `json:"extension"`
	Version        *VersionRef                  `json:"version"`
}

type ExtensionRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type VersionRef struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Digest string `json:"digest"`
}

func CreateInvokeExtensionJob(
	tx *gorm.DB,
	organizationID uuid.UUID,
	version *ExtensionVersion,
	referenceID uuid.UUID,
	target *extensions.InvocationTarget,
) (*RunnerJob, error) {
	extension, err := FindExtension(organizationID, version.ExtensionID.String())
	if err != nil {
		return nil, err
	}

	spec := RunnerJobSpec{
		InvokeExtension: &InvokeExtensionJobSpec{
			Target:         target,
			OrganizationID: organizationID.String(),
			Extension: &ExtensionRef{
				ID:   extension.ID.String(),
				Name: extension.Name,
			},
			Version: &VersionRef{
				ID:     version.ID.String(),
				Name:   version.Name,
				Digest: version.Digest,
			},
		},
	}

	now := time.Now()
	job := &RunnerJob{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		State:          RunnerJobStatePending,
		Type:           RunnerJobTypeInvokeExtension,
		CreatedAt:      &now,
		UpdatedAt:      &now,
		Spec:           datatypes.NewJSONType(&spec),
		ReferenceID:    referenceID,
	}

	err = tx.Create(job).Error
	if err != nil {
		return nil, err
	}

	return job, nil
}

func FindRunnerJob(id uuid.UUID) (*RunnerJob, error) {
	var job RunnerJob
	err := database.Conn().
		Where("id = ?", id).
		First(&job).
		Error

	if err != nil {
		return nil, err
	}

	return &job, nil
}

func ListPendingRunnerJobs() ([]RunnerJob, error) {
	var jobs []RunnerJob
	err := database.Conn().
		Where("state = ?", RunnerJobStatePending).
		Order("created_at DESC").
		Find(&jobs).
		Error

	if err != nil {
		return nil, err
	}

	return jobs, nil
}

func LockRunnerJob(tx *gorm.DB, id uuid.UUID) (*RunnerJob, error) {
	var job RunnerJob
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("id = ?", id).
		Where("state = ?", RunnerJobStatePending).
		First(&job).
		Error

	if err != nil {
		return nil, err
	}

	return &job, nil
}

func (j *RunnerJob) UpdateState(state string) error {
	now := time.Now()
	j.State = state
	j.UpdatedAt = &now
	return database.Conn().Save(j).Error
}

func (j *RunnerJob) Finish(result string, resultReason string) error {
	now := time.Now()

	return database.Conn().Transaction(func(tx *gorm.DB) error {
		j.State = RunnerJobStateFinished
		j.Result = result
		j.ResultReason = resultReason
		j.UpdatedAt = &now
		err := tx.Save(j).Error
		if err != nil {
			return err
		}

		runner, err := FindRunnerInTransaction(tx, j.OrganizationID, j.RunnerID)
		if err != nil {
			return err
		}

		runner.State = RunnerStateIdle
		runner.UpdatedAt = &now
		err = tx.Save(runner).Error
		if err != nil {
			return err
		}

		return nil
	})
}
