package me

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__DescribeLastLocation(t *testing.T) {
	r := support.Setup(t)
	ctx := notificationSettingsContext(r.User.String(), r.Organization.ID.String())

	t.Run("no saved location returns empty response", func(t *testing.T) {
		resp, err := DescribeLastLocation(ctx)
		require.NoError(t, err)
		assert.Nil(t, resp.LastLocation)
	})

	t.Run("returns the saved location", func(t *testing.T) {
		path := "/" + r.Organization.Slug + "/apps/deploy?run=42&node=approve-1"
		_, err := SaveLastLocation(ctx, &pb.SaveLastLocationRequest{Path: path})
		require.NoError(t, err)

		resp, err := DescribeLastLocation(ctx)
		require.NoError(t, err)
		require.NotNil(t, resp.LastLocation)
		assert.Equal(t, path, resp.LastLocation.Path)
		assert.NotNil(t, resp.LastLocation.UpdatedAt)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		_, err := DescribeLastLocation(context.Background())
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, code)
	})
}

func Test__SaveLastLocation(t *testing.T) {
	r := support.Setup(t)
	ctx := notificationSettingsContext(r.User.String(), r.Organization.ID.String())

	t.Run("persists and overwrites the previous location", func(t *testing.T) {
		first := "/" + r.Organization.Slug + "/apps/deploy?run=1"
		second := "/" + r.Organization.Slug + "/apps/deploy?run=2"
		resp, err := SaveLastLocation(ctx, &pb.SaveLastLocationRequest{Path: first})
		require.NoError(t, err)
		assert.Equal(t, first, resp.LastLocation.Path)

		resp, err = SaveLastLocation(ctx, &pb.SaveLastLocationRequest{Path: second})
		require.NoError(t, err)
		assert.Equal(t, second, resp.LastLocation.Path)
	})

	t.Run("rejects a path that belongs to another organization", func(t *testing.T) {
		_, err := SaveLastLocation(ctx, &pb.SaveLastLocationRequest{Path: "/other-org/apps"})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("rejects a missing path", func(t *testing.T) {
		_, err := SaveLastLocation(ctx, &pb.SaveLastLocationRequest{})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("rejects a path that is not relative to the app origin", func(t *testing.T) {
		for _, path := range []string{"not-a-path", "//evil.com", "https://evil.com/phish"} {
			_, err := SaveLastLocation(ctx, &pb.SaveLastLocationRequest{Path: path})
			code, _, ok := grpcerrors.HandlerStatus(err)
			assert.True(t, ok, "path %q should be rejected", path)
			assert.Equal(t, codes.InvalidArgument, code, "path %q should be rejected", path)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		_, err := SaveLastLocation(context.Background(), &pb.SaveLastLocationRequest{Path: "/" + r.Organization.Slug})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, code)
	})
}
