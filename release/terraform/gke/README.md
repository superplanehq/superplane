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

Deletion protection is enabled for the GKE cluster and the Cloud SQL instance.
Set both flags to false in `terraform.tfvars`.

```hcl
gke_deletion_protection = false
sql_deletion_protection = false
```

Apply the flags:

```bash
terraform apply
```

Destroy the deployment:

```bash
terraform destroy
```

The static IP is not a Terraform resource. Delete it after destroy:

```bash
gcloud compute addresses delete superplane-ip --global
```
