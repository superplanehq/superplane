# -----------------------------------------------------------------------------
# Runners
# -----------------------------------------------------------------------------

locals {
  runners_enabled                    = var.enable_runners
  fleet_manager_enabled              = var.enable_runners && nonsensitive(var.installation_admin_token != "")
  fleet_manager_service_account_name = "superplane-fleet-manager"
  runner_network_tag                 = "superplane-runner"
  runner_zones                       = length(var.runner_zones) > 0 ? var.runner_zones : [var.zone]
  runner_fleet_images = {
    for fleet in var.runner_fleets : fleet.id => fleet.image != "" ? fleet.image : "projects/${var.project_id}/global/images/family/superplane-runner-${fleet.architecture}"
  }

  fleet_manager_config = local.fleet_manager_enabled ? yamlencode({
    id                     = var.cluster_name
    superplaneUrl          = "http://superplane-api.${var.superplane_namespace}.svc.cluster.local:8000"
    installationAdminToken = var.installation_admin_token
    runnerReleaseBaseUrl   = var.runner_release_base_url
    fleets = [for fleet in var.runner_fleets : {
      id           = fleet.id
      provider     = "gcp"
      warmCapacity = fleet.warm_capacity
      maxCapacity  = fleet.max_capacity
      gcp = {
        projectId    = var.project_id
        zones        = local.runner_zones
        machineType  = fleet.machine_type
        image        = local.runner_fleet_images[fleet.id]
        architecture = fleet.architecture
        subnetwork   = google_compute_subnetwork.runners[0].id
        networkTags  = [local.runner_network_tag]
        diskSizeGb   = fleet.disk_size_gb
      }
    }]
  }) : ""
}

resource "google_compute_subnetwork" "runners" {
  count                    = local.runners_enabled ? 1 : 0
  name                     = "${var.cluster_name}-runners"
  region                   = var.region
  network                  = var.network
  ip_cidr_range            = var.runner_subnet_cidr
  private_ip_google_access = true

  lifecycle {
    precondition {
      condition     = var.enable_gcs_blob_storage
      error_message = "Runners need enable_gcs_blob_storage = true. The API and worker pods do not share filesystem blob storage."
    }
  }
}

# Runner VMs have no external IP. With private nodes, the cluster NAT already
# covers all subnetworks in the region.
resource "google_compute_router" "runners" {
  count   = local.runners_enabled && !var.enable_private_nodes ? 1 : 0
  name    = "${var.cluster_name}-runners-router"
  region  = var.region
  network = var.network
}

resource "google_compute_router_nat" "runners" {
  count                              = local.runners_enabled && !var.enable_private_nodes ? 1 : 0
  name                               = "${var.cluster_name}-runners-nat"
  router                             = google_compute_router.runners[0].name
  region                             = var.region
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "LIST_OF_SUBNETWORKS"

  subnetwork {
    name                    = google_compute_subnetwork.runners[0].id
    source_ip_ranges_to_nat = ["ALL_IP_RANGES"]
  }

  log_config {
    enable = true
    filter = "ERRORS_ONLY"
  }
}

# Runners only make outbound connections. This rule has priority over the
# default network rules that allow SSH, RDP, ICMP, and internal traffic.
resource "google_compute_firewall" "runners_deny_ingress" {
  count       = local.runners_enabled ? 1 : 0
  name        = "${var.cluster_name}-runners-deny-ingress"
  network     = var.network
  direction   = "INGRESS"
  priority    = 900
  target_tags = [local.runner_network_tag]

  source_ranges = ["0.0.0.0/0"]

  deny {
    protocol = "all"
  }
}

resource "google_service_account" "fleet_manager" {
  count        = local.runners_enabled ? 1 : 0
  account_id   = "${local.service_account_prefix}-fleet-mgr"
  display_name = "SuperPlane Fleet Manager"
}

resource "google_project_iam_member" "fleet_manager_instance_admin" {
  count   = local.runners_enabled ? 1 : 0
  project = var.project_id
  role    = "roles/compute.instanceAdmin.v1"
  member  = "serviceAccount:${google_service_account.fleet_manager[0].email}"
}

resource "google_service_account_iam_member" "fleet_manager_workload_identity" {
  count              = local.runners_enabled ? 1 : 0
  service_account_id = google_service_account.fleet_manager[0].name
  role               = "roles/iam.workloadIdentityUser"
  member             = "serviceAccount:${var.project_id}.svc.id.goog[${var.superplane_namespace}/${local.fleet_manager_service_account_name}]"

  # The workload identity pool exists only after the cluster exists.
  depends_on = [google_container_cluster.superplane]
}

resource "kubernetes_secret" "fleet_manager" {
  count = local.fleet_manager_enabled ? 1 : 0

  metadata {
    name      = "superplane-fleet-manager"
    namespace = kubernetes_namespace.superplane.metadata[0].name
  }

  data = {
    "fleet-manager.yaml" = local.fleet_manager_config
  }

  lifecycle {
    precondition {
      condition     = var.fleet_manager_image_tag != ""
      error_message = "Set fleet_manager_image_tag when installation_admin_token is set."
    }
  }
}
