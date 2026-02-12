# Azure Integration for SuperPlane

This integration connects SuperPlane with Microsoft Azure to automate VM-centric workflows.
It currently provides:

- `azure.onVirtualMachineCreated` trigger (Event Grid driven)
- `azure.createVirtualMachine` action (VM provisioning with dynamic networking UX)

Website: https://azure.microsoft.com/

## Authentication

The integration follows a dual-mode authentication strategy.

### Production Mode (Workload Identity Federation / OIDC)

Production uses Azure Workload Identity Federation with no client secret storage.

- Provider reads a signed OIDC assertion from `AZURE_FEDERATED_TOKEN_FILE`
- Provider authenticates using `azidentity.NewClientAssertionCredential`
- Azure AD exchanges assertion for Azure access token

This is the secure default for hosted and production environments.

### Local Development Mode (`az login`)

For local development, setup and iteration can use Azure CLI authentication (`az login`), while SuperPlane keeps the same federated flow at runtime.

Typical local flow:

1. Run `az login` (for portal/CLI setup and quick validation)
2. Set a reachable public issuer URL for your local app (tunnel)
3. Regenerate/update the OIDC token file referenced by `AZURE_FEDERATED_TOKEN_FILE`

This provides a seamless developer workflow without switching production auth architecture.

## Components

## Trigger: `azure.onVirtualMachineCreated`

Starts workflows when Azure emits `Microsoft.Resources.ResourceWriteSuccess` for VM resources via Event Grid.

Key behavior:

- Handles Event Grid subscription validation handshake
- Filters to VM resources (`Microsoft.Compute/virtualMachines`)
- Emits event payload when provisioning state is `Succeeded`

## Action: `azure.createVirtualMachine`

Creates a new Azure VM using Azure SDK long-running operations.

### UX and Networking Model (Important)

Users do **not** need to manually create a NIC.

- User selects: Resource Group -> Virtual Network -> Subnet (cascading dropdowns)
- Backend automatically performs networking plumbing:
  - resolves subnet
  - creates/attaches NIC in background
  - optionally creates/attaches Public IP

An existing `networkInterfaceId` is still accepted as an optional advanced override.

### Supported VM Configuration

- Basics: Resource Group, Region, VM Name, Image, Size
- Networking: VNet + Subnet cascading selection
- Disks: OS Disk Type (`Standard_LRS`, `StandardSSD_LRS`, `Premium_LRS`)
- Public IP: optional Standard SKU Public IP
- Advanced: Custom Data / cloud-init (base64 encoded before Azure API call)

### Action Output

The action returns:

- `id` (VM resource ID)
- `privateIp`
- `publicIp` (empty when no Public IP is attached)
- plus operational metadata (`name`, `provisioningState`, `location`, `size`, `adminUsername`)

## Prerequisites

Before using the integration:

1. Create an Azure App Registration
2. Configure Federated Credential(s) for SuperPlane issuer + subject + audience
3. Grant RBAC permissions

### Required Azure Resources

- Azure Subscription
- Resource Group(s) where VMs will be created
- Existing VNet/Subnet (for implicit NIC path)
- Azure App Registration (Microsoft Entra ID)
- Federated Credential on that app registration

### Recommended RBAC Roles

At minimum, assign permissions at Resource Group (or Subscription) scope.

- `Contributor` (simplest recommended role for full VM provisioning path)

Or use least-privilege split roles:

- `Virtual Machine Contributor`
- `Network Contributor`

If using Event Grid provisioning externally, include corresponding Event Grid role(s).

## Troubleshooting

### `failed to read OIDC token ... AZURE_FEDERATED_TOKEN_FILE ... no such file or directory`

- Ensure env var is set in the running app container/process
- Ensure token file exists and is readable at that path
- In local dev, run `az login` and refresh your local federation setup/token generation flow

### `AADSTS90061` / `AADSTS50166` External OIDC endpoint failed

- Your public issuer URL is stale/unreachable
- Refresh tunnel URL
- Update app `BASE_URL`/`WEBHOOKS_BASE_URL`
- Update Azure federated credential issuer to match exactly
- Regenerate token file and restart app

## Development Validation

Run Azure integration tests:

```bash
go test ./pkg/integrations/azure/...
```

