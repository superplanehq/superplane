# Azure runner images

This Packer template builds Ubuntu 24.04 images for amd64 or arm64. It
publishes each image to an Azure Compute Gallery. The images contain the
static tools that runner tasks use. They do not contain a runner binary
or registration credentials.

Fleet Manager selects the gallery image for each fleet. VM custom data
downloads the configured runner release, verifies its checksum, and runs the
release `install.sh`.

Create the gallery and image definitions with the AKS Terraform stack in
[release/terraform/aks](../../../terraform/aks) before you build.

## Trusted Launch

`trusted_launch` defaults to true. Both templates use this variable. The
build VM then uses Trusted Launch.
Use true for a private gallery whose image definition is `TrustedLaunch`.
The AKS Terraform stack creates that private gallery. Keep the default
for the development path in `release/terraform/aks`.

Set `trusted_launch` to false for a community gallery or a direct shared
gallery. The image definition must be `TrustedLaunchSupported`. Azure
rejects a community or direct shared image that contains Trusted Launch
VM Guest State. With false, the template sets `security_type` to null and
turns off secure boot and vTPM. Plugin azure-arm 2.3.3 then omits
`securityProfile`. The Gen2 source image builds a Standard Gen2 VM.
Virtual machines created from that image can still use Trusted Launch.

## Build

Install Packer 1.16.1 or later. Authenticate with Azure CLI. Initialize the
required Packer plugins:

```bash
az login
packer init release/runner/packer/azure/runner.pkr.hcl
```

Copy `runner.pkrvars.hcl.example` and set the gallery names from
`terraform output`. Then build one architecture:

```bash
packer build \
  -var-file=release/runner/packer/azure/runner.pkrvars.hcl \
  -var architecture=amd64 \
  -var vm_size=Standard_D2ds_v4 \
  -var image_name=superplane-runner-amd64 \
  -var image_version=1.0.0 \
  release/runner/packer/azure/runner.pkr.hcl

packer build \
  -var-file=release/runner/packer/azure/runner.pkrvars.hcl \
  -var architecture=arm64 \
  -var vm_size=Standard_D2pds_v5 \
  -var image_name=superplane-runner-arm64 \
  -var image_version=1.0.0 \
  release/runner/packer/azure/runner.pkr.hcl
```

Each command builds one architecture. Use a new `image_version` for every
build. The image definition must already exist in the gallery. The AKS
Terraform stack creates `superplane-runner-amd64` and
`superplane-runner-arm64` with `TrustedLaunch`.

`runner.pkr.hcl` and `runner.local.pkr.hcl` accept the same optional
variables. Leave `build_resource_group_name` empty to let Packer create a
temporary resource group in `location`. Set it to build in an existing
resource group. Packer 2.3.3 rejects a template that sets both `location`
and `build_resource_group_name` on the source. This template omits
`location` on the source when the build group is set. Keep `location` equal
to that group's region. Replica regions are `location` plus
`replication_regions`.
