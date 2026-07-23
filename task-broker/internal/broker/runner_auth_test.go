package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/models"
	"github.com/superplane/runner/shared/runnerregistrationtoken"
	brokermodels "github.com/superplane/runner/task-broker/internal/models"
	"github.com/superplane/runner/task-broker/internal/store/testdb"
)

func TestRunnerRegistrationJWTClaimAndComplete(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	ctx := context.Background()
	if err := st.CreateFleet(ctx, &brokermodels.Fleet{
		ID: "fleet-a", Provisioner: "aws", CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateTask(ctx, &models.Task{
		ID: "task-a", FleetID: "fleet-a", Status: models.StatusQueued,
		CreatedAt: time.Now().UTC(), Command: []string{"true"},
	}); err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(NewRouter(&Server{Store: st}, RouterOptions{AuthToken: "control"}))
	defer ts.Close()

	registration := createTestRegistration(t, "control", "fleet-a")
	accessToken := registerTestRunner(t, ts.URL, registration, "i-runner", "fleet-a")

	// Same registration JWT must not work twice.
	reuseBody, _ := json.Marshal(api.RegisterRunnerRequest{RunnerID: "i-other", FleetID: "fleet-a"})
	reuseReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/runners/register", bytes.NewReader(reuseBody))
	reuseReq.Header.Set("Authorization", "Bearer "+registration)
	reuseReq.Header.Set("Content-Type", "application/json")
	reuseResp, err := http.DefaultClient.Do(reuseReq)
	if err != nil {
		t.Fatal(err)
	}
	_ = reuseResp.Body.Close()
	if reuseResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("reuse registration status = %d, want 401", reuseResp.StatusCode)
	}

	claimBody, _ := json.Marshal(api.ClaimTaskRequest{
		RunnerID: "i-runner", FleetID: "fleet-a", LeaseSeconds: 60,
	})
	claimReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/tasks/claim", bytes.NewReader(claimBody))
	claimReq.Header.Set("Authorization", "Bearer "+accessToken)
	claimReq.Header.Set("Content-Type", "application/json")
	claimResp, err := http.DefaultClient.Do(claimReq)
	if err != nil {
		t.Fatal(err)
	}
	defer claimResp.Body.Close()
	if claimResp.StatusCode != http.StatusOK {
		t.Fatalf("claim status = %d", claimResp.StatusCode)
	}
	var claim api.ClaimTaskResponse
	if err := json.NewDecoder(claimResp.Body).Decode(&claim); err != nil {
		t.Fatal(err)
	}
	if claim.Task == nil || claim.Task.ID != "task-a" {
		t.Fatalf("claim = %#v", claim)
	}

	completeBody, _ := json.Marshal(api.CompleteTaskRequest{RunnerID: "i-runner", ExitCode: 0})
	completeReq, _ := http.NewRequest(
		http.MethodPost, ts.URL+"/v1/tasks/task-a/complete", bytes.NewReader(completeBody),
	)
	completeReq.Header.Set("Authorization", "Bearer "+accessToken)
	completeReq.Header.Set("Content-Type", "application/json")
	completeResp, err := http.DefaultClient.Do(completeReq)
	if err != nil {
		t.Fatal(err)
	}
	defer completeResp.Body.Close()
	if completeResp.StatusCode != http.StatusNoContent {
		t.Fatalf("complete status = %d", completeResp.StatusCode)
	}

	controlClaim, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/tasks/claim", bytes.NewReader(claimBody))
	controlClaim.Header.Set("Authorization", "Bearer control")
	resp, err := http.DefaultClient.Do(controlClaim)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("control token claim status = %d, want 401", resp.StatusCode)
	}

	revokeReq, _ := http.NewRequest(http.MethodDelete, ts.URL+"/v1/runners/self", nil)
	revokeReq.Header.Set("Authorization", "Bearer "+accessToken)
	revokeResp, err := http.DefaultClient.Do(revokeReq)
	if err != nil {
		t.Fatal(err)
	}
	_ = revokeResp.Body.Close()
	if revokeResp.StatusCode != http.StatusNoContent {
		t.Fatalf("revoke status = %d", revokeResp.StatusCode)
	}
	revokedClaim, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/tasks/claim", bytes.NewReader(claimBody))
	revokedClaim.Header.Set("Authorization", "Bearer "+accessToken)
	revokedResp, err := http.DefaultClient.Do(revokedClaim)
	if err != nil {
		t.Fatal(err)
	}
	defer revokedResp.Body.Close()
	if revokedResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("revoked token claim status = %d, want 401", revokedResp.StatusCode)
	}
}

func createTestRegistration(t *testing.T, secret, fleetID string) string {
	t.Helper()
	token, err := runnerregistrationtoken.Mint(fleetID, secret, time.Now().UTC().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func registerTestRunner(t *testing.T, baseURL, registrationToken, runnerID, fleetID string) string {
	t.Helper()
	body, _ := json.Marshal(api.RegisterRunnerRequest{RunnerID: runnerID, FleetID: fleetID})
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/v1/runners/register", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+registrationToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register runner status = %d", resp.StatusCode)
	}
	var out api.RegisterRunnerResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out.AccessToken
}
