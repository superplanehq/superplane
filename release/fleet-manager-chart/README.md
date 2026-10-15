# Standalone Fleet Manager

The Fleet Manager chart provisions runner VMs for one SuperPlane installation.
Install the chart after Terraform creates the cluster, network, and workload
identity resources. The SuperPlane setup wizard creates a normal personal token
for the signed-in installation administrator. It puts the token in the generated
configuration only. The standalone chart stores the complete configuration in
one `<release>-config` Secret.

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
| AWS | `aws.ami` | `runner_image_amd64` or `runner_image_arm64` |
| AWS | `serviceAccount.annotations.eks.amazonaws.com/role-arn` | `fleet_manager_role_arn` |
| GCP | `superplane.url` | `superplane_url` |
| GCP | `gcp.projectId` | `runner_project` |
| GCP | `gcp.zones` | `runner_zones` |
| GCP | `gcp.subnetwork` | `runner_subnetwork` |
| GCP | `gcp.networkTags` | `runner_network_tag` |
| GCP | `gcp.image` | `runner_image_amd64` or `runner_image_arm64` |
| GCP | `serviceAccount.annotations.iam.gke.io/gcp-service-account` | `fleet_manager_service_account_email` |
| Azure | `superplane.url` | `superplane_url` |
| Azure | `azure.subscriptionId` | `runner_subscription_id` |
| Azure | `azure.resourceGroup` | `runner_resource_group` |
| Azure | `azure.location` | `runner_location` |
| Azure | `azure.zones` | `runner_zones` |
| Azure | `azure.subnetId` | `runner_subnet_id` |
| Azure | `azure.networkSecurityGroupId` | `runner_nsg_id` |
| Azure | `azure.identityId` | `runner_identity_id` |
| Azure | `azure.imageId` | `runner_image_amd64` or `runner_image_arm64` |
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

Build the amd64 and arm64 images with Packer. Set `runner_image_amd64` and
`runner_image_arm64` before you run `terraform apply`. The setup wizard does not
let you select a fleet when its architecture does not have an image.

## Install and upgrade

Complete **Set up runner fleets** during owner setup. For installations that
use the repository Terraform stacks, SuperPlane loads the provider, network,
location, images, and workload-identity fields from the chart configuration.
Select the provider and fleets. Then, run the generated Helm heredoc. The
command contains the complete values file and the personal token.

The chart creates `fleet-manager-config` for the normal `fleet-manager` release.
Helm stores the Secret content, including `installationAdminToken`, in release
metadata. Restrict access to Helm release Secrets and command history.

The post-install and post-upgrade Job creates missing fleets and updates the
Runner version and specification of existing fleets. It preserves the current
`enabled` state. Removing values does not delete a fleet.

## Token rotation

Open the setup wizard again to create a new personal token and generate a new
Helm command. Run the command to replace the configuration Secret. Revoke the
old token in personal-token settings after Fleet Manager starts to reconcile.
If you reload the wizard before you install, it creates another token because
SuperPlane cannot retrieve an existing plaintext token. Revoke unused tokens
manually.

## Remove a fleet

1. Disable the fleet in SuperPlane.
2. Keep the fleet in `values.yaml` until Fleet Manager removes its runner VMs.
3. Remove the fleet from `values.yaml`.
4. Upgrade the Helm release.

## Migrate an integrated Fleet Manager

1. Export the current Fleet Manager configuration.
2. Preserve the current Fleet Manager ID.
3. Generate the standalone chart command from the setup wizard.
4. Scale the integrated Fleet Manager Deployment to zero.
5. Upgrade the SuperPlane chart to remove the integrated Deployment.
6. Install the standalone Fleet Manager chart in the SuperPlane namespace.
7. Verify that the startup Job updated every fleet.
8. Verify that Fleet Manager adopted the existing runner VMs.
9. Revoke the old personal token.

Delete credential resources from the removed flow after the upgrade:

```sh
kubectl --namespace superplane delete secret superplane-fleet-manager-token --ignore-not-found
kubectl --namespace superplane delete job superplane-fleet-manager-credential --ignore-not-found
```

The charts do not remove these old resources automatically.

Changing the Fleet Manager ID can leave existing cloud resources unmanaged.

## Roll back

Use `helm rollback fleet-manager <revision>` to restore a previous chart
revision. Select a previous immutable Runner version in the values and upgrade
the release when the runner binary must also roll back. Do not rebuild or
republish an existing component version.
