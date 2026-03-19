package hub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/extensions"
	"github.com/superplanehq/superplane/pkg/extensions/hub/protocol"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	appjwt "github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

var ErrNoWorkersAvailable = errors.New("no workers available")

const bundleTokenSubject = "extension-bundle"

type Hub struct {
	Addr             string
	ExtensionStorage *extensions.Storage
	Signer           *appjwt.Signer
	Upgrader         websocket.Upgrader
	httpServer       *http.Server
	mu               sync.Mutex
	runners          map[string]*RunnerSession
}

type RunnerSession struct {
	runner      *models.Runner
	conn        *websocket.Conn
	send        chan any
	done        chan struct{}
	connectedAt time.Time
	lastSeenAt  time.Time
	busy        bool
	closeOnce   sync.Once
}

func New(addr string, extensionStorage *extensions.Storage, signer *appjwt.Signer) *Hub {
	hub := &Hub{
		Addr:             addr,
		ExtensionStorage: extensionStorage,
		Signer:           signer,
		runners:          make(map[string]*RunnerSession),
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool {
				return true
			},
		},
	}

	hub.httpServer = &http.Server{
		Addr:    addr,
		Handler: hub.Routes(),
	}

	return hub
}

func (h *Hub) Start(ctx context.Context) error {
	log.Printf("Starting extension worker hub on %s", h.Addr)

	//
	// Start the job polling loop.
	// TODO: add RabbitMQ consumer here to speed things up.
	//
	go h.pollJobs(ctx)

	//
	// Serve HTTP / WebSocket API endpoints for worker communication.
	//
	return h.httpServer.ListenAndServe()
}

func (h *Hub) pollJobs(ctx context.Context) {
	logrus.Info("Polling jobs from database")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():

			//
			// TODO
			// Make sure you do not shutdown until all current
			// jobs are finished - cancel them after some timeout.
			//

			return
		case <-ticker.C:

			jobs, err := models.ListPendingRunnerJobs()
			if err != nil {
				logrus.Errorf("Error finding pending jobs: %v", err)
			}

			for _, job := range jobs {
				logrus.Infof("Processing job %s", job.ID)
				err := h.LockAndProcessJob(job.ID)
				if err != nil {
					logrus.Errorf("Error locking and processing job %s: %v", job.ID, err)
					continue
				}
			}
		}
	}
}

func (h *Hub) LockAndProcessJob(id uuid.UUID) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		job, err := models.LockRunnerJob(tx, id)
		if err != nil {
			return err
		}

		return h.dispatchJob(tx, job)
	})
}

/*
 * These are the routes exposed for workers to use.
 */
func (h *Hub) Routes() http.Handler {
	mux := http.NewServeMux()

	/*
	 * Endpoint used for workers to register with the hub.
	 * Authentication here is a short-lived JWT token generated
	 * by SuperPlane's signer with the (organizationId, poolId, workerId)
	 * claims that the worker can use to register.
	 */
	mux.HandleFunc("/api/v1/register", h.handleRegister)

	/*
	 * Endpoint for workers for fetching extension files.
	 * Authentication here is done through a short-lived JWT token
	 * generated and sent with `job.assign` message to the worker.
	 *
	 * TODO: probably easier to store / retrieve tarballs here.
	 *
	 * Available uses are:
	 * - /api/v1/extensions/bundle.js
	 * - /api/v1/extensions/manifest.json
	 */
	mux.HandleFunc("/api/v1/extensions/", h.handleExtensionFile)

	return mux
}

func (h *Hub) handleRegister(w http.ResponseWriter, r *http.Request) {
	log.Printf("Handling register request")

	token := strings.TrimSpace(r.URL.Query().Get(protocol.QueryToken))

	if token == "" {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	claims, err := h.Signer.ValidateAndGetClaims(token)
	if err != nil {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	organizationID := claimString(claims, "organizationId")
	poolID := claimString(claims, "poolId")
	runnerID := claimString(claims, "runnerId")

	log.Printf("Worker %s registering with organization %s and pool %s", runnerID, organizationID, poolID)

	pool, err := models.FindPoolForOrganization(uuid.MustParse(organizationID))
	if err != nil {
		logrus.Errorf("Error finding pool %s for organization: %v", poolID, err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	runner, err := pool.FindRunner(uuid.MustParse(runnerID))
	if err != nil {
		logrus.Errorf("Error finding runner %s for pool %s: %v", runnerID, poolID, err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	if _, ok := h.runners[runner.ID.String()]; ok {
		logrus.Errorf("Runner %s is already registered", runnerID)
		http.Error(w, "", http.StatusConflict)
		return
	}

	if err := runner.UpdateState(models.RunnerStateIdle); err != nil {
		logrus.Errorf("Error updating runner %s state to idle: %v", runnerID, err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	conn, err := h.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("upgrade websocket: %v", err), http.StatusBadRequest)
		return
	}

	h.registerRunner(runner, conn)
}

func (h *Hub) registerRunner(runner *models.Runner, conn *websocket.Conn) *RunnerSession {
	h.mu.Lock()
	defer h.mu.Unlock()

	session := &RunnerSession{
		runner:      runner,
		conn:        conn,
		send:        make(chan any, 16),
		done:        make(chan struct{}),
		connectedAt: time.Now().UTC(),
		lastSeenAt:  time.Now().UTC(),
	}

	runnerID := runner.ID.String()
	h.runners[runnerID] = session

	go h.writeLoop(session)
	go h.readLoop(session)

	return session
}

func claimString(claims map[string]any, key string) string {
	value, ok := claims[key]
	if !ok {
		return ""
	}

	text, ok := value.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(text)
}

func (h *Hub) handleExtensionFile(w http.ResponseWriter, r *http.Request) {
	log.Printf("Handling extension file request")

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := strings.TrimSpace(r.URL.Query().Get(protocol.QueryToken))
	if token == "" {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	claims, err := h.Signer.ValidateAndGetClaims(token)
	if err != nil {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	//
	// TODO: what is this subject check doing?
	//
	if claimString(claims, "sub") != bundleTokenSubject {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	organizationID := claimString(claims, "organizationId")
	extension := claimString(claims, "extension")
	version := claimString(claims, "version")

	//
	// Find the file from the path
	//
	fileName := strings.TrimPrefix(r.URL.Path, "/api/v1/extensions/")
	if fileName != "bundle.js" && fileName != "manifest.json" {
		http.NotFound(w, r)
		return
	}

	var (
		content     []byte
		readErr     error
		contentType string
	)

	//
	// TODO: Using a signed URL + redirect here would be better.
	//

	switch fileName {
	case "bundle.js":
		content, readErr = h.ExtensionStorage.ReadVersionBundleJS(organizationID, extension, version)
		contentType = "application/javascript"

	case "manifest.json":
		content, readErr = h.ExtensionStorage.ReadVersionManifestJSON(organizationID, extension, version)
		contentType = "application/json"

	default:
		http.NotFound(w, r)
		return
	}

	if readErr != nil {
		http.Error(w, readErr.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write(content)
}

func (h *Hub) readLoop(runner *RunnerSession) {
	defer h.unregisterRunner(runner)

	for {
		_, payload, err := runner.conn.ReadMessage()
		if err != nil {
			return
		}

		if err := h.handleRunnerMessage(runner, payload); err != nil {
			return
		}
	}
}

func (h *Hub) writeLoop(runner *RunnerSession) {
	ticker := time.NewTicker(15 * time.Second)
	defer func() {
		ticker.Stop()
		h.unregisterRunner(runner)
	}()

	for {
		select {
		case <-runner.done:
			return
		case message, ok := <-runner.send:
			if !ok {
				return
			}

			if err := runner.conn.WriteJSON(message); err != nil {
				return
			}
		case <-ticker.C:
			if err := runner.conn.WriteJSON(protocol.NewPing()); err != nil {
				return
			}
		}
	}
}

func (h *Hub) handleRunnerMessage(runner *RunnerSession, payload []byte) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	var envelope protocol.Envelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return fmt.Errorf("decode runner message: %w", err)
	}

	runner.lastSeenAt = time.Now().UTC()

	switch envelope.Type {
	case protocol.MessageTypeJobComplete:
		logrus.Infof("Received job complete message from runner %s", runner.runner.ID)
		return h.handleJobCompleteMessage(payload)

	case protocol.MessageTypePong:
		logrus.Infof("Received pong message from runner %s", runner.runner.ID)
		return nil
	}

	logrus.Infof("Received unknown message from runner %s: %s", runner.runner.ID, envelope.Type)
	return nil
}

func (h *Hub) handleJobCompleteMessage(payload []byte) error {
	var message protocol.JobCompleteMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		return err
	}

	switch message.JobType {
	case protocol.JobTypeInvokeExtension:
		return h.handleJobCompletionOutput(message.JobType, message.JobID, message.Result)
	case protocol.JobTypeExecuteCode:
		return h.handleJobCompletionOutput(message.JobType, message.JobID, message.Result)
	default:
		return fmt.Errorf("job type %s is not supported", message.JobType)
	}
}

func (h *Hub) handleJobCompletionOutput(jobType, jobID string, output *protocol.JobOutput) error {
	if output == nil {
		logrus.Infof("Job %s failed without output details", jobID)
		h.failJob(jobType, jobID, fmt.Errorf("job completed without output details"))
		return nil
	}

	if output.Success {
		logrus.Infof("Job %s completed successfully", jobID)
		h.completeJob(jobType, jobID, output.Output)
		return nil
	}

	if output.Error == nil {
		logrus.Infof("Job %s failed without error details", jobID)
		h.failJob(jobType, jobID, fmt.Errorf("job failed without error details"))
		return nil
	}

	logrus.Infof("Job %s failed with error: %s: %s", jobID, output.Error.Code, output.Error.Message)
	h.failJob(jobType, jobID, fmt.Errorf("%s: %s", output.Error.Code, output.Error.Message))
	return nil
}

func (h *Hub) unregisterRunner(runner *RunnerSession) {
	runner.closeOnce.Do(func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		if current, ok := h.runners[runner.runner.ID.String()]; ok && current == runner {
			delete(h.runners, runner.runner.ID.String())
		}

		close(runner.done)
		if runner.conn != nil {
			_ = runner.conn.Close()
		}
	})
}

func (h *Hub) generateBundleToken(job *models.RunnerJob) (string, error) {
	if job.Type != models.RunnerJobTypeInvokeExtension {
		return "", fmt.Errorf("job type %s is not supported", job.Type)
	}

	spec := job.Spec.Data()
	if spec == nil {
		return "", fmt.Errorf("missing job spec")
	}

	return h.Signer.GenerateWithClaims(bundleTokenSubject, 15*time.Minute, map[string]any{
		"organizationId": job.OrganizationID.String(),
		"extension":      spec.InvokeExtension.Extension.Name,
		"version":        spec.InvokeExtension.Version.Name,
	})
}

func (h *Hub) dispatchJob(tx *gorm.DB, job *models.RunnerJob) error {
	runner, err := models.OccupyRunner(tx, job.OrganizationID, job)
	if err != nil {
		return fmt.Errorf("error occupying runner: %w", err)
	}

	message, err := h.buildJobMessage(job)
	if err != nil {
		return fmt.Errorf("error building job payload: %w", err)
	}

	return h.queueMessage(h.runners[runner.ID.String()], message)
}

func (h *Hub) buildJobMessage(job *models.RunnerJob) (*protocol.JobAssignMessage, error) {
	spec := job.Spec.Data()
	if spec == nil {
		return nil, fmt.Errorf("missing job spec")
	}

	switch job.Type {
	case models.RunnerJobTypeInvokeExtension:
		return h.buildInvokeExtensionJobMessage(job, spec.InvokeExtension)
	case models.RunnerJobTypeExecuteCode:
		return h.buildExecuteCodeJob(job, spec.ExecuteCode)
	default:
		return nil, fmt.Errorf("job type %s is not supported", job.Type)
	}
}

func (h *Hub) buildExecuteCodeJob(job *models.RunnerJob, executeCode *models.ExecuteCodeJobSpec) (*protocol.JobAssignMessage, error) {
	return &protocol.JobAssignMessage{
		Type:           protocol.MessageTypeJobAssign,
		OrganizationID: job.OrganizationID.String(),
		JobType:        protocol.JobTypeExecuteCode,
		JobID:          job.ID.String(),
		ExecuteCode: &protocol.ExecuteCode{
			Code:    executeCode.Code,
			Timeout: executeCode.Timeout,
		},
	}, nil
}

func (h *Hub) buildInvokeExtensionJobMessage(job *models.RunnerJob, invokeExtension *models.InvokeExtensionJobSpec) (*protocol.JobAssignMessage, error) {
	bundleToken, err := h.generateBundleToken(job)
	if err != nil {
		return nil, fmt.Errorf("generate bundle token: %w", err)
	}

	payload := extensions.ExecuteInvocationPayload{
		Target: *invokeExtension.Target,
		Context: &extensions.InvocationContext{
			Configuration: map[string]any{},
			Metadata:      nil,
		},
		Invocation: &extensions.ExecuteInvocation{
			Data: map[string]any{
				"foo": "bar",
			},
		},
	}

	invocation, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal invocation payload: %w", err)
	}

	return &protocol.JobAssignMessage{
		Type:           protocol.MessageTypeJobAssign,
		OrganizationID: job.OrganizationID.String(),
		JobType:        protocol.JobTypeInvokeExtension,
		JobID:          job.ID.String(),
		InvokeExtension: &protocol.InvokeExtension{
			Extension: &protocol.ExtensionRef{
				ID:   invokeExtension.Extension.ID,
				Name: invokeExtension.Extension.Name,
			},
			Version: &protocol.VersionRef{
				ID:     invokeExtension.Version.ID,
				Name:   invokeExtension.Version.Name,
				Digest: invokeExtension.Version.Digest,
			},
			BundleToken: bundleToken,
			Invocation:  invocation,
		},
	}, nil
}

func (h *Hub) queueMessage(runner *RunnerSession, message any) error {
	select {
	case <-runner.done:
		return fmt.Errorf("runner %s is disconnected", runner.runner.ID.String())
	case runner.send <- message:
		return nil
	default:
		return fmt.Errorf("runner %s send queue is full", runner.runner.ID.String())
	}
}

func (h *Hub) completeJob(jobType string, jobID string, output json.RawMessage) {
	switch jobType {
	case protocol.JobTypeExecuteCode:
		h.completeExecuteCodeJob(jobType, jobID, output)

	case protocol.JobTypeInvokeExtension:
		h.completeInvokeExtensionJob(jobType, jobID, output)

	default:
		h.failJob(jobType, jobID, fmt.Errorf("no completion handler for job type %s", jobType))
	}
}

func (h *Hub) completeInvokeExtensionJob(jobType, jobID string, output json.RawMessage) {
	job, err := models.FindRunnerJob(uuid.MustParse(jobID))
	if err != nil {
		logrus.Errorf("Error finding job %s: %v", jobID, err)
		return
	}

	spec := job.Spec.Data()
	switch spec.InvokeExtension.Target.Operation {
	case extensions.InvocationOperationExecute:
		h.finishJob(jobID, models.RunnerJobResultPassed, "")
		execution, events, err := h.completeInvokeExtensionExecuteJob(job, output)
		if err != nil {
			logrus.Errorf("Error completing invoke extension execute job %s: %v", jobID, err)
			return
		}

		messages.NewCanvasExecutionMessage(
			execution.WorkflowID.String(),
			execution.ID.String(),
			execution.NodeID,
		).Publish()

		for _, event := range events {
			messages.NewCanvasEventCreatedMessage(event.WorkflowID.String(), &event).Publish()
		}

	default:
		h.failJob(jobType, jobID, fmt.Errorf("no completion handler for operation %s", spec.InvokeExtension.Target.Operation))
		return
	}
}

func (h *Hub) completeInvokeExtensionExecuteJob(job *models.RunnerJob, payload json.RawMessage) (*models.CanvasNodeExecution, []models.CanvasEvent, error) {
	output := extensions.ExecuteInvocationOutput{}
	if err := json.Unmarshal(payload, &output); err != nil {
		return nil, nil, fmt.Errorf("unmarshal output: %w", err)
	}

	logrus.Infof("Complete invoke extension execute job %s with output: %+v", job.ID, output)

	execution, err := models.FindUnscopedNodeExecution(job.ReferenceID)
	if err != nil {
		return nil, nil, fmt.Errorf("error finding node execution: %w", err)
	}

	return h.completeExecutionStateEffects(execution, output.Effects.ExecutionState)
}

func (h *Hub) completeExecuteCodeJob(_ string, jobID string, output json.RawMessage) {
	job, err := models.FindRunnerJob(uuid.MustParse(jobID))
	if err != nil {
		logrus.Errorf("Error finding job %s: %v", jobID, err)
		return
	}

	h.finishJob(jobID, models.RunnerJobResultPassed, "")

	execution, events, err := h.completeExecuteCodeExecutionJob(job, output)
	if err != nil {
		logrus.Errorf("Error completing execute code job %s: %v", jobID, err)
		return
	}

	messages.NewCanvasExecutionMessage(
		execution.WorkflowID.String(),
		execution.ID.String(),
		execution.NodeID,
	).Publish()

	for _, event := range events {
		messages.NewCanvasEventCreatedMessage(event.WorkflowID.String(), &event).Publish()
	}
}

func (h *Hub) completeExecuteCodeExecutionJob(job *models.RunnerJob, payload json.RawMessage) (*models.CanvasNodeExecution, []models.CanvasEvent, error) {
	var output struct {
		Effects struct {
			ExecutionState extensions.InvocationExecutionStateEffects `json:"executionState"`
		} `json:"effects"`
	}

	if err := json.Unmarshal(payload, &output); err != nil {
		return nil, nil, fmt.Errorf("unmarshal output: %w", err)
	}

	logrus.Infof("Complete execute code job %s with output: %+v", job.ID, output)

	execution, err := models.FindUnscopedNodeExecution(job.ReferenceID)
	if err != nil {
		return nil, nil, fmt.Errorf("error finding node execution: %w", err)
	}

	return h.completeExecutionStateEffects(execution, output.Effects.ExecutionState)
}

func (h *Hub) completeExecutionStateEffects(
	execution *models.CanvasNodeExecution,
	state extensions.InvocationExecutionStateEffects,
) (*models.CanvasNodeExecution, []models.CanvasEvent, error) {
	if !state.Passed {
		message := "execution failed"
		if state.Failed != nil && strings.TrimSpace(state.Failed.Message) != "" {
			message = state.Failed.Message
		}

		return execution, nil, execution.Fail(models.CanvasNodeExecutionResultReasonError, message)
	}

	outputs := map[string][]any{}
	for _, emission := range state.Emissions {
		outputs[emission.Channel] = append(outputs[emission.Channel], emission.Payloads...)
	}

	events, err := execution.Pass(outputs)
	return execution, events, err
}

func (h *Hub) failJob(jobType string, jobID string, err error) {
	//
	// Finish runner job first.
	//
	h.finishJob(jobID, models.RunnerJobResultFailed, err.Error())

	//
	// Update runner job reference next.
	//
	switch jobType {
	case protocol.JobTypeExecuteCode:
		h.failExecuteCodeJob(jobID, err)
	case protocol.JobTypeInvokeExtension:
		h.failInvokeExtensionJob(jobID, err)
	}
}

func (h *Hub) failExecuteCodeJob(jobID string, executionError error) {
	job, err := models.FindRunnerJob(uuid.MustParse(jobID))
	if err != nil {
		logrus.Errorf("Error finding job %s: %v", jobID, err)
		return
	}

	execution, err := models.FindUnscopedNodeExecution(job.ReferenceID)
	if err != nil {
		logrus.Errorf("Error finding node execution %s: %v", job.ReferenceID, err)
		return
	}

	err = execution.Fail(models.CanvasNodeExecutionResultReasonError, executionError.Error())
	if err != nil {
		logrus.Errorf("Error failing node execution %s: %v", job.ReferenceID, err)
		return
	}

	messages.NewCanvasExecutionMessage(
		execution.WorkflowID.String(),
		execution.ID.String(),
		execution.NodeID,
	).Publish()
}

func (h *Hub) failInvokeExtensionJob(jobID string, executionError error) {
	job, err := models.FindRunnerJob(uuid.MustParse(jobID))
	if err != nil {
		logrus.Errorf("Error finding job %s: %v", jobID, err)
		return
	}

	execution, err := models.FindUnscopedNodeExecution(job.ReferenceID)
	if err != nil {
		logrus.Errorf("Error finding node execution %s: %v", job.ReferenceID, err)
		return
	}

	err = execution.Fail(models.CanvasNodeExecutionResultReasonError, executionError.Error())
	if err != nil {
		logrus.Errorf("Error failing node execution %s: %v", job.ReferenceID, err)
		return
	}

	messages.NewCanvasExecutionMessage(
		execution.WorkflowID.String(),
		execution.ID.String(),
		execution.NodeID,
	).Publish()
}

/*
 * Finishes job and marks worker as idle in the database.
 */
func (h *Hub) finishJob(jobID string, result string, resultReason string) {
	job, err := models.FindRunnerJob(uuid.MustParse(jobID))
	if err != nil {
		logrus.Errorf("Error finding job %s: %v", jobID, err)
		return
	}

	err = job.Finish(result, resultReason)
	if err != nil {
		logrus.Errorf("Error finishing job %s: %v", jobID, err)
		return
	}
}
