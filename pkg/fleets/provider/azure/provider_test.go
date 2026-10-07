package azureprovider

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/superplanehq/superplane/pkg/fleets/artifact"
	"github.com/superplanehq/superplane/pkg/fleets/provider"
)

type fakeCompute struct {
	nics           map[string]armnetwork.Interface
	vms            map[string]armcompute.VirtualMachine
	createNICInput *armnetwork.Interface
	createVMInputs []armcompute.VirtualMachine
	createVMErrors []error
	deleteVMName   string
	deleteVMError  error
	deleteNICName  string
	deleteNICError error
}

func (f *fakeCompute) CreateNIC(
	_ context.Context,
	_, name string,
	nic armnetwork.Interface,
) (armnetwork.Interface, error) {
	f.createNICInput = &nic
	copied := nic
	copied.ID = to.Ptr("/nics/" + name)
	copied.Name = to.Ptr(name)
	if f.nics == nil {
		f.nics = map[string]armnetwork.Interface{}
	}
	f.nics[name] = copied
	return copied, nil
}

func (f *fakeCompute) DeleteNIC(_ context.Context, _, name string) error {
	f.deleteNICName = name
	if f.deleteNICError != nil {
		return f.deleteNICError
	}
	delete(f.nics, name)
	return nil
}

func (f *fakeCompute) CreateVM(
	_ context.Context,
	_, name string,
	vm armcompute.VirtualMachine,
) (armcompute.VirtualMachine, error) {
	f.createVMInputs = append(f.createVMInputs, vm)
	index := len(f.createVMInputs) - 1
	if index < len(f.createVMErrors) && f.createVMErrors[index] != nil {
		return armcompute.VirtualMachine{}, f.createVMErrors[index]
	}
	copied := vm
	copied.Name = to.Ptr(name)
	if copied.Properties == nil {
		copied.Properties = &armcompute.VirtualMachineProperties{}
	}
	copied.Properties.ProvisioningState = to.Ptr("Creating")
	now := time.Now().UTC()
	copied.Properties.TimeCreated = &now
	if f.vms == nil {
		f.vms = map[string]armcompute.VirtualMachine{}
	}
	f.vms[name] = copied
	return copied, nil
}

func (f *fakeCompute) DeleteVM(_ context.Context, _, name string) error {
	f.deleteVMName = name
	if f.deleteVMError != nil {
		return f.deleteVMError
	}
	delete(f.vms, name)
	return nil
}

func (f *fakeCompute) ListVMs(_ context.Context, _ string) ([]armcompute.VirtualMachine, error) {
	vms := make([]armcompute.VirtualMachine, 0, len(f.vms))
	for _, vm := range f.vms {
		vms = append(vms, vm)
	}
	return vms, nil
}

func TestCreateTagsAzureResourcesWithRunnerIdentity(t *testing.T) {
	client := &fakeCompute{}
	azureProvider := newTestProvider(t, client)
	resource, err := azureProvider.Create(context.Background(), provider.CreateRequest{
		RunnerID:      "09fe835e-9f8d-4ea4-9f78-962494c90e21",
		FleetID:       "linux-amd64",
		RunnerVersion: "1.2.3",
		Bootstrap:     []byte("#!/bin/bash\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resource.ID != "sprunner-09fe835e9f8d4ea49f78962494c90e21" ||
		resource.RunnerID != "09fe835e-9f8d-4ea4-9f78-962494c90e21" {
		t.Fatalf("resource = %#v", resource)
	}
	if client.createNICInput == nil {
		t.Fatal("expected NIC create")
	}
	if client.createNICInput.Properties == nil ||
		len(client.createNICInput.Properties.IPConfigurations) != 1 ||
		client.createNICInput.Properties.IPConfigurations[0].Properties.PublicIPAddress != nil {
		t.Fatalf("NIC has a public IP: %#v", client.createNICInput.Properties)
	}
	if len(client.createVMInputs) != 1 {
		t.Fatalf("VM creates = %d", len(client.createVMInputs))
	}
	vm := client.createVMInputs[0]
	if len(vm.Zones) != 1 || deref(vm.Zones[0]) != "1" {
		t.Fatalf("zones = %#v", vm.Zones)
	}
	if vm.Properties == nil || vm.Properties.SecurityProfile == nil ||
		deref(vm.Properties.SecurityProfile.SecurityType) != armcompute.SecurityTypesTrustedLaunch ||
		vm.Properties.SecurityProfile.UefiSettings == nil ||
		!deref(vm.Properties.SecurityProfile.UefiSettings.SecureBootEnabled) ||
		!deref(vm.Properties.SecurityProfile.UefiSettings.VTpmEnabled) {
		t.Fatalf("security profile = %#v", vm.Properties.SecurityProfile)
	}
	decoded, err := base64.StdEncoding.DecodeString(deref(vm.Properties.OSProfile.CustomData))
	if err != nil || string(decoded) != "#!/bin/bash\n" {
		t.Fatalf("custom data = %q, err = %v", decoded, err)
	}
	tags := ptrMap(vm.Tags)
	if tags[TagKeyRunnerID] != resource.RunnerID ||
		tags[TagKeyFleetID] != "linux-amd64" ||
		tags[TagKeyRunnerVersion] != "1.2.3" ||
		tags[TagKeyFleetManagerID] != "fleet-manager-production" ||
		tags["Environment"] != "production" {
		t.Fatalf("tags = %#v", tags)
	}
	if _, exists := tags["superplane_managed_runner"]; exists {
		t.Fatalf("legacy managed tag is present: %#v", tags)
	}
	if vm.Properties.StorageProfile == nil ||
		vm.Properties.StorageProfile.OSDisk == nil ||
		vm.Properties.StorageProfile.OSDisk.DiffDiskSettings == nil ||
		deref(vm.Properties.StorageProfile.OSDisk.DiffDiskSettings.Option) != armcompute.DiffDiskOptionsLocal {
		t.Fatalf("ephemeral OS disk = %#v", vm.Properties.StorageProfile)
	}
}

func TestListFiltersAzureResourcesByFleetManagerAndFleet(t *testing.T) {
	client := &fakeCompute{vms: map[string]armcompute.VirtualMachine{
		"owned": {
			Name: to.Ptr("owned"),
			Tags: map[string]*string{
				TagKeyFleetManagerID: to.Ptr("fleet-manager-production"),
				TagKeyFleetID:        to.Ptr("linux-amd64"),
				TagKeyRunnerID:       to.Ptr("runner-1"),
			},
			Properties: &armcompute.VirtualMachineProperties{
				ProvisioningState: to.Ptr("Succeeded"),
			},
		},
		"other-fleet": {
			Name: to.Ptr("other-fleet"),
			Tags: map[string]*string{
				TagKeyFleetManagerID: to.Ptr("fleet-manager-production"),
				TagKeyFleetID:        to.Ptr("linux-arm64"),
				TagKeyRunnerID:       to.Ptr("runner-2"),
			},
		},
		"other-manager": {
			Name: to.Ptr("other-manager"),
			Tags: map[string]*string{
				TagKeyFleetManagerID: to.Ptr("other"),
				TagKeyFleetID:        to.Ptr("linux-amd64"),
				TagKeyRunnerID:       to.Ptr("runner-3"),
			},
		},
	}}
	azureProvider := newTestProvider(t, client)
	resources, err := azureProvider.List(context.Background(), "linux-amd64")
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 || resources[0].ID != "owned" || resources[0].RunnerID != "runner-1" {
		t.Fatalf("resources = %#v", resources)
	}
}

func TestCreateRetriesNextZoneWhenAzureCapacityIsUnavailable(t *testing.T) {
	client := &fakeCompute{
		createVMErrors: []error{
			&azcore.ResponseError{ErrorCode: "ZonalAllocationFailed", StatusCode: http.StatusConflict},
			nil,
		},
	}
	azureProvider := newTestProvider(t, client)
	if _, err := azureProvider.Create(context.Background(), provider.CreateRequest{
		RunnerID:      "runner-1",
		FleetID:       "fleet-a",
		RunnerVersion: "1.0.0",
		Bootstrap:     []byte("bootstrap"),
	}); err != nil {
		t.Fatal(err)
	}
	if len(client.createVMInputs) != 2 ||
		deref(client.createVMInputs[0].Zones[0]) != "1" ||
		deref(client.createVMInputs[1].Zones[0]) != "2" {
		t.Fatalf("zone attempts = %#v", client.createVMInputs)
	}
}

func TestBuildBootstrapDownloadsPublicArtifact(t *testing.T) {
	azureProvider := newTestProvider(t, &fakeCompute{})
	script, err := azureProvider.BuildBootstrap(provider.RunnerBootstrap{
		RunnerID:          "runner-1",
		FleetID:           "fleet-a",
		RunnerAPIURL:      "https://superplane.example",
		RegistrationToken: "short-lived-registration-token",
		Artifact: artifact.Artifact{
			URL:    "https://downloads.example/runner/v1.2.3/runner-linux-amd64.tar.gz",
			SHA256: strings.Repeat("a", 64),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	body := string(script)
	for _, expected := range []string{
		"https://downloads.example/runner/v1.2.3/runner-linux-amd64.tar.gz",
		"sha256sum --check --strict",
		"--extract",
		"--gzip",
		"set +x",
		`RUNNER_API_URL="https://superplane.example" \`,
		`RUNNER_REGISTRATION_TOKEN="short-lived-registration-token" \`,
		`"$bundle_dir/install.sh"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("bootstrap does not contain %q:\n%s", expected, body)
		}
	}
	if strings.Contains(body, "amazon-cloudwatch-agent") {
		t.Fatalf("Azure bootstrap must not configure CloudWatch:\n%s", body)
	}
	command := exec.Command("bash", "-n")
	command.Stdin = strings.NewReader(body)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("bootstrap shell syntax: %v\n%s", err, output)
	}
}

func TestDeleteTreatsMissingAzureVMAsSuccess(t *testing.T) {
	client := &fakeCompute{
		deleteVMError:  &azcore.ResponseError{StatusCode: http.StatusNotFound},
		deleteNICError: &azcore.ResponseError{StatusCode: http.StatusNotFound},
	}
	azureProvider := newTestProvider(t, client)
	if err := azureProvider.Delete(context.Background(), provider.Resource{ID: "sprunner-gone"}); err != nil {
		t.Fatal(err)
	}
}

func TestNewRejectsResourceTagsThatOverrideReservedTags(t *testing.T) {
	_, err := New(&fakeCompute{}, Config{
		FleetManagerID:         "fleet-manager-production",
		SubscriptionID:         "sub",
		ResourceGroup:          "runners",
		Location:               "eastus",
		ImageID:                "/images/runner",
		VMSize:                 "Standard_D2ds_v4",
		Architecture:           "amd64",
		SubnetID:               "/subnets/runners",
		NetworkSecurityGroupID: "/nsgs/runners",
		IdentityID:             "/identities/runner",
		Zones:                  []string{"1"},
		ResourceTags:           map[string]string{TagKeyFleetID: "overridden"},
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "reserved Azure resource tag") {
		t.Fatalf("error = %v", err)
	}
}

func newTestProvider(t *testing.T, client ComputeAPI) *Provider {
	t.Helper()
	azureProvider, err := New(client, Config{
		FleetManagerID:         "fleet-manager-production",
		SubscriptionID:         "00000000-0000-0000-0000-000000000000",
		ResourceGroup:          "superplane-runners",
		Location:               "eastus",
		ImageID:                "/galleries/runners/images/superplane-runner-amd64/versions/1.0.0",
		VMSize:                 "Standard_D2ds_v4",
		Architecture:           "amd64",
		SubnetID:               "/subnets/runners",
		NetworkSecurityGroupID: "/nsgs/runners",
		IdentityID:             "/identities/runner",
		Zones:                  []string{"1", "2"},
		DiskSizeGB:             30,
		EphemeralOSDisk:        true,
		ResourceTags:           map[string]string{"Environment": "production"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return azureProvider
}

func ptrMap(tags map[string]*string) map[string]string {
	values := make(map[string]string, len(tags))
	for key, value := range tags {
		if value != nil {
			values[key] = *value
		}
	}
	return values
}

func deref[T any](value *T) T {
	if value == nil {
		var zero T
		return zero
	}
	return *value
}

func TestCreateDoesNotIgnoreUnexpectedAzureErrors(t *testing.T) {
	client := &fakeCompute{
		createVMErrors: []error{errors.New("quota exceeded")},
	}
	azureProvider := newTestProvider(t, client)
	_, err := azureProvider.Create(context.Background(), provider.CreateRequest{
		RunnerID:      "runner-1",
		FleetID:       "fleet-a",
		RunnerVersion: "1.0.0",
		Bootstrap:     []byte("bootstrap"),
	})
	if err == nil || !strings.Contains(err.Error(), "quota exceeded") {
		t.Fatalf("error = %v", err)
	}
	if client.deleteNICName != "sprunner-runner1-nic" &&
		!strings.HasSuffix(client.deleteNICName, "-nic") {
		t.Fatalf("NIC cleanup name = %q", client.deleteNICName)
	}
}
