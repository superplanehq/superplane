# Utils

Docker image for managing SuperPlane infrastructure on GKE and EKS.

## Image

`Dockerfile` installs:

- Terraform
- Helm
- kubectl
- gcloud
- AWS CLI
- eksctl
- vim and jq

The base image is the pinned Helm release.

## Usage

From this directory, run:

```bash
make shell
```

This builds the image and opens a shell with:

- gcloud credentials mounted at `/root/.config/gcloud`
- AWS credentials mounted at `/root/.aws`
- The install stacks mounted at `/workspace`
