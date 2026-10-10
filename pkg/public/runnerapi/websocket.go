package runnerapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	runnerHelloTimeout = 15 * time.Second
	runnerReadTimeout  = 60 * time.Second
	runnerWriteTimeout = 10 * time.Second
	runnerPingInterval = 20 * time.Second
	runnerMessageLimit = 2 * 1024 * 1024
)

var runnerUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return strings.TrimSpace(r.Header.Get("Origin")) == ""
	},
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

type runnerConnection struct {
	socket *websocket.Conn
	mu     sync.Mutex
}

func (c *runnerConnection) write(message any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.socket.SetWriteDeadline(time.Now().Add(runnerWriteTimeout))
	return c.socket.WriteJSON(message)
}

func (s *Server) connectRunner(w http.ResponseWriter, r *http.Request) {
	authenticatedRunner, ok := runnerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "runner credential is required")
		return
	}

	socket, err := runnerUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	connection := &runnerConnection{socket: socket}
	defer socket.Close()
	socket.SetReadLimit(runnerMessageLimit)
	_ = socket.SetReadDeadline(time.Now().Add(runnerHelloTimeout))

	var hello helloMessage
	if err := socket.ReadJSON(&hello); err != nil || hello.Type != messageTypeHello {
		_ = connection.write(protocolErrorMessage{
			Type:    messageTypeError,
			Code:    "hello_required",
			Message: "the first runner message must be hello",
		})
		return
	}

	fleet, err := models.FindRunnerFleet(database.DB(r.Context()), authenticatedRunner.FleetID)
	if err != nil ||
		hello.RunnerID != authenticatedRunner.ID.String() ||
		hello.FleetID != fleet.Slug ||
		hello.Version != authenticatedRunner.RunnerVersion {
		_ = connection.write(protocolErrorMessage{
			Type:    messageTypeError,
			Code:    "identity_mismatch",
			Message: "runner identity does not match its credential",
		})
		return
	}

	connectionID := uuid.New()
	if err := authenticatedRunner.OpenConnection(database.DB(r.Context()), connectionID, time.Now()); err != nil {
		_ = connection.write(protocolErrorMessage{
			Type:    messageTypeError,
			Code:    "connection_rejected",
			Message: "runner connection was rejected",
		})
		return
	}
	s.trackConnection(authenticatedRunner.ID, connection)
	defer s.untrackConnection(authenticatedRunner.ID, connection)
	defer func() {
		_ = authenticatedRunner.CloseConnection(database.DB(r.Context()), connectionID, time.Now())
	}()

	if err := s.reconcileRunnerConnection(
		r.Context(),
		connection,
		authenticatedRunner,
		connectionID,
		hello.CurrentTaskID,
	); err != nil {
		s.writeConnectionError(connection, err, "")
		return
	}

	stopPings := make(chan struct{})
	defer close(stopPings)
	go s.sendRunnerPings(connection, stopPings)

	_ = socket.SetReadDeadline(time.Now().Add(runnerReadTimeout))
	for {
		_, raw, err := socket.ReadMessage()
		if err != nil {
			return
		}
		_ = socket.SetReadDeadline(time.Now().Add(runnerReadTimeout))

		var envelope messageEnvelope
		if err := json.Unmarshal(raw, &envelope); err != nil {
			s.writeConnectionError(connection, errInvalidRunnerMessage, "")
			continue
		}

		switch envelope.Type {
		case messageTypePong:
			if err := authenticatedRunner.TouchConnection(
				database.DB(r.Context()),
				connectionID,
				time.Now(),
			); err != nil {
				return
			}
			if authenticatedRunner.State != models.RunnerStateBusy {
				if err := s.reconcileRunnerConnection(
					r.Context(),
					connection,
					authenticatedRunner,
					connectionID,
					"",
				); err != nil {
					s.writeConnectionError(connection, err, "")
					return
				}
			}
		case messageTypeShutdownRequest:
			var shutdown shutdownRequestMessage
			if err := json.Unmarshal(raw, &shutdown); err != nil {
				s.writeConnectionError(connection, errInvalidRunnerMessage, "")
				continue
			}
			taskID, shouldShutdown, err := s.requestRunnerShutdown(
				r.Context(),
				authenticatedRunner,
				connectionID,
				shutdown,
			)
			if err != nil {
				s.writeConnectionError(connection, err, shutdown.RequestID)
				continue
			}
			if shouldShutdown {
				_ = connection.write(typedMessage{Type: messageTypeShutdown})
				return
			}
			if err := connection.write(taskControlMessage{
				Type:      messageTypeCancel,
				TaskID:    taskID.String(),
				RequestID: shutdown.RequestID,
				Reason:    shutdown.Reason,
			}); err != nil {
				return
			}
		case messageTypeComplete:
			var complete completeMessage
			if err := json.Unmarshal(raw, &complete); err != nil {
				s.writeConnectionError(connection, errInvalidRunnerMessage, "")
				continue
			}
			if err := s.completeRunnerTask(
				r.Context(),
				authenticatedRunner,
				connectionID,
				complete,
			); err != nil {
				s.writeConnectionError(connection, err, complete.RequestID)
				continue
			}
			if err := connection.write(ackMessage{
				Type:      messageTypeAck,
				RequestID: complete.RequestID,
			}); err != nil {
				return
			}
			if authenticatedRunner.State == models.RunnerStateTerminated {
				_ = connection.write(typedMessage{Type: messageTypeShutdown})
				return
			}
			if err := s.reconcileRunnerConnection(
				r.Context(),
				connection,
				authenticatedRunner,
				connectionID,
				"",
			); err != nil {
				s.writeConnectionError(connection, err, "")
				return
			}
		default:
			s.writeConnectionError(connection, errInvalidRunnerMessage, "")
		}
	}
}

func (s *Server) requestRunnerShutdown(
	ctx context.Context,
	runner *models.Runner,
	connectionID uuid.UUID,
	request shutdownRequestMessage,
) (uuid.UUID, bool, error) {
	if _, err := uuid.Parse(request.RequestID); err != nil ||
		request.Reason != shutdownReasonSignal {
		return uuid.Nil, false, errInvalidRunnerMessage
	}
	reportedTaskID, err := optionalRunnerTaskID(request.CurrentTaskID)
	if err != nil {
		return uuid.Nil, false, errInvalidRunnerMessage
	}

	var taskID uuid.UUID
	var shouldShutdown bool
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		current, err := models.FindRunner(tx, runner.ID)
		if err != nil {
			return err
		}
		*runner = *current
		if runner.CurrentConnectionID == nil || *runner.CurrentConnectionID != connectionID {
			return models.ErrRunnerConnectionReplaced
		}
		if runner.State == models.RunnerStateTerminated {
			shouldShutdown = true
			return nil
		}
		if runner.State == models.RunnerStateIdle && reportedTaskID == nil {
			if err := runner.Terminate(tx, models.RunnerTerminationInterrupted); err != nil {
				return err
			}
			shouldShutdown = true
			return nil
		}
		if runner.State != models.RunnerStateBusy || !runner.Ephemeral {
			return errRunnerStateMismatch
		}
		task, err := runner.FindActiveTask(tx)
		if err != nil ||
			reportedTaskID == nil ||
			task.ID != *reportedTaskID {
			return errRunnerStateMismatch
		}
		if err := task.RequestCancel(tx, time.Now()); err != nil {
			return err
		}
		taskID = task.ID
		return nil
	})
	return taskID, shouldShutdown, err
}

var (
	errInvalidRunnerMessage = errors.New("runner message is invalid")
	errRunnerStateMismatch  = errors.New("runner state does not match durable state")
)

func (s *Server) reconcileRunnerConnection(
	ctx context.Context,
	connection *runnerConnection,
	runner *models.Runner,
	connectionID uuid.UUID,
	reportedTaskID string,
) error {
	var task *models.RunnerTask
	var shouldSendTask bool
	var shouldStartTask bool
	var shouldShutdown bool
	var shouldCancel bool
	var reservedTask *models.RunnerTask

	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		current, err := models.FindRunner(tx, runner.ID)
		if err != nil {
			return err
		}
		*runner = *current
		if runner.CurrentConnectionID == nil || *runner.CurrentConnectionID != connectionID {
			return models.ErrRunnerConnectionReplaced
		}

		reportedID, err := optionalRunnerTaskID(reportedTaskID)
		if err != nil {
			return errInvalidRunnerMessage
		}
		active, activeErr := runner.FindActiveTask(tx)

		if runner.State == models.RunnerStateTerminated {
			if reportedID == nil {
				shouldShutdown = true
				return nil
			}
			terminal, findErr := models.FindRunnerTask(tx, *reportedID)
			if findErr != nil || terminal.RunnerID == nil || *terminal.RunnerID != runner.ID {
				return errRunnerStateMismatch
			}
			return nil
		}

		if reportedID != nil {
			if errors.Is(activeErr, models.ErrRunnerTaskNotFound) {
				terminal, findErr := models.FindRunnerTask(tx, *reportedID)
				if findErr == nil &&
					terminal.RunnerID != nil &&
					*terminal.RunnerID == runner.ID &&
					terminal.IsTerminal() {
					task = terminal
					return nil
				}
			}
			if activeErr != nil || active.ID != *reportedID {
				return errRunnerStateMismatch
			}
			if active.State == models.RunnerTaskStateReserved {
				shouldStartTask = true
			}
			task = active
			shouldCancel = active.CancelRequestedAt != nil
			return nil
		}

		switch {
		case activeErr == nil && active.State == models.RunnerTaskStateRunning:
			return errRunnerStateMismatch
		case activeErr == nil:
			task = active
		case errors.Is(activeErr, models.ErrRunnerTaskNotFound):
			if runner.State != models.RunnerStateIdle {
				return errRunnerStateMismatch
			}
			task, err = runner.ReserveNextTask(tx)
			if errors.Is(err, models.ErrRunnerTaskNotFound) {
				task = nil
				return nil
			}
			if err != nil {
				return err
			}
			reservedTask = task
		default:
			return activeErr
		}
		shouldSendTask = task != nil
		return nil
	})
	if err != nil {
		return err
	}
	if reservedTask != nil {
		telemetry.RecordRunnerTaskQueueDuration(ctx, reservedTask)
	}

	if shouldStartTask {
		if err := s.initializeTaskLogs(ctx, task.ID); err != nil {
			return err
		}
		if err := s.startRunnerTask(ctx, runner, task.ID, connectionID); err != nil {
			return err
		}
	}
	if shouldShutdown {
		return connection.write(typedMessage{Type: messageTypeShutdown})
	}
	if shouldCancel {
		return connection.write(taskControlMessage{Type: messageTypeCancel, TaskID: task.ID.String()})
	}
	if !shouldSendTask {
		return nil
	}

	if err := s.initializeTaskLogs(ctx, task.ID); err != nil {
		return err
	}
	payload, err := s.encryptor.Decrypt(ctx, task.PayloadCiphertext, []byte(task.ID.String()))
	if err != nil {
		return fmt.Errorf("decrypt runner task payload: %w", err)
	}
	if !json.Valid(payload) {
		return errors.New("runner task payload is invalid")
	}
	if err := connection.write(taskMessage{
		Type: messageTypeTask,
		Task: json.RawMessage(payload),
	}); err != nil {
		return err
	}

	return s.startRunnerTask(ctx, runner, task.ID, connectionID)
}

func (s *Server) startRunnerTask(
	ctx context.Context,
	runner *models.Runner,
	taskID uuid.UUID,
	connectionID uuid.UUID,
) error {
	return database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		currentRunner, err := models.FindRunner(tx, runner.ID)
		if err != nil {
			return err
		}
		currentTask, err := models.FindRunnerTask(tx, taskID)
		if err != nil {
			return err
		}
		if currentRunner.CurrentConnectionID == nil ||
			*currentRunner.CurrentConnectionID != connectionID {
			return models.ErrRunnerConnectionReplaced
		}
		if err := currentTask.Start(tx, currentRunner, s.activeLogStoreName, time.Now()); err != nil {
			return err
		}
		*runner = *currentRunner
		return nil
	})
}

func (s *Server) initializeTaskLogs(ctx context.Context, taskID uuid.UUID) error {
	store := runnerlogs.Current()
	if store == nil || store.Name() != s.activeLogStoreName {
		return models.ErrRunnerTaskLogStoreRequired
	}
	return store.Initialize(ctx, taskID)
}

func (s *Server) completeRunnerTask(
	ctx context.Context,
	runner *models.Runner,
	connectionID uuid.UUID,
	complete completeMessage,
) error {
	if _, err := uuid.Parse(complete.RequestID); err != nil {
		return errInvalidRunnerMessage
	}
	taskID, err := uuid.Parse(complete.TaskID)
	if err != nil {
		return errInvalidRunnerMessage
	}
	if len(complete.Result) > 0 && !json.Valid(complete.Result) {
		return errInvalidRunnerMessage
	}

	hash, err := completionHash(complete)
	if err != nil {
		return errInvalidRunnerMessage
	}

	var completedTask *models.RunnerTask
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		currentRunner, err := models.FindRunner(tx, runner.ID)
		if err != nil {
			return err
		}
		if currentRunner.CurrentConnectionID == nil ||
			*currentRunner.CurrentConnectionID != connectionID {
			return models.ErrRunnerConnectionReplaced
		}
		task, err := models.FindRunnerTask(tx, taskID)
		if err != nil {
			return err
		}
		completedNow, err := task.Complete(
			tx,
			currentRunner,
			hash,
			datatypes.JSON(complete.Result),
			complete.ExitCode,
			complete.Error,
			complete.Canceled,
			time.Now(),
		)
		if err != nil {
			return err
		}
		if completedNow {
			completedTask = task
		}
		*runner = *currentRunner
		return nil
	})
	if err == nil && completedTask != nil {
		telemetry.RecordRunnerTaskExecutionDuration(ctx, completedTask)
	}
	return err
}

func completionHash(complete completeMessage) (string, error) {
	value := struct {
		TaskID   string          `json:"task_id"`
		ExitCode int32           `json:"exit_code"`
		Error    string          `json:"error,omitempty"`
		Result   json.RawMessage `json:"result,omitempty"`
		Canceled bool            `json:"canceled,omitempty"`
	}{
		TaskID:   complete.TaskID,
		ExitCode: complete.ExitCode,
		Error:    complete.Error,
		Result:   complete.Result,
		Canceled: complete.Canceled,
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func optionalRunnerTaskID(value string) (*uuid.UUID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (s *Server) sendRunnerPings(connection *runnerConnection, stop <-chan struct{}) {
	ticker := time.NewTicker(runnerPingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if err := connection.write(typedMessage{Type: messageTypePing}); err != nil {
				_ = connection.socket.Close()
				return
			}
		case <-stop:
			return
		}
	}
}

func (s *Server) writeConnectionError(
	connection *runnerConnection,
	err error,
	requestID string,
) {
	code := "internal_error"
	message := "runner operation failed"
	switch {
	case errors.Is(err, errInvalidRunnerMessage):
		code = "invalid_message"
		message = "runner message is invalid"
	case errors.Is(err, errRunnerStateMismatch),
		errors.Is(err, models.ErrRunnerTaskNotStartable),
		errors.Is(err, models.ErrRunnerTaskNotCompletable):
		code = "state_mismatch"
		message = "runner state does not match durable state"
	case errors.Is(err, models.ErrRunnerTaskCompletionConflict):
		code = "completion_conflict"
		message = "task already has a different terminal result"
	case errors.Is(err, models.ErrRunnerConnectionReplaced):
		code = "connection_replaced"
		message = "another connection replaced this runner connection"
	}
	_ = connection.write(protocolErrorMessage{
		Type:      messageTypeError,
		Code:      code,
		Message:   message,
		RequestID: requestID,
	})
}
