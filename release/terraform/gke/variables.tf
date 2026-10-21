# -----------------------------------------------------------------------------
# Required Variables
# -----------------------------------------------------------------------------

variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "domain_name" {
  description = "Domain name for SuperPlane (e.g., superplane.example.com)"
  type        = string
}

variable "static_ip_name" {
  description = "Name of the pre-created global static IP address for the ingress"
  type        = string
}

variable "letsencrypt_email" {
  description = "Email address for Let's Encrypt certificate notifications"
  type        = string
}

# -----------------------------------------------------------------------------
# Optional Variables - GCP
# -----------------------------------------------------------------------------

variable "region" {
  description = "GCP region for resources"
  type        = string
  default     = "us-central1"
}

variable "zone" {
  description = "GCP zone for the GKE cluster"
  type        = string
  default     = "us-central1-a"
}

# -----------------------------------------------------------------------------
# Optional Variables - GKE Cluster
# -----------------------------------------------------------------------------

variable "cluster_name" {
  description = "Name of the GKE cluster"
  type        = string
  default     = "superplane"
}

variable "cluster_version" {
  description = "Kubernetes version for the GKE cluster"
  type        = string
  default     = "1.36"
}

variable "node_count" {
  description = "Number of nodes in the GKE cluster"
  type        = number
  default     = 2
}

variable "machine_type" {
  description = "Machine type for GKE nodes. Use e2-standard-2 or a larger type. The API and the runner pods share one log disk, so they run on one node. e2-medium does not have enough CPU for that node."
  type        = string
  default     = "e2-standard-2"
}

# -----------------------------------------------------------------------------
# Optional Variables - Cloud SQL
# -----------------------------------------------------------------------------

variable "db_instance_name" {
  description = "Name of the Cloud SQL instance"
  type        = string
  default     = "superplane-db"
}

variable "db_version" {
  description = "PostgreSQL version for Cloud SQL"
  type        = string
  default     = "POSTGRES_17"
}

variable "db_tier" {
  description = "Machine tier for Cloud SQL (e.g., db-custom-2-4096 for 2 vCPUs, 4GB RAM)"
  type        = string
  default     = "db-custom-2-4096"
}

variable "db_name" {
  description = "Name of the database to create"
  type        = string
  default     = "superplane"
}

variable "db_password" {
  description = "Password for the database. If not provided, a random password will be generated."
  type        = string
  default     = ""
  sensitive   = true
}

# -----------------------------------------------------------------------------
# Optional Variables - SuperPlane
# -----------------------------------------------------------------------------

variable "superplane_namespace" {
  description = "Kubernetes namespace for SuperPlane"
  type        = string
  default     = "superplane"
}

variable "superplane_image_registry" {
  description = "Registry that hosts the SuperPlane image, without the image name."
  type        = string
  default     = "ghcr.io/superplanehq"
}

variable "fleet_manager_image_registry" {
  description = "Registry that hosts the standalone Fleet Manager image. Terraform uses this value only to grant GKE node pull access for a private GCP registry."
  type        = string
  default     = "ghcr.io/superplanehq"
}

variable "superplane_image_tag" {
  description = "SuperPlane image tag. Empty uses the chart appVersion. Versioned releases use vMAJOR.MINOR.PATCH with an optional prerelease. Legacy selfhosted and local tags remain supported."
  type        = string
  default     = ""

  validation {
    condition     = var.superplane_image_tag == "" || can(regex("^v[0-9]+\\.[0-9]+\\.[0-9]+(-[0-9A-Za-z.-]+)?$", var.superplane_image_tag)) || strcontains(var.superplane_image_tag, "selfhosted") || startswith(var.superplane_image_tag, "local-")
    error_message = "Use a vMAJOR.MINOR.PATCH release tag, a legacy selfhosted tag, a local- tag, or leave the value empty."
  }
}

variable "github_enterprise_allowed_hosts" {
  description = "Optional comma-separated GitHub Enterprise Cloud host allowlist. Empty allows any host that SuperPlane accepts."
  type        = string
  default     = ""
}

variable "github_enterprise_extra_ca_certs_secret" {
  description = "Optional Kubernetes secret name with a CA bundle for GitHub Enterprise TLS. Empty skips the mount."
  type        = string
  default     = ""
}

variable "superplane_chart_path" {
  description = "Path to a local SuperPlane Helm chart, for example ../../superplane-helm-chart/helm. Leave empty to install the published chart from oci://ghcr.io/superplanehq."
  type        = string
  default     = ""
}

variable "superplane_chart_version" {
  description = "Published SuperPlane Helm chart version, for example 1.0.0-rc3. Required when superplane_chart_path is empty. Chart versions 0.30.0 and lower are the previous chart line."
  type        = string
  default     = "1.0.0-rc3"
}

# -----------------------------------------------------------------------------
# Optional Variables - Blob Storage
# -----------------------------------------------------------------------------

variable "enable_gcs_blob_storage" {
  description = "Store blobs (task logs and files) in a GCS bucket. SuperPlane uses Workload Identity to access the bucket. Runners require this."
  type        = bool
  default     = false
}

variable "blob_bucket_name" {
  description = "Name of the GCS bucket for blobs. Leave empty to use <project_id>-superplane-blobs."
  type        = string
  default     = ""
}

variable "blob_bucket_force_destroy" {
  description = "Delete all objects in the blob bucket on terraform destroy."
  type        = bool
  default     = false
}

# -----------------------------------------------------------------------------
# Optional Variables - Runners
# -----------------------------------------------------------------------------

variable "enable_runners" {
  description = "Enable the runner API and create the runner network and the Fleet Manager service account. Requires enable_gcs_blob_storage."
  type        = bool
  default     = false
}

variable "runner_subnet_cidr" {
  description = "CIDR range of the runner subnetwork. It must not overlap other subnetworks or GKE ranges in the network."
  type        = string
  default     = "172.20.0.0/24"
}

variable "runner_zones" {
  description = "Zones for runner VMs. Fleet Manager tries the next zone when a zone has no capacity. Leave empty to use var.zone."
  type        = list(string)
  default     = []
}

variable "runner_image_amd64" {
  description = "Packer-built GCP image resource for linux/amd64 runner VMs."
  type        = string
  default     = ""
}

variable "runner_image_arm64" {
  description = "Packer-built GCP image resource for linux/arm64 runner VMs."
  type        = string
  default     = ""
}

# -----------------------------------------------------------------------------
# Optional Variables - Network
# -----------------------------------------------------------------------------

variable "network" {
  description = "VPC network to use"
  type        = string
  default     = "default"
}

# -----------------------------------------------------------------------------
# Optional Variables - Security
# -----------------------------------------------------------------------------

variable "enable_private_nodes" {
  description = "Whether to enable private nodes (nodes without public IPs)"
  type        = bool
  default     = true
}

variable "master_authorized_cidr_blocks" {
  description = "CIDR blocks authorized to access the GKE master (VPN IPs)"
  type = list(object({
    cidr_block   = string
    display_name = string
  }))
  default = []
}

variable "master_ipv4_cidr_block" {
  description = "CIDR block for the GKE master (required when enable_private_nodes is true)"
  type        = string
  default     = "172.16.0.0/28"
}

variable "gke_deletion_protection" {
  description = "Enable deletion protection on the GKE cluster. Set this to false and apply before destroy."
  type        = bool
  default     = true
}

variable "sql_deletion_protection" {
  description = "Enable deletion protection on the Cloud SQL instance. Set this to false and apply before destroy."
  type        = bool
  default     = true
}
