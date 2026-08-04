package dispatch

import (
	"context"
	"strings"
)

const ProvisionerAWSLambda = "aws-lambda"

type Dispatcher interface {
	Dispatch(ctx context.Context, fleetID, taskID string) error
}

type NoOp struct{}

func (NoOp) Dispatch(context.Context, string, string) error { return nil }

type Resolver struct {
	LambdaClient LambdaInvoker
}

func (r *Resolver) For(provisioner, dispatchTarget string) (Dispatcher, bool) {
	switch strings.TrimSpace(provisioner) {
	case ProvisionerAWSLambda:
		var client LambdaInvoker
		if r != nil {
			client = r.LambdaClient
		}
		return &Lambda{Client: client, FunctionName: strings.TrimSpace(dispatchTarget)}, true
	default:
		return NoOp{}, false
	}
}

func (r *Resolver) Provisioners() []string {
	return []string{ProvisionerAWSLambda}
}
