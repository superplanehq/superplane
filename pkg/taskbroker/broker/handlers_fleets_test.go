package broker

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/superplanehq/superplane/pkg/taskbroker/shared/api"
)

func TestListFleetsReturnsMetadata(t *testing.T) {
	st := openStore(t)

	srv := &Server{Store: st}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "tok"}))
	defer ts.Close()

	regBody, err := json.Marshal(api.RegisterFleetRequest{
		ID:          "e1-tiny-amd64",
		Provisioner: "aws",
		Arch:        "amd64",
		Size:        "t3.micro",
	})
	if err != nil {
		t.Fatal(err)
	}
	regReq, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/fleets", bytes.NewReader(regBody))
	if err != nil {
		t.Fatal(err)
	}
	regReq.Header.Set("Content-Type", "application/json")
	regReq.Header.Set("Authorization", "Bearer tok")
	regResp, err := ts.Client().Do(regReq)
	if err != nil {
		t.Fatal(err)
	}
	_ = regResp.Body.Close()
	if regResp.StatusCode != http.StatusCreated {
		t.Fatalf("register status: %d", regResp.StatusCode)
	}

	listReq, err := http.NewRequest(http.MethodGet, ts.URL+"/v1/fleets", nil)
	if err != nil {
		t.Fatal(err)
	}
	listReq.Header.Set("Authorization", "Bearer tok")
	listResp, err := ts.Client().Do(listReq)
	if err != nil {
		t.Fatal(err)
	}
	defer listResp.Body.Close()
	if listResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(listResp.Body)
		t.Fatalf("list status: %d body: %s", listResp.StatusCode, string(body))
	}

	var fleets []api.FleetResponse
	if err := json.NewDecoder(listResp.Body).Decode(&fleets); err != nil {
		t.Fatal(err)
	}
	if len(fleets) != 1 {
		t.Fatalf("fleets: %#v", fleets)
	}
	got := fleets[0]
	if got.ID != "e1-tiny-amd64" || got.Provisioner != "aws" || got.Arch != "amd64" || got.Size != "t3.micro" {
		t.Fatalf("fleet metadata: %#v", got)
	}
	if got.CreatedAt == 0 {
		t.Fatal("expected created_at_unix")
	}

	// Upsert updates catalog fields visible on list.
	upsertBody, err := json.Marshal(api.RegisterFleetRequest{
		ID:          "e1-tiny-amd64",
		Provisioner: "aws",
		Arch:        "amd64",
		Size:        "t3.small",
	})
	if err != nil {
		t.Fatal(err)
	}
	upsertReq, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/fleets", bytes.NewReader(upsertBody))
	if err != nil {
		t.Fatal(err)
	}
	upsertReq.Header.Set("Content-Type", "application/json")
	upsertReq.Header.Set("Authorization", "Bearer tok")
	upsertResp, err := ts.Client().Do(upsertReq)
	if err != nil {
		t.Fatal(err)
	}
	_ = upsertResp.Body.Close()
	if upsertResp.StatusCode != http.StatusCreated {
		t.Fatalf("upsert status: %d", upsertResp.StatusCode)
	}

	listResp2, err := ts.Client().Do(listReq)
	if err != nil {
		t.Fatal(err)
	}
	defer listResp2.Body.Close()
	if err := json.NewDecoder(listResp2.Body).Decode(&fleets); err != nil {
		t.Fatal(err)
	}
	if len(fleets) != 1 || fleets[0].Size != "t3.small" {
		t.Fatalf("after upsert: %#v", fleets)
	}
}
