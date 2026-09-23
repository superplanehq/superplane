# -----------------------------------------------------------------------------
# GKE Cluster Outputs
# -----------------------------------------------------------------------------

output "cluster_name" {
  description = "Name of the GKE cluster"
  value       = google_container_cluster.superplane.name
}

output "cluster_endpoint" {
  description = "GKE cluster endpoint"
  value       = google_container_cluster.superplane.endpoint
  sensitive   = true
}

output "cluster_ca_certificate" {
  description = "GKE cluster CA certificate (base64 encoded)"
  value       = google_container_cluster.superplane.master_auth[0].cluster_ca_certificate
  sensitive   = true
}

# -----------------------------------------------------------------------------
# Database Outputs
# -----------------------------------------------------------------------------

output "database_instance_name" {
  description = "Name of the Cloud SQL instance"
  value       = google_sql_database_instance.superplane.name
}

output "database_private_ip" {
  description = "Private IP address of the Cloud SQL instance"
  value       = google_sql_database_instance.superplane.private_ip_address
}

output "database_connection_name" {
  description = "Connection name for Cloud SQL instance"
  value       = google_sql_database_instance.superplane.connection_name
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
  value       = "gcloud container clusters get-credentials ${var.cluster_name} --zone=${var.zone} --project=${var.project_id}"
}
