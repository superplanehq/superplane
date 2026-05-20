package runners

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/runners/models"
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

func (s *postgresStore) CreateFleet(name, authToken string) (*models.RunnerFleet, error) {
	fleet := &models.RunnerFleet{
		Name:      name,
		AuthToken: authToken,
	}
	if err := s.db().Create(fleet).Error; err != nil {
		return nil, err
	}
	return fleet, nil
}

func (s *postgresStore) ListFleets() ([]models.RunnerFleet, error) {
	var fleets []models.RunnerFleet
	if err := s.db().Order("created_at ASC").Find(&fleets).Error; err != nil {
		return nil, err
	}
	return fleets, nil
}

func (s *postgresStore) FindFleet(id uuid.UUID) (*models.RunnerFleet, error) {
	var fleet models.RunnerFleet
	if err := s.db().First(&fleet, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &fleet, nil
}

func (s *postgresStore) FindFleetByAuthToken(token string) (*models.RunnerFleet, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var fleet models.RunnerFleet
	if err := s.db().First(&fleet, "auth_token = ?", token).Error; err != nil {
		return nil, err
	}
	return &fleet, nil
}

func (s *postgresStore) DeleteFleet(id uuid.UUID) error {
	return s.db().Delete(&models.RunnerFleet{}, "id = ?", id).Error
}

func (s *postgresStore) FindTask(id uuid.UUID) (*models.RunnerTask, error) {
	var task models.RunnerTask
	if err := s.db().First(&task, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *postgresStore) FindTaskByExecutionID(executionID uuid.UUID) (*models.RunnerTask, error) {
	var task models.RunnerTask
	if err := s.db().Where("execution_id = ?", executionID).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *postgresStore) EnqueueJob(fleetID, executionID uuid.UUID, spec models.JobSpec) (*models.RunnerTask, error) {
	id := uuid.New()
	task := &models.RunnerTask{
		ID:          id,
		FleetID:     fleetID,
		FleetTaskID: id.String(),
		ExecutionID: executionID,
		Status:      models.TaskStatusQueued,
		Spec:        datatypes.NewJSONType(spec),
	}
	if err := s.db().Create(task).Error; err != nil {
		return nil, err
	}
	return task, nil
}

func (s *postgresStore) ClaimNextQueuedJob(fleetID uuid.UUID) (*models.RunnerTask, error) {
	var task models.RunnerTask
	err := s.db().Transaction(func(tx *gorm.DB) error {
		var taskIDStr string
		if err := tx.Raw(`
			SELECT id::text FROM runner_tasks
			WHERE fleet_id = ? AND status = ?
			ORDER BY created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		`, fleetID, models.TaskStatusQueued).Scan(&taskIDStr).Error; err != nil {
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
		res := tx.Model(&models.RunnerTask{}).
			Where("id = ? AND status = ?", taskID, models.TaskStatusQueued).
			Updates(map[string]any{
				"status":        models.TaskStatusDispatched,
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

func (s *postgresStore) CompleteJob(taskID uuid.UUID, req models.FleetCompleteRequest) (*models.RunnerTask, error) {
	var task models.RunnerTask
	err := s.db().Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&task, "id = ?", taskID).Error; err != nil {
			return err
		}
		if task.IsTerminal() {
			return nil
		}

		status := models.TaskStatusFailed
		if req.Canceled {
			status = models.TaskStatusCanceled
		} else if req.ExitCode == 0 && strings.TrimSpace(req.Error) == "" {
			status = models.TaskStatusSucceeded
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
		if err := tx.Model(&models.RunnerTask{}).Where("id = ?", taskID).Updates(updates).Error; err != nil {
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
func FleetTaskFromRunnerTask(t *models.RunnerTask) *models.FleetTask {
	if t == nil {
		return nil
	}
	ft := &models.FleetTask{
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
