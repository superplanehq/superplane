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
# Blob Storage Outputs
# -----------------------------------------------------------------------------

output "blob_bucket_name" {
  description = "GCS bucket for SuperPlane blobs"
  value       = local.gcs_blob_storage_enabled ? google_storage_bucket.blobs[0].name : null
}

output "app_service_account_email" {
  description = "GCP service account that SuperPlane pods use through Workload Identity"
  value       = local.gcs_blob_storage_enabled ? google_service_account.app[0].email : null
}

# -----------------------------------------------------------------------------
# Runner Outputs
# -----------------------------------------------------------------------------

output "runner_subnetwork" {
  description = "Subnetwork for runner VMs"
  value       = local.runners_enabled ? google_compute_subnetwork.runners[0].id : null
}

output "runner_network_tag" {
  description = "Network tag on runner VMs"
  value       = local.runners_enabled ? local.runner_network_tag : null
}

output "runner_image_families" {
  description = "Image families that Fleet Manager uses when a fleet has no image"
  value       = local.runners_enabled ? local.runner_fleet_images : null
}

output "fleet_manager_service_account_email" {
  description = "GCP service account that Fleet Manager uses through Workload Identity"
  value       = local.runners_enabled ? google_service_account.fleet_manager[0].email : null
}

output "fleet_manager_deployed" {
  description = "Whether Terraform deployed Fleet Manager"
  value       = local.fleet_manager_enabled
}

# -----------------------------------------------------------------------------
# kubectl Configuration Command
# -----------------------------------------------------------------------------

output "kubectl_config_command" {
  description = "Command to configure kubectl to connect to the cluster"
  value       = "gcloud container clusters get-credentials ${var.cluster_name} --zone=${var.zone} --project=${var.project_id}"
}
