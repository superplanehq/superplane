# SuperPlane on Azure Kubernetes Service (AKS)

Deploy SuperPlane to AKS with PostgreSQL Flexible Server and Azure Blob storage.

## Prerequisites

- [Terraform](https://www.terraform.io/downloads) >= 1.5.0
- [Azure CLI](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli) installed and authenticated
- An Azure subscription with permission to create AKS, PostgreSQL, storage, and networking resources

## Deploy

```bash
az login
az account set --subscription SUBSCRIPTION_ID

cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your values

terraform init
terraform apply
```

The deployment takes 15-20 minutes.

## Configure kubectl

```bash
az aks get-credentials --resource-group superplane --name superplane --admin
```

## Configure DNS

After deployment, get the load balancer IP:

```bash
kubectl get svc -n ingress-nginx ingress-nginx-controller
```

Create an A record in your DNS provider pointing your domain to that IP address.

## Verify

```bash
kubectl get pods -n superplane
kubectl get certificate -n superplane
```

Access SuperPlane at `https://your-domain.com`

## Runners

The stack creates a runner resource group, a NAT-backed subnet with no inbound Internet access, a Compute Gallery, and identities for Fleet Manager and runner VMs.

Build Ubuntu images with [release/runner/packer/azure](../../runner/packer/azure/README.md). Publish them into the gallery image definitions that Terraform creates.

Set `fleet_manager_config` in `terraform.tfvars` to start Fleet Manager. The YAML body must use provider `azure` and the subnet, NSG, identity, and image IDs from `terraform output`.

## Destroy

PostgreSQL Flexible Server keeps backups for seven days. Destroy the deployment:

```bash
terraform destroy
```

## Notes

- New Azure virtual networks do not provide default outbound access. AKS nodes and runner VMs use the NAT Gateway.
- PostgreSQL is private to the virtual network. SuperPlane uses TLS (`sslmode=require`).
- Azure does not allow the administrator login `postgres`. The default username is `superplane`.
- Blob objects stay private. SuperPlane signs downloads with a user-delegation SAS.
- The NGINX Ingress Controller creates a public Standard Load Balancer for inbound HTTPS.
