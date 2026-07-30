package dispatch

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

func TestResolverForNoOpProvisioner(t *testing.T) {
	r := &Resolver{}
	for _, provisioner := range []string{"", "aws", "local"} {
		d, needsDispatch := r.For(provisioner, "")
		if needsDispatch {
			t.Fatalf("provisioner %q: expected no dispatch needed", provisioner)
		}
		if _, ok := d.(NoOp); !ok {
			t.Fatalf("provisioner %q: expected NoOp, got %T", provisioner, d)
		}
		if err := d.Dispatch(context.Background(), "fleet", "task"); err != nil {
			t.Fatalf("NoOp.Dispatch returned error: %v", err)
		}
	}
}

func TestResolverForLambdaProvisioner(t *testing.T) {
	r := &Resolver{LambdaClient: fakeInvoker(func(ctx context.Context, in *lambda.InvokeInput, _ ...func(*lambda.Options)) (*lambda.InvokeOutput, error) {
		return &lambda.InvokeOutput{}, nil
	})}
	d, needsDispatch := r.For("aws-lambda", "my-func")
	if !needsDispatch {
		t.Fatal("expected dispatch needed for aws-lambda provisioner")
	}
	lam, ok := d.(*Lambda)
	if !ok {
		t.Fatalf("expected *Lambda, got %T", d)
	}
	if lam.FunctionName != "my-func" {
		t.Fatalf("function name = %q", lam.FunctionName)
	}
}

type fakeInvoker func(ctx context.Context, in *lambda.InvokeInput, optFns ...func(*lambda.Options)) (*lambda.InvokeOutput, error)

func (f fakeInvoker) Invoke(ctx context.Context, in *lambda.InvokeInput, optFns ...func(*lambda.Options)) (*lambda.InvokeOutput, error) {
	return f(ctx, in, optFns...)
}

func TestLambdaDispatchInvokesAsyncWithEmptyPayload(t *testing.T) {
	var gotFunctionName string
	var gotInvocationType string
	client := fakeInvoker(func(_ context.Context, in *lambda.InvokeInput, _ ...func(*lambda.Options)) (*lambda.InvokeOutput, error) {
		gotFunctionName = *in.FunctionName
		gotInvocationType = string(in.InvocationType)
		return &lambda.InvokeOutput{}, nil
	})
	l := &Lambda{Client: client, FunctionName: "runner-lambda-x"}
	if err := l.Dispatch(context.Background(), "fleet-1", "task-1"); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if gotFunctionName != "runner-lambda-x" {
		t.Fatalf("function name = %q", gotFunctionName)
	}
	if gotInvocationType != "Event" {
		t.Fatalf("invocation type = %q, want async Event", gotInvocationType)
	}
}

func TestLambdaDispatchRequiresFunctionName(t *testing.T) {
	l := &Lambda{Client: fakeInvoker(func(context.Context, *lambda.InvokeInput, ...func(*lambda.Options)) (*lambda.InvokeOutput, error) {
		t.Fatal("should not invoke without a function name")
		return nil, nil
	})}
	if err := l.Dispatch(context.Background(), "fleet-1", "task-1"); err == nil {
		t.Fatal("expected error for missing function name")
	}
}

func TestLambdaDispatchRequiresClient(t *testing.T) {
	l := &Lambda{FunctionName: "f"}
	if err := l.Dispatch(context.Background(), "fleet-1", "task-1"); err == nil {
		t.Fatal("expected error for missing client")
	}
}
