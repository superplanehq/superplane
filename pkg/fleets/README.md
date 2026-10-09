# Fleet Manager

This process reconciles SuperPlane runner demand with configured infrastructure
providers. A fleet can use AWS, Azure, GCP, or Docker. Fleet Manager only uses the
installation admin HTTP API.

Set `FLEET_MANAGER_CONFIG` to the JSON or YAML configuration body, or set
`FLEET_MANAGER_CONFIG_FILE` to a configuration path. `FLEET_MANAGER_CONFIG`
takes precedence when both variables are set. File configuration accepts JSON
files with a `.json` extension and YAML files with a `.yml` or `.yaml`
extension. The default file path is `/etc/superplane/fleet-manager.yaml`.

`INSTALLATION_ADMIN_TOKEN` overrides `installationAdminToken` from either
configuration source.

Set the top-level `id` to a stable identifier for the Fleet Manager
deployment. AWS and Azure providers apply it as the
`superplane_fleet_manager_id` resource tag and use it with the fleet ID to
find owned runners. GCP providers apply it as the
`superplane_fleet_manager_id` instance label.

Each AWS fleet can define additional `resourceTags`. Fleet Manager applies
them to runner instances and root volumes. Additional tags cannot override the
reserved name, Fleet Manager, fleet, runner, runner version, or architecture
tags. The legacy `superplane_managed_runner` tag is also reserved and omitted
from new resources. A fleet can define at most 44 additional tags because EC2
allows 50 tags per resource and Fleet Manager applies six reserved tags.

Use a personal API token that belongs to an installation administrator. The
Fleet Manager sends it as an HTTP bearer token. Runner instances receive only
their short-lived registration token.

The local Compose service waits for owner setup and manages a development
token automatically. It does not require `INSTALLATION_ADMIN_TOKEN`.

AWS fleets require `runnerReleaseBaseUrl`. Fleet Manager selects a release
from this layout:

```text
<runnerReleaseBaseUrl>/<release-id>/runner-linux-amd64.tar.gz
<runnerReleaseBaseUrl>/<release-id>/runner-linux-arm64.tar.gz
<runnerReleaseBaseUrl>/<release-id>/checksums.txt
```

Set the fleet's `runnerVersion` to an immutable release ID. Supported IDs are
`sha:<40-character-lowercase-git-sha>` and `v<semantic-version>`, for example
`sha:0123456789abcdef0123456789abcdef01234567` or `v1.2.3`.
SHA builds require a clean checkout whose `HEAD` matches the release ID, and
the uploader resumes partial SHA releases only after hashing the existing
object bytes and verifying that they match the local artifacts. A published
`checksums.txt` marks the release complete and prevents later uploads.
SHA uploads require an AWS CLI v2 release that supports
`s3api put-object --if-none-match`.

Each archive contains the runner binary and `install.sh`. Fleet Manager reads
the selected archive checksum from `checksums.txt`. AWS bootstrap downloads
the archive, verifies its SHA-256, extracts it, and runs the bundled installer.
Set `aws.region` on every AWS fleet. Fleet Manager currently uses one AWS
SDK credential configuration and creates a regional EC2 client for each fleet.

Set `aws.cloudWatch.logGroupName` on a fleet to send
`superplane-runner.service` output to CloudWatch Logs. The runner AMI must
contain the Amazon CloudWatch Agent. Fleet Manager writes the runtime agent
configuration, uses the fleet's `aws.region`, names the stream after the EC2
instance ID, and starts the agent before it installs and starts the runner.
Logging setup is best effort and does not prevent the runner from starting.
The log group must already exist, and `aws.iamInstanceProfile` must name an
instance profile that permits `logs:CreateLogStream`, `logs:DescribeLogGroups`,
`logs:DescribeLogStreams`, and `logs:PutLogEvents` for that log group. The
example configuration uses a placeholder profile name with this expected
access.

The same instance profile enables Session Manager access when its EC2 role
allows `ssm:UpdateInstanceInformation` and the
`ssmmessages:CreateControlChannel`, `ssmmessages:CreateDataChannel`,
`ssmmessages:OpenControlChannel`, and `ssmmessages:OpenDataChannel` actions.
Its trust policy must allow EC2 to assume the role. The example sets
`aws.iamInstanceProfile` to `superplane-runner-instance-profile` and leaves
`aws.keyName` empty because Session Manager does not require an SSH key.

Azure fleets also require `runnerReleaseBaseUrl`. Bootstrap uses the same
archive layout and `install.sh` flow as AWS. Set `azure.imageId` to a
Compute Gallery image version. Set `azure.vmSize` to a Trusted Launch size.
`Standard_D2ds_v4` is the default for amd64. An arm64 fleet must set
`azure.vmSize`. Set `azure.ephemeralOSDisk` to `true`
only when that size has a local SSD. Fleet Manager creates one virtual
machine per runner. It places the NIC on `azure.subnetId`, applies
`azure.networkSecurityGroupId`, and assigns `azure.identityId`. It does not
give the VM a public IP. Set `azure.zones` to the availability zones that
the fleet may use. Fleet Manager retries the next zone when Azure reports
insufficient capacity.

See `config.azure.example.json` for a complete Azure fleet.

GCP fleets also require `runnerReleaseBaseUrl`. Bootstrap uses the same
archive layout and `install.sh` flow as AWS. Fleet Manager uses Application
Default Credentials, for example GKE Workload Identity. The credential needs
`roles/compute.instanceAdmin.v1` in `gcp.projectId`.

Fleet Manager creates one Compute Engine instance per runner. It sends the
bootstrap script as `user-data` metadata, and cloud-init runs it once at
first boot. Set `gcp.image` to an image or image family from
`release/runner/packer/gce`. Set `gcp.subnetwork` to a subnetwork with Cloud
NAT. Instances get no external IP address. Set `gcp.zones` to the zones that
the fleet may use. Fleet Manager tries the next zone when a zone has no
capacity.

`gcp.machineType` defaults to `e2-standard-4` for amd64 and `t2a-standard-4`
for arm64. `gcp.diskType` defaults to `pd-balanced`. Leave
`gcp.serviceAccountEmail` empty to start instances without a service
account. Runner tasks then cannot get Google Cloud credentials from the
metadata server. Use `gcp.networkTags` to apply firewall rules.

GCP labels allow only lowercase letters, digits, underscores, and dashes. The
top-level `id` and each GCP fleet `id` must follow that rule. Fleet Manager
reserves the `superplane_fleet_manager_id`, `superplane_fleet_id`, and
`superplane_runner_arch` labels. It stores the runner ID and runner version in
instance metadata.

See `config.gcp.example.json` for a complete GCP fleet.

Docker fleets use a configured runner image instead of a release artifact.
The local development configuration uses the tool-rich
`superplane-runner-local:dev` image.

Fleet Manager creates unbound ephemeral runners. An idle runner can reserve the
next queued task. The target capacity is the number of queued tasks plus
`warmCapacity`. Set `warmCapacity` to the number of idle runners that the
fleet must keep when no tasks are queued. Set `maxCapacity` to limit the total
number of pending, idle, and busy runners. A value of `0`, or no value, permits
unlimited capacity. Set `warmCapacity` and `maxCapacity` to the same positive
value for a fixed-size fleet.
