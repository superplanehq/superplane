# SuperPlane Installation with Terraform

Terraform configurations to deploy [SuperPlane](https://github.com/superplanehq/superplane) on
managed Kubernetes clusters.

## Supported Platforms

| Platform | Directory | Status |
|----------|-----------|--------|
| Google Kubernetes Engine (GKE) | [`gke/`](./gke/) | Ready |
| Amazon Elastic Kubernetes Service (EKS) | [`eks/`](./eks/) | Ready |
| Azure Kubernetes Service (AKS) | [`aks/`](./aks/) | Ready |

## Quick Start

### GKE (Google Cloud)

```bash
cd gke
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars

terraform init
terraform apply
```

See [`gke/README.md`](./gke/README.md) for full instructions.

### EKS (AWS)

```bash
cd eks
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars

terraform init
terraform apply
```

See [`eks/README.md`](./eks/README.md) for full instructions.

### AKS (Azure)

```bash
cd aks
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars

terraform init
terraform apply
```

See [`aks/README.md`](./aks/README.md) for full instructions.

## What Gets Created

Each deployment creates:

- **Kubernetes cluster** (GKE, EKS, or AKS)
- **Managed PostgreSQL database** (Cloud SQL, RDS, or Flexible Server)
- **VPC networking** with private subnets for database
- **Load balancer** for ingress
- **cert-manager** for automatic SSL certificates
- **SuperPlane** application deployment

The AKS stack also creates Azure Blob storage, a runner subnet, and a Compute Gallery.

## Requirements

- Terraform >= 1.5.0
- Cloud provider CLI (gcloud, aws, or az) authenticated
- kubectl

## Checks

```bash
make check
make format
```

These targets build the utils image and run Terraform in that container.
Validate does not call AWS, GCP, or Azure.

## Architecture

```
                    ┌─────────────────┐
                    │    Internet     │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  Load Balancer  │
                    │  (GCE/ALB/Azure)│
                    └────────┬────────┘
                             │
         ┌───────────────────┼───────────────────┐
         │        Kubernetes Cluster             │
         │                   │                   │
         │          ┌────────▼────────┐          │
         │          │   SuperPlane    │          │
         │          └────────┬────────┘          │
         │                   │                   │
         └───────────────────┼───────────────────┘
                             │
                    ┌────────▼────────┐
                    │   PostgreSQL    │
                    │ Cloud SQL/RDS/  │
                    │ Flexible Server │
                    └─────────────────┘
```

## License

Apache 2.0
