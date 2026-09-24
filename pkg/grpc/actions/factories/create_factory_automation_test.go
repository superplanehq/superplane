package factories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__CreateFactoryAutomation(t *testing.T) {
	r := support.Setup(t)

	t.Run("feature off -> error", func(t *testing.T) {
		_, err := CreateFactoryAutomation(context.Background(), IntakeDependencies{}, r.Organization.ID.String(), &pb.CreateFactoryAutomationRequest{
			FactoryId: "00000000-0000-0000-0000-000000000001",
			Name:      "Preview environment",
			ColumnKey: models.CanvasColumnKeyVerify,
		})
		code, message, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
		assert.Equal(t, "Custom automations are not enabled for this organization.", message)
	})

	t.Run("feature on continues past the flag", func(t *testing.T) {
		require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryCustomAutomations))

		_, err := CreateFactoryAutomation(context.Background(), IntakeDependencies{}, r.Organization.ID.String(), &pb.CreateFactoryAutomationRequest{
			FactoryId: "00000000-0000-0000-0000-000000000001",
			Name:      "Preview environment",
			ColumnKey: "invalid",
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})
}
