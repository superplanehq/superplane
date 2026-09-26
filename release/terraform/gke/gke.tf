# -----------------------------------------------------------------------------
# GKE Cluster
# -----------------------------------------------------------------------------

resource "google_container_cluster" "superplane" {
  name     = var.cluster_name
  location = var.zone

  deletion_protection = var.gke_deletion_protection

  initial_node_count       = var.node_count
  remove_default_node_pool = false

  min_master_version = var.cluster_version
  release_channel {
    channel = "REGULAR"
  }

  network = var.network

  # Enable Workload Identity
  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }

  private_cluster_config {
    enable_private_nodes    = var.enable_private_nodes
    enable_private_endpoint = false
    master_ipv4_cidr_block  = var.enable_private_nodes ? var.master_ipv4_cidr_block : null
  }

  dynamic "master_authorized_networks_config" {
    for_each = length(var.master_authorized_cidr_blocks) > 0 ? [1] : []
    content {
      dynamic "cidr_blocks" {
        for_each = var.master_authorized_cidr_blocks
        content {
          cidr_block   = cidr_blocks.value.cidr_block
          display_name = cidr_blocks.value.display_name
        }
      }
    }
  }

  logging_config {
    enable_components = ["SYSTEM_COMPONENTS", "WORKLOADS"]
  }

  monitoring_config {
    enable_components = ["SYSTEM_COMPONENTS"]
  }

  node_config {
    machine_type = var.machine_type

    oauth_scopes = [
      "https://www.googleapis.com/auth/devstorage.read_only",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring",
    ]

    workload_metadata_config {
      mode = "GKE_METADATA"
    }
  }

  depends_on = [
    google_project_service.container
  ]
}

# -----------------------------------------------------------------------------
# Cloud NAT for Private Nodes
# -----------------------------------------------------------------------------

# Cloud Router for NAT (required for private nodes to access internet)
resource "google_compute_router" "nat_router" {
  count   = var.enable_private_nodes ? 1 : 0
  name    = "${var.cluster_name}-nat-router"
  region  = var.region
  network = "projects/${var.project_id}/global/networks/${var.network}"
}

# Cloud NAT (allows private nodes to pull container images)
resource "google_compute_router_nat" "nat" {
  count                              = var.enable_private_nodes ? 1 : 0
  name                               = "${var.cluster_name}-nat"
  router                             = google_compute_router.nat_router[0].name
  region                             = var.region
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"

  log_config {
    enable = true
    filter = "ERRORS_ONLY"
  }
}
