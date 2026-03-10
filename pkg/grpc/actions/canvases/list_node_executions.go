package canvases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func ListNodeExecutions(ctx context.Context, registry *registry.Registry, workflowID, nodeID string, pbStates []pb.CanvasNodeExecution_State, pbResults []pb.CanvasNodeExecution_Result, limit uint32, before *timestamppb.Timestamp) (*pb.ListNodeExecutionsResponse, error) {
	wfID, err := uuid.Parse(workflowID)
	if err != nil {
		return nil, err
	}

	workflowNode, err := models.FindCanvasNode(database.Conn(), wfID, nodeID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.NotFound, "canvas node not found")
		}

		return nil, err
	}

	states, err := validateExecutionStates(pbStates)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	results, err := validateExecutionResults(pbResults)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	limit = getLimit(limit)
	beforeTime := getBefore(before)

	//
	// List and count executions
	//
	executions, err := models.ListNodeExecutions(wfID, nodeID, states, results, int(limit), beforeTime)
	if err != nil {
		return nil, err
	}

	totalCount, err := models.CountNodeExecutions(wfID, nodeID, states, results)
	if err != nil {
		return nil, err
	}

	serialized, err := SerializeNodeExecutionsForSingleNode(workflowNode, executions)
	if err != nil {
		return nil, err
	}

	return &pb.ListNodeExecutionsResponse{
		Executions:    serialized,
		TotalCount:    uint32(totalCount),
		HasNextPage:   hasNextPage(len(executions), int(limit), totalCount),
		LastTimestamp: getLastExecutionTimestamp(executions),
	}, nil
}

func SerializeNodeExecutionsForSingleNode(node *models.CanvasNode, executions []models.CanvasNodeExecution) ([]*pb.CanvasNodeExecution, error) {
	if node.Type != models.NodeTypeBlueprint {
		return SerializeNodeExecutions(executions, []models.CanvasNodeExecution{})
	}

	childExecutions, err := models.FindChildExecutionsForMultiple(executionIDs(executions))
	if err != nil {
		return nil, err
	}

	return SerializeNodeExecutions(executions, childExecutions)
}

func SerializeNodeExecutions(executions []models.CanvasNodeExecution, childExecutions []models.CanvasNodeExecution) ([]*pb.CanvasNodeExecution, error) {
	var rootEvents, inputEvents, outputEvents []models.CanvasEvent
	var rootEventsErr, inputEventsErr, outputEventsErr error
	var cancelledByUsers []models.User
	var cancelledByUsersErr error
	var wg sync.WaitGroup

	//
	// Fetch all execution resources in parallel
	//
	wg.Add(4)

	//
	// Root events
	//
	go func() {
		defer wg.Done()
		rootEvents, rootEventsErr = models.FindCanvasEvents(rootEventIDs(executions))
	}()

	//
	// Input events
	//
	go func() {
		defer wg.Done()
		inputEvents, inputEventsErr = models.FindCanvasEvents(eventIDs(executions))
	}()

	//
	// Output events
	//
	go func() {
		defer wg.Done()
		outputEvents, outputEventsErr = models.FindCanvasEventsForExecutions(executionIDs(executions))
	}()

	//
	// Cancelled-by users
	//
	go func() {
		defer wg.Done()
		cancelledByUsers, cancelledByUsersErr = models.FindMaybeDeletedUsersByIDs(cancelledByIDs(executions))
	}()

	wg.Wait()

	if rootEventsErr != nil {
		return nil, fmt.Errorf("error finding root events: %v", rootEventsErr)
	}
	if inputEventsErr != nil {
		return nil, fmt.Errorf("error finding input events: %v", inputEventsErr)
	}
	if outputEventsErr != nil {
		return nil, fmt.Errorf("error finding output events: %v", outputEventsErr)
	}
	if cancelledByUsersErr != nil {
		return nil, fmt.Errorf("error finding cancelled-by users: %v", cancelledByUsersErr)
	}

	cancelledByUsersByID := make(map[uuid.UUID]models.User, len(cancelledByUsers))
	for _, user := range cancelledByUsers {
		cancelledByUsersByID[user.ID] = user
	}

	//
	// Combine everything into the response
	//
	result := make([]*pb.CanvasNodeExecution, 0, len(executions))
	for _, execution := range executions {
		rootEvent, err := getRootEventForExecution(execution, rootEvents)
		if err != nil {
			return nil, err
		}

		input, err := getInputForExecution(execution, inputEvents)
		if err != nil {
			return nil, err
		}

		outputs, err := getOutputsForExecution(execution, outputEvents)
		if err != nil {
			return nil, err
		}

		metadataMap := execution.Metadata.Data()
		metadata, err := structpb.NewStruct(metadataMap)
		if err != nil {
			return nil, err
		}

		configuration, err := structpb.NewStruct(execution.Configuration.Data())
		if err != nil {
			return nil, err
		}

		pbExecution := &pb.CanvasNodeExecution{
			Id:                  execution.ID.String(),
			CanvasId:            execution.WorkflowID.String(),
			NodeId:              execution.NodeID,
			ParentExecutionId:   execution.GetParentExecutionID(),
			PreviousExecutionId: execution.GetPreviousExecutionID(),
			State:               NodeExecutionStateToProto(execution.State),
			Result:              NodeExecutionResultToProto(execution.Result),
			ResultReason:        NodeExecutionResultReasonToProto(execution.ResultReason),
			ResultMessage:       execution.ResultMessage,
			CreatedAt:           timestamppb.New(*execution.CreatedAt),
			UpdatedAt:           timestamppb.New(*execution.UpdatedAt),
			Metadata:            metadata,
			Configuration:       configuration,
			Input:               input,
			Outputs:             outputs,
			RootEvent:           rootEvent,
			CancelledBy:         cancelledByRef(execution.CancelledBy, cancelledByUsersByID),
		}

		if len(childExecutions) == 0 {
			result = append(result, pbExecution)
			continue
		}

		children := filterChildrenForParent(execution.ID, childExecutions)
		childExecutions, err := SerializeNodeExecutions(children, []models.CanvasNodeExecution{})
		if err != nil {
			return nil, err
		}

		pbExecution.ChildExecutions = append(pbExecution.ChildExecutions, childExecutions...)
		result = append(result, pbExecution)
	}

	return result, nil
}

func filterChildrenForParent(parentExecutionID uuid.UUID, childExecutions []models.CanvasNodeExecution) []models.CanvasNodeExecution {
	children := []models.CanvasNodeExecution{}
	for _, child := range childExecutions {
		if child.ParentExecutionID.String() == parentExecutionID.String() {
			children = append(children, child)
		}
	}

	return children
}

func validateExecutionStates(in []pb.CanvasNodeExecution_State) ([]string, error) {
	if len(in) == 0 {
		return []string{}, nil
	}

	states := []string{}
	for _, s := range in {
		state, err := ProtoToNodeExecutionState(s)
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}

	return states, nil
}

func validateExecutionResults(in []pb.CanvasNodeExecution_Result) ([]string, error) {
	if len(in) == 0 {
		return []string{}, nil
	}

	results := []string{}
	for _, r := range in {
		result, err := ProtoToNodeExecutionResult(r)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

func ProtoToNodeExecutionState(state pb.CanvasNodeExecution_State) (string, error) {
	switch state {
	case pb.CanvasNodeExecution_STATE_PENDING:
		return models.CanvasNodeExecutionStatePending, nil
	case pb.CanvasNodeExecution_STATE_STARTED:
		return models.CanvasNodeExecutionStateStarted, nil
	case pb.CanvasNodeExecution_STATE_FINISHED:
		return models.CanvasNodeExecutionStateFinished, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "invalid execution state: %v", state)
	}
}

func ProtoToNodeExecutionResult(result pb.CanvasNodeExecution_Result) (string, error) {
	switch result {
	case pb.CanvasNodeExecution_RESULT_PASSED:
		return models.CanvasNodeExecutionResultPassed, nil
	case pb.CanvasNodeExecution_RESULT_FAILED:
		return models.CanvasNodeExecutionResultFailed, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "invalid execution result: %v", result)
	}
}

func NodeExecutionStateToProto(state string) pb.CanvasNodeExecution_State {
	switch state {
	case models.CanvasNodeExecutionStatePending:
		return pb.CanvasNodeExecution_STATE_PENDING
	case models.CanvasNodeExecutionStateStarted:
		return pb.CanvasNodeExecution_STATE_STARTED
	case models.CanvasNodeExecutionStateFinished:
		return pb.CanvasNodeExecution_STATE_FINISHED
	default:
		return pb.CanvasNodeExecution_STATE_UNKNOWN
	}
}

func NodeExecutionResultToProto(result string) pb.CanvasNodeExecution_Result {
	switch result {
	case models.CanvasNodeExecutionResultPassed:
		return pb.CanvasNodeExecution_RESULT_PASSED
	case models.CanvasNodeExecutionResultFailed:
		return pb.CanvasNodeExecution_RESULT_FAILED
	case models.CanvasNodeExecutionResultCancelled:
		return pb.CanvasNodeExecution_RESULT_CANCELLED
	default:
		return pb.CanvasNodeExecution_RESULT_UNKNOWN
	}
}

func NodeExecutionResultReasonToProto(reason string) pb.CanvasNodeExecution_ResultReason {
	switch reason {
	case models.CanvasNodeExecutionResultReasonOk:
		return pb.CanvasNodeExecution_RESULT_REASON_OK
	case models.CanvasNodeExecutionResultReasonError:
		return pb.CanvasNodeExecution_RESULT_REASON_ERROR
	case models.CanvasNodeExecutionResultReasonErrorResolved:
		return pb.CanvasNodeExecution_RESULT_REASON_ERROR_RESOLVED
	default:
		return pb.CanvasNodeExecution_RESULT_REASON_OK
	}
}

func getLastExecutionTimestamp(executions []models.CanvasNodeExecution) *timestamppb.Timestamp {
	if len(executions) > 0 {
		return timestamppb.New(*executions[len(executions)-1].CreatedAt)
	}
	return nil
}

func getLimit(limit uint32) uint32 {
	if limit == 0 || limit > 100 {
		return 100
	}
	return limit
}

func getBefore(before *timestamppb.Timestamp) *time.Time {
	if before == nil {
		return nil
	}
	t := before.AsTime()
	return &t
}

func hasNextPage(numResults, limit int, totalCount int64) bool {
	return int64(numResults) >= int64(limit) && int64(numResults) < totalCount
}

func executionIDs(executions []models.CanvasNodeExecution) []string {
	ids := make([]string, len(executions))
	for i, execution := range executions {
		ids[i] = execution.ID.String()
	}
	return ids
}

func cancelledByIDs(executions []models.CanvasNodeExecution) []uuid.UUID {
	idsMap := make(map[uuid.UUID]struct{})
	for _, execution := range executions {
		if execution.CancelledBy == nil {
			continue
		}
		idsMap[*execution.CancelledBy] = struct{}{}
	}

	ids := make([]uuid.UUID, 0, len(idsMap))
	for id := range idsMap {
		ids = append(ids, id)
	}

	return ids
}

func eventIDs(executions []models.CanvasNodeExecution) []string {
	ids := make([]string, len(executions))
	for i, execution := range executions {
		ids[i] = execution.EventID.String()
	}

	return ids
}

func rootEventIDs(executions []models.CanvasNodeExecution) []string {
	ids := make([]string, len(executions))
	for i, execution := range executions {
		ids[i] = execution.RootEventID.String()
	}

	return ids
}

func cancelledByRef(id *uuid.UUID, users map[uuid.UUID]models.User) *pb.UserRef {
	if id == nil {
		return nil
	}

	user, ok := users[*id]
	name := ""
	if ok {
		name = user.Name
	}

	return &pb.UserRef{Id: id.String(), Name: name}
}

func getInputForExecution(execution models.CanvasNodeExecution, events []models.CanvasEvent) (*structpb.Struct, error) {
	for _, event := range events {
		if event.ID.String() == execution.EventID.String() {
			eventData, ok := event.Data.Data().(map[string]any)
			if !ok {
				return nil, fmt.Errorf("event data cannot be turned into input for execution %s", execution.ID.String())
			}

			data, err := structpb.NewStruct(eventData)
			if err != nil {
				return nil, err
			}

			return data, nil
		}
	}

	return nil, fmt.Errorf("input not found for execution %s", execution.ID.String())
}

func getRootEventForExecution(execution models.CanvasNodeExecution, rootEvents []models.CanvasEvent) (*pb.CanvasEvent, error) {
	for _, rootEvent := range rootEvents {
		if rootEvent.ID.String() == execution.RootEventID.String() {
			data, ok := rootEvent.Data.Data().(map[string]any)
			if !ok {
				return nil, fmt.Errorf("event data is not a map[string]any")
			}

			s, err := structpb.NewStruct(data)
			if err != nil {
				return nil, err
			}

			return &pb.CanvasEvent{
				Id:         rootEvent.ID.String(),
				CanvasId:   rootEvent.WorkflowID.String(),
				NodeId:     rootEvent.NodeID,
				Channel:    rootEvent.Channel,
				CustomName: valueOrEmpty(rootEvent.CustomName),
				Data:       s,
				CreatedAt:  timestamppb.New(*rootEvent.CreatedAt),
			}, nil
		}
	}

	return nil, fmt.Errorf("input not found for execution %s", execution.ID.String())
}

func getOutputsForExecution(execution models.CanvasNodeExecution, events []models.CanvasEvent) (*structpb.Struct, error) {
	outputMap := map[string][]any{}
	for _, event := range events {
		if event.ExecutionID.String() == execution.ID.String() {
			if _, ok := outputMap[event.Channel]; !ok {
				outputMap[event.Channel] = []any{event.Data.Data()}
			} else {
				outputMap[event.Channel] = append(outputMap[event.Channel], event.Data.Data())
			}
		}
	}

	m, err := json.Marshal(outputMap)
	if err != nil {
		return nil, err
	}

	var outputs map[string]any
	err = json.Unmarshal(m, &outputs)
	if err != nil {
		return nil, err
	}

	data, err := structpb.NewStruct(outputs)
	if err != nil {
		return nil, err
	}

	return data, nil
}
