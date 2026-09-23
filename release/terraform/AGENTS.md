# Install Terraform

Read this file when you edit `release/terraform`. For repository setup, read
[AGENTS.md](../../AGENTS.md) first.

`gke/` and `eks/` are the install stacks. `utils/Makefile` builds one image:
`build`, `gke.shell`, and `eks.shell`.

## Checks

From this directory:

```bash
make format
make check
```

From the repository root, run `make terraform.format` and `make terraform.check`.

`make check` builds the utils image and runs `terraform fmt` and
`terraform validate` in that container. Do not install Terraform on the host.
Validate does not call AWS or GCP.

CI runs `make terraform.check` when files under `release/terraform` change.
