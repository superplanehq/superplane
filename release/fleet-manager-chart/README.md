# Standalone Fleet Manager

The Fleet Manager chart provisions runner VMs for one SuperPlane installation.
Install the chart after Terraform creates the cluster, network, and workload
identity resources. The SuperPlane chart creates a dedicated service-token
Secret in its namespace. Fleet Manager values reference the Secret name and
key. The credential value does not enter Helm values, Terraform state, browser
storage, logs, or API responses.

## Release coordinates

- Chart: `oci://ghcr.io/superplanehq/fleet-manager-chart`
- Fleet Manager image: `ghcr.io/superplanehq/fleet-manager:vX.Y.Z`
- Runner archives:
  `https://github.com/superplanehq/superplane/releases/download/runner%2F{version}`

Use an exact chart version, image version, and Runner version. Do not use a
floating tag.

Before the first component release, initialize the GHCR chart package with a
disposable prerelease version. Link it to `superplanehq/superplane`, enable
repository permission inheritance, and set its visibility to public. GitHub
creates command-line GHCR packages as private by default. The release pipeline
verifies the chart anonymously and refuses to reuse an existing release
version.

## Terraform output mapping

| Provider | Chart field | Terraform output |
| --- | --- | --- |
| AWS | `superplane.url` | `superplane_url` |
| AWS | `aws.region` | `runner_region` |
| AWS | `aws.subnetIds` | `runner_private_subnet_ids` |
| AWS | `aws.securityGroupIds` | `runner_security_group_id` |
| AWS | `aws.iamInstanceProfile` | `runner_instance_profile_name` |
| AWS | `serviceAccount.annotations.eks.amazonaws.com/role-arn` | `fleet_manager_role_arn` |
| GCP | `superplane.url` | `superplane_url` |
| GCP | `gcp.projectId` | `runner_project` |
| GCP | `gcp.zones` | `runner_zones` |
| GCP | `gcp.subnetwork` | `runner_subnetwork` |
| GCP | `gcp.networkTags` | `runner_network_tag` |
| GCP | `serviceAccount.annotations.iam.gke.io/gcp-service-account` | `fleet_manager_service_account_email` |
| Azure | `superplane.url` | `superplane_url` |
| Azure | `azure.subscriptionId` | `runner_subscription_id` |
| Azure | `azure.resourceGroup` | `runner_resource_group` |
| Azure | `azure.location` | `runner_location` |
| Azure | `azure.zones` | `runner_zones` |
| Azure | `azure.subnetId` | `runner_subnet_id` |
| Azure | `azure.networkSecurityGroupId` | `runner_nsg_id` |
| Azure | `azure.identityId` | `runner_identity_id` |
| Azure | `serviceAccount.annotations.azure.workload.identity/client-id` | `fleet_manager_identity_client_id` |
| Azure | `serviceAccount.name` | `fleet_manager_service_account_name` |
| Azure | `podLabels.azure.workload.identity/use` | Set to `"true"` |

AWS IRSA, GKE Workload Identity, and AKS Workload Identity grant cloud access
to the chart service account. Keep the default service-account name unless you
also update the provider trust binding.

For Azure, add the workload identity label to `values.yaml`:

```yaml
podLabels:
  azure.workload.identity/use: "true"
```

Public VM images are not part of this release. Replace these placeholders
before production use:

- AWS: `<PUBLIC_AMI_ID_FOR_REGION_AND_ARCH>`
- GCP: `<PUBLIC_GCP_IMAGE_OR_FAMILY>`
- Azure: `<PUBLIC_AZURE_IMAGE_VERSION_ID>`

## Install and upgrade

Complete **Set up runner fleets** during owner setup. For installations that
use the repository Terraform stacks, SuperPlane loads the provider, network,
location, and workload-identity fields from the chart configuration. Review
the values, replace the public VM image placeholder, and save `values.yaml`.

The automatically provisioned credential requires Fleet Manager and
SuperPlane in the same namespace. Confirm that the Secret exists without
reading its value:

```sh
kubectl --namespace superplane get secret superplane-fleet-manager-token
```

Install or upgrade an exact chart version:

```sh
helm upgrade --install fleet-manager \
  oci://ghcr.io/superplanehq/fleet-manager-chart \
  --version X.Y.Z \
  --namespace superplane \
  --create-namespace \
  --values values.yaml \
  --wait \
  --timeout 10m
```

The pre-install and pre-upgrade Job creates missing fleets and updates the
Runner version and specification of existing fleets. It preserves the current
`enabled` state. Removing values does not delete a fleet.

For an advanced cross-namespace installation, use an external secret manager.
Set `fleetManagerSetup.credentials.provision=false` in the SuperPlane chart and
provision the same dedicated credential Secret in both namespaces. Do not copy
the credential through Helm values or command-line arguments.

## Token rotation

For an automatically provisioned credential, delete the old Secret and upgrade
the SuperPlane chart. Its pre-upgrade hook creates a new random credential.
Then restart Fleet Manager:

```sh
kubectl --namespace superplane rollout restart deployment/fleet-manager-fleet-manager
kubectl --namespace superplane rollout status deployment/fleet-manager-fleet-manager
```

Confirm that Fleet Manager reconciles each configured fleet after the restart.
For an externally managed credential, rotate the Secret in both namespaces
through the external secret manager.

## Remove a fleet

1. Disable the fleet in SuperPlane.
2. Keep the fleet in `values.yaml` until Fleet Manager removes its runner VMs.
3. Remove the fleet from `values.yaml`.
4. Upgrade the Helm release.

## Migrate an integrated Fleet Manager

1. Export the current Fleet Manager configuration.
2. Preserve the current Fleet Manager ID.
3. Prepare standalone chart values that reference the dedicated Secret.
4. Scale the integrated Fleet Manager Deployment to zero.
5. Upgrade the SuperPlane chart. This upgrade creates the dedicated Secret and
   removes the integrated Deployment.
6. Install the standalone Fleet Manager chart in the SuperPlane namespace.
7. Verify that the startup Job updated every fleet.
8. Verify that Fleet Manager adopted the existing runner VMs.
9. Revoke the old personal token.

Changing the Fleet Manager ID can leave existing cloud resources unmanaged.

## Roll back

Use `helm rollback fleet-manager <revision>` to restore a previous chart
revision. Select a previous immutable Runner version in the values and upgrade
the release when the runner binary must also roll back. Do not rebuild or
republish an existing component version.
