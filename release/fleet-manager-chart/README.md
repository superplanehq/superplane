# Standalone Fleet Manager

The Fleet Manager chart provisions runner VMs for one SuperPlane installation.
Install the chart after Terraform creates the cluster, network, and workload
identity resources. The chart does not accept an installation token as a
value. Create the token Secret before the Helm installation.

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
| AWS | `aws.region` | `runner_region` |
| AWS | `aws.subnetIds` | `runner_private_subnet_ids` |
| AWS | `aws.securityGroupIds` | `runner_security_group_id` |
| AWS | `aws.iamInstanceProfile` | `runner_instance_profile_name` |
| AWS | `serviceAccount.annotations.eks.amazonaws.com/role-arn` | `fleet_manager_role_arn` |
| GCP | `gcp.projectId` | `runner_project` |
| GCP | `gcp.zones` | `runner_zones` |
| GCP | `gcp.subnetwork` | `runner_subnetwork` |
| GCP | `gcp.networkTags` | `runner_network_tag` |
| GCP | `serviceAccount.annotations.iam.gke.io/gcp-service-account` | `fleet_manager_service_account_email` |
| Azure | `azure.subscriptionId` | `runner_subscription_id` |
| Azure | `azure.resourceGroup` | `runner_resource_group` |
| Azure | `azure.location` | `runner_location` |
| Azure | `azure.zones` | `runner_zones` |
| Azure | `azure.subnetId` | `runner_subnet_id` |
| Azure | `azure.networkSecurityGroupId` | `runner_nsg_id` |
| Azure | `azure.identityId` | `runner_identity_id` |
| Azure | `serviceAccount.annotations.azure.workload.identity/client-id` | `fleet_manager_identity_client_id` |
| Azure | `serviceAccount.name` | `fleet_manager_service_account_name` |

AWS IRSA, GKE Workload Identity, and AKS Workload Identity grant cloud access
to the chart service account. Keep the default service-account name unless you
also update the provider trust binding.

Public VM images are not part of this release. Replace these placeholders
before production use:

- AWS: `<PUBLIC_AMI_ID_FOR_REGION_AND_ARCH>`
- GCP: `<PUBLIC_GCP_IMAGE_OR_FAMILY>`
- Azure: `<PUBLIC_AZURE_IMAGE_VERSION_ID>`

## Install and upgrade

Create a personal token from the Fleet Manager setup page. Enter it
interactively so it does not enter shell history:

```sh
read -rsp 'Fleet Manager token: ' FLEET_MANAGER_TOKEN; echo
kubectl --namespace superplane create secret generic superplane-fleet-manager-token \
  --from-literal=INSTALLATION_ADMIN_TOKEN="$FLEET_MANAGER_TOKEN" \
  --dry-run=client --output=yaml | kubectl apply -f -
unset FLEET_MANAGER_TOKEN
```

Install or upgrade an exact chart version:

```sh
helm upgrade --install fleet-manager \
  oci://ghcr.io/superplanehq/fleet-manager-chart \
  --version X.Y.Z \
  --namespace superplane \
  --create-namespace \
  --values values.yaml
```

The pre-install and pre-upgrade Job creates missing fleets and updates the
Runner version and specification of existing fleets. It preserves the current
`enabled` state. Removing values does not delete a fleet.

## Token rotation

Create a new token, update the Secret, and restart Fleet Manager:

```sh
kubectl --namespace superplane rollout restart deployment/fleet-manager-fleet-manager
kubectl --namespace superplane rollout status deployment/fleet-manager-fleet-manager
```

Verify reconciliation before you revoke the old token.

## Remove a fleet

1. Disable the fleet in SuperPlane.
2. Keep the fleet in `values.yaml` until Fleet Manager removes its runner VMs.
3. Remove the fleet from `values.yaml`.
4. Upgrade the Helm release.

## Migrate an integrated Fleet Manager

1. Export the current Fleet Manager configuration.
2. Preserve the current Fleet Manager ID.
3. Create a new token Secret with a different name.
4. Prepare standalone chart values.
5. Scale the integrated Fleet Manager Deployment to zero.
6. Upgrade the SuperPlane chart to a version without the integrated Deployment.
7. Install the standalone Fleet Manager chart.
8. Verify that the startup Job updated every fleet.
9. Verify that Fleet Manager adopted the existing runner VMs.
10. Revoke the old token.

Changing the Fleet Manager ID can leave existing cloud resources unmanaged.

## Roll back

Use `helm rollback fleet-manager <revision>` to restore a previous chart
revision. Select a previous immutable Runner version in the values and upgrade
the release when the runner binary must also roll back. Do not rebuild or
republish an existing component version.
