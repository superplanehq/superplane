# AWS runner AMIs

This Packer template builds Ubuntu 24.04 AMIs for amd64 or arm64. The AMIs
contain the static tools that runner tasks use. They do not contain a runner
binary or registration credentials.

Fleet Manager selects the AMI for each fleet. Instance userdata downloads the
configured runner release, verifies its checksum, and runs the release
`install.sh`. The AMIs include the Amazon CloudWatch Agent, but Fleet Manager
supplies its region and log-group configuration at instance launch.

## Build

Install Packer 1.16.1 or later and export AWS credentials that can create an
EC2-backed AMI. Initialize the required Packer plugins:

```bash
packer init release/runner/packer/aws/runner.pkr.hcl
```

Then build one AMI by passing its architecture and build instance type:

```bash
packer build \
  -var architecture=amd64 \
  -var instance_type=t3.medium \
  release/runner/packer/aws/runner.pkr.hcl

packer build \
  -var architecture=arm64 \
  -var instance_type=t4g.medium \
  release/runner/packer/aws/runner.pkr.hcl
```

Each command builds one architecture. It writes the AMI ID to
`release/runner/packer/aws/manifest.json`. AMI IDs are specific to the selected
AWS region. Copy or consume the manifest before the next build replaces it.

You can also override optional Packer variables. For example:

```bash
packer build \
  -var architecture=amd64 \
  -var instance_type=t3.medium \
  -var aws_region=us-west-2 \
  -var ami_name_prefix=superplane-runner \
  release/runner/packer/aws/runner.pkr.hcl
```
