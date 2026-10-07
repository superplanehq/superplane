package gcpprovider

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/superplanehq/superplane/pkg/fleets/artifact"
	"github.com/superplanehq/superplane/pkg/fleets/provider"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

const testRunnerID = "09fe835e-9f8d-4ea4-9f78-962494c90e21"

type insertCall struct {
	project  string
	zone     string
	instance *compute.Instance
}

type fakeCompute struct {
	inserts      []insertCall
	insertErrors []error
	deletes      []string
	deleteError  error
	listFilter   string
	listOutput   []*compute.Instance
}

func (f *fakeCompute) InsertInstance(
	_ context.Context,
	project, zone string,
	instance *compute.Instance,
) error {
	f.inserts = append(f.inserts, insertCall{project: project, zone: zone, instance: instance})
	index := len(f.inserts) - 1
	if index < len(f.insertErrors) {
		return f.insertErrors[index]
	}
	return nil
}

func (f *fakeCompute) DeleteInstance(_ context.Context, project, zone, name string) error {
	f.deletes = append(f.deletes, project+"/"+zone+"/"+name)
	return f.deleteError
}

func (f *fakeCompute) ListInstances(
	_ context.Context,
	_ string,
	filter string,
) ([]*compute.Instance, error) {
	f.listFilter = filter
	return f.listOutput, nil
}

func TestCreateStartsLabeledInstanceWithoutExternalAddress(t *testing.T) {
	client := &fakeCompute{}
	gcpProvider := newTestProvider(t, client, func(config *Config) {
		config.Labels = map[string]string{"environment": "production"}
	})

	resource, err := gcpProvider.Create(context.Background(), provider.CreateRequest{
		RunnerID:      testRunnerID,
		FleetID:       "e1-large-amd64",
		RunnerVersion: "v1.2.3",
		Bootstrap:     []byte("#!/bin/bash\n"),
	})
	if err != nil {
		t.Fatal(err)
	}

	expectedName := "superplane-runner-" + testRunnerID
	if resource.ID != "us-central1-a/"+expectedName || resource.RunnerID != testRunnerID {
		t.Fatalf("resource = %#v", resource)
	}
	if len(client.inserts) != 1 {
		t.Fatalf("insert calls = %d", len(client.inserts))
	}
	call := client.inserts[0]
	instance := call.instance
	if call.project != "my-project" || call.zone != "us-central1-a" || instance.Name != expectedName {
		t.Fatalf("insert = %s/%s/%s", call.project, call.zone, instance.Name)
	}
	if instance.MachineType != "zones/us-central1-a/machineTypes/e2-standard-4" {
		t.Fatalf("machine type = %q", instance.MachineType)
	}
	for key, expected := range map[string]string{
		LabelFleetManagerID: "gke",
		LabelFleetID:        "e1-large-amd64",
		LabelArchitecture:   "amd64",
		"environment":       "production",
	} {
		if instance.Labels[key] != expected {
			t.Fatalf("label %s = %q, labels = %#v", key, instance.Labels[key], instance.Labels)
		}
	}
	for key, expected := range map[string]string{
		metadataKeyUserData:      "#!/bin/bash\n",
		MetadataKeyRunnerID:      testRunnerID,
		MetadataKeyRunnerVersion: "v1.2.3",
		metadataKeyBlockSSHKeys:  "true",
	} {
		if value := metadataValue(instance.Metadata, key); value != expected {
			t.Fatalf("metadata %s = %q", key, value)
		}
	}
	networkInterface := instance.NetworkInterfaces[0]
	if networkInterface.Subnetwork != testSubnetwork || len(networkInterface.AccessConfigs) != 0 {
		t.Fatalf("network interface = %#v", networkInterface)
	}
	disk := instance.Disks[0]
	if !disk.Boot || !disk.AutoDelete ||
		disk.InitializeParams.SourceImage != testImage ||
		disk.InitializeParams.DiskSizeGb != 30 ||
		disk.InitializeParams.DiskType != "zones/us-central1-a/diskTypes/pd-balanced" {
		t.Fatalf("boot disk = %#v", disk.InitializeParams)
	}
	if len(instance.ServiceAccounts) != 0 {
		t.Fatalf("instance must not get a service account: %#v", instance.ServiceAccounts)
	}
	shielded := instance.ShieldedInstanceConfig
	if shielded == nil || !shielded.EnableSecureBoot || !shielded.EnableVtpm || !shielded.EnableIntegrityMonitoring {
		t.Fatalf("shielded instance config = %#v", shielded)
	}
	if instance.Tags == nil || len(instance.Tags.Items) != 1 || instance.Tags.Items[0] != "superplane-runner" {
		t.Fatalf("network tags = %#v", instance.Tags)
	}
}

func TestCreateAttachesConfiguredServiceAccount(t *testing.T) {
	client := &fakeCompute{}
	gcpProvider := newTestProvider(t, client, func(config *Config) {
		config.ServiceAccountEmail = "runner@my-project.iam.gserviceaccount.com"
	})

	if _, err := gcpProvider.Create(context.Background(), testCreateRequest()); err != nil {
		t.Fatal(err)
	}

	accounts := client.inserts[0].instance.ServiceAccounts
	if len(accounts) != 1 || accounts[0].Email != "runner@my-project.iam.gserviceaccount.com" {
		t.Fatalf("service accounts = %#v", accounts)
	}
}

func TestCreateRetriesNextZoneWhenZoneHasNoCapacity(t *testing.T) {
	client := &fakeCompute{insertErrors: []error{
		&operationError{codes: []string{"ZONE_RESOURCE_POOL_EXHAUSTED"}, message: "no capacity"},
	}}
	gcpProvider := newTestProvider(t, client, nil)

	resource, err := gcpProvider.Create(context.Background(), testCreateRequest())
	if err != nil {
		t.Fatal(err)
	}

	if len(client.inserts) != 2 || client.inserts[1].zone != "us-central1-b" {
		t.Fatalf("insert calls = %#v", client.inserts)
	}
	if client.inserts[1].instance.MachineType != "zones/us-central1-b/machineTypes/e2-standard-4" {
		t.Fatalf("second zone machine type = %q", client.inserts[1].instance.MachineType)
	}
	if !strings.HasPrefix(resource.ID, "us-central1-b/") {
		t.Fatalf("resource ID = %q", resource.ID)
	}
}

func TestCreateStopsOnErrorsOtherThanCapacity(t *testing.T) {
	client := &fakeCompute{insertErrors: []error{
		&googleapi.Error{Code: http.StatusForbidden, Message: "permission denied"},
	}}
	gcpProvider := newTestProvider(t, client, nil)

	_, err := gcpProvider.Create(context.Background(), testCreateRequest())
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("error = %v", err)
	}
	if len(client.inserts) != 1 {
		t.Fatalf("insert calls = %d", len(client.inserts))
	}
}

func TestCreateTreatsExistingInstanceAsCreated(t *testing.T) {
	client := &fakeCompute{insertErrors: []error{
		&googleapi.Error{Code: http.StatusConflict, Message: "already exists"},
	}}
	gcpProvider := newTestProvider(t, client, nil)

	resource, err := gcpProvider.Create(context.Background(), testCreateRequest())
	if err != nil {
		t.Fatal(err)
	}
	if resource.ID != "us-central1-a/superplane-runner-"+testRunnerID {
		t.Fatalf("resource ID = %q", resource.ID)
	}
}

func TestListFindsInstancesByFleetManagerAndFleetLabels(t *testing.T) {
	created := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	runnerID := testRunnerID
	client := &fakeCompute{listOutput: []*compute.Instance{
		{
			Name:              "superplane-runner-" + testRunnerID,
			Zone:              "https://www.googleapis.com/compute/v1/projects/my-project/zones/us-central1-b",
			Status:            "RUNNING",
			CreationTimestamp: created.Format(time.RFC3339),
			Labels:            map[string]string{LabelFleetID: "e1-large-amd64"},
			Metadata: &compute.Metadata{Items: []*compute.MetadataItems{
				{Key: MetadataKeyRunnerID, Value: &runnerID},
			}},
		},
		{
			Name:   "not-a-runner",
			Zone:   "https://www.googleapis.com/compute/v1/projects/my-project/zones/us-central1-a",
			Labels: map[string]string{LabelFleetID: "e1-large-amd64"},
		},
	}}
	gcpProvider := newTestProvider(t, client, nil)

	resources, err := gcpProvider.List(context.Background(), "e1-large-amd64")
	if err != nil {
		t.Fatal(err)
	}

	expectedFilter := `(labels.superplane_fleet_manager_id = "gke") AND (labels.superplane_fleet_id = "e1-large-amd64")`
	if client.listFilter != expectedFilter {
		t.Fatalf("filter = %q", client.listFilter)
	}
	if len(resources) != 1 {
		t.Fatalf("resources = %#v", resources)
	}
	resource := resources[0]
	if resource.ID != "us-central1-b/superplane-runner-"+testRunnerID ||
		resource.RunnerID != testRunnerID ||
		resource.FleetID != "e1-large-amd64" ||
		resource.State != "RUNNING" ||
		!resource.CreatedAt.Equal(created) {
		t.Fatalf("resource = %#v", resource)
	}
}

func TestDeleteUsesZoneFromResourceID(t *testing.T) {
	client := &fakeCompute{}
	gcpProvider := newTestProvider(t, client, nil)

	err := gcpProvider.Delete(context.Background(), provider.Resource{
		ID:       "us-central1-b/superplane-runner-abc",
		RunnerID: "abc",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(client.deletes) != 1 || client.deletes[0] != "my-project/us-central1-b/superplane-runner-abc" {
		t.Fatalf("deletes = %#v", client.deletes)
	}
}

func TestDeleteTreatsMissingOrDeletingInstanceAsSuccess(t *testing.T) {
	for name, deleteError := range map[string]error{
		"missing": &googleapi.Error{Code: http.StatusNotFound},
		"deleting": &googleapi.Error{
			Code:   http.StatusBadRequest,
			Errors: []googleapi.ErrorItem{{Reason: "resourceNotReady"}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			gcpProvider := newTestProvider(t, &fakeCompute{deleteError: deleteError}, nil)
			err := gcpProvider.Delete(context.Background(), provider.Resource{
				ID: "us-central1-a/superplane-runner-gone",
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDeleteRejectsResourceIDWithoutZone(t *testing.T) {
	gcpProvider := newTestProvider(t, &fakeCompute{}, nil)
	err := gcpProvider.Delete(context.Background(), provider.Resource{ID: "superplane-runner-abc"})
	if err == nil {
		t.Fatal("expected resource ID without zone to fail")
	}
}

func TestNewRejectsInvalidLabels(t *testing.T) {
	tests := map[string]func(*Config){
		"reserved label":         func(config *Config) { config.Labels = map[string]string{LabelFleetID: "x"} },
		"uppercase label key":    func(config *Config) { config.Labels = map[string]string{"Environment": "x"} },
		"invalid label value":    func(config *Config) { config.Labels = map[string]string{"environment": "Prod 1"} },
		"invalid fleet manager":  func(config *Config) { config.FleetManagerID = "GKE" },
		"missing zones":          func(config *Config) { config.Zones = nil },
		"unsupported arch":       func(config *Config) { config.Architecture = "386" },
		"missing subnetwork":     func(config *Config) { config.Subnetwork = "" },
		"missing source image":   func(config *Config) { config.Image = "" },
		"missing machine type":   func(config *Config) { config.MachineType = "" },
		"missing project":        func(config *Config) { config.ProjectID = "" },
		"too many custom labels": func(config *Config) { config.Labels = tooManyLabels() },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			config := testConfig()
			mutate(&config)
			if _, err := New(&fakeCompute{}, config, nil); err == nil {
				t.Fatal("expected configuration to fail")
			}
		})
	}
}

func TestBuildBootstrapRendersValidShellScript(t *testing.T) {
	gcpProvider := newTestProvider(t, &fakeCompute{}, nil)
	sha := strings.Repeat("a", 64)

	body, err := gcpProvider.BuildBootstrap(provider.RunnerBootstrap{
		RunnerID:          testRunnerID,
		FleetID:           "e1-large-amd64",
		RunnerAPIURL:      "https://superplane.example",
		RegistrationToken: "registration-token",
		Artifact: artifact.Artifact{
			URL:    "https://downloads.example/runner/v1.2.3/runner-linux-amd64.tar.gz",
			SHA256: sha,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	script := string(body)
	for _, expected := range []string{
		`"https://downloads.example/runner/v1.2.3/runner-linux-amd64.tar.gz"`,
		sha + "  $bundle_dir/runner.tar.gz",
		`RUNNER_API_URL="https://superplane.example"`,
		`RUNNER_REGISTRATION_TOKEN="registration-token"`,
	} {
		if !strings.Contains(script, expected) {
			t.Fatalf("bootstrap is missing %q:\n%s", expected, script)
		}
	}
	command := exec.Command("bash", "-n")
	command.Stdin = strings.NewReader(script)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("bootstrap shell syntax: %v\n%s", err, output)
	}
}

func TestBuildBootstrapRejectsInvalidChecksum(t *testing.T) {
	gcpProvider := newTestProvider(t, &fakeCompute{}, nil)
	_, err := gcpProvider.BuildBootstrap(provider.RunnerBootstrap{
		RunnerID:          testRunnerID,
		FleetID:           "e1-large-amd64",
		RunnerAPIURL:      "https://superplane.example",
		RegistrationToken: "registration-token",
		Artifact:          artifact.Artifact{URL: "https://downloads.example/runner.tar.gz", SHA256: "short"},
	})
	if err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("error = %v", err)
	}
}

const (
	testImage      = "projects/my-project/global/images/family/superplane-runner-amd64"
	testSubnetwork = "projects/my-project/regions/us-central1/subnetworks/superplane-runners"
)

func testConfig() Config {
	return Config{
		FleetManagerID: "gke",
		ProjectID:      "my-project",
		Zones:          []string{"us-central1-a", "us-central1-b"},
		MachineType:    "e2-standard-4",
		Image:          testImage,
		Architecture:   "amd64",
		Subnetwork:     testSubnetwork,
		NetworkTags:    []string{"superplane-runner"},
		DiskSizeGB:     30,
		DiskType:       "pd-balanced",
	}
}

func newTestProvider(t *testing.T, client ComputeAPI, mutate func(*Config)) *Provider {
	t.Helper()
	config := testConfig()
	if mutate != nil {
		mutate(&config)
	}
	gcpProvider, err := New(client, config, nil)
	if err != nil {
		t.Fatal(err)
	}
	return gcpProvider
}

func testCreateRequest() provider.CreateRequest {
	return provider.CreateRequest{
		RunnerID:      testRunnerID,
		FleetID:       "e1-large-amd64",
		RunnerVersion: "v1.2.3",
		Bootstrap:     []byte("#!/bin/bash\n"),
	}
}

func tooManyLabels() map[string]string {
	labels := map[string]string{}
	for index := 0; index <= maxCustomLabels; index++ {
		labels[fmt.Sprintf("label_%d", index)] = "x"
	}
	return labels
}
