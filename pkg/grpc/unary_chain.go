package grpc

import (
	"context"

	recovery "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/superplanehq/superplane/pkg/authorization"
	"google.golang.org/grpc"
)

type UnaryChain struct {
	interceptors []grpc.UnaryServerInterceptor
}

func NewUnaryChain(authService authorization.Authorization) *UnaryChain {
	recoveryOpts := []recovery.Option{
		recovery.WithRecoveryHandler(customFunc),
	}

	return &UnaryChain{
		interceptors: []grpc.UnaryServerInterceptor{
			recovery.UnaryServerInterceptor(recoveryOpts...),
			authorization.NewAuthorizationInterceptor(authService).UnaryInterceptor(),
			sanitizeErrorUnaryInterceptor(),
		},
	}
}

func (c *UnaryChain) Invoke(
	ctx context.Context,
	fullMethod string,
	req any,
	handler func(context.Context, any) (any, error),
) (any, error) {
	if c == nil || len(c.interceptors) == 0 {
		return handler(ctx, req)
	}

	call := handler
	for i := len(c.interceptors) - 1; i >= 0; i-- {
		interceptor := c.interceptors[i]
		next := call
		info := &grpc.UnaryServerInfo{FullMethod: fullMethod}
		call = func(ctx context.Context, req any) (any, error) {
			return interceptor(ctx, req, info, next)
		}
	}

	return call(ctx, req)
}
