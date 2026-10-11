package azureprovider

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
)

type ComputeAPI interface {
	CreateNIC(
		ctx context.Context,
		resourceGroup, name string,
		nic armnetwork.Interface,
	) (armnetwork.Interface, error)
	DeleteNIC(ctx context.Context, resourceGroup, name string) error
	CreateVM(
		ctx context.Context,
		resourceGroup, name string,
		vm armcompute.VirtualMachine,
	) (armcompute.VirtualMachine, error)
	DeleteVM(ctx context.Context, resourceGroup, name string) error
	ListVMs(ctx context.Context, resourceGroup string) ([]armcompute.VirtualMachine, error)
	ListNICs(ctx context.Context, resourceGroup string) ([]armnetwork.Interface, error)
}

type sdkCompute struct {
	vms  *armcompute.VirtualMachinesClient
	nics *armnetwork.InterfacesClient
}

func NewSDK(subscriptionID string, credential azcore.TokenCredential) (ComputeAPI, error) {
	if subscriptionID == "" {
		return nil, fmt.Errorf("Azure subscription ID is required")
	}
	if credential == nil {
		return nil, fmt.Errorf("Azure credential is required")
	}
	vms, err := armcompute.NewVirtualMachinesClient(subscriptionID, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("create Azure virtual machine client: %w", err)
	}
	nics, err := armnetwork.NewInterfacesClient(subscriptionID, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("create Azure network interface client: %w", err)
	}
	return &sdkCompute{vms: vms, nics: nics}, nil
}

func (s *sdkCompute) CreateNIC(
	ctx context.Context,
	resourceGroup, name string,
	nic armnetwork.Interface,
) (armnetwork.Interface, error) {
	poller, err := s.nics.BeginCreateOrUpdate(ctx, resourceGroup, name, nic, nil)
	if err != nil {
		return armnetwork.Interface{}, err
	}
	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return armnetwork.Interface{}, err
	}
	return resp.Interface, nil
}

func (s *sdkCompute) DeleteNIC(ctx context.Context, resourceGroup, name string) error {
	poller, err := s.nics.BeginDelete(ctx, resourceGroup, name, nil)
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (s *sdkCompute) CreateVM(
	ctx context.Context,
	resourceGroup, name string,
	vm armcompute.VirtualMachine,
) (armcompute.VirtualMachine, error) {
	poller, err := s.vms.BeginCreateOrUpdate(ctx, resourceGroup, name, vm, nil)
	if err != nil {
		return armcompute.VirtualMachine{}, err
	}
	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return armcompute.VirtualMachine{}, err
	}
	return resp.VirtualMachine, nil
}

func (s *sdkCompute) DeleteVM(ctx context.Context, resourceGroup, name string) error {
	poller, err := s.vms.BeginDelete(ctx, resourceGroup, name, &armcompute.VirtualMachinesClientBeginDeleteOptions{
		ForceDeletion: to.Ptr(true),
	})
	if err != nil {
		return err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	return err
}

func (s *sdkCompute) ListVMs(
	ctx context.Context,
	resourceGroup string,
) ([]armcompute.VirtualMachine, error) {
	pager := s.vms.NewListPager(resourceGroup, nil)
	var vms []armcompute.VirtualMachine
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, vm := range page.Value {
			if vm != nil {
				vms = append(vms, *vm)
			}
		}
	}
	return vms, nil
}

func (s *sdkCompute) ListNICs(
	ctx context.Context,
	resourceGroup string,
) ([]armnetwork.Interface, error) {
	pager := s.nics.NewListPager(resourceGroup, nil)
	var nics []armnetwork.Interface
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, nic := range page.Value {
			if nic != nil {
				nics = append(nics, *nic)
			}
		}
	}
	return nics, nil
}
