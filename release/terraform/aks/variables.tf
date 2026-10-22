# -----------------------------------------------------------------------------
# Required Variables
# -----------------------------------------------------------------------------

variable "subscription_id" {
  description = "Azure subscription ID"
  type        = string
}

variable "domain_name" {
  description = "Domain name for SuperPlane (e.g., superplane.example.com)"
  type        = string
}

variable "letsencrypt_email" {
  description = "Email address for Let's Encrypt certificate notifications"
  type        = string
}

# -----------------------------------------------------------------------------
# Optional Variables - Azure
# -----------------------------------------------------------------------------

variable "location" {
  description = "Azure region for resources"
  type        = string
  default     = "eastus"
}

variable "resource_group_name" {
  description = "Name of the application resource group"
  type        = string
  default     = "superplane"
}

# -----------------------------------------------------------------------------
# Optional Variables - AKS Cluster
# -----------------------------------------------------------------------------

variable "cluster_name" {
  description = "Name of the AKS cluster"
  type        = string
  default     = "superplane"
}

variable "kubernetes_version" {
  description = "Kubernetes version for the AKS cluster. Empty uses the Azure default."
  type        = string
  default     = ""
}

variable "node_count" {
  description = "Number of nodes in the AKS system pool"
  type        = number
  default     = 1
}

variable "node_vm_size" {
  description = "VM size for AKS nodes. Default fits a 4-vCPU regional quota."
  type        = string
  default     = "Standard_D2s_v4"
}

# -----------------------------------------------------------------------------
# Optional Variables - PostgreSQL
# -----------------------------------------------------------------------------

variable "create_postgresql_flexible_server" {
  description = "Create Azure Database for PostgreSQL Flexible Server. Set false when the subscription has no Flexible Server SKUs in the region (az postgres flexible-server list-skus --location eastus returns []). Helm then runs Postgres in the cluster."
  type        = bool
  default     = true
}

variable "db_server_name" {
  description = "Name of the PostgreSQL Flexible Server"
  type        = string
  default     = "superplane-db"
}

variable "db_engine_version" {
  description = "PostgreSQL major version for Flexible Server"
  type        = string
  default     = "16"
}

variable "db_sku_name" {
  description = "SKU for PostgreSQL Flexible Server. Default is burstable for small subscriptions."
  type        = string
  default     = "B_Standard_B2s"
}

variable "db_storage_mb" {
  description = "Allocated storage for PostgreSQL in MB"
  type        = number
  default     = 32768
}

variable "db_name" {
  description = "Name of the database to create"
  type        = string
  default     = "superplane"
}

variable "db_username" {
  description = "Administrator username for PostgreSQL. Azure does not allow postgres."
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
  description = "SuperPlane image tag override. Empty uses the chart appVersion. Versioned, legacy selfhosted, and local tags are supported."
  type        = string
  default     = ""

  validation {
    condition     = var.superplane_image_tag == "" || can(regex("^v[0-9]+\\.[0-9]+\\.[0-9]+(-[0-9A-Za-z.-]+)?$", var.superplane_image_tag)) || strcontains(var.superplane_image_tag, "selfhosted") || startswith(var.superplane_image_tag, "local-")
    error_message = "Use a vMAJOR.MINOR.PATCH release tag, a legacy selfhosted tag, a local- tag, or leave the value empty."
  }
}

variable "image_registry" {
  description = "Container registry for the SuperPlane image. Use an ACR login server for local builds."
  type        = string
  default     = "ghcr.io/superplanehq"
}

variable "image_name" {
  description = "SuperPlane image repository name"
  type        = string
  default     = "superplane"
}

variable "helm_chart_repository" {
  description = "OCI or HTTP Helm repository. Ignored when helm_chart_path is set."
  type        = string
  default     = "oci://ghcr.io/superplanehq"
}

variable "helm_chart_name" {
  description = "Helm chart name when helm_chart_path is empty"
  type        = string
  default     = "superplane-chart"
}

variable "helm_chart_path" {
  description = "Local SuperPlane Helm chart path for chart development. Empty installs the published chart."
  type        = string
  default     = ""
}

variable "superplane_chart_version" {
  description = "Published SuperPlane Helm chart version with a matching versioned image. Ignored when helm_chart_path is set."
  type        = string
  default     = "1.0.0-rc3"
}

variable "fleet_manager_version" {
  description = "Fleet Manager release the owner-setup wizard installs. Use v0.1.3 or later with this chart. The Fleet Manager binary syncs fleets."
  type        = string
  default     = "v0.1.3"

  validation {
    condition     = can(regex("^v\\d+\\.\\d+\\.\\d+$", var.fleet_manager_version))
    error_message = "Set fleet_manager_version to vMAJOR.MINOR.PATCH, for example v0.1.3."
  }
}

# Latest stable chart on https://charts.jetstack.io as of 2026-10-14.
# cert-manager 1.21 supports Kubernetes 1.33-1.36.
# aks.tf leaves kubernetes_version empty, so AKS uses the N-1 minor version.
# The October 2026 AKS calendar includes 1.34, 1.35, and 1.36.
# GKE and EKS do not pin this chart. EKS pins the Traefik chart.
variable "cert_manager_chart_version" {
  description = "cert-manager Helm chart version"
  type        = string
  default     = "v1.21.2"
}

# Latest chart on https://kubernetes.github.io/ingress-nginx as of 2026-10-14.
# Controller 1.15.1 was published on 2026-03-19. kubeVersion is >=1.21.0-0.
# That release tested Kubernetes 1.35. The project is retired. See README.md.
variable "ingress_nginx_chart_version" {
  description = "ingress-nginx Helm chart version"
  type        = string
  default     = "4.15.1"
}

variable "container_registry_id" {
  description = "Azure Container Registry resource ID. Empty skips the AcrPull role assignment."
  type        = string
  default     = ""
}

variable "runner_zones" {
  description = "Availability zones that Fleet Manager can use for runner VMs."
  type        = list(string)
  default     = ["1"]
}

variable "runner_image_amd64" {
  description = "linux/amd64 image version ID. Use a /CommunityGalleries/... ID published by SuperPlane, or a customer gallery version ID. The image version is independent of the runner version."
  type        = string
  default     = "/CommunityGalleries/Superplane-ba852f02-dd01-43eb-b8c4-67a57fdd8af8/Images/superplane-runner-amd64/Versions/1.14623.0"
}

variable "runner_image_arm64" {
  description = "linux/arm64 image version ID. Use a /CommunityGalleries/... ID published by SuperPlane, or a customer gallery version ID. The image version is independent of the runner version."
  type        = string
  default     = "/CommunityGalleries/Superplane-ba852f02-dd01-43eb-b8c4-67a57fdd8af8/Images/superplane-runner-arm64/Versions/1.14623.0"
}

# -----------------------------------------------------------------------------
# Optional Variables - Network
# -----------------------------------------------------------------------------

variable "vnet_cidr" {
  description = "CIDR block for the virtual network"
  type        = string
  default     = "10.0.0.0/16"
}

variable "aks_subnet_cidr" {
  description = "CIDR block for AKS nodes"
  type        = string
  default     = "10.0.0.0/20"
}

variable "postgres_subnet_cidr" {
  description = "CIDR block for PostgreSQL Flexible Server"
  type        = string
  default     = "10.0.16.0/24"
}

variable "runner_subnet_cidr" {
  description = "CIDR block for runner virtual machines"
  type        = string
  default     = "10.0.17.0/24"
}

variable "aks_service_cidr" {
  description = "Kubernetes service CIDR. Must not overlap the virtual network."
  type        = string
  default     = "172.16.0.0/16"
}

variable "aks_dns_service_ip" {
  description = "kube-dns address. Must sit inside aks_service_cidr."
  type        = string
  default     = "172.16.0.10"
}
