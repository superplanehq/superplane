# SuperPlane on Azure Kubernetes Service (AKS)

This stack installs SuperPlane on AKS. It uses Azure Blob Storage for
platform files. PostgreSQL Flexible Server is optional. If Azure does not
offer Flexible Server SKUs in the region, Helm runs Postgres in the cluster.

The Helm chart in this repository includes Azure blob settings, workload
identity labels, the runner API, and Fleet Manager. Terraform uses that local
chart by default.

## Prerequisites

Install these tools:

- Terraform 1.5.0 or later
- Azure CLI
- Docker with Buildx
- Helm 3
- kubectl
- Packer 1.16.1 or later (runner images only)

You also need:

- An Azure subscription
- Permission to create AKS, networking, storage, and identities
- A DNS name that you control
- A container registry that AKS can pull (GHCR public images, or ACR for local builds)

## Quota and SKU checks

New subscriptions often have a small regional vCPU quota. The default node
pool is one `Standard_D2s_v4` node (2 vCPUs).

Check Flexible Server SKUs before you apply:

```bash
az postgres flexible-server list-skus --location eastus
```

If the command returns `[]`, set `create_postgresql_flexible_server = false`.
The error `Version should be in: []` is that Azure limit. It is not a SuperPlane
version string.

Confirm the node size exists in the region:

```bash
az vm list-skus --location eastus --size Standard_D2s_v4 --output table
```

Do not use `Standard_D4s_v5` or `Standard_D2ds_v5` unless the subscription
lists those sizes. Many new subscriptions allow `*_v4` only.

## 1. Sign in to Azure

```bash
az login
az account set --subscription SUBSCRIPTION_ID
```

## 2. Build images from this repository

Generate protobuf and OpenAPI files first (`make dev.up` then `make pb.gen`
if they are missing).

```bash
cd /path/to/superplane
export IMAGE_TAG="local-$(git rev-parse --short HEAD)"

make image.build IMAGE=superplane IMAGE_TAG="$IMAGE_TAG"

docker build \
  --platform linux/amd64 \
  -f release/fleet-manager/Dockerfile \
  --target runtime \
  -t "superplane-fleet-manager:${IMAGE_TAG}" \
  .
```

Do not run `release/superplane-image/build.sh` for this path. That script
pushes to GHCR.

## 3. Push images to a registry

AKS cannot pull images that exist only on your laptop. Create ACR, or use
GHCR if you can push there.

```bash
export ACR_NAME="YOUR_ACR_NAME"
az group create --name superplane-images --location eastus
az acr create \
  --resource-group superplane-images \
  --name "$ACR_NAME" \
  --sku Basic \
  --location eastus
az acr login --name "$ACR_NAME"
export ACR_LOGIN="$(az acr show -n "$ACR_NAME" --query loginServer -o tsv)"
export ACR_ID="$(az acr show -n "$ACR_NAME" --query id -o tsv)"

docker tag "superplane:${IMAGE_TAG}" "${ACR_LOGIN}/superplane:${IMAGE_TAG}"
docker tag "superplane-fleet-manager:${IMAGE_TAG}" \
  "${ACR_LOGIN}/superplane-fleet-manager:${IMAGE_TAG}"
docker push "${ACR_LOGIN}/superplane:${IMAGE_TAG}"
docker push "${ACR_LOGIN}/superplane-fleet-manager:${IMAGE_TAG}"
```

## 4. Configure Terraform

```bash
cd release/terraform/aks
cp terraform.tfvars.example terraform.tfvars
```

Set at least:

```hcl
subscription_id     = "YOUR_SUBSCRIPTION_ID"
domain_name         = "superplane.example.com"
letsencrypt_email   = "admin@example.com"
location            = "eastus"
node_count          = 1
node_vm_size        = "Standard_D2s_v4"
superplane_image_tag = "local-YOURSHA"
image_registry      = "YOURACR.azurecr.io"
container_registry_id = "/subscriptions/.../registries/YOURACR"
helm_chart_path     = "../../superplane-helm-chart/helm"
```

If `az postgres flexible-server list-skus --location eastus` is empty:

```hcl
create_postgresql_flexible_server = false
```

Leave `fleet_manager_config` empty until owner setup is complete.

Do not keep a `*_override.tf` file. Chart path and image registry are
Terraform variables.

## 5. Apply

```bash
cd release/terraform/aks
terraform init
terraform apply
```

The first apply takes 15-20 minutes. Terraform assigns AcrPull when
`container_registry_id` is set. You do not need `-target` for a new install.

If you already created AKS without `container_registry_id`, attach ACR
once, then apply again:

```bash
az aks update --resource-group superplane --name superplane --attach-acr "$ACR_NAME"
```

## 6. Configure kubectl

```bash
az aks get-credentials --resource-group superplane --name superplane --admin
```

kubectl uses `localhost:8080` until this command runs.

## 7. Configure DNS

```bash
kubectl get svc -n ingress-nginx ingress-nginx-controller \
  -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
```

Create a DNS A record for `domain_name` that points to that IP. Let's
Encrypt HTTP-01 needs the public name. A local `/etc/hosts` file does not
issue the certificate.

## 8. Open SuperPlane before DNS

HTTPS on the load balancer IP fails until the certificate is ready. Port
forward to the API:

```bash
kubectl port-forward -n superplane svc/superplane-api 8000:8000
```

Open `http://localhost:8000`. Complete owner setup. Create an admin API
token if you will start Fleet Manager.

## 9. Verify

```bash
kubectl get pods -n superplane
kubectl get certificate -n superplane
```

API, workers, and websocket must be `Running`. The certificate becomes
`True` after DNS propagates.

Access SuperPlane at `https://YOUR_DOMAIN`.

## 10. Optional: Azure runner VMs

Terraform already creates the gallery, runner subnet, NSG, and identities.

Build gallery images:

```bash
az login
packer init release/runner/packer/azure/runner.pkr.hcl
packer build \
  -var-file=release/runner/packer/azure/runner.pkrvars.hcl \
  -var architecture=amd64 \
  -var vm_size=Standard_D2ds_v4 \
  -var image_name=superplane-runner-amd64 \
  -var image_version=1.0.0 \
  release/runner/packer/azure/runner.pkr.hcl
```

Use a size from `az vm list-skus`. Do not use `*_v5` unless the
subscription allows it.

Build runner archives:

```bash
release/runner/build.sh "v0.0.0-local"
```

Host the archives on a public HTTPS prefix:

```text
<base>/<release-id>/runner-linux-amd64.tar.gz
<base>/<release-id>/checksums.txt
```

The SuperPlane blob account is private. Use a separate public storage
account or another HTTPS host.

Set `fleet_manager_config` from `terraform output` and apply again.

## Destroy

```bash
cd release/terraform/aks
terraform destroy
```

Flexible Server keeps backups for seven days when you created that server.

## Troubleshooting

### AKS SKU rejected

Set `node_vm_size` to a size from the Azure error list. `Standard_D2s_v4`
is the default.

### AKS service CIDR overlap

The stack sets `aks_service_cidr = 172.16.0.0/16`. Do not use the Azure
default `10.0.0.0/16` with this virtual network.

### Insufficient vCPU quota

Lower `node_count` to 1. Use `Standard_D2s_v4`. Raise quota before you
grow the pool.

### kube_admin_config empty list

Terraform reads `kube_admin_config_raw`. Run `terraform apply` again after
the cluster exists.

### Flexible Server `Version should be in: []`

Set `create_postgresql_flexible_server = false`.

### Storage 403 Key based authentication

The storage account keeps shared access keys so Terraform can wait for
the blob data plane. Pods still use workload identity.

### Helm labels unmarshal bool

Workload identity labels use `type = "string"` in Terraform. The chart
quotes label values.

### Helm context deadline exceeded

The SuperPlane release waits 20 minutes. Check pods:

```bash
kubectl get pods -n superplane
kubectl logs -n superplane deploy/superplane-api --tail=80
```

### Dirty database / CrashLoopBackOff

API, workers, and websocket used to run migrations together. A killed
`CREATE INDEX CONCURRENTLY` leaves `schema_migrations.dirty = true`.

Helm runs migrations in one hook Job. API, workers, and websocket start
the server and do not migrate.

To repair a dirty database:

```bash
kubectl exec -n superplane postgres-0 -- \
  psql -U superplane -d superplane \
  -c "SELECT version, dirty FROM schema_migrations;"
```

Drop any `INVALID` index. Set `dirty = false` only after the index is
valid. Restart the API deployment.

## Azure resources this stack creates

Defaults use name prefix `superplane` in `eastus`.

### Resource groups

| Resource | Default name | Purpose |
| --- | --- | --- |
| Resource group | `superplane` | AKS, network, storage, identities, DNS |
| Resource group | `superplane-runners` | Compute Gallery and runner VM identity |

### Networking

| Resource | Default name | Purpose |
| --- | --- | --- |
| Virtual network | `superplane-vnet` | `10.0.0.0/16` |
| Subnet | `superplane-aks` | AKS nodes (`10.0.0.0/20`) |
| Subnet | `superplane-postgres` | Flexible Server delegation (`10.0.16.0/24`) |
| Subnet | `superplane-runners` | Runner VMs (`10.0.17.0/24`) |
| NAT Gateway | `superplane-nat` | Outbound Internet for nodes and runners |
| Public IP | `superplane-nat` | NAT Gateway address |
| NSG | `superplane-aks` | AKS subnet |
| NSG | `superplane-runners` | Denies inbound Internet to runners |

AKS Overlay uses service CIDR `172.16.0.0/16` and kube-dns `172.16.0.10`.

### Kubernetes (Azure)

| Resource | Default name | Purpose |
| --- | --- | --- |
| AKS cluster | `superplane` | Control plane. SKU Standard. Workload identity and OIDC on. |
| Role assignment | Network Contributor | AKS cluster identity on the virtual network |
| Role assignment | AcrPull | Kubelet identity on ACR when `container_registry_id` is set |

The NGINX Ingress Controller creates a public Standard Load Balancer and
a public IP. Azure creates those objects. They are not Terraform
resources in this folder.

### Data

| Resource | Default name | Purpose |
| --- | --- | --- |
| Storage account | `sp` + 8 random chars | Private blob store |
| Blob container | `superplane` | Platform files |
| Private DNS zone | `privatelink.postgres.database.azure.com` | Flexible Server DNS |
| Private DNS VNet link | `superplane-postgres` | Links the zone to the VNet |
| PostgreSQL Flexible Server | `superplane-db` | Created only when `create_postgresql_flexible_server` is true |
| PostgreSQL database | `superplane` | Application database |
| PostgreSQL setting | `require_secure_transport=on` | TLS required |

### Identities and access

| Resource | Default name | Purpose |
| --- | --- | --- |
| User-assigned identity | `superplane-app` | Workload identity for API, workers, websocket |
| User-assigned identity | `superplane-fleet-manager` | Workload identity for Fleet Manager |
| User-assigned identity | `superplane-runner` | Assigned to runner VMs |
| Federated credential | `superplane-app` | `system:serviceaccount:superplane:superplane` |
| Federated credential | `superplane-fleet-manager` | `system:serviceaccount:superplane:superplane-fleet-manager` |
| Role | Storage Blob Data Contributor | App identity on the storage account |
| Role | Storage Blob Delegator | User-delegation SAS |
| Role | Virtual Machine Contributor | Fleet Manager on `superplane-runners` |
| Role | Network Contributor | Fleet Manager on `superplane-runners` |
| Role | Reader | Fleet Manager on the gallery |
| Role | Managed Identity Operator | Fleet Manager on the runner identity |

### Runner images

| Resource | Default name | Purpose |
| --- | --- | --- |
| Compute Gallery | `superplanerunners` | Trusted Launch image definitions |
| Gallery image | `superplane-runner-amd64` | linux/amd64 runner base |
| Gallery image | `superplane-runner-arm64` | linux/arm64 runner base |

Packer publishes image versions into those definitions. Terraform does
not build the versions.

### Not Azure services

These run in the cluster:

- cert-manager (Let's Encrypt)
- ingress-nginx
- SuperPlane API, workers, websocket
- RabbitMQ
- Postgres StatefulSet when Flexible Server is off
- Fleet Manager Deployment when `fleet_manager_config` is set

### Created outside Terraform

Create these yourself when you need them:

- Azure Container Registry (local image builds)
- DNS A record
- Packer image versions
- Public HTTPS host for runner release archives
- Let's Encrypt certificate (cert-manager creates it after DNS)

## Notes

- The AKS NSG allows inbound TCP 80, 443, and node ports 30000-32767 from
  the Internet. Let's Encrypt HTTP-01 and the public load balancer need
  those ports. The runner NSG still denies inbound Internet.
- New Azure virtual networks do not provide default outbound access. Nodes
  and runner VMs use the NAT Gateway.
- Azure does not allow the administrator login `postgres`. The default
  username is `superplane`.
- Blob objects stay private. SuperPlane signs downloads with a
  user-delegation SAS.
- Terraform reads AKS kubeconfig from `kube_admin_config_raw`.
