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
  value       = "kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].hostname}'"
}

# -----------------------------------------------------------------------------
# Next Steps
# -----------------------------------------------------------------------------

output "next_steps" {
  description = "Instructions to complete the setup"
  value       = <<-EOT

    ============================================================
    NEXT STEPS - Configure DNS
    ============================================================

    1. Configure kubectl:
       ${local.kubectl_command}

    2. Get the Load Balancer hostname:
       kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].hostname}'

    3. Create a CNAME record in your DNS provider:
       - Type: CNAME
       - Name: ${var.domain_name}
       - Value: <hostname from step 2>

    4. Wait for DNS propagation and certificate issuance (~5-10 min)

    5. Access SuperPlane at: https://${var.domain_name}

    ============================================================
  EOT
}

locals {
  kubectl_command = "aws eks update-kubeconfig --region ${var.region} --name ${var.cluster_name}"
}
