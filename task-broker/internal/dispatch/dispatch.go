// Package dispatch starts execution for tasks whose fleet cannot pull work on
// its own (e.g. AWS Lambda). Fleets that poll for work themselves (e.g. EC2)
// get NoOp.
package dispatch

import (
	"context"
	"strings"
)

// ProvisionerAWSLambda is the fleet provisioner value that routes through the
// Lambda dispatcher.
const ProvisionerAWSLambda = "aws-lambda"

// Dispatcher starts one execution unit for a task. The invocation carries no
// task payload — it is a doorbell; the execution unit claims its own task
// from task-broker via the normal claim endpoint.
type Dispatcher interface {
	Dispatch(ctx context.Context, fleetID, taskID string) error
}

// NoOp is used by fleets whose capacity pulls work on its own (EC2 runners
// poll task-broker directly; nothing needs to be kicked off here).
type NoOp struct{}

func (NoOp) Dispatch(context.Context, string, string) error { return nil }

// Resolver builds the Dispatcher for a fleet based on its provisioner.
type Resolver struct {
	// LambdaClient invokes Lambda functions. Required only for fleets with
	// Provisioner == ProvisionerAWSLambda.
	LambdaClient LambdaInvoker
}

// For returns the Dispatcher for a fleet and whether the fleet needs an
// active dispatch call at all (false for NoOp fleets, so callers can skip
// the work of invoking and recording a dispatch attempt).
func (r *Resolver) For(provisioner, lambdaFunctionName string) (Dispatcher, bool) {
	switch strings.TrimSpace(provisioner) {
	case ProvisionerAWSLambda:
		var client LambdaInvoker
		if r != nil {
			client = r.LambdaClient
		}
		return &Lambda{Client: client, FunctionName: strings.TrimSpace(lambdaFunctionName)}, true
	default:
		return NoOp{}, false
	}
}
