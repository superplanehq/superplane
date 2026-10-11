# -----------------------------------------------------------------------------
# GCS Blob Storage
# -----------------------------------------------------------------------------

locals {
  gcs_blob_storage_enabled = var.enable_gcs_blob_storage
  blob_bucket_name         = var.blob_bucket_name != "" ? var.blob_bucket_name : "${var.project_id}-superplane-blobs"
  app_service_account_name = "superplane"
  # Service account IDs allow at most 30 characters.
  service_account_prefix = substr(var.cluster_name, 0, 20)
}

# SuperPlane signs GCS download URLs with the IAM signBlob API.
resource "google_project_service" "iamcredentials" {
  count              = local.gcs_blob_storage_enabled ? 1 : 0
  service            = "iamcredentials.googleapis.com"
  disable_on_destroy = false
}

resource "google_storage_bucket" "blobs" {
  count    = local.gcs_blob_storage_enabled ? 1 : 0
  name     = local.blob_bucket_name
  location = upper(var.region)

  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"
  force_destroy               = var.blob_bucket_force_destroy
}

resource "google_service_account" "app" {
  count        = local.gcs_blob_storage_enabled ? 1 : 0
  account_id   = "${local.service_account_prefix}-app"
  display_name = "SuperPlane application"
}

resource "google_storage_bucket_iam_member" "app_blobs" {
  count  = local.gcs_blob_storage_enabled ? 1 : 0
  bucket = google_storage_bucket.blobs[0].name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.app[0].email}"
}

resource "google_service_account_iam_member" "app_sign_blob" {
  count              = local.gcs_blob_storage_enabled ? 1 : 0
  service_account_id = google_service_account.app[0].name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:${google_service_account.app[0].email}"
}

resource "google_service_account_iam_member" "app_workload_identity" {
  count              = local.gcs_blob_storage_enabled ? 1 : 0
  service_account_id = google_service_account.app[0].name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[${var.superplane_namespace}/${local.app_service_account_name}]"

  # The workload identity pool exists only after the cluster exists.
  depends_on = [google_container_cluster.superplane]
}
