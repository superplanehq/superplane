package actions

import (
	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pbSuperplane "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ValidateUUIDs(ids ...string) error {
	for _, id := range ids {
		_, err := uuid.Parse(id)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "invalid UUID: %s", id)
		}
	}

	return nil
}

func ExecutionResultToProto(result string) pbSuperplane.Execution_Result {
	switch result {
	case models.StageExecutionResultFailed:
		return pbSuperplane.Execution_RESULT_FAILED
	case models.StageExecutionResultPassed:
		return pbSuperplane.Execution_RESULT_PASSED
	default:
		return pbSuperplane.Execution_RESULT_UNKNOWN
	}
}
