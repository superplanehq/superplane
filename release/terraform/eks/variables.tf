# -----------------------------------------------------------------------------
# Required Variables
# -----------------------------------------------------------------------------

variable "domain_name" {
  description = "Domain name for SuperPlane (e.g., superplane.example.com)"
  type        = string
}

variable "letsencrypt_email" {
  description = "Email address for Let's Encrypt certificate notifications"
  type        = string
}


# -----------------------------------------------------------------------------
# Optional Variables - AWS
# -----------------------------------------------------------------------------

variable "region" {
  description = "AWS region for resources"
  type        = string
  default     = "us-east-1"
}

variable "availability_zones" {
  description = "Availability zones for the VPC subnets (minimum 2 required by AWS)"
  type        = list(string)
  default     = ["us-east-1a", "us-east-1b"]
}

# -----------------------------------------------------------------------------
# Optional Variables - EKS Cluster
# -----------------------------------------------------------------------------

variable "cluster_name" {
  description = "Name of the EKS cluster"
  type        = string
  default     = "superplane"
}

variable "cluster_version" {
  description = "Kubernetes version for the EKS cluster"
  type        = string
  default     = "1.31"
}

variable "node_count" {
  description = "Number of nodes in the EKS cluster"
  type        = number
  default     = 2
}

variable "instance_type" {
  description = "Instance type for EKS nodes"
  type        = string
  default     = "t3.medium"
}

# -----------------------------------------------------------------------------
# Optional Variables - RDS
# -----------------------------------------------------------------------------

variable "db_instance_identifier" {
  description = "Identifier for the RDS instance"
  type        = string
  default     = "superplane-db"
}

variable "db_engine_version" {
  description = "PostgreSQL version for RDS"
  type        = string
  default     = "17"
}

variable "db_instance_class" {
  description = "Instance class for RDS (e.g., db.t3.medium)"
  type        = string
  default     = "db.t3.medium"
}

variable "db_allocated_storage" {
  description = "Allocated storage for RDS in GB"
  type        = number
  default     = 20
}

variable "db_name" {
  description = "Name of the database to create"
  type        = string
  default     = "superplane"
}

variable "db_username" {
  description = "Master username for the database"
  type        = string
  default     = "postgres"
}

variable "db_password" {
  description = "Password for the database. If not provided, a random password will be generated."
  type        = string
  default     = ""
  sensitive   = true
}

variable "rds_deletion_protection" {
  description = "Enable deletion protection for RDS instance. Set to false to allow deletion during terraform destroy."
  type        = bool
  default     = true
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

variable "vpc_cidr" {
  description = "CIDR block for the VPC"
  type        = string
  default     = "10.0.0.0/16"
}

# -----------------------------------------------------------------------------
# Optional Variables - Security
# -----------------------------------------------------------------------------

variable "enable_secrets_encryption" {
  description = "Enable encryption of Kubernetes secrets at rest using KMS. Note: Cannot be enabled on existing clusters."
  type        = bool
  default     = false
}

variable "vpn_cidr_blocks" {
  description = "CIDR blocks allowed to access the EKS API server (VPN IPs). Set to restrict public access."
  type        = list(string)
  default     = []
}

variable "enable_public_access" {
  description = "Whether to enable public access to the EKS API server. Set to false to restrict to VPN only."
  type        = bool
  default     = true
}
