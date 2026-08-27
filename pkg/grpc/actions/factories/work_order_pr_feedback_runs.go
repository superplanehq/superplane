package factories

import (
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

type prFeedbackRunHit struct {
	handler models.FactoryPRFeedbackHandler
	run     models.CanvasRun
	prURL   string
}

func loadPRFeedbackRunsByWorkOrderID(
	db *gorm.DB,
	factory *models.Factory,
	orderIDs []uuid.UUID,
) (map[uuid.UUID][]*pb.WorkOrderPRFeedbackRun, error) {
	result := map[uuid.UUID][]*pb.WorkOrderPRFeedbackRun{}
	if len(orderIDs) == 0 {
		return result, nil
	}

	keysByOrder, err := models.ListPRArtifactKeysByWorkOrderIDs(db, factory.ID, orderIDs)
	if err != nil {
		return nil, err
	}

	urls := make([]string, 0)
	orderByURL := map[string]uuid.UUID{}
	for orderID, keys := range keysByOrder {
		for _, key := range keys {
			urls = append(urls, key)
			orderByURL[key] = orderID
		}
	}
	if len(urls) == 0 {
		return result, nil
	}

	hits, err := findPRFeedbackRunsForURLs(db, factory, urls)
	if err != nil {
		return nil, err
	}

	hitsByOrder := map[uuid.UUID][]prFeedbackRunHit{}
	for _, hit := range hits {
		orderID, ok := orderByURL[hit.prURL]
		if !ok {
			continue
		}
		hitsByOrder[orderID] = append(hitsByOrder[orderID], hit)
	}

	for orderID, orderHits := range hitsByOrder {
		serialized, err := serializeWorkOrderPRFeedbackRuns(db, factory, orderID, orderHits)
		if err != nil {
			return nil, err
		}
		result[orderID] = serialized
	}

	return result, nil
}

func findPRFeedbackRunsForURLs(
	db *gorm.DB,
	factory *models.Factory,
	urls []string,
) ([]prFeedbackRunHit, error) {
	if len(urls) == 0 {
		return nil, nil
	}

	handlers, err := factory.ListPRFeedbackHandlers(db)
	if err != nil {
		return nil, err
	}
	if len(handlers) == 0 {
		return nil, nil
	}

	handlerByCanvas := make(map[uuid.UUID]models.FactoryPRFeedbackHandler, len(handlers))
	canvasIDs := make([]uuid.UUID, 0, len(handlers))
	for _, handler := range handlers {
		handlerByCanvas[handler.CanvasID] = handler
		canvasIDs = append(canvasIDs, handler.CanvasID)
	}

	events, err := models.ListRootEventsForPullRequestURLs(db, canvasIDs, urls)
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, nil
	}

	runKeys := make([]models.CanvasRunKey, 0, len(events))
	urlByRun := map[uuid.UUID]string{}
	handlerByRun := map[uuid.UUID]models.FactoryPRFeedbackHandler{}
	for _, event := range events {
		handler, ok := handlerByCanvas[event.WorkflowID]
		if !ok {
			continue
		}
		payload := prFeedbackRootPayload(event)
		prURL := prFeedbackPullRequestURL(payload)
		if prURL == "" {
			continue
		}
		runKeys = append(runKeys, models.CanvasRunKey{WorkflowID: event.WorkflowID, RunID: event.RunID})
		urlByRun[event.RunID] = prURL
		handlerByRun[event.RunID] = handler
	}

	runs, err := models.FindCanvasRunsByKeys(db, runKeys)
	if err != nil {
		return nil, err
	}

	hits := make([]prFeedbackRunHit, 0, len(runs))
	for _, run := range runs {
		handler, ok := handlerByRun[run.ID]
		if !ok {
			continue
		}
		hits = append(hits, prFeedbackRunHit{
			handler: handler,
			run:     run,
			prURL:   urlByRun[run.ID],
		})
	}

	sort.SliceStable(hits, func(i, j int) bool {
		return canvasRunCreatedAt(hits[i].run).Before(canvasRunCreatedAt(hits[j].run))
	})

	return hits, nil
}

func serializeWorkOrderPRFeedbackRuns(
	db *gorm.DB,
	factory *models.Factory,
	orderID uuid.UUID,
	hits []prFeedbackRunHit,
) ([]*pb.WorkOrderPRFeedbackRun, error) {
	runsByHandler := map[uuid.UUID][]models.CanvasRun{}
	handlerByID := map[uuid.UUID]models.FactoryPRFeedbackHandler{}
	for _, hit := range hits {
		handlerByID[hit.handler.ID] = hit.handler
		runsByHandler[hit.handler.ID] = append(runsByHandler[hit.handler.ID], hit.run)
	}

	serializedByRunID := map[uuid.UUID]*pb.FactoryPRFeedbackHandlerRun{}
	for handlerID, runs := range runsByHandler {
		handler := handlerByID[handlerID]
		serialized, err := serializePRFeedbackRuns(db, factory, &handler, runs)
		if err != nil {
			return nil, err
		}
		for i, run := range runs {
			serialized[i].WorkOrderId = orderID.String()
			serializedByRunID[run.ID] = serialized[i]
		}
	}

	result := make([]*pb.WorkOrderPRFeedbackRun, 0, len(hits))
	for _, hit := range hits {
		run, ok := serializedByRunID[hit.run.ID]
		if !ok {
			continue
		}
		result = append(result, &pb.WorkOrderPRFeedbackRun{
			HandlerId:   hit.handler.ID.String(),
			HandlerName: hit.handler.Name(),
			CanvasId:    hit.handler.CanvasID.String(),
			Run:         run,
		})
	}

	return result, nil
}

func canvasRunCreatedAt(run models.CanvasRun) time.Time {
	if run.CreatedAt == nil {
		return time.Time{}
	}
	return *run.CreatedAt
}
