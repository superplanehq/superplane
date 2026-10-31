package runners_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/datatypes"
)

func TestOrganizationRunnerAPIAuthorizationAndCleanup(t *testing.T) {
	s := newLifecycleSteps(t)
	s.createFleet()
	created := s.createRunner(false, nil)
	ownerToken := s.token
	db := database.DB(t.Context())
	account, err := models.CreateAccount("Fleet reader", "fleet-reader@example.com")
	require.NoError(t, err)
	reader, err := models.CreateUser(s.resource.Organization.ID, account.ID, account.Email, account.Name)
	require.NoError(t, err)
	require.NoError(t, s.resource.AuthService.AssignRole(reader.ID.String(), models.RoleOrgOperator,
		s.resource.Organization.ID.String(), models.DomainTypeOrganization))
	s.token = apiToken(t, reader.ID)
	base := "/api/v1/organizations/" + s.resource.Organization.ID.String() + "/runner-fleets"
	for _, path := range []string{base, s.path, s.path + "/capacity", s.path + "/runners", s.path + "/runners/" + created.Runner.Id} {
		s.request(http.MethodGet, path, nil, http.StatusOK)
	}
	for _, request := range []struct{ method, path, body string }{
		{http.MethodPost, base, `{"fleetId":"denied","runnerVersion":"dev","spec":{}}`},
		{http.MethodPatch, s.path, `{"enabled":false}`},
		{http.MethodPost, s.path + "/runners", `{"ephemeral":true}`},
		{http.MethodDelete, s.path + "/runners/" + created.Runner.Id, ""},
		{http.MethodPost, s.path + "/runners/" + created.Runner.Id + ":provider-deleted", `{"fleetManagerId":"manager","provider":"test"}`},
	} {
		s.request(request.method, request.path, []byte(request.body), http.StatusNotFound)
	}
	s.token = ownerToken
	directCleanupPath := s.path + "/runners/" + created.Runner.Id + ":provider-deleted"
	s.request(http.MethodPost, directCleanupPath, []byte(`{}`), http.StatusBadRequest)
	directRunner, err := models.FindRunner(db, uuid.MustParse(created.Runner.Id))
	require.NoError(t, err)
	require.Nil(t, directRunner.ProviderDeletedAt)
	foreign, err := models.CreateOrganization("Foreign fleet organization", "")
	require.NoError(t, err)
	foreignFleet := &models.RunnerFleet{ID: uuid.New(), Slug: "foreign", ScopeType: models.RunnerFleetScopeOrganization,
		ScopeID: &foreign.ID, Enabled: true, RunnerVersion: "dev", Spec: datatypes.NewJSONType(models.RunnerFleetSpec{})}
	require.NoError(t, foreignFleet.Create(db))
	foreignRunner := &models.Runner{ID: uuid.New(), FleetID: foreignFleet.ID, State: models.RunnerStatePending, RunnerVersion: "dev"}
	require.NoError(t, db.Create(foreignRunner).Error)
	for _, path := range []string{base + "/foreign", base + "/foreign/runners", s.path + "/runners/" + foreignRunner.ID.String(),
		"/api/v1/organizations/" + foreign.ID.String() + "/runner-fleets/direct"} {
		s.request(http.MethodGet, path, nil, http.StatusNotFound)
	}
	page := &pb.ListFleetRunnersResponse{}
	require.NoError(t, protojson.Unmarshal(s.request(http.MethodGet, s.path+"/runners?after_id="+foreignRunner.ID.String(), nil, http.StatusOK), page))
	require.Empty(t, page.Runners)
	owned, err := models.FindOrganizationRunnerFleet(db, s.resource.Organization.ID, "direct")
	require.NoError(t, err)
	// A matching fleet key cannot make a foreign organization's task eligible.
	task := &models.RunnerTask{ID: uuid.New(), OrganizationID: foreign.ID, FleetID: owned.ID,
		Backend: models.RunnerTaskBackendIntegrated, State: models.RunnerTaskStateQueued,
		PayloadCiphertext: []byte(`{}`), QueuedAt: time.Now()}
	require.NoError(t, db.Create(task).Error)
	s.request(http.MethodPost, s.path+"/runners", []byte(`{"taskId":"`+task.ID.String()+`","ephemeral":true,"idempotencyKey":"foreign-task"}`), http.StatusBadRequest)
	validTask := s.createTask("echo registration-ownership")
	s.waitTask(validTask.ID, models.RunnerTaskStateQueued)
	boundRunner := s.createRunner(true, &validTask.ID)
	require.NoError(t, db.Model(validTask).Update("organization_id", foreign.ID).Error)
	s.assertRegistrationRejected(boundRunner, http.StatusUnauthorized)
	// Provider cleanup remains authorized after feature disablement.
	ownedRunner := &pb.CreateFleetRunnerResponse{}
	require.NoError(t, protojson.Unmarshal(s.request(http.MethodPost, s.path+"/runners",
		[]byte(`{"ephemeral":true,"fleetManagerId":"manager","provider":"test"}`), http.StatusOK), ownedRunner))
	require.NoError(t, models.DisableExperimentalFeature(s.resource.Organization.ID, features.FeatureOrganizationRunners))
	require.NoError(t, owned.SetEnabled(db, false))
	list := &pb.ListManagedRunnerFleetsResponse{}
	require.NoError(t, protojson.Unmarshal(s.request(http.MethodGet, base, nil, http.StatusOK), list))
	require.Len(t, list.Fleets, 1)
	require.False(t, list.Fleets[0].Enabled)
	for _, request := range []struct{ method, path, body string }{
		{http.MethodPost, base, `{"fleetId":"disabled","runnerVersion":"dev","spec":{}}`},
		{http.MethodPatch, s.path, `{"enabled":true}`},
		{http.MethodPost, s.path + "/runners", `{"ephemeral":true}`},
	} {
		body := s.request(request.method, request.path, []byte(request.body), http.StatusBadRequest)
		require.Contains(t, string(body), "organization_runners")
	}
	s.request(http.MethodDelete, s.path+"/runners/"+ownedRunner.Runner.Id, nil, http.StatusOK)
	for _, body := range []string{`{}`, `{"fleetManagerId":"manager"}`, `{"provider":"test"}`,
		`{"fleetManagerId":"wrong","provider":"test"}`, `{"fleetManagerId":"manager","provider":"wrong"}`} {
		s.request(http.MethodPost, s.path+"/runners/"+ownedRunner.Runner.Id+":provider-deleted", []byte(body), http.StatusBadRequest)
	}
	s.request(http.MethodPost, s.path+"/runners/"+ownedRunner.Runner.Id+":provider-deleted",
		[]byte(`{"fleetManagerId":"manager","provider":"test"}`), http.StatusOK)
	runner, err := models.FindRunner(db, uuid.MustParse(ownedRunner.Runner.Id))
	require.NoError(t, err)
	require.NotNil(t, runner.ProviderDeletedAt)

	// An administrator can confirm direct runner cleanup without provider tags.
	s.request(http.MethodDelete, s.path+"/runners/"+created.Runner.Id, nil, http.StatusOK)
	for _, body := range []string{`{"fleetManagerId":"manager","provider":"test"}`,
		`{"fleetManagerId":"manager"}`, `{"provider":"test"}`} {
		s.request(http.MethodPost, directCleanupPath, []byte(body), http.StatusBadRequest)
	}
	directRunner, err = models.FindRunner(db, uuid.MustParse(created.Runner.Id))
	require.NoError(t, err)
	require.Nil(t, directRunner.ProviderDeletedAt)
	confirmed := &pb.MarkFleetRunnerProviderDeletedResponse{}
	require.NoError(t, protojson.Unmarshal(s.request(http.MethodPost, directCleanupPath, []byte(`{}`), http.StatusOK), confirmed))
	require.NotNil(t, confirmed.Runner.ProviderDeletedAt)
	directRunner, err = models.FindRunner(db, directRunner.ID)
	require.NoError(t, err)
	require.NotNil(t, directRunner.ProviderDeletedAt)
	require.WithinDuration(t, confirmed.Runner.ProviderDeletedAt.AsTime(), *directRunner.ProviderDeletedAt, time.Microsecond)
	// Repeated confirmation preserves the first deletion timestamp.
	repeated := &pb.MarkFleetRunnerProviderDeletedResponse{}
	require.NoError(t, protojson.Unmarshal(s.request(http.MethodPost, directCleanupPath, []byte(`{}`), http.StatusOK), repeated))
	require.Equal(t, directRunner.ProviderDeletedAt.UTC(), repeated.Runner.ProviderDeletedAt.AsTime())
}

func apiToken(t *testing.T, userID uuid.UUID) string {
	t.Helper()
	token, err := crypto.Base64String(32)
	require.NoError(t, err)
	require.NoError(t, models.CreateUserAPIToken(database.DB(t.Context()), models.NewUserAPIToken(userID, "fleet-test", crypto.HashToken(token))))
	return token
}
