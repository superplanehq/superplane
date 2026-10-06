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
  default     = 2
}

variable "node_vm_size" {
  description = "VM size for AKS nodes"
  type        = string
  default     = "Standard_D4s_v5"
}

# -----------------------------------------------------------------------------
# Optional Variables - PostgreSQL
# -----------------------------------------------------------------------------

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
  description = "SKU for PostgreSQL Flexible Server"
  type        = string
  default     = "GP_Standard_D2s_v3"
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
  description = "SuperPlane image tag (e.g., stable, beta, v0.4)"
  type        = string
  default     = "stable"
}

variable "fleet_manager_image_tag" {
  description = "Fleet Manager image tag. Empty uses superplane_image_tag."
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
