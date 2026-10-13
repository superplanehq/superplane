# Azure runner images

This Packer template builds Ubuntu 24.04 images for amd64 or arm64. It
publishes Trusted Launch images to an Azure Compute Gallery. The images
contain the static tools that runner tasks use. They do not contain a runner
binary or registration credentials.

Fleet Manager selects the gallery image for each fleet. VM custom data
downloads the configured runner release, verifies its checksum, and runs the
release `install.sh`.

Create the gallery and image definitions with the AKS Terraform stack in
[release/terraform/aks](../../../terraform/aks) before you build.

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
build. The image definition must already exist in the gallery. Terraform
creates `superplane-runner-amd64` and `superplane-runner-arm64` with
`TrustedLaunchSupported`.
