package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/task-broker/internal/dispatch"
	brokermodels "github.com/superplane/runner/task-broker/internal/models"
	"github.com/superplane/runner/task-broker/internal/store/testdb"
)

func TestCreateTaskRejectsDockerOnFleetWithoutDockerSupport(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	noDocker := false
	if err := st.CreateFleet(context.Background(), &brokermodels.Fleet{
		ID: "fleet-no-docker", Provisioner: "aws-lambda", Arch: "amd64", Size: "small",
		CreatedAt: time.Now().UTC(), LambdaFunctionName: "fn", SupportsDocker: &noDocker,
	}); err != nil {
		t.Fatal(err)
	}

	srv := &Server{Store: st}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "token"}))
	defer ts.Close()

	status, body := postTask(t, ts, `{
		"fleet_id": "fleet-no-docker",
		"webhook_url": "https://example.com/hook",
		"execution_mode": "docker",
		"docker_image": "alpine",
		"commands": [{"command": "echo hi"}]
	}`)
	if status != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", status, body)
	}
	if !strings.Contains(body, "does not support docker") {
		t.Fatalf("body=%s", body)
	}
}

func TestCreateTaskRejectsExecutionTimeoutAboveFleetCap(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	capSeconds := 900
	if err := st.CreateFleet(context.Background(), &brokermodels.Fleet{
		ID: "fleet-capped", Provisioner: "aws-lambda", Arch: "amd64", Size: "small",
		CreatedAt: time.Now().UTC(), LambdaFunctionName: "fn", MaxExecutionTimeoutSeconds: &capSeconds,
	}); err != nil {
		t.Fatal(err)
	}

	srv := &Server{Store: st}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "token"}))
	defer ts.Close()

	status, body := postTask(t, ts, `{
		"fleet_id": "fleet-capped",
		"webhook_url": "https://example.com/hook",
		"execution_timeout_seconds": 1000,
		"commands": [{"command": "echo hi"}]
	}`)
	if status != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", status, body)
	}
	if !strings.Contains(body, "exceeds fleet") {
		t.Fatalf("body=%s", body)
	}
}

func TestCreateTaskClampsOmittedTimeoutToFleetCap(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	capSeconds := 900
	if err := st.CreateFleet(context.Background(), &brokermodels.Fleet{
		ID: "fleet-capped-2", Provisioner: "aws-lambda", Arch: "amd64", Size: "small",
		CreatedAt: time.Now().UTC(), LambdaFunctionName: "fn", MaxExecutionTimeoutSeconds: &capSeconds,
	}); err != nil {
		t.Fatal(err)
	}

	srv := &Server{Store: st}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "token"}))
	defer ts.Close()

	status, body := postTask(t, ts, `{
		"fleet_id": "fleet-capped-2",
		"webhook_url": "https://example.com/hook",
		"commands": [{"command": "echo hi"}]
	}`)
	if status != http.StatusCreated {
		t.Fatalf("status=%d body=%s", status, body)
	}
	var created api.BrokerCreateTaskResponse
	if err := json.Unmarshal([]byte(body), &created); err != nil {
		t.Fatal(err)
	}
	task, err := st.GetTask(context.Background(), created.ID)
	if err != nil || task == nil {
		t.Fatalf("get task: %v %#v", err, task)
	}
	if task.ExecutionTimeoutSeconds == nil || *task.ExecutionTimeoutSeconds != capSeconds {
		t.Fatalf("expected timeout clamped to %d, got %#v", capSeconds, task.ExecutionTimeoutSeconds)
	}
}

func TestCreateTaskDispatchesToLambdaFleet(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	if err := st.CreateFleet(context.Background(), &brokermodels.Fleet{
		ID: "fleet-lambda-dispatch", Provisioner: "aws-lambda", Arch: "amd64", Size: "small",
		CreatedAt: time.Now().UTC(), LambdaFunctionName: "runner-lambda-fn",
	}); err != nil {
		t.Fatal(err)
	}

	var invokeCount int
	var gotFunctionName string
	fake := fakeLambdaInvoker(func(_ context.Context, in *lambda.InvokeInput, _ ...func(*lambda.Options)) (*lambda.InvokeOutput, error) {
		invokeCount++
		gotFunctionName = *in.FunctionName
		return &lambda.InvokeOutput{}, nil
	})

	srv := &Server{Store: st, Dispatch: &dispatch.Resolver{LambdaClient: fake}}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "token"}))
	defer ts.Close()

	status, body := postTask(t, ts, `{
		"fleet_id": "fleet-lambda-dispatch",
		"webhook_url": "https://example.com/hook",
		"commands": [{"command": "echo hi"}]
	}`)
	if status != http.StatusCreated {
		t.Fatalf("status=%d body=%s", status, body)
	}
	if invokeCount != 1 {
		t.Fatalf("expected exactly one lambda invoke, got %d", invokeCount)
	}
	if gotFunctionName != "runner-lambda-fn" {
		t.Fatalf("function name = %q", gotFunctionName)
	}

	candidates, err := st.ClaimDispatchCandidates(context.Background(), dispatch.ProvisionerAWSLambda, time.Minute, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 0 {
		t.Fatalf("expected no sweep candidates after successful dispatch: %#v", candidates)
	}
}

type fakeLambdaInvoker func(ctx context.Context, in *lambda.InvokeInput, optFns ...func(*lambda.Options)) (*lambda.InvokeOutput, error)

func (f fakeLambdaInvoker) Invoke(ctx context.Context, in *lambda.InvokeInput, optFns ...func(*lambda.Options)) (*lambda.InvokeOutput, error) {
	return f(ctx, in, optFns...)
}

func postTask(t *testing.T, ts *httptest.Server, body string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/tasks", bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, strings.TrimSpace(string(respBody))
}
