# Azure Integration for SuperPlane

This integration connects SuperPlane with Microsoft Azure to automate VM-centric workflows.
It currently provides:

- `azure.onVirtualMachineCreated` trigger (Event Grid driven)
- `azure.onVirtualMachineDeleted` trigger (Event Grid driven)
- `azure.createVirtualMachine` action (VM provisioning with dynamic networking UX)

Website: https://azure.microsoft.com/

## Architecture

The integration uses a thin ARM REST client (`armClient`) instead of per-resource SDK packages.
Only `azcore` and `azidentity` are used from the Azure SDK — for OIDC authentication and token management.
All resource operations (VMs, NICs, Public IPs, Resource Groups, SKUs, Event Grid subscriptions) go through
direct ARM REST API calls with automatic LRO polling and pagination.

## Authentication

The integration uses Azure Workload Identity Federation with no client secret storage.

- SuperPlane signs OIDC JWTs using its own keys (`ctx.OIDC.Sign()`)
- The signed JWT is used as a client assertion via `azidentity.NewClientAssertionCredential`
- Azure AD exchanges the assertion for an Azure access token

The OIDC issuer is the SuperPlane base URL. The subject is `app-installation:<integration-id>`
and the audience is the integration ID.

## Components

## Trigger: `azure.onVirtualMachineCreated`

Starts workflows when Azure emits `Microsoft.Resources.ResourceWriteSuccess` for VM resources via Event Grid.

## Trigger: `azure.onVirtualMachineDeleted`

Starts workflows when Azure emits `Microsoft.Resources.ResourceDeleteSuccess` for VM resources via Event Grid.

Key behavior:

- Automatically creates Event Grid subscriptions at the Azure subscription scope
- Handles Event Grid subscription validation handshake (via validationUrl GET)
- Filters to VM resources (`Microsoft.Compute/virtualMachines`)
- Emits event payload when provisioning state is `Succeeded`
- Cleans up Event Grid subscriptions when the trigger is removed

## Action: `azure.createVirtualMachine`

Creates a new Azure VM using ARM REST API long-running operations.

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
- `EventGrid EventSubscription Contributor`

## Troubleshooting

### `AADSTS90061` / `AADSTS50166` External OIDC endpoint failed

- Your public issuer URL is stale/unreachable
- Refresh tunnel URL
- Update app `BASE_URL`/`WEBHOOKS_BASE_URL`
- Update Azure federated credential issuer to match exactly
- Restart the app to re-sync the integration

## Development Validation

Run Azure integration tests:

```bash
go test ./pkg/integrations/azure/...
```
