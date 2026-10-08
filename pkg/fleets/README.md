# Fleet Manager

This process reconciles SuperPlane runner demand with configured infrastructure
providers. A fleet can use AWS or Docker. Fleet Manager only uses the
installation admin HTTP API.

Set `FLEET_MANAGER_CONFIG` to the JSON or YAML configuration body, or set
`FLEET_MANAGER_CONFIG_FILE` to a configuration path. `FLEET_MANAGER_CONFIG`
takes precedence when both variables are set. File configuration accepts JSON
files with a `.json` extension and YAML files with a `.yml` or `.yaml`
extension. The default file path is `/etc/superplane/fleet-manager.yaml`.

`INSTALLATION_ADMIN_TOKEN` overrides `installationAdminToken` from either
configuration source.

Set the top-level `id` to a stable identifier for the Fleet Manager
deployment. AWS providers apply it as the `superplane_fleet_manager_id`
resource tag and use it with the fleet ID to find owned runners.

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

Docker fleets use a configured runner image instead of a release artifact.
The local development configuration uses the tool-rich
`superplane-runner-local:dev` image.

The runner sends OS, architecture, and hostname as registration fields. The
server records the registration request's IP in the runner's `ip` field when available.
It uses the rightmost `X-Forwarded-For` address from the ingress proxy, or the
connection peer address for direct requests. Fleet Manager adds the
`fleet_manager_id` tag. Use `--tag key=value` more than once, or set `--tags`
or `RUNNER_TAGS` to comma-separated `key=value` pairs. Quote a pair as a CSV
field if its value contains a comma. Use `--tags-from-ec2-metadata` or
`--tags-from-gcp-metadata` to add cloud instance tags. AWS Fleet Manager enables
EC2 tags for its runners. The admin runner list and describe API return the
host fields and tags.

Fleet Manager creates unbound ephemeral runners. An idle runner can reserve the
next queued task. The target capacity is the number of queued tasks plus
`warmCapacity`. Set `warmCapacity` to the number of idle runners that the
fleet must keep when no tasks are queued. Set `maxCapacity` to limit the total
number of pending, idle, and busy runners. A value of `0`, or no value, permits
unlimited capacity. Set `warmCapacity` and `maxCapacity` to the same positive
value for a fixed-size fleet.
