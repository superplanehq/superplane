# -----------------------------------------------------------------------------
# EKS Cluster Outputs
# -----------------------------------------------------------------------------

output "cluster_name" {
  description = "Name of the EKS cluster"
  value       = aws_eks_cluster.superplane.name
}

output "cluster_endpoint" {
  description = "EKS cluster endpoint"
  value       = aws_eks_cluster.superplane.endpoint
  sensitive   = true
}

output "cluster_ca_certificate" {
  description = "EKS cluster CA certificate (base64 encoded)"
  value       = aws_eks_cluster.superplane.certificate_authority[0].data
  sensitive   = true
}

# -----------------------------------------------------------------------------
# Database Outputs
# -----------------------------------------------------------------------------

output "database_endpoint" {
  description = "RDS instance endpoint"
  value       = aws_db_instance.superplane.endpoint
}

output "database_address" {
  description = "RDS instance address (hostname only)"
  value       = aws_db_instance.superplane.address
}

# -----------------------------------------------------------------------------
# VPC Outputs
# -----------------------------------------------------------------------------

output "vpc_id" {
  description = "ID of the VPC"
  value       = aws_vpc.superplane.id
}

output "runner_private_subnet_ids" {
  description = "Private subnet IDs for runner VMs"
  value       = var.enable_runners ? aws_subnet.private[*].id : []
}

output "runner_security_group_id" {
  description = "Outbound-only security group ID for runner VMs"
  value       = var.enable_runners ? aws_security_group.runners[0].id : null
}

output "fleet_manager_role_arn" {
  description = "IAM role ARN for the standalone Fleet Manager service account"
  value       = var.enable_runners ? aws_iam_role.fleet_manager[0].arn : null
}

output "runner_instance_profile_name" {
  description = "IAM instance profile name for runner VMs"
  value       = var.enable_runners ? aws_iam_instance_profile.runner[0].name : null
}

output "runner_region" {
  description = "AWS region for runner VMs"
  value       = var.enable_runners ? var.region : null
}

output "runner_image_amd64" {
  description = "Packer-built AMI configured for linux/amd64 runner VMs"
  value       = var.enable_runners ? var.runner_image_amd64 : null
}

output "runner_image_arm64" {
  description = "Packer-built AMI configured for linux/arm64 runner VMs"
  value       = var.enable_runners ? var.runner_image_arm64 : null
}

# -----------------------------------------------------------------------------
# Runner Storage Outputs
# -----------------------------------------------------------------------------

output "runner_active_logs_efs_id" {
  description = "ID of the EFS file system used for runner active logs"
  value       = aws_efs_file_system.runner_active_logs.id
}

output "runner_active_logs_efs_access_point_id" {
  description = "ID of the EFS access point used for runner active logs"
  value       = aws_efs_access_point.runner_active_logs.id
}

output "blob_storage_bucket" {
  description = "S3 bucket used for shared SuperPlane blobs"
  value       = aws_s3_bucket.blobs.id
}

# -----------------------------------------------------------------------------
# SuperPlane Outputs
# -----------------------------------------------------------------------------

output "superplane_namespace" {
  description = "Kubernetes namespace where SuperPlane is deployed"
  value       = var.superplane_namespace
}

output "superplane_url" {
  description = "URL to access SuperPlane"
  value       = "https://${var.domain_name}"
}

# -----------------------------------------------------------------------------
# kubectl Configuration Command
# -----------------------------------------------------------------------------

output "kubectl_config_command" {
  description = "Command to configure kubectl to connect to the cluster"
  value       = "aws eks update-kubeconfig --region ${var.region} --name ${var.cluster_name}"
}

# -----------------------------------------------------------------------------
# Load Balancer Output
# -----------------------------------------------------------------------------

output "load_balancer_hostname_command" {
  description = "Command to get the NLB hostname for DNS configuration"
  value       = "kubectl get svc -n traefik traefik -o jsonpath='{.status.loadBalancer.ingress[0].hostname}'"
}
