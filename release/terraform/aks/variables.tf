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
  description = "SuperPlane image tag. Empty uses the chart's <git-sha>-selfhosted image. The plain <git-sha> and stable images load UI files from the hosted CDN and fail CORS on a customer domain. A local build can use a local- tag."
  type        = string
  default     = ""

  validation {
    condition     = var.superplane_image_tag == "" || strcontains(var.superplane_image_tag, "selfhosted") || startswith(var.superplane_image_tag, "local-")
    error_message = "Set superplane_image_tag to a tag that contains selfhosted, for example <git-sha>-selfhosted, or leave it empty. A local image can use a local- tag."
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

variable "fleet_manager_image_registry" {
  description = "Container registry for Fleet Manager. Empty uses image_registry."
  type        = string
  default     = ""
}

variable "fleet_manager_image_name" {
  description = "Fleet Manager image repository name"
  type        = string
  default     = "superplane-fleet-manager"
}

variable "fleet_manager_image_tag" {
  description = "Fleet Manager image tag. Empty uses superplane_image_tag. Set a tag when Fleet Manager is enabled and superplane_image_tag is empty."
  type        = string
  default     = ""
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
  description = "Path to the Helm chart in this repository. Empty uses helm_chart_repository."
  type        = string
  default     = "../../superplane-helm-chart/helm"
}

variable "container_registry_id" {
  description = "Azure Container Registry resource ID. Empty skips the AcrPull role assignment."
  type        = string
  default     = ""
}

variable "fleet_manager_config" {
  description = "Fleet Manager YAML or JSON body. Empty disables the Fleet Manager Deployment."
  type        = string
  default     = ""
  sensitive   = true
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
