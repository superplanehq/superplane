# -----------------------------------------------------------------------------
# VPC Peering for Cloud SQL Private IP
# -----------------------------------------------------------------------------

resource "google_compute_global_address" "private_ip_range" {
  name          = "google-managed-services-${var.network}"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  network       = "projects/${var.project_id}/global/networks/${var.network}"

  depends_on = [
    google_project_service.servicenetworking
  ]
}

resource "google_service_networking_connection" "private_vpc_connection" {
  network                 = "projects/${var.project_id}/global/networks/${var.network}"
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_ip_range.name]

  depends_on = [
    google_project_service.servicenetworking,
    time_sleep.wait_for_instance_deletion
  ]
}

# Wait for Cloud SQL instance to be fully deleted before attempting to delete the connection
# Using time_sleep avoids circular dependency issues that null_resource polling creates
resource "time_sleep" "wait_for_instance_deletion" {
  depends_on = [
    google_sql_database_instance.superplane
  ]

  destroy_duration = "60s"
}

# -----------------------------------------------------------------------------
# Cloud SQL PostgreSQL Instance
# -----------------------------------------------------------------------------

resource "google_sql_database_instance" "superplane" {
  name             = var.db_instance_name
  database_version = var.db_version
  region           = var.region

  deletion_protection = var.sql_deletion_protection

  settings {
    tier    = var.db_tier
    edition = "ENTERPRISE"

    ip_configuration {
      ipv4_enabled    = false
      private_network = "projects/${var.project_id}/global/networks/${var.network}"
    }

    backup_configuration {
      enabled            = true
      start_time         = "03:00"
      binary_log_enabled = false

      backup_retention_settings {
        retained_backups = 7
      }
    }

    insights_config {
      query_insights_enabled  = true
      query_string_length     = 1024
      record_application_tags = true
      record_client_address   = true
    }

    database_flags {
      name  = "log_connections"
      value = "on"
    }

    database_flags {
      name  = "log_disconnections"
      value = "on"
    }

    database_flags {
      name  = "log_checkpoints"
      value = "on"
    }

    database_flags {
      name  = "log_lock_waits"
      value = "on"
    }

    database_flags {
      name  = "log_min_duration_statement"
      value = "1000"
    }
  }

  depends_on = [
    google_service_networking_connection.private_vpc_connection,
    google_project_service.sqladmin
  ]
}

# Set root password
resource "google_sql_user" "postgres" {
  name     = "postgres"
  instance = google_sql_database_instance.superplane.name
  password = local.db_password

  # Ensure database is deleted before user during destroy
  # Terraform destroys in reverse dependency order, so database will be destroyed first
  depends_on = [
    google_sql_database.superplane
  ]
}

# Create the SuperPlane database
resource "google_sql_database" "superplane" {
  name     = var.db_name
  instance = google_sql_database_instance.superplane.name
}
