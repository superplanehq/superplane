# SuperPlane Installation with Terraform

Terraform configurations to deploy [SuperPlane](https://github.com/superplanehq/superplane) on
managed Kubernetes clusters.

## Supported Platforms

| Platform | Directory | Status |
|----------|-----------|--------|
| Google Kubernetes Engine (GKE) | [`gke/`](./gke/) | ✅ Ready |
| Amazon Elastic Kubernetes Service (EKS) | [`eks/`](./eks/) | ✅ Ready |

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

## What Gets Created

Each deployment creates:

- **Kubernetes cluster** (GKE or EKS)
- **Managed PostgreSQL database** (Cloud SQL or RDS)
- **VPC networking** with private subnets for database
- **Load balancer** for ingress
- **cert-manager** for automatic SSL certificates
- **SuperPlane** application deployment

## Requirements

- Terraform >= 1.5.0
- Cloud provider CLI (gcloud or aws) authenticated
- kubectl

## Architecture

```
                    ┌─────────────────┐
                    │    Internet     │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  Load Balancer  │
                    │  (GCE/ALB)      │
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
                    │ (Cloud SQL/RDS) │
                    └─────────────────┘
```

## License

Apache 2.0
