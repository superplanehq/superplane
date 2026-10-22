# -----------------------------------------------------------------------------
# GCS Blob Storage
# -----------------------------------------------------------------------------

locals {
  blob_bucket_name         = var.blob_bucket_name != "" ? var.blob_bucket_name : "${var.project_id}-superplane-blobs"
  app_service_account_name = "superplane"
  # Service account IDs allow at most 30 characters.
  service_account_prefix = substr(var.cluster_name, 0, 20)
}

# SuperPlane signs GCS download URLs with the IAM signBlob API.
resource "google_project_service" "iamcredentials" {
  service            = "iamcredentials.googleapis.com"
  disable_on_destroy = false
}

resource "google_storage_bucket" "blobs" {
  name     = local.blob_bucket_name
  location = upper(var.region)

  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
  force_destroy               = var.blob_bucket_force_destroy

  # Finished runner logs are a signed GET. The browser fetches that URL
  # from the SuperPlane origin, so the bucket must allow that origin.
  cors {
    origin          = ["https://${var.domain_name}"]
    method          = ["GET", "HEAD"]
    response_header = ["Content-Type", "Content-Encoding", "Content-Length"]
    max_age_seconds = 3600
  }
}

resource "google_service_account" "app" {
  account_id   = "${local.service_account_prefix}-app"
  display_name = "SuperPlane application"
}

resource "google_storage_bucket_iam_member" "app_blobs" {
  bucket = google_storage_bucket.blobs.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.app.email}"
}

resource "google_service_account_iam_member" "app_sign_blob" {
  service_account_id = google_service_account.app.name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:${google_service_account.app.email}"
}

resource "google_service_account_iam_member" "app_workload_identity" {
  service_account_id = google_service_account.app.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[${var.superplane_namespace}/${local.app_service_account_name}]"

  # The workload identity pool exists only after the cluster exists.
  depends_on = [google_container_cluster.superplane]
}

# Installs that enabled GCS blob storage before it became mandatory keep
# these resources in state under the indexed address.
moved {
  from = google_project_service.iamcredentials[0]
  to   = google_project_service.iamcredentials
}

moved {
  from = google_storage_bucket.blobs[0]
  to   = google_storage_bucket.blobs
}

moved {
  from = google_service_account.app[0]
  to   = google_service_account.app
}

moved {
  from = google_storage_bucket_iam_member.app_blobs[0]
  to   = google_storage_bucket_iam_member.app_blobs
}

moved {
  from = google_service_account_iam_member.app_sign_blob[0]
  to   = google_service_account_iam_member.app_sign_blob
}

moved {
  from = google_service_account_iam_member.app_workload_identity[0]
  to   = google_service_account_iam_member.app_workload_identity
}
