package grpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	pb "github.com/superplanehq/superplane/pkg/protos/actions"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/test/support/impl"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type unavailableAction struct {
	core.Action
}

func (a unavailableAction) IsAvailable() bool {
	return false
}

func TestActionService_DescribeActionRejectsUnavailableActions(t *testing.T) {
	r := &registry.Registry{
		Actions: map[string]core.Action{
			"unavailable": registry.NewPanicableAction(unavailableAction{
				Action: impl.NewDummyAction(impl.DummyActionOptions{Name: "unavailable"}),
			}),
		},
	}

	service := NewActionService(r)

	_, err := service.DescribeAction(context.Background(), &pb.DescribeActionRequest{Name: "unavailable"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}
