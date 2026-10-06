package factories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/factories/vcs"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
)

func TestDescribeFactoryPullRequestMergeability_BitbucketFactory(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory := bitbucketFactory(t, r)
	order, err := factory.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
	require.NoError(t, err)
	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL:   "https://github.com/acme/app/pull/42",
		State: models.FactoryPullRequestStateOpen,
	})
	require.NoError(t, err)

	_, err = DescribeFactoryPullRequestMergeability(t.Context(), IntakeDependencies{}, r.Organization.ID.String(), &pb.DescribeFactoryPullRequestMergeabilityRequest{
		FactoryId: factory.ID.String(),
		PrId:      pullRequest.ID.String(),
	})
	require.ErrorIs(t, err, vcs.ErrNotSupported)
	_, message, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, "SuperPlane does not support this for Bitbucket.", message)
}

func TestMergeFactoryPullRequest_BitbucketFactory(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactoryPullRequestMerge))

	factory := bitbucketFactory(t, r)
	order, err := factory.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
	require.NoError(t, err)
	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL:   "https://github.com/acme/app/pull/42",
		State: models.FactoryPullRequestStateOpen,
	})
	require.NoError(t, err)
	const headSHA = "abc123"
	require.NoError(t, pullRequest.SetMergeability(db, models.FactoryPullRequestMergeabilitySnapshot{
		Mergeable:      true,
		HeadSHA:        headSHA,
		AllowedMethods: "SQUASH",
	}))

	_, err = MergeFactoryPullRequest(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.MergeFactoryPullRequestRequest{
		FactoryId:       factory.ID.String(),
		PrId:            pullRequest.ID.String(),
		MergeMethod:     pb.FactoryPullRequestMergeability_MERGE_METHOD_SQUASH,
		ExpectedHeadSha: headSHA,
	})
	require.ErrorIs(t, err, vcs.ErrNotSupported)
	_, message, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, "SuperPlane does not support this for Bitbucket.", message)

	stored, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: pullRequest.ID})
	require.NoError(t, err)
	assert.Equal(t, models.FactoryPullRequestStateOpen, stored.State)
}

func TestSendWorkOrderToBacklog_BitbucketFactoryCannotClose(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())
	factory := bitbucketFactory(t, r)
	order, err := factory.CreateWorkOrder(db, "Closed task", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateOpen,
		Actor:   &r.User,
	})
	require.NoError(t, err)
	_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateClosed,
		Result:  models.FactoryWorkOrderResultFailed,
		Actor:   &r.User,
	})
	require.NoError(t, err)
	_, err = order.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL:   "https://github.com/acme/app/pull/42",
		State: models.FactoryPullRequestStateOpen,
	})
	require.NoError(t, err)

	_, err = SendWorkOrderToBacklog(ctx, IntakeDependencies{}, r.Organization.ID.String(), &pb.SendWorkOrderToBacklogRequest{
		FactoryId:         factory.ID.String(),
		OrderId:           order.ID.String(),
		ClosePullRequests: true,
	})
	require.ErrorIs(t, err, vcs.ErrNotSupported)
	_, message, ok := grpcerrors.HandlerStatus(err)
	require.True(t, ok)
	assert.Equal(t, "SuperPlane cannot close a Bitbucket pull request.", message)

	reloaded, err := factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateClosed, reloaded.State)
}

func bitbucketFactory(t *testing.T, r *support.ResourceRegistry) *models.Factory {
	t.Helper()
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	provider := models.ProviderBitbucket
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSProvider: &provider,
	}))
	return factory
}
