# SuperPlane on Azure Kubernetes Service (AKS)

This stack installs SuperPlane on AKS from published images and a published
Helm chart. You do not build images on the install path. Azure Blob Storage
holds platform files. PostgreSQL Flexible Server is optional. If Azure does
not offer Flexible Server SKUs in the region, Helm runs Postgres in the
cluster.

## Prerequisites

Install these tools:

- Terraform 1.5.0 or later
- Azure CLI
- Helm 3
- kubectl
- Docker (optional, to check the published images)

You also need:

- An Azure subscription
- Permission to create AKS, networking, storage, and identities
- A DNS name that you control

AKS pulls `ghcr.io/superplanehq/superplane` and
`ghcr.io/superplanehq/fleet-manager`. Those images are public.

## Quota and SKU checks

New subscriptions often have a small regional vCPU quota. The default node
pool is one `Standard_D2s_v4` node (2 vCPUs).

Fleet Manager runners use the same regional vCPU quota as the AKS nodes.
Family quotas can still have spare capacity when the regional total is full.

Estimate the total before you apply:

```text
node_count × node vCPUs + fleet maxCapacity × runner vCPUs
```

Check the quota:

```bash
az vm list-usage --location <region> -o table | grep -Ei "Total Regional vCPUs|<family>"
```

A new subscription often has a Total Regional vCPUs limit of 4.
The default install then has room for one runner.
Request a quota increase before you raise `maxCapacity`.

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

## 2. Confirm the published artifacts

The defaults pin these artifacts:

- Chart `oci://ghcr.io/superplanehq/superplane-chart` version
  `0.0.0-f711ba6b1ebf19ae26825b8a6397b9ef24832f68`
- App image `ghcr.io/superplanehq/superplane:f711ba6b1ebf19ae26825b8a6397b9ef24832f68-selfhosted`

Confirm them before you apply:

```bash
SHA=f711ba6b1ebf19ae26825b8a6397b9ef24832f68
helm show chart oci://ghcr.io/superplanehq/superplane-chart --version "0.0.0-${SHA}"
docker manifest inspect "ghcr.io/superplanehq/superplane:${SHA}-selfhosted"
```

`helm show chart` must print `appVersion: ${SHA}-selfhosted`.
Leave `superplane_image_tag` empty. The chart uses that appVersion.

## 3. Configure Terraform

```bash
cd release/terraform/aks
cp terraform.tfvars.example terraform.tfvars
```

Set these values:

```hcl
subscription_id   = "YOUR_SUBSCRIPTION_ID"
domain_name       = "superplane.example.com"
letsencrypt_email = "admin@example.com"
```

If `az postgres flexible-server list-skus --location eastus` is empty:

```hcl
create_postgresql_flexible_server = false
```

Terraform does not install Fleet Manager. The SuperPlane chart creates its
dedicated service-token Secret in Kubernetes. The credential does not enter
Terraform state.
Leave `helm_chart_path` empty. Leave `superplane_image_tag` empty.

## 4. Apply

```bash
cd release/terraform/aks
terraform init
terraform apply
```

The first apply takes 15-20 minutes. You do not need `-target` for a new
install.

## 5. Configure kubectl

```bash
az aks get-credentials --resource-group superplane --name superplane --admin
```

kubectl uses `localhost:8080` until this command runs.

## 6. Configure DNS

```bash
kubectl get svc -n ingress-nginx ingress-nginx-controller \
  -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
```

Create a DNS A record for `domain_name` that points to that IP. Let's
Encrypt HTTP-01 needs the public name. A local `/etc/hosts` file does not
issue the certificate.

## 7. Open SuperPlane before DNS

HTTPS on the load balancer IP fails until the certificate is ready. Port
forward to the API:

```bash
kubectl port-forward -n superplane svc/superplane-api 8000:8000
```

Open `http://localhost:8000`. Complete owner setup.

## 8. Verify

```bash
kubectl get pods -n superplane
kubectl get certificate -n superplane
```

API, workers, and websocket must be `Running`. The certificate becomes
`True` after DNS propagates.

Access SuperPlane at `https://YOUR_DOMAIN`.

## Update SuperPlane

Change `superplane_chart_version` to a published chart version
`0.0.0-<git-sha>`. Run the confirm commands in section 2 with the new SHA.
Then run `terraform apply`.

Leave `superplane_image_tag` empty. The chart appVersion selects the
self-hosted app image.

## Third-party chart versions

The defaults are:

- `cert_manager_chart_version` = `v1.21.2`
- `ingress_nginx_chart_version` = `4.15.1`

Set another value in `terraform.tfvars` only when you need a different
chart.

cert-manager 1.21 supports Kubernetes 1.33 through 1.36.
This stack leaves `kubernetes_version` empty. AKS then uses the N-1 minor
version. The October 2026 AKS calendar includes 1.34, 1.35, and 1.36.
Release 1.21 covers those versions. It stops at Kubernetes 1.36.

GKE and EKS do not pin cert-manager. EKS pins the Traefik chart at `41.7.0`.

ingress-nginx `4.15.1` is the latest chart on
`https://kubernetes.github.io/ingress-nginx` (14 October 2026).
The chart allows Kubernetes `>=1.21.0-0`. Controller `1.15.1` tested
Kubernetes 1.35.

## Ingress NGINX retirement

This stack installs ingress-nginx. The Ingress class is `nginx`.
Kubernetes retired Ingress NGINX in March 2026. The project ships no new
releases, bug fixes, or security updates after that date. Published Helm
charts remain available. This stack does not replace the controller.

- https://kubernetes.io/blog/2025/11/11/ingress-nginx-retirement/
- https://kubernetes.io/blog/2026/01/29/ingress-nginx-statement/
- https://github.com/kubernetes/ingress-nginx (archived)

## Development: local images and chart

Use this path to test a chart or image from this repository. The install
path above does not use it.

Generate protobuf and OpenAPI files first (`make dev.up`, then `make pb.gen`
if they are missing).

```bash
cd /path/to/superplane
export IMAGE_TAG="local-$(git rev-parse --short HEAD)"

make image.build IMAGE=superplane IMAGE_TAG="$IMAGE_TAG"

docker build \
  --platform linux/amd64 \
  -f release/fleet-manager/Dockerfile \
  --target runtime \
  -t "fleet-manager:${IMAGE_TAG}" \
  .
```

Do not run `release/superplane-image/build.sh` for this path. That script
pushes to GHCR.

AKS cannot pull images that exist only on your laptop. Create ACR:

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
docker tag "fleet-manager:${IMAGE_TAG}" "${ACR_LOGIN}/fleet-manager:${IMAGE_TAG}"
docker push "${ACR_LOGIN}/superplane:${IMAGE_TAG}"
docker push "${ACR_LOGIN}/fleet-manager:${IMAGE_TAG}"
```

Set these values in `terraform.tfvars`:

```hcl
helm_chart_path         = "../../superplane-helm-chart/helm"
superplane_image_tag    = "local-YOURSHA"
image_registry          = "YOURACR.azurecr.io"
container_registry_id   = "/subscriptions/.../registries/YOURACR"
```

`helm_chart_path` selects the chart in this repository. Terraform then
ignores `superplane_chart_version`. A local chart requires
`superplane_image_tag`. The local `Chart.yaml` appVersion `0.1.0` is a
placeholder. It is not a published image.

Terraform assigns AcrPull when `container_registry_id` is set.
If the cluster already exists without that value, attach ACR once, then
apply again:

```bash
az aks update --resource-group superplane --name superplane --attach-acr "$ACR_NAME"
```

Do not keep a `*_override.tf` file. Chart path and image registry are
Terraform variables.

## Runners

Terraform sets `fleet_manager_version` to `v0.1.2`.
The owner-setup wizard writes that chart version and that image tag.
Version `0.1.2` accepts a community gallery image ID and an empty `zones` list.

Terraform creates the runner subnet, network security group, and identities.
Use the SuperPlane community gallery for the VM image. Do not run Packer in
the customer subscription.

Download the manifest for the runner version:

```text
https://superplanehq-releases.s3.amazonaws.com/selfhosted/<version>/azure.json
```

Build `azure.imageId` from that file:

```text
/CommunityGalleries/<communityGalleryPublicName>/Images/<definition>/Versions/<image version>
```

Use `superplane-runner-amd64` or `superplane-runner-arm64`. The fleet
`architecture` must match that definition. Read `azure.regions` in the
manifest. Set `azure.location` to a region in that list.

Choose zones in the customer subscription:

```bash
az vm list-skus --location <region> --size <vmSize>
```

Zone numbers are not the same in every subscription. Leave `azure.zones`
empty when the region or size has no availability zones.

Open **Set up runner fleets** during owner setup. Terraform passes the runner
network, location, identity, and cluster values to SuperPlane. The wizard
preloads these values. Replace the Azure image placeholder with the community
gallery image version. Save `values.yaml` and run the Helm command that the wizard shows.

This stack still creates an empty customer Compute Gallery and two image
definitions. You can ignore them when `azure.imageId` points at the
community gallery. Turning that gallery off in Terraform would destroy
galleries that already contain image versions.

## Development: local runner images

Use this path to publish a runner image into the customer gallery. The
install path above does not use it.

Install Packer 1.16.1 or later.

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

Open **Set up runner fleets** during owner setup. Terraform passes the runner
network, location, identity, public SuperPlane URL, and cluster values to
SuperPlane. The wizard preloads these values. Replace the Azure image
placeholder with the immutable gallery image version that Packer created. Save
`values.yaml` and run the displayed Helm command.
See [the standalone chart guide](../../fleet-manager-chart/README.md).

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
- Fleet Manager, when installed separately with its standalone Helm chart

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
