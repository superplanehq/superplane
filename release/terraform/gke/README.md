# SuperPlane on Google Kubernetes Engine (GKE)

Deploy SuperPlane to GKE with Cloud SQL PostgreSQL.

## Prerequisites

- [Terraform](https://www.terraform.io/downloads) >= 1.5.0
- [gcloud CLI](https://cloud.google.com/sdk/docs/install) installed and authenticated
- A GCP project with billing enabled

## Pre-deployment Steps

### 1. Authenticate with GCP

```bash
gcloud auth application-default login
gcloud config set project YOUR_PROJECT_ID
```

### 2. Create a Static IP Address

```bash
gcloud compute addresses create superplane-ip --global --ip-version=IPV4
gcloud compute addresses describe superplane-ip --global --format='get(address)'
```

### 3. Configure DNS

Create an A record pointing your domain to the static IP address.

## Deploy

```bash
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your values

terraform init
terraform apply
```

## Configure kubectl

```bash
gcloud container clusters get-credentials superplane --zone=us-central1-a --project=YOUR_PROJECT_ID
```

## Verify

```bash
kubectl get pods -n superplane
kubectl get certificate -n superplane
```

Access SuperPlane at `https://your-domain.com`

## Destroy

```bash
gcloud sql instances patch superplane-db --no-deletion-protection
terraform destroy
gcloud compute addresses delete superplane-ip --global
```
