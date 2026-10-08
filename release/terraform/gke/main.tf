# -----------------------------------------------------------------------------
# Enable Required GCP APIs
# -----------------------------------------------------------------------------

resource "google_project_service" "container" {
  service            = "container.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "servicenetworking" {
  service            = "servicenetworking.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "sqladmin" {
  service            = "sqladmin.googleapis.com"
  disable_on_destroy = false
}

data "google_project" "current" {
  project_id = var.project_id
}

locals {
  # gcr.io/<project> and <region>-docker.pkg.dev/<project>/... live in this
  # project. The default GKE node service account cannot pull them until it
  # has Artifact Registry reader.
  pulls_from_gcp_registry = (
    startswith(var.superplane_image_registry, "gcr.io/${var.project_id}") ||
    can(regex("docker\\.pkg\\.dev/${var.project_id}(/|$)", var.superplane_image_registry)) ||
    startswith(var.fleet_manager_image_registry, "gcr.io/${var.project_id}") ||
    can(regex("docker\\.pkg\\.dev/${var.project_id}(/|$)", var.fleet_manager_image_registry))
  )
}

resource "google_project_service" "artifactregistry" {
  count              = local.pulls_from_gcp_registry ? 1 : 0
  service            = "artifactregistry.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_iam_member" "gke_nodes_artifact_registry" {
  count   = local.pulls_from_gcp_registry ? 1 : 0
  project = var.project_id
  role    = "roles/artifactregistry.reader"
  member  = "serviceAccount:${data.google_project.current.number}-compute@developer.gserviceaccount.com"

  depends_on = [google_project_service.artifactregistry]
}

# -----------------------------------------------------------------------------
# Random password for database (if not provided)
# -----------------------------------------------------------------------------

resource "random_password" "db_password" {
  count   = var.db_password == "" ? 1 : 0
  length  = 32
  special = false
}

locals {
  db_password = var.db_password != "" ? var.db_password : random_password.db_password[0].result
}

# -----------------------------------------------------------------------------
# Random secrets for SuperPlane
# -----------------------------------------------------------------------------

resource "random_password" "session_secret" {
  length  = 64
  special = false
}

resource "random_password" "jwt_secret" {
  length  = 64
  special = false
}

resource "random_password" "encryption_key" {
  length  = 32
  special = false
}

resource "tls_private_key" "oidc" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "time_static" "oidc_key" {}
