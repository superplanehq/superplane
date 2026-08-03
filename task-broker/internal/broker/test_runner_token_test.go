package broker

import (
	"context"
	"testing"
	"time"

	"github.com/superplane/runner/shared/opaquetoken"
	brokermodels "github.com/superplane/runner/task-broker/internal/models"
	taskstore "github.com/superplane/runner/task-broker/internal/store"
)

func mintRunnerAccessToken(t *testing.T, ctx context.Context, st taskstore.Store, fleetID, runnerID string) string {
	t.Helper()
	now := time.Now().UTC()
	if err := st.CreateFleet(ctx, &brokermodels.Fleet{
		ID: fleetID, Provisioner: "test", Arch: "amd64", Size: "local", CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	access, err := opaquetoken.Generate()
	if err != nil {
		t.Fatal(err)
	}
	if err := st.RegisterRunnerWithJTI(
		ctx,
		"jti-"+runnerID,
		runnerID,
		fleetID,
		opaquetoken.Hash(access),
		now,
	); err != nil {
		t.Fatal(err)
	}
	return access
}
