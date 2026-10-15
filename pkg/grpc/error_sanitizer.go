package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/grpc/errors"
)

func SanitizeError(requestCtx context.Context, err error) error {
	return grpcerrors.SanitizeError(requestCtx, err)
}
