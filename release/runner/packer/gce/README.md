# GCE runner images

This Packer template builds Ubuntu 24.04 Compute Engine images for amd64 or
arm64. The images contain the same task tools as the AWS runner AMIs. They do
not contain a runner binary or registration credentials. They do not contain
the AWS CLI, the CloudWatch Agent, or the SSM Agent.

Fleet Manager selects the image for each fleet. It sends the bootstrap script
as `user-data` metadata. cloud-init runs the script once at first boot. The
script downloads the configured runner release, verifies its checksum, and
runs the release `install.sh`.

`scripts/install.sh` installs the task tools for this image. It does
not install the AWS CLI, the CloudWatch Agent, or the SSM Agent. The
media tools script stays in `../aws/scripts` because it is not specific
to one cloud. Keep the tool version defaults the same as
`../aws/runner.pkr.hcl`.

## Build

Install Packer 1.16.1 or later. Authenticate with an account that can create
instances and images in the build project:

```bash
gcloud auth application-default login
gcloud services enable compute.googleapis.com --project=my-gcp-project
packer init release/runner/packer/gce/runner.pkr.hcl
```

Build one image per architecture. Each build adds a new image to the
`superplane-runner-amd64` or `superplane-runner-arm64` family:

```bash
packer build \
  -var project_id=my-gcp-project \
  -var architecture=amd64 \
  -var machine_type=e2-standard-4 \
  release/runner/packer/gce/runner.pkr.hcl

packer build \
  -var project_id=my-gcp-project \
  -var zone=us-central1-a \
  -var architecture=arm64 \
  -var machine_type=t2a-standard-4 \
  release/runner/packer/gce/runner.pkr.hcl
```

T2A Arm machines are available only in some zones. Set `zone` to a zone that
has the Arm machine type that you select.

The build writes the image name to `release/runner/packer/gce/manifest.json`.
Point a fleet at the family to use the newest image:

```text
projects/my-gcp-project/global/images/family/superplane-runner-amd64
```

Point a fleet at one image name to pin a build.

## Network access

By default, the build instance gets an external IP address in the `default`
network, and Packer connects over SSH. The network must allow inbound TCP 22.

Set `use_iap=true` to give the build instance no external IP address. Packer
then connects through Identity-Aware Proxy. Allow inbound TCP 22 from
`35.235.240.0/20` and give the subnetwork Cloud NAT so the build can download
packages:

```bash
packer build \
  -var project_id=my-gcp-project \
  -var architecture=amd64 \
  -var machine_type=e2-standard-4 \
  -var use_iap=true \
  -var subnetwork=superplane-runners \
  release/runner/packer/gce/runner.pkr.hcl
```
