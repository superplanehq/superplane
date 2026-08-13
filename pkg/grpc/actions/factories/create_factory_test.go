package factories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__CreateFactory(t *testing.T) {
	r := support.Setup(t)

	t.Run("empty name -> error", func(t *testing.T) {
		_, err := CreateFactory(context.Background(), r.Organization.ID.String(), &pb.CreateFactoryRequest{
			Name: "   ",
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("uses the key the caller sent", func(t *testing.T) {
		response, err := CreateFactory(context.Background(), r.Organization.ID.String(), &pb.CreateFactoryRequest{
			Name: support.RandomName("factory"),
			Key:  "chos",
		})
		require.NoError(t, err)
		assert.Equal(t, "CHOS", response.Factory.Key)
	})

	t.Run("duplicate key -> error", func(t *testing.T) {
		_, err := CreateFactory(context.Background(), r.Organization.ID.String(), &pb.CreateFactoryRequest{
			Name: support.RandomName("factory"),
			Key:  "DUPE",
		})
		require.NoError(t, err)

		_, err = CreateFactory(context.Background(), r.Organization.ID.String(), &pb.CreateFactoryRequest{
			Name: support.RandomName("factory"),
			Key:  "DUPE",
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.AlreadyExists, code)
	})

	// Callers that omit the key never picked one, so sharing leading
	// letters with an existing workspace must not fail the request.
	t.Run("omitted key falls back to a free name-derived key", func(t *testing.T) {
		first, err := CreateFactory(context.Background(), r.Organization.ID.String(), &pb.CreateFactoryRequest{
			Name: "Payments platform",
		})
		require.NoError(t, err)
		assert.Equal(t, "PAYME", first.Factory.Key)

		second, err := CreateFactory(context.Background(), r.Organization.ID.String(), &pb.CreateFactoryRequest{
			Name: "Payments platform reboot",
		})
		require.NoError(t, err)
		assert.NotEqual(t, first.Factory.Key, second.Factory.Key)
		assert.NotEmpty(t, second.Factory.Key)
	})
}
