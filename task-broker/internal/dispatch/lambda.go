package dispatch

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

// LambdaInvoker is the subset of *lambda.Client used for dispatch, so tests
// can substitute a fake.
type LambdaInvoker interface {
	Invoke(ctx context.Context, in *lambda.InvokeInput, optFns ...func(*lambda.Options)) (*lambda.InvokeOutput, error)
}

// Lambda dispatches a task by asynchronously invoking a Lambda function.
// The invoke payload is empty: the invocation is a doorbell that claims its
// own task from task-broker after it starts running.
type Lambda struct {
	Client       LambdaInvoker
	FunctionName string
}

func (l *Lambda) Dispatch(ctx context.Context, fleetID, taskID string) error {
	if l == nil || l.Client == nil {
		return fmt.Errorf("lambda dispatcher: no client configured for fleet %s", fleetID)
	}
	if l.FunctionName == "" {
		return fmt.Errorf("lambda dispatcher: no function name configured for fleet %s", fleetID)
	}
	_, err := l.Client.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   aws.String(l.FunctionName),
		InvocationType: types.InvocationTypeEvent,
		Payload:        []byte("{}"),
	})
	if err != nil {
		return fmt.Errorf("lambda dispatcher: invoke %s for task %s: %w", l.FunctionName, taskID, err)
	}
	return nil
}
