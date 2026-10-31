package runners_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	billingorganizations "github.com/superplanehq/superplane/ee/billing/organizations"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/enterprise"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/grpc"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/public"
	"github.com/superplanehq/superplane/pkg/public/runnerapi"
	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
	runnerlogsfs "github.com/superplanehq/superplane/pkg/runners/logs/fs"
	"github.com/superplanehq/superplane/pkg/workers"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"go.opentelemetry.io/otel"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestDirectOrganizationRunnerLifecycle(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "runner")
	build := exec.CommandContext(t.Context(), "go", "build", "-buildvcs=false", "-o", binary, "../../../cmd/runner")
	output, err := build.CombinedOutput()
	require.NoError(t, err, "%s", output)
	for _, ephemeral := range []bool{true, false} {
		name := "reusable"
		if ephemeral {
			name = "ephemeral"
		}
		t.Run(name, func(t *testing.T) {
			s := newLifecycleSteps(t)
			s.createFleet()
			first := s.createTask("echo first-log; sleep 1; echo completed-log")
			s.waitTask(first.ID, models.RunnerTaskStateQueued)
			created := s.createRunner(ephemeral, &first.ID)
			unused := s.createRunner(true, nil)
			processDone := s.startRunner(binary, created)
			s.waitTask(first.ID, models.RunnerTaskStateRunning)
			queued := s.createTask("echo successive-log")
			require.NoError(t, models.DisableExperimentalFeature(s.resource.Organization.ID, features.FeatureOrganizationRunners))
			s.assertUnusedRegistrationRejected(unused)
			s.waitTask(first.ID, models.RunnerTaskStateSucceeded)
			s.assertLogs(first.ID, "completed-log")
			require.Eventually(t, func() bool {
				task, err := models.FindRunnerTask(database.DB(t.Context()), queued.ID)
				return err == nil && task.State == models.RunnerTaskStatePending && task.FleetID == first.FleetID
			}, 5*time.Second, 20*time.Millisecond)
			if ephemeral {
				s.assertProcessExited(processDone)
				s.request(http.MethodDelete, s.path+"/runners/"+unused.Runner.Id, nil, http.StatusOK)
				s.compactLogs(first.ID, "completed-log")
				return
			}
			require.NoError(t, models.EnableExperimentalFeature(s.resource.Organization.ID, features.FeatureOrganizationRunners))
			s.waitTask(queued.ID, models.RunnerTaskStateSucceeded)
			s.assertLogs(queued.ID, "successive-log")
			require.Eventually(t, func() bool {
				runner, err := models.FindRunner(database.DB(t.Context()), uuid.MustParse(created.Runner.Id))
				return err == nil && runner.State == models.RunnerStateIdle
			}, 5*time.Second, 20*time.Millisecond)
			cancelled := s.createTask("echo cancellation-log; exec sleep 20")
			s.waitTask(cancelled.ID, models.RunnerTaskStateRunning)
			require.NoError(t, models.DisableExperimentalFeature(s.resource.Organization.ID, features.FeatureOrganizationRunners))
			// Use the same cancellation path as a workflow execution.
			require.NoError(t, contexts.NewRunnerTaskContext(database.DB(t.Context()), s.resource.Encryptor, s.resource.Organization.ID).RequestCancel(cancelled.ID.String()))
			s.waitTask(cancelled.ID, models.RunnerTaskStateCanceled)
			s.assertLogs(cancelled.ID, "cancellation-log")
			s.request(http.MethodDelete, s.path+"/runners/"+created.Runner.Id, nil, http.StatusOK)
			s.assertProcessExited(processDone)
			s.compactLogs(cancelled.ID, "cancellation-log")
		})
	}
}

type lifecycleSteps struct {
	t        *testing.T
	resource *support.ResourceRegistry
	api      *httptest.Server
	gateway  *httptest.Server
	store    runnerlogs.Store
	path     string
	token    string
}

func newLifecycleSteps(t *testing.T) *lifecycleSteps {
	t.Helper()
	r := support.Setup(t)
	signer := jwt.NewSigner("organization-runner-e2e")
	store, err := runnerlogsfs.New(t.TempDir())
	require.NoError(t, err)
	require.NoError(t, store.Setup(runnerlogs.SetupContext{Context: t.Context(), MeterProvider: otel.GetMeterProvider()}))
	previous := runnerlogs.Current()
	runnerlogs.SetCurrent(store)
	t.Cleanup(func() { runnerlogs.SetCurrent(previous) })
	gateway, err := runnerapi.NewServer(signer, r.Encryptor, runnerlogs.StoreFS, nil)
	require.NoError(t, err)
	runnerHTTP := httptest.NewServer(gateway.Handler())
	t.Cleanup(runnerHTTP.Close)
	api, err := public.NewServer(r.Encryptor, r.Registry, signer, support.NewOIDCProvider(), "", "", "", "test", "/app/templates", r.AuthService, false)
	require.NoError(t, err)
	enterpriseRegistry := enterprise.NewRegistry()
	enterpriseRegistry.Register(enterprise.Billing, billingorganizations.New())
	services, err := grpc.NewServices(grpc.ServicesConfig{Enterprise: enterpriseRegistry, AuthService: r.AuthService, Encryptor: r.Encryptor,
		Registry: r.Registry, OIDCProvider: support.NewOIDCProvider(), JWTSigner: signer, RunnerAPIBaseURL: runnerHTTP.URL})
	require.NoError(t, err)
	require.NoError(t, api.RegisterGRPCGateway(services))
	apiHTTP := httptest.NewServer(api.Router)
	t.Cleanup(apiHTTP.Close)
	rawToken, err := crypto.Base64String(32)
	require.NoError(t, err)
	require.NoError(t, models.CreateUserAPIToken(database.DB(t.Context()), models.NewUserAPIToken(r.User, "direct-runner-test", crypto.HashToken(rawToken))))
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureOrganizationRunners))
	workerCtx, cancel := context.WithCancel(t.Context())
	gateway.StartControlNotifications(workerCtx, uuid.NewString())
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		workers.NewRunnerTaskScheduler(20 * time.Millisecond).Start(workerCtx)
	}()
	t.Cleanup(func() { cancel(); <-workerDone })
	return &lifecycleSteps{t: t, resource: r, api: apiHTTP, gateway: runnerHTTP, store: store,
		path: "/api/v1/organizations/" + r.Organization.ID.String() + "/runner-fleets/direct", token: rawToken}
}

func (s *lifecycleSteps) request(method, path string, body []byte, expected int) []byte {
	s.t.Helper()
	req, err := http.NewRequestWithContext(s.t.Context(), method, s.api.URL+path, bytes.NewReader(body))
	require.NoError(s.t, err)
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("x-organization-id", s.resource.Organization.ID.String())
	req.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(req)
	require.NoError(s.t, err)
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	require.NoError(s.t, err)
	require.Equal(s.t, expected, response.StatusCode, "%s", data)
	return data
}

func (s *lifecycleSteps) createFleet() {
	s.request(http.MethodPost, "/api/v1/organizations/"+s.resource.Organization.ID.String()+"/runner-fleets",
		[]byte(`{"fleetId":"direct","runnerVersion":"dev","spec":{"operatingSystem":"linux","architecture":"amd64"}}`), http.StatusOK)
}

func (s *lifecycleSteps) createTask(script string) *models.RunnerTask {
	s.t.Helper()
	id := uuid.New()
	payload, err := json.Marshal(map[string]any{"id": id.String(), "command": []string{"sh", "-c", script}, "execution_mode": "host"})
	require.NoError(s.t, err)
	require.NoError(s.t, contexts.NewRunnerTaskContext(database.DB(s.t.Context()), s.resource.Encryptor,
		s.resource.Organization.ID).Create(id.String(), "direct", payload))
	task, err := models.FindRunnerTask(database.DB(s.t.Context()), id)
	require.NoError(s.t, err)
	return task
}

func (s *lifecycleSteps) createRunner(ephemeral bool, taskID *uuid.UUID) *pb.CreateFleetRunnerResponse {
	s.t.Helper()
	request := &pb.CreateFleetRunnerRequest{Ephemeral: ephemeral, IdempotencyKey: uuid.NewString()}
	if taskID != nil {
		request.TaskId = new(taskID.String())
	}
	body, err := protojson.Marshal(request)
	require.NoError(s.t, err)
	response := &pb.CreateFleetRunnerResponse{}
	require.NoError(s.t, protojson.Unmarshal(s.request(http.MethodPost, s.path+"/runners", body, http.StatusOK), response))
	return response
}

func (s *lifecycleSteps) startRunner(binary string, created *pb.CreateFleetRunnerResponse) <-chan error {
	s.t.Helper()
	runCtx, cancel := context.WithCancel(s.t.Context())
	command := exec.CommandContext(runCtx, binary)
	command.Env = append(os.Environ(), "RUNNER_API_URL="+created.RunnerApiUrl,
		"RUNNER_REGISTRATION_TOKEN="+created.RegistrationToken, "RUNNER_TASK_WORK_DIRECTORY="+s.t.TempDir(),
		"RUNNER_LOG_SPOOL_DIRECTORY="+s.t.TempDir())
	var output bytes.Buffer
	command.Stdout, command.Stderr = &output, &output
	require.NoError(s.t, command.Start())
	done := make(chan error, 1)
	go func() { done <- command.Wait(); close(done) }()
	s.t.Cleanup(func() {
		cancel()
		<-done
		if s.t.Failed() {
			s.t.Logf("runner output: %s", output.String())
		}
	})
	return done
}

func (s *lifecycleSteps) waitTask(id uuid.UUID, state string) {
	s.t.Helper()
	require.Eventually(s.t, func() bool {
		task, err := models.FindRunnerTask(database.DB(s.t.Context()), id)
		return err == nil && task.State == state
	}, 10*time.Second, 20*time.Millisecond, "task %s must reach %s", id, state)
}

func (s *lifecycleSteps) assertProcessExited(done <-chan error) {
	s.t.Helper()
	select {
	case err := <-done:
		require.NoError(s.t, err)
	case <-time.After(10 * time.Second):
		s.t.Fatal("runner did not exit after acknowledgement or termination")
	}
}

func (s *lifecycleSteps) assertUnusedRegistrationRejected(created *pb.CreateFleetRunnerResponse) {
	s.assertRegistrationRejected(created, http.StatusForbidden)
}

func (s *lifecycleSteps) assertRegistrationRejected(created *pb.CreateFleetRunnerResponse, expected int) {
	s.t.Helper()
	req, err := http.NewRequestWithContext(s.t.Context(), http.MethodPost, s.gateway.URL+"/runner/v1/register", bytes.NewBufferString(`{"version":"dev"}`))
	require.NoError(s.t, err)
	req.Header.Set("Authorization", "Bearer "+created.RegistrationToken)
	response, err := http.DefaultClient.Do(req)
	require.NoError(s.t, err)
	defer response.Body.Close()
	require.Equal(s.t, expected, response.StatusCode)
	grant, err := models.FindRunnerRegistration(database.DB(s.t.Context()), uuid.MustParse(created.Runner.Id))
	require.NoError(s.t, err)
	require.Nil(s.t, grant.ConsumedAt)
}

func (s *lifecycleSteps) assertLogs(id uuid.UUID, text string) {
	s.t.Helper()
	logs, err := s.store.ReadAfter(s.t.Context(), id, "")
	require.NoError(s.t, err)
	defer logs.Content.Close()
	content, err := io.ReadAll(logs.Content)
	require.NoError(s.t, err)
	require.Contains(s.t, string(content), text)
}

func (s *lifecycleSteps) compactLogs(id uuid.UUID, text string) {
	s.t.Helper()
	provider, err := filesystem.New(s.t.TempDir())
	require.NoError(s.t, err)
	installationID, err := models.GetInstallationID(database.DB(s.t.Context()))
	require.NoError(s.t, err)
	compactor := workers.NewRunnerTaskLogCompactor(provider, s.store, installationID, time.Second, 0)
	require.NoError(s.t, compactor.Process(s.t.Context()))
	lifecycle, err := (&models.RunnerTask{ID: id}).FindLifecycle(database.DB(s.t.Context()))
	require.NoError(s.t, err)
	require.Equal(s.t, models.RunnerTaskLogStateArchived, lifecycle.State)
	require.Nil(s.t, lifecycle.CleanupAfter)
	_, err = s.store.ReadAfter(s.t.Context(), id, "")
	require.ErrorIs(s.t, err, runnerlogs.ErrNotFound)
	final, err := provider.Get(s.t.Context(), *lifecycle.FinalObjectKey)
	require.NoError(s.t, err)
	defer final.Close()
	uncompressed, err := gzip.NewReader(final)
	require.NoError(s.t, err)
	defer uncompressed.Close()
	content, err := io.ReadAll(uncompressed)
	require.NoError(s.t, err)
	require.Contains(s.t, string(content), text)
}
