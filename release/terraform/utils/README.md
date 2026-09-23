# Utils

Docker image for managing SuperPlane infrastructure on GKE and EKS.

## Image

`Dockerfile` installs:

- Terraform
- Helm
- kubectl
- gcloud
- gke-gcloud-auth-plugin
- AWS CLI
- eksctl
- vim and jq

The base image is the pinned Helm release.

## Usage

From this directory, run:

```bash
make gke.shell
make eks.shell
```

Both targets build the same image and mount the install stacks at `/workspace`.
`make gke.shell` mounts gcloud credentials at `/root/.config/gcloud`.
`make eks.shell` mounts AWS credentials at `/root/.aws`.
