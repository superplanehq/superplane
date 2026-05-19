package public

import (
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/runners"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

// HandleRunnerTaskComplete receives the completion webhook from fleet-manager.
// Route: POST /api/v1/webhooks/runner/complete/{runnerTaskID}
func (s *Server) HandleRunnerTaskComplete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	runnerTaskIDStr := vars["runnerTaskID"]
	runnerTaskID, err := uuid.Parse(runnerTaskIDStr)
	if err != nil {
		http.Error(w, "runner task not found", http.StatusNotFound)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxEventSize)
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		if _, ok := err.(*http.MaxBytesError); ok {
			http.Error(w, fmt.Sprintf("request body too large (max %d bytes)", MaxEventSize), http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "error reading request body", http.StatusBadRequest)
		return
	}

	store := runners.NewPostgresStore()
	runnerTask, err := store.FindTask(runnerTaskID)
	if err != nil {
		http.Error(w, "runner task not found", http.StatusNotFound)
		return
	}

	tx := database.Conn()
	execution, err := models.FindNodeExecutionByID(tx, runnerTask.ExecutionID)
	if err != nil {
		log.Errorf("runner webhook: execution %s not found: %v", runnerTask.ExecutionID, err)
		http.Error(w, "execution not found", http.StatusNotFound)
		return
	}

	node, err := models.FindCanvasNode(tx, execution.WorkflowID, execution.NodeID)
	if err != nil {
		log.Errorf("runner webhook: node %s not found in canvas %s: %v", execution.NodeID, execution.WorkflowID, err)
		http.Error(w, "node not found", http.StatusNotFound)
		return
	}

	action, err := s.registry.GetAction("runner")
	if err != nil {
		http.Error(w, "runner action not registered", http.StatusInternalServerError)
		return
	}

	newEvents := []models.CanvasEvent{}
	onNewEvents := func(events []models.CanvasEvent) {
		newEvents = append(newEvents, events...)
	}

	code, _, err := action.HandleWebhook(core.WebhookRequestContext{
		Body:          body,
		Headers:       r.Header,
		WorkflowID:    execution.WorkflowID.String(),
		NodeID:        execution.NodeID,
		Configuration: execution.Configuration.Data(),
		Metadata:      contexts.NewExecutionMetadataContext(tx, execution),
		Logger:        logging.ForExecution(execution, nil),
		HTTP:          s.registry.HTTPContext(),
		FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
			exec, lookupErr := models.FirstNodeExecutionByKVInTransaction(tx, execution.WorkflowID, execution.NodeID, key, value)
			if lookupErr != nil {
				return nil, lookupErr
			}
			return &core.ExecutionContext{
				ID:             exec.ID,
				WorkflowID:     exec.WorkflowID.String(),
				NodeID:         exec.NodeID,
				BaseURL:        s.BaseURL,
				Configuration:  exec.Configuration.Data(),
				HTTP:           s.registry.HTTPContext(),
				Metadata:       contexts.NewExecutionMetadataContext(tx, exec),
				NodeMetadata:   contexts.NewNodeMetadataContext(tx, node),
				ExecutionState: contexts.NewExecutionStateContext(tx, exec, onNewEvents),
				Requests:       contexts.NewExecutionRequestContext(tx, exec),
				Logger:         logging.ForExecution(exec, nil),
				Notifications:  contexts.NewNotificationContext(tx, uuid.Nil, exec.WorkflowID),
				CanvasMemory:   contexts.NewCanvasMemoryContext(tx, exec.WorkflowID),
			}, nil
		},
	})
	if err != nil {
		log.Errorf("runner webhook: handle webhook error: %v", err)
		http.Error(w, "error processing runner task", http.StatusInternalServerError)
		return
	}

	for _, event := range newEvents {
		messages.PublishCanvasEventCreatedMessage(&event)
	}

	if code != 0 && code != http.StatusOK {
		w.WriteHeader(code)
		return
	}

	w.WriteHeader(http.StatusOK)
}
