package runners

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type postgresStore struct{}

func NewPostgresStore() Store {
	return &postgresStore{}
}

func (s *postgresStore) db() *gorm.DB {
	return database.Conn()
}

func (s *postgresStore) CreateFleet(name, mode, fleetURL, authToken string, labels []string) (*RunnerFleet, error) {
	if labels == nil {
		labels = []string{}
	}
	if mode == "" {
		mode = FleetModeBridge
	}
	fleet := &RunnerFleet{
		Name:      name,
		Mode:      mode,
		FleetURL:  strings.TrimSpace(fleetURL),
		AuthToken: authToken,
		Labels:    datatypes.NewJSONType(labels),
	}
	if err := s.db().Create(fleet).Error; err != nil {
		return nil, err
	}
	return fleet, nil
}

func (s *postgresStore) ListFleets() ([]RunnerFleet, error) {
	var fleets []RunnerFleet
	if err := s.db().Order("created_at ASC").Find(&fleets).Error; err != nil {
		return nil, err
	}
	return fleets, nil
}

func (s *postgresStore) FindFleet(id uuid.UUID) (*RunnerFleet, error) {
	var fleet RunnerFleet
	if err := s.db().First(&fleet, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &fleet, nil
}

func (s *postgresStore) FindFleetByAuthToken(token string) (*RunnerFleet, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var fleet RunnerFleet
	if err := s.db().First(&fleet, "auth_token = ?", token).Error; err != nil {
		return nil, err
	}
	return &fleet, nil
}

func (s *postgresStore) DeleteFleet(id uuid.UUID) error {
	return s.db().Delete(&RunnerFleet{}, "id = ?", id).Error
}

func (s *postgresStore) CreateTask(id uuid.UUID, fleetID uuid.UUID, fleetTaskID string, executionID uuid.UUID) (*RunnerTask, error) {
	task := &RunnerTask{
		ID:          id,
		FleetID:     fleetID,
		FleetTaskID: fleetTaskID,
		ExecutionID: executionID,
		Status:      TaskStatusQueued,
		Spec:        datatypes.NewJSONType(JobSpec{}),
	}
	if err := s.db().Create(task).Error; err != nil {
		return nil, err
	}
	return task, nil
}

func (s *postgresStore) FindTask(id uuid.UUID) (*RunnerTask, error) {
	var task RunnerTask
	if err := s.db().First(&task, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *postgresStore) FindTaskByExecutionID(executionID uuid.UUID) (*RunnerTask, error) {
	var task RunnerTask
	if err := s.db().Where("execution_id = ?", executionID).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *postgresStore) EnqueueJob(fleetID, executionID uuid.UUID, spec JobSpec) (*RunnerTask, error) {
	id := uuid.New()
	task := &RunnerTask{
		ID:          id,
		FleetID:     fleetID,
		FleetTaskID: id.String(),
		ExecutionID: executionID,
		Status:      TaskStatusQueued,
		Spec:        datatypes.NewJSONType(spec),
	}
	if err := s.db().Create(task).Error; err != nil {
		return nil, err
	}
	return task, nil
}

func (s *postgresStore) ClaimNextQueuedJob(fleetID uuid.UUID) (*RunnerTask, error) {
	var task RunnerTask
	err := s.db().Transaction(func(tx *gorm.DB) error {
		var taskIDStr string
		if err := tx.Raw(`
			SELECT id::text FROM runner_tasks
			WHERE fleet_id = ? AND status = ?
			ORDER BY created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		`, fleetID, TaskStatusQueued).Scan(&taskIDStr).Error; err != nil {
			return err
		}
		if strings.TrimSpace(taskIDStr) == "" {
			return nil
		}
		taskID, err := uuid.Parse(taskIDStr)
		if err != nil {
			return err
		}
		task.ID = taskID
		now := time.Now().UTC()
		res := tx.Model(&RunnerTask{}).
			Where("id = ? AND status = ?", taskID, TaskStatusQueued).
			Updates(map[string]any{
				"status":        TaskStatusDispatched,
				"dispatched_at": now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			task.ID = uuid.Nil
			return nil
		}
		return tx.First(&task, "id = ?", taskID).Error
	})
	if err != nil {
		return nil, err
	}
	if task.ID == uuid.Nil {
		return nil, nil
	}
	return &task, nil
}

func (s *postgresStore) CompleteJob(taskID uuid.UUID, req FleetCompleteRequest) (*RunnerTask, error) {
	var task RunnerTask
	err := s.db().Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&task, "id = ?", taskID).Error; err != nil {
			return err
		}
		if task.IsTerminal() {
			return nil
		}

		status := TaskStatusFailed
		if req.Canceled {
			status = TaskStatusCanceled
		} else if req.ExitCode == 0 && strings.TrimSpace(req.Error) == "" {
			status = TaskStatusSucceeded
		}

		now := time.Now().UTC()
		updates := map[string]any{
			"status":       status,
			"exit_code":    req.ExitCode,
			"output":       req.Output,
			"error":        req.Error,
			"completed_at": now,
		}
		if len(req.Result) > 0 {
			updates["result"] = datatypes.JSON(req.Result)
		}
		if req.TaskLog != nil {
			updates["task_log"] = datatypes.NewJSONType(req.TaskLog)
		}
		if err := tx.Model(&RunnerTask{}).Where("id = ?", taskID).Updates(updates).Error; err != nil {
			return err
		}
		return tx.First(&task, "id = ?", taskID).Error
	})
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// FleetTaskFromRunnerTask builds a FleetTask for the runner component finish path.
func FleetTaskFromRunnerTask(t *RunnerTask) *FleetTask {
	if t == nil {
		return nil
	}
	ft := &FleetTask{
		TaskID: t.ID.String(),
		Status: t.Status,
		Output: t.Output,
		Error:  t.Error,
	}
	if t.ExitCode != nil {
		ft.ExitCode = t.ExitCode
	}
	if len(t.Result) > 0 {
		ft.Result = json.RawMessage(t.Result)
	}
	if sink := t.TaskLog.Data(); sink != nil {
		ft.TaskLog = TaskLogToFleetLog(sink)
		if sink.CloudWatch != nil {
			ft.CloudWatchLogGroup = sink.CloudWatch.LogGroupName
			ft.CloudWatchLogStream = sink.CloudWatch.LogStreamName
		}
	}
	return ft
}

// ErrTaskAlreadyTerminal is returned when completing an already-finished task.
var ErrTaskAlreadyTerminal = errors.New("runner task already terminal")
