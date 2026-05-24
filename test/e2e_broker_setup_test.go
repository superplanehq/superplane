package e2e_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/superplane/runner/shared/api"
)

const e2eBrokerAuthToken = E2EAuthToken

type brokerStack struct {
	BrokerURL string
	FleetID   string
	AuthToken string
}

func startBrokerStack(t *testing.T, root string, binDir string) brokerStack {
	t.Helper()

	fleetID := "e2e-" + strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())

	brokerDSN := os.Getenv("TEST_DATABASE_URL")
	if brokerDSN == "" {
		t.Skip("TEST_DATABASE_URL unset — required for task-broker e2e (see README)")
	}

	brokerBin := filepath.Join(binDir, "task-broker")
	cmd := exec.Command("go", "build", "-o", brokerBin, "./task-broker/cmd/task-broker")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build task-broker: %v\n%s", err, out)
	}

	brokerAddr := freeTCPAddr(t)
	brokerURL := "http://" + brokerAddr

	brokerCmd := exec.Command(brokerBin)
	brokerCmd.Env = subprocessEnv(
		"DATABASE_URL="+brokerDSN,
		"LISTEN_ADDR="+brokerAddr,
		"AUTH_TOKEN="+e2eBrokerAuthToken,
		"REAP_INTERVAL_SEC=3600",
	)
	brokerCmd.Stdout = io.Discard
	brokerCmd.Stderr = newTestLogWriter(t, "task-broker")
	if err := brokerCmd.Start(); err != nil {
		t.Fatalf("start task-broker: %v", err)
	}
	t.Cleanup(func() {
		_ = brokerCmd.Process.Signal(syscall.SIGTERM)
		_, _ = brokerCmd.Process.Wait()
	})

	waitReady(t, brokerURL+"/healthz", 10*time.Second)

	regBody, err := json.Marshal(api.RegisterFleetRequest{
		ID:     fleetID,
		Labels: []string{"e2e"},
	})
	if err != nil {
		t.Fatal(err)
	}
	regReq, err := http.NewRequest(http.MethodPost, brokerURL+"/v1/fleets", bytes.NewReader(regBody))
	if err != nil {
		t.Fatal(err)
	}
	regReq.Header.Set("Content-Type", "application/json")
	regReq.Header.Set("Authorization", "Bearer "+e2eBrokerAuthToken)
	regResp, err := http.DefaultClient.Do(regReq)
	if err != nil {
		t.Fatal(err)
	}
	regRespBody, _ := io.ReadAll(regResp.Body)
	_ = regResp.Body.Close()
	if regResp.StatusCode != http.StatusCreated {
		t.Fatalf("register fleet: %s", string(regRespBody))
	}

	return brokerStack{
		BrokerURL: brokerURL,
		FleetID:   fleetID,
		AuthToken: e2eBrokerAuthToken,
	}
}

func setE2EBrokerAuth(req *http.Request, token string) {
	req.Header.Set("Authorization", "Bearer "+token)
}

func freeTCPAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	return addr
}
