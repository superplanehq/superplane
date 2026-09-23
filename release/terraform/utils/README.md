# Utils

Docker images for managing Superplane infrastructure on different cloud providers.

## Available Images

| Image            | Purpose                                 |
| ---------------- | --------------------------------------- |
| `Dockerfile.gke` | Google Kubernetes Engine (GKE) on GCP   |
| `Dockerfile.eks` | Elastic Kubernetes Service (EKS) on AWS |

Each image comes pre-installed with:

- Terraform
- Helm
- kubectl
- Cloud-specific CLI (gcloud / aws-cli)
- Common utilities (vim, jq)

## Usage

From this directory, run:

```bash
# GKE
make gke.shell

# EKS
make eks.shell
```

This builds the image and drops you into an interactive shell with:

- Cloud credentials mounted (persisted across sessions)
- Workspace mounted at `/workspace`
