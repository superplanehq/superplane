# SuperPlane on Google Kubernetes Engine (GKE)

This guide installs SuperPlane in your own GCP project. One Terraform apply
creates a GKE cluster, a Cloud SQL PostgreSQL instance, runner infrastructure,
and the SuperPlane Helm release.

A runner is a short-lived Compute Engine VM that runs one task, for example a
Run Bash node or a factory agent. Fleet Manager runs in the cluster. It creates
a runner VM when a task is queued and deletes the VM when the task ends.

Do the steps in this order:

1. [Prepare the project](#1-prepare-the-project).
2. [Install SuperPlane](#2-install-superplane).
3. [Install Fleet Manager](#3-install-fleet-manager).

## What Terraform creates

| Part | Resources | Variable |
| --- | --- | --- |
| SuperPlane | GKE cluster, Cloud NAT (private nodes), Cloud SQL, cert-manager, Helm release | Always |
| Blob storage | GCS bucket, `superplane-app` service account, Workload Identity binding | `enable_gcs_blob_storage` |
| Runner network | Runner subnetwork, deny-all ingress firewall rule, `superplane-fleet-mgr` service account | `enable_runners` |
| Fleet Manager infrastructure | Service account, Workload Identity binding, and setup metadata | `enable_runners` |

A new install sets both flags to `true` in `terraform.tfvars` before the first
apply. The variable defaults stay `false`. An existing install that omitted
these settings keeps local blob files.

## Prerequisites

- [Terraform](https://developer.hashicorp.com/terraform/install) 1.5.0 or later.
- [gcloud CLI](https://cloud.google.com/sdk/docs/install) and `kubectl`.
- A GCP project with billing enabled.
- An account with the Owner role on the project, or with permission to create
  GKE, Cloud SQL, Compute Engine, GCS, and IAM resources.
- A DNS zone where you can create an A record.
- For runners: [Packer](https://developer.hashicorp.com/packer/install) 1.16.1
  or later, `curl`, and `jq`.

Run all `terraform` commands in this directory (`release/terraform/gke`).

## 1. Prepare the project

### Authenticate

```bash
export PROJECT_ID=my-gcp-project

gcloud auth login
gcloud auth application-default login
gcloud config set project "$PROJECT_ID"
```

### Enable the APIs

Terraform enables the GKE, Cloud SQL, Service Networking, and IAM Credentials
APIs. Enable Compute Engine first, because you need it for the static IP:

```bash
gcloud services enable compute.googleapis.com iam.googleapis.com --project="$PROJECT_ID"
```

### Create a static IP address

```bash
gcloud compute addresses create superplane-ip --global --ip-version=IPV4
gcloud compute addresses describe superplane-ip --global --format='get(address)'
```

### Configure DNS

Create an A record for your SuperPlane domain, for example
`superplane.example.com`. Point it to the static IP address.

Confirm public DNS before you apply. Let's Encrypt uses public resolvers,
not your local resolver:

```bash
dig +short superplane.example.com @8.8.8.8
```

The answer must be the static IP address. If your local resolver still
fails, flush it or use `8.8.8.8`. The apex nameservers must match the
registrar. Mixed nameservers (for example DigitalOcean and Squarespace)
make some resolvers return `NXDOMAIN`.

## 2. Install SuperPlane

### Configure Terraform

```bash
cp terraform.tfvars.example terraform.tfvars
```

Set the required values in `terraform.tfvars`:

```hcl
project_id        = "my-gcp-project"
domain_name       = "superplane.example.com"
static_ip_name    = "superplane-ip"
letsencrypt_email = "admin@example.com"

# Confirm the release exists before you apply.
superplane_image_tag     = ""
superplane_chart_version = "1.0.0"

enable_gcs_blob_storage = true
enable_runners          = true
```

Set both flags in this file. Do not rely on the variable defaults. Those
defaults stay `false` so an existing install keeps its current blob storage.

The default node type is `e2-standard-2`. Keep that type, or use a larger
type. The API, the runner API, and the runner workers share one log disk.
They must run on one node. `e2-medium` does not have enough CPU for that
node. The Helm release then fails at the timeout.

Select a published stable or release-candidate version. Confirm the image and
chart before you apply:

```bash
export SUPERPLANE_VERSION=v1.0.0

docker manifest inspect ghcr.io/superplanehq/superplane:${SUPERPLANE_VERSION}
helm show chart oci://ghcr.io/superplanehq/superplane-chart --version "${SUPERPLANE_VERSION#v}"
```

Leave `superplane_image_tag` empty. The chart appVersion selects the matching
versioned image. Legacy self-hosted tags remain valid as explicit overrides.

If the chart is not published yet, set `superplane_chart_path` to
`../../superplane-helm-chart/helm` and leave `superplane_chart_version`
empty. Also set `superplane_image_tag`. The checked-in chart appVersion
is a placeholder, not a published image.

With `enable_gcs_blob_storage = false`, SuperPlane stores blobs on pod
volumes. Each pod has its own volume, and the data is lost when a pod restarts.
Use this setting only for a short evaluation.

### Apply

```bash
terraform init
terraform apply
```

The apply takes approximately 20 minutes. It creates the cluster, the database,
the SuperPlane release, the blob bucket, and the runner infrastructure.

The runner infrastructure includes:

- The runner subnetwork (`172.20.0.0/24` by default). Runner VMs have no
  external IP address. Cloud NAT gives them outbound internet access.
- A firewall rule that blocks all inbound traffic to runner VMs.
- The Fleet Manager service account, with `roles/compute.instanceAdmin.v1` on
  the project.
- A runner API Deployment. Runners connect to
  `https://superplane.example.com/runner/v1`.
- Fleet Manager setup metadata, including the configured runner images.

If `172.20.0.0/24` overlaps a range in your network, set `runner_subnet_cidr`
before you apply.

### Verify

```bash
gcloud container clusters get-credentials superplane --zone=us-central1-a --project="$PROJECT_ID"

kubectl get pods -n superplane
kubectl get certificate -n superplane
```

The apply waits until `https://superplane.example.com/health` responds.
The GCE controller skips the certificate check on the first sync.
Terraform syncs the Ingress twice after Helm, then waits for the
certificate. This wait can take 15 minutes.

All pods must be `Running`. The namespace must have one Certificate, and
that Certificate must be `Ready`.

Open `https://superplane.example.com`. Create the owner account. The owner
account is the installation admin.

## 3. Install Fleet Manager

Do these steps after the installation in step 2 works.

### 3.1 Build the runner image

Build the image with Packer. Follow
[release/runner/packer/gce/README.md](../../runner/packer/gce/README.md).
For the default fleet, build the amd64 image:

```bash
cd ../../..   # repository root
packer init release/runner/packer/gce/runner.pkr.hcl
packer build \
  -var project_id="$PROJECT_ID" \
  -var architecture=amd64 \
  -var machine_type=e2-standard-4 \
  release/runner/packer/gce/runner.pkr.hcl
cd release/terraform/gke
```

The image goes into the `superplane-runner-amd64` image family. Set the
immutable image resource in `terraform.tfvars`, and apply the stack again:

```hcl
runner_image_amd64 = "projects/PROJECT/global/images/IMAGE_NAME"
runner_image_arm64 = "projects/PROJECT/global/images/ARM_IMAGE_NAME"
```

The default build uses a public IP address and SSH on TCP 22. If the
network blocks inbound SSH, build after the apply. Use Identity-Aware Proxy
and the runner subnetwork:

```bash
cd ../../..   # repository root
packer build \
  -var project_id="$PROJECT_ID" \
  -var architecture=amd64 \
  -var machine_type=e2-standard-4 \
  -var use_iap=true \
  -var subnetwork=superplane-runners \
  release/runner/packer/gce/runner.pkr.hcl
cd release/terraform/gke
```

Allow inbound TCP 22 from `35.235.240.0/20` for IAP.

### 3.2 Install the standalone Fleet Manager

Open **Set up runner fleets** during owner setup. Terraform passes the public
SuperPlane URL and these values to SuperPlane. The wizard preloads them into
the generated configuration:

```bash
terraform output runner_project
terraform output runner_zones
terraform output runner_subnetwork
terraform output runner_network_tag
terraform output runner_image_amd64
terraform output runner_image_arm64
terraform output fleet_manager_service_account_email
```

Select only fleets whose image output is set. SuperPlane creates a personal
token when you generate the install command. Run the displayed Helm commands.
The standalone chart stores the complete configuration in
`fleet-manager-config`.

The chart startup Job creates or updates the fleets after the configuration
Secret exists. For complete install, upgrade, token rotation, fleet removal, and
rollback instructions, see
[the standalone chart guide](../../fleet-manager-chart/README.md).

Confirm that Fleet Manager runs:

```bash
kubectl get pods -n superplane \
  -l app.kubernetes.io/instance=fleet-manager,app.kubernetes.io/name=fleet-manager
kubectl logs -n superplane deployment/fleet-manager-fleet-manager
```

The log must not show configuration or permission errors.

To change fleets, edit the standalone chart values and run `helm upgrade`.

### 3.3 Run a task

1. Create a canvas.
2. Add a Manual Run trigger.
3. Add a Run Bash node. Set **Machine type** to `e1-large-amd64`. Use this
   script:

   ```bash
   uname -a
   ```

4. Run the canvas.

Watch the runner VM:

```bash
gcloud compute instances list \
  --filter="labels.superplane_fleet_id=e1-large-amd64" \
  --format="table(name,zone.basename(),status,creationTimestamp)"
```

Read the boot log of a runner VM:

```bash
gcloud compute instances get-serial-port-output <instance-name> --zone=<zone>
```

The task must complete, and the task log must show the `uname` output. The VM
must stop and Fleet Manager must delete it after the task.

## Configuration reference

| Variable | Default | Description |
| --- | --- | --- |
| `superplane_image_registry` | `ghcr.io/superplanehq` | Registry that hosts the SuperPlane image. |
| `fleet_manager_image_registry` | `ghcr.io/superplanehq` | Registry that hosts the standalone Fleet Manager image. Set this for private GCP registries so nodes keep pull access. |
| `superplane_image_tag` | empty | Image tag override. Empty uses the chart appVersion. A local chart requires an explicit tag. Versioned, legacy self-hosted, and local tags are supported. |
| `enable_gcs_blob_storage` | `false` | Store blobs in GCS. Set this to `true` in `terraform.tfvars` for a new install. |
| `blob_bucket_name` | `<project_id>-superplane-blobs` | Name of the blob bucket. |
| `blob_bucket_force_destroy` | `false` | Delete the bucket objects on destroy. |
| `superplane_chart_path` | empty | Path to a local chart. Empty installs the published chart. |
| `superplane_chart_version` | empty | Published chart version, for example `1.0.0` or `1.0.0-rc1`. Required when `superplane_chart_path` is empty. |
| `enable_runners` | `false` | Enable the runner API and create the runner network. Set this to `true` in `terraform.tfvars` for a new install. |
| `runner_subnet_cidr` | `172.20.0.0/24` | Range of the runner subnetwork. |
| `runner_zones` | `[zone]` | Zones for runner VMs. |

For an arm64 fleet, build the arm64 image and use zones that have the machine
type. For example, T2A machines are not available in all zones.

## Troubleshooting

**The UI loads HTML but scripts fail CORS.** Use a versioned release image or
a legacy self-hosted image. Apply again, then hard-refresh the browser.

**The API or the runner workers stay `Pending`.** The node does not have
enough CPU. Those pods share one log disk, so they must run on one node.
Set `machine_type` to `e2-standard-2` or a larger type. A change of
`machine_type` replaces the cluster.

**Pods stay in `ImagePullBackOff` with HTTP 403.** You pointed
`superplane_image_registry` at `gcr.io/<project>` or Artifact Registry.
Terraform grants
`roles/artifactregistry.reader` to the default Compute Engine service
account. Wait one minute, then delete the failing pods.

**The certificate is not Ready.** Public DNS must match the static IP:

```bash
dig +short superplane.example.com @8.8.8.8
```

One Certificate must exist in the namespace:
`kubectl get certificate -n superplane`. Let's Encrypt needs HTTP on
port 80. Wait up to 15 minutes.

**Fleet Manager restarts.** Read the log with
`kubectl logs -n superplane deployment/fleet-manager-fleet-manager --previous`.
A configuration error names the field that is not correct.

**Fleet Manager cannot create VMs.** Make sure that the Kubernetes service
account has the Workload Identity annotation:

```bash
kubectl get serviceaccount superplane-fleet-manager -n superplane -o yaml
```

**The VM starts, but the runner does not register.** Read the serial port
output. The bootstrap script must download the runner release and connect to
`https://<domain>/runner/v1`. Make sure that Cloud NAT exists for the runner
subnetwork and that the domain has a valid certificate.

**VM creation fails with `ZONE_RESOURCE_POOL_EXHAUSTED`.** The zone has no
capacity for the machine type. Add more zones to `runner_zones`. Fleet Manager
tries the next zone.

**Task logs or file downloads fail.** SuperPlane signs GCS URLs with the IAM
signBlob API. Make sure that the IAM Credentials API is enabled and that the
`superplane` Kubernetes service account has the Workload Identity annotation.

## Security

- Runner VMs run task code. A task can run any command. Use a dedicated GCP
  project for SuperPlane.
- Runner VMs have no service account, so tasks get no GCP credentials. Runner
  VMs have no external IP address. The firewall rule blocks all inbound
  traffic. Project SSH keys are blocked.
- Fleet Manager has `roles/compute.instanceAdmin.v1` on the project. It can
  change all VMs in the project. This is another reason to use a dedicated
  project.
- The Terraform state contains the database password and the session and
  encryption keys. Store the state in an
  encrypted remote backend with restricted access.

## Destroy

Stop the runners first. Uninstall the standalone Fleet Manager chart. Then
delete runner VMs that remain:

```bash
# Read the Fleet Manager ID from the installed standalone chart.
FLEET_MANAGER_ID=$(helm get values fleet-manager -n superplane --all --output json | jq -er '.manager.id')
helm uninstall fleet-manager -n superplane

gcloud compute instances list --filter="labels.superplane_fleet_manager_id=${FLEET_MANAGER_ID}" \
  --format="value(name,zone.basename())" |
  while read -r name zone; do gcloud compute instances delete "$name" --zone="$zone" --quiet; done
```

Deletion protection is enabled for the GKE cluster and the Cloud SQL instance.
Set both flags to false in `terraform.tfvars`. To delete the blob bucket with
its objects, also set `blob_bucket_force_destroy`:

```hcl
gke_deletion_protection   = false
sql_deletion_protection   = false
blob_bucket_force_destroy = true
```

Apply the flags, then destroy:

```bash
terraform apply
terraform destroy
```

Terraform does not manage the static IP address or the runner images. Delete
them after the destroy:

```bash
gcloud compute addresses delete superplane-ip --global
gcloud compute images list --filter="family~^superplane-runner-" --format="value(name)" |
  xargs -r -n1 gcloud compute images delete --quiet
```
