package workers

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

type RunInitializer struct {
	semaphore *semaphore.Weighted
	registry  *registry.Registry
	logger    *log.Entry
}

func NewRunInitializer(registry *registry.Registry) *RunInitializer {
	return &RunInitializer{
		registry:  registry,
		semaphore: semaphore.NewWeighted(25),
		logger:    log.WithFields(log.Fields{"worker": "RunInitializer"}),
	}
}

func (w *RunInitializer) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runs, err := models.ListPendingRuns(database.Conn())
			if err != nil {
				w.logger.Errorf("Error listing pending runs: %v", err)
				continue
			}

			for _, run := range runs {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.logger.Errorf("Error acquiring semaphore: %v", err)
					continue
				}

				go func(run models.CanvasRun) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessRun(run); err != nil {
						w.logger.Errorf("Error processing run %s: %v", run.ID, err)
					}
				}(run)
			}
		}
	}
}

func (w *RunInitializer) LockAndProcessRun(run models.CanvasRun) error {
	logger := w.logger.WithFields(log.Fields{"run": run.ID})
	logger.Infof("Locking and processing run")

	newEvents := []models.CanvasEvent{}

	err := database.Conn().Transaction(func(tx *gorm.DB) error {
		locked, err := models.LockCanvasRunInTransaction(tx, run.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				logger.Infof("Run already processed - skipping")
				return nil
			}

			return err
		}

		newEvents, err = w.processSubRun(tx, locked)
		if err != nil {
			return err
		}

		newEvents = append(newEvents, newEvents...)
		return nil
	})

	if err != nil {
		return err
	}

	for _, event := range newEvents {
		messages.PublishCanvasEventCreatedMessage(&event)
	}

	return nil
}

func (w *RunInitializer) processSubRun(tx *gorm.DB, run *models.CanvasRun) ([]models.CanvasEvent, error) {
	initIndex := slices.IndexFunc(run.Callbacks, func(callback core.RunCallbackDefinition) bool {
		return callback.Kind == core.RunCallbackKindInit
	})

	if initIndex == -1 {
		return nil, fmt.Errorf("init callback definition not found")
	}

	newEvents, err := w.executeCallback(tx, run, run.Callbacks[initIndex])
	if err != nil {
		return nil, fmt.Errorf("execute callback: %w", err)
	}

	if err := run.Start(tx); err != nil {
		return nil, fmt.Errorf("start run: %w", err)
	}

	return newEvents, nil
}

func (w *RunInitializer) executeCallback(tx *gorm.DB, run *models.CanvasRun, callback core.RunCallbackDefinition) ([]models.CanvasEvent, error) {
	//
	// TODO: handle parent callbacks too
	//
	switch callback.Ref {
	case core.RunCallbackRefTarget:
		return w.executeCallbackOnTarget(tx, run, callback)
	default:
		return nil, fmt.Errorf("invalid callback reference: %s", callback.Ref)
	}
}

func (w *RunInitializer) executeCallbackOnTarget(tx *gorm.DB, run *models.CanvasRun, callback core.RunCallbackDefinition) ([]models.CanvasEvent, error) {
	newEvents := []models.CanvasEvent{}
	onNewEvents := func(events []models.CanvasEvent) {
		newEvents = append(newEvents, events...)
	}

	data, err := run.Input.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshal input: %w", err)
	}

	var input map[string]any
	if err := models.UnmarshalJSONValue(data, &input); err != nil {
		return nil, fmt.Errorf("unmarshal input: %w", err)
	}

	targetNode, err := run.FindTargetNode(tx)
	if err != nil {
		return nil, fmt.Errorf("find target node: %w", err)
	}

	//
	// TODO: handle actions too?
	//

	ref := targetNode.Ref.Data()
	trigger, err := w.registry.GetTrigger(ref.Trigger.Name)
	if err != nil {
		return nil, fmt.Errorf("get trigger: %w", err)
	}

	_, err = trigger.HandleHook(core.TriggerHookContext{
		Name:          callback.Hook,
		Logger:        logging.ForNode(*targetNode),
		Configuration: targetNode.Configuration.Data(),
		HTTP:          w.registry.HTTPContextInTransaction(tx),
		Metadata:      contexts.NewNodeMetadataContext(tx, targetNode),
		Requests:      contexts.NewNodeRequestContext(tx, targetNode),
		Events:        contexts.NewEventContext(tx, targetNode, run, onNewEvents),
		Parameters:    input,
	})

	if err != nil {
		return nil, fmt.Errorf("handle hook: %w", err)
	}

	return newEvents, nil
}
