package runnerapi

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"net"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/mcp"
	"github.com/superplanehq/superplane/pkg/models"
	runnercontrol "github.com/superplanehq/superplane/pkg/runners/control"
	"gorm.io/gorm"
)

func (s *Server) dispatchWaitingRunner(ctx context.Context, connection *runnerConnection, runnerID, connectionID uuid.UUID, fleetSlug, fleetScope string, stop <-chan struct{}) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			runner, err := models.FindRunner(database.DB(ctx), runnerID)
			if err != nil || runner.State != models.RunnerStateIdle {
				continue
			}
			if err := s.reconcileRunnerConnection(ctx, connection, runner, connectionID, "", fleetSlug, fleetScope); err != nil && !errors.Is(err, models.ErrRunnerPreparationUnavailable) {
				log.WithError(err).WithField("runner", runnerID).Warn("Runner assignment polling failed")
			}
		}
	}
}

func (s *Server) prepareAndDeliver(ctx context.Context, connection *runnerConnection, runnerID, connectionID, taskID uuid.UUID) {
	if !connection.deliveryMu.TryLock() {
		return
	}
	defer connection.deliveryMu.Unlock()
	db := database.DB(ctx)
	task, err := models.FindRunnerTask(db, taskID)
	if err != nil {
		return
	}
	preparation, err := models.ClaimRunnerTaskPreparation(db, taskID, runnerID, connectionID, time.Now())
	if errors.Is(err, models.ErrRunnerPreparationExpired) {
		s.failPreparation(ctx, task, preparation, "Runner credential preparation exceeded five minutes.")
		return
	}
	if err != nil {
		return
	}
	var payload []byte
	if preparation.PreparedAt != nil {
		task, err = models.FindRunnerTask(db, taskID)
		if err != nil {
			return
		}
		payload, err = s.encryptor.Decrypt(ctx, task.PayloadCiphertext, []byte(taskID.String()))
	} else {
		attemptCtx, cancel := context.WithDeadline(ctx, *preparation.AttemptExpiresAt)
		httpClient := &preparationHTTP{ctx: attemptCtx, http: s.registry.HTTPContext()}
		if logErr := s.initializeTaskLogs(attemptCtx, taskID); logErr != nil {
			err = &runnerLogInitializationError{cause: logErr}
		} else {
			payload, err = s.refreshTaskPayload(attemptCtx, task, preparation, httpClient)
		}
		if err == nil {
			var ciphertext []byte
			ciphertext, err = s.encryptor.Encrypt(attemptCtx, payload, []byte(taskID.String()))
			if err == nil {
				err = preparation.Accept(database.DB(attemptCtx), runnerID, connectionID, ciphertext, preparedPayloadValidUntil(payload), time.Now(), func(tx *gorm.DB) error { return s.validateRefreshResources(tx, task, payload) })
			}
		}
		cancel()
		if err != nil {
			if errors.Is(err, models.ErrRunnerPreparationUnavailable) || errors.Is(err, context.Canceled) && ctx.Err() != nil {
				return
			}
			if !preparation.Deadline.After(time.Now()) {
				s.failPreparation(ctx, task, preparation, "Runner credential preparation exceeded five minutes.")
				return
			}
			if retryablePreparationError(err) || httpClient.retryable {
				delay := max(preparationRetryDelay(preparation.AttemptCount), httpClient.retryAfter)
				retryAt := minTime(time.Now().Add(delay), preparation.Deadline)
				if retryErr := preparation.Retry(database.DB(context.WithoutCancel(ctx)), retryAt, time.Now()); retryErr != nil {
					log.WithError(retryErr).WithField("task", taskID).Warn("Failed to persist runner preparation retry")
				}
				log.WithFields(log.Fields{"task": taskID, "attempt": preparation.AttemptCount, "retry_at": retryAt, "deadline": preparation.Deadline}).Warn("Runner credential preparation will retry")
				return
			}
			s.failPreparation(ctx, task, preparation, "Runner credentials are no longer valid or authorized.")
			return
		}
	}
	if err != nil || !json.Valid(payload) {
		s.failPreparation(ctx, task, preparation, "The saved runner payload is invalid.")
		return
	}
	guardPayload := append([]byte(nil), payload...)
	var document map[string]any
	if err := json.Unmarshal(payload, &document); err != nil {
		s.failPreparation(ctx, task, preparation, "The saved runner payload is invalid.")
		return
	}
	delete(document, "credential_refreshes")
	payload, err = json.Marshal(document)
	if err != nil {
		return
	}
	if err := task.BeginDispatch(db, runnerID, connectionID, preparation, time.Now(), func(tx *gorm.DB) error { return s.validateRefreshResources(tx, task, guardPayload) }); err != nil {
		if errors.Is(err, errInvalidCredentialDescriptor) {
			s.failPreparation(ctx, task, preparation, "Runner credentials are no longer valid or authorized.")
			return
		}
		if retryablePreparationError(err) {
			if !preparation.Deadline.After(time.Now()) {
				s.failPreparation(ctx, task, preparation, "Runner credential preparation exceeded five minutes.")
				return
			}
			_ = preparation.DelayDelivery(db, minTime(time.Now().Add(preparationRetryDelay(preparation.AttemptCount)), preparation.Deadline), time.Now())
		}
		return
	}
	if err := connection.write(taskMessage{Type: messageTypeTask, Task: json.RawMessage(payload)}); err != nil {
		// The durable marker keeps this reservation assigned even if a partial
		// socket write may already have started execution.
		_ = connection.socket.Close()
		return
	}
	runner, err := models.FindRunner(db, runnerID)
	if err != nil {
		return
	}
	if err := s.startRunnerTask(ctx, runner, taskID, connectionID); err != nil {
		_ = connection.socket.Close()
	}
}

func (s *Server) failPreparation(ctx context.Context, task *models.RunnerTask, preparation *models.RunnerTaskPreparation, diagnostic string) {
	ctx = context.WithoutCancel(ctx)
	if s.failPreparationState(ctx, task, preparation, diagnostic) {
		s.finishFailedPreparation(ctx, task)
	}
}

func (s *Server) failPreparationState(ctx context.Context, task *models.RunnerTask, preparation *models.RunnerTaskPreparation, diagnostic string) bool {
	if preparation != nil && !preparation.Deadline.After(time.Now()) && preparation.PreparedAt == nil {
		preparation = nil
	}
	if err := task.FailPreparation(database.DB(ctx), diagnostic, preparation, time.Now()); err != nil {
		return false
	}
	log.WithFields(log.Fields{"task": task.ID, "organization": task.OrganizationID, "reserved_at": task.ReservedAt, "diagnostic": diagnostic}).Warn("Runner preparation failed before dispatch")
	return true
}

func (s *Server) finishFailedPreparation(ctx context.Context, task *models.RunnerTask) {
	_ = runnercontrol.PublishOrganizationCapacity(ctx, task.OrganizationID)
	if s.nodeRequestExecutor != nil {
		s.waitForTaskExecutionRequest(ctx, task)
	}
}

func (s *Server) recoverPreparations(ctx context.Context) {
	// Provider calls and completion notifications have independent bounded
	// workers. Neither can delay release of other expired reservations.
	go s.recoverRunnerCredentialCleanup(ctx)
	const notificationConcurrency = 4
	completed := make(chan *models.RunnerTask, 64)
	for range notificationConcurrency {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case task := <-completed:
					s.finishFailedPreparation(ctx, task)
				}
			}
		}()
	}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		s.sweepPreparations(ctx, completed)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Server) sweepPreparations(ctx context.Context, completed chan<- *models.RunnerTask) {
	tasks, err := models.ListExpiredRunnerPreparations(database.DB(ctx), time.Now())
	if err != nil {
		log.WithError(err).Warn("Failed to list expired runner preparations")
		return
	}
	for i := range tasks {
		if ctx.Err() != nil {
			return
		}
		if !s.failPreparationState(ctx, &tasks[i], nil, "Runner credential preparation exceeded five minutes.") {
			continue
		}
		select {
		case completed <- &tasks[i]:
		default:
			// The terminal result is durable. Existing execution and capacity
			// polling recover notifications when the bounded queue is full.
		}
	}
}

func (s *Server) recoverRunnerCredentialCleanup(ctx context.Context) {
	const cleanupConcurrency = 4
	workers := make(chan struct{}, cleanupConcurrency)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		cleanup, err := models.ListRunnerCredentialCleanup(database.DB(ctx), time.Now())
		if err != nil {
			log.WithError(err).Warn("Failed to list runner credential cleanup")
		}
		for i := range cleanup {
			if ctx.Err() != nil {
				return
			}
			select {
			case workers <- struct{}{}:
				go func(record models.RunnerCredentialCleanup) {
					defer func() { <-workers }()
					claimed, err := record.Claim(database.DB(ctx), time.Now())
					if err != nil {
						log.WithError(err).WithField("credential_cleanup", record.ID).Warn("Failed to claim runner credential cleanup")
						return
					}
					if !claimed {
						return
					}
					attemptCtx, cancel := context.WithTimeout(ctx, models.RunnerPreparationAttemptTimeout)
					err = s.cleanupRunnerCredential(attemptCtx, &record, &preparationHTTP{ctx: attemptCtx, http: s.registry.HTTPContext()})
					cancel()
					if err == nil {
						return
					}
					_ = record.Retry(database.DB(context.WithoutCancel(ctx)), time.Now())
					log.WithFields(log.Fields{"task": record.TaskID, "credential_cleanup": record.ID, "expires_at": record.ExpiresAt}).Warn("Runner credential cleanup failed; finite expiry remains active")
				}(cleanup[i])
			default:
				// Persisted records remain eligible for the next sweep.
				break
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Log initialization shares the bounded preparation attempt. A storage outage
// retries under the same durable deadline as credential preparation.
type runnerLogInitializationError struct{ cause error }

func (e *runnerLogInitializationError) Error() string {
	return "runner log initialization is temporarily unavailable"
}
func (e *runnerLogInitializationError) Unwrap() error { return e.cause }

func retryablePreparationError(err error) bool {
	if errors.Is(err, mcp.ErrFactoryAgentResourceRefreshBusy) {
		return true
	}
	var mcpStorage *mcp.FactoryAgentResourceRefreshStorageError
	if errors.As(err, &mcpStorage) {
		return true
	}
	var logs *runnerLogInitializationError
	if errors.As(err, &logs) {
		return true
	}
	var storage *credentialStorageError
	if errors.As(err, &storage) {
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var network net.Error
	if errors.As(err, &network) {
		return true
	}
	var provider *core.ProviderAPIError
	if errors.As(err, &provider) {
		return provider.IsTransport() || provider.IsRateLimited() || provider.IsUnavailable()
	}
	var token *mcp.TokenEndpointError
	if errors.As(err, &token) {
		return token.StatusCode == 429 || token.StatusCode >= 500
	}
	return false
}

func preparationRetryDelay(attempt int) time.Duration {
	seconds := 30
	if attempt <= 4 {
		seconds = 1 << max(attempt, 1)
	}
	delay := time.Duration(seconds) * time.Second
	return delay + time.Duration(rand.Int64N(int64(delay/5)+1))
}

func minTime(first, second time.Time) time.Time {
	if first.Before(second) {
		return first
	}
	return second
}
