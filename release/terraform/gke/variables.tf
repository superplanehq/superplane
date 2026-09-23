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
  default     = "1.34"
}

variable "node_count" {
  description = "Number of nodes in the GKE cluster"
  type        = number
  default     = 2
}

variable "machine_type" {
  description = "Machine type for GKE nodes"
  type        = string
  default     = "e2-medium"
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

variable "superplane_image_tag" {
  description = "SuperPlane image tag (e.g., stable, beta, v0.4)"
  type        = string
  default     = "stable"
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
  description = "Enable deletion protection on the GKE cluster"
  type        = bool
  default     = true
}

variable "sql_deletion_protection" {
  description = "Enable deletion protection on the Cloud SQL instance"
  type        = bool
  default     = true
}
