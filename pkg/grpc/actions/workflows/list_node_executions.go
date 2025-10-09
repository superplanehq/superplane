package workflows

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListNodeExecutions(ctx context.Context, registry *registry.Registry, workflowID, nodeID string, pbStates []pb.WorkflowNodeExecution_State, pbResults []pb.WorkflowNodeExecution_Result, limit uint32, before *timestamppb.Timestamp) (*pb.ListNodeExecutionsResponse, error) {
	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
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

	var executions []models.WorkflowNodeExecution
	query := database.Conn().
		Where("workflow_id = ?", workflowUUID).
		Where("node_id = ?", nodeID).
		Order("created_at DESC").
		Limit(int(limit))

	if len(states) > 0 {
		query = query.Where("state IN ?", states)
	}

	if len(results) > 0 {
		query = query.Where("result IN ?", results)
	}

	if beforeTime != nil {
		query = query.Where("created_at < ?", beforeTime)
	}

	if err := query.Find(&executions).Error; err != nil {
		return nil, err
	}

	var totalCount int64
	countQuery := database.Conn().
		Model(&models.WorkflowNodeExecution{}).
		Where("workflow_id = ?", workflowUUID).
		Where("node_id = ?", nodeID)

	if len(states) > 0 {
		countQuery = countQuery.Where("state IN ?", states)
	}

	if len(results) > 0 {
		countQuery = countQuery.Where("result IN ?", results)
	}

	if err := countQuery.Count(&totalCount).Error; err != nil {
		return nil, err
	}

	serialized, err := SerializeNodeExecutions(executions)
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

func SerializeNodeExecutions(executions []models.WorkflowNodeExecution) ([]*pb.WorkflowNodeExecution, error) {
	result := make([]*pb.WorkflowNodeExecution, 0, len(executions))

	for _, execution := range executions {
		// Get inputs from parent execution (computed field for API)
		var input *structpb.Struct
		inputData, err := execution.GetInputs()
		if err == nil {
			input, err = structpb.NewStruct(inputData)
			if err != nil {
				return nil, err
			}
		}

		outputsData := execution.Outputs.Data()
		outputsMap := make(map[string]any, len(outputsData))
		for k, v := range outputsData {
			outputsMap[k] = v
		}
		outputs, err := structpb.NewStruct(outputsMap)
		if err != nil {
			return nil, err
		}

		metadata, err := structpb.NewStruct(execution.Metadata.Data())
		if err != nil {
			return nil, err
		}

		configuration, err := structpb.NewStruct(execution.Configuration.Data())
		if err != nil {
			return nil, err
		}

		parentExecID := ""
		if execution.ParentExecutionID != nil {
			parentExecID = execution.ParentExecutionID.String()
		}

		blueprintID := ""
		if execution.BlueprintID != nil {
			blueprintID = execution.BlueprintID.String()
		}

		previousExecID := ""
		if execution.PreviousExecutionID != nil {
			previousExecID = execution.PreviousExecutionID.String()
		}

		previousOutputBranch := ""
		if execution.PreviousOutputBranch != nil {
			previousOutputBranch = *execution.PreviousOutputBranch
		}

		previousOutputIndex := int32(0)
		if execution.PreviousOutputIndex != nil {
			previousOutputIndex = int32(*execution.PreviousOutputIndex)
		}

		result = append(result, &pb.WorkflowNodeExecution{
			Id:                    execution.ID.String(),
			WorkflowId:            execution.WorkflowID.String(),
			NodeId:                execution.NodeID,
			ParentExecutionId:     parentExecID,
			BlueprintId:           blueprintID,
			State:                 NodeExecutionStateToProto(execution.State),
			Result:                NodeExecutionResultToProto(execution.Result),
			ResultReason:          NodeExecutionResultReasonToProto(execution.ResultReason),
			ResultMessage:         execution.ResultMessage,
			Input:                 input,
			Outputs:               outputs,
			CreatedAt:             timestamppb.New(*execution.CreatedAt),
			UpdatedAt:             timestamppb.New(*execution.UpdatedAt),
			Metadata:              metadata,
			Configuration:         configuration,
			PreviousExecutionId:   previousExecID,
			PreviousOutputBranch:  previousOutputBranch,
			PreviousOutputIndex:   previousOutputIndex,
		})
	}

	return result, nil
}

func validateExecutionStates(in []pb.WorkflowNodeExecution_State) ([]string, error) {
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

func validateExecutionResults(in []pb.WorkflowNodeExecution_Result) ([]string, error) {
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

func ProtoToNodeExecutionState(state pb.WorkflowNodeExecution_State) (string, error) {
	switch state {
	case pb.WorkflowNodeExecution_STATE_PENDING:
		return models.WorkflowNodeExecutionStatePending, nil
	case pb.WorkflowNodeExecution_STATE_WAITING:
		return models.WorkflowNodeExecutionStateWaiting, nil
	case pb.WorkflowNodeExecution_STATE_STARTED:
		return models.WorkflowNodeExecutionStateStarted, nil
	case pb.WorkflowNodeExecution_STATE_ROUTING:
		return models.WorkflowNodeExecutionStateRouting, nil
	case pb.WorkflowNodeExecution_STATE_FINISHED:
		return models.WorkflowNodeExecutionStateFinished, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "invalid execution state: %v", state)
	}
}

func ProtoToNodeExecutionResult(result pb.WorkflowNodeExecution_Result) (string, error) {
	switch result {
	case pb.WorkflowNodeExecution_RESULT_PASSED:
		return models.WorkflowNodeExecutionResultPassed, nil
	case pb.WorkflowNodeExecution_RESULT_FAILED:
		return models.WorkflowNodeExecutionResultFailed, nil
	default:
		return "", status.Errorf(codes.InvalidArgument, "invalid execution result: %v", result)
	}
}

func NodeExecutionStateToProto(state string) pb.WorkflowNodeExecution_State {
	switch state {
	case models.WorkflowNodeExecutionStatePending:
		return pb.WorkflowNodeExecution_STATE_PENDING
	case models.WorkflowNodeExecutionStateWaiting:
		return pb.WorkflowNodeExecution_STATE_WAITING
	case models.WorkflowNodeExecutionStateStarted:
		return pb.WorkflowNodeExecution_STATE_STARTED
	case models.WorkflowNodeExecutionStateRouting:
		return pb.WorkflowNodeExecution_STATE_ROUTING
	case models.WorkflowNodeExecutionStateFinished:
		return pb.WorkflowNodeExecution_STATE_FINISHED
	default:
		return pb.WorkflowNodeExecution_STATE_UNKNOWN
	}
}

func NodeExecutionResultToProto(result string) pb.WorkflowNodeExecution_Result {
	switch result {
	case models.WorkflowNodeExecutionResultPassed:
		return pb.WorkflowNodeExecution_RESULT_PASSED
	case models.WorkflowNodeExecutionResultFailed:
		return pb.WorkflowNodeExecution_RESULT_FAILED
	default:
		return pb.WorkflowNodeExecution_RESULT_UNKNOWN
	}
}

func NodeExecutionResultReasonToProto(reason string) pb.WorkflowNodeExecution_ResultReason {
	switch reason {
	case "ok":
		return pb.WorkflowNodeExecution_RESULT_REASON_OK
	case "error":
		return pb.WorkflowNodeExecution_RESULT_REASON_ERROR
	default:
		return pb.WorkflowNodeExecution_RESULT_REASON_OK
	}
}

func getLastExecutionTimestamp(executions []models.WorkflowNodeExecution) *timestamppb.Timestamp {
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
