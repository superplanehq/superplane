# -----------------------------------------------------------------------------
# Random password for database (if not provided)
# -----------------------------------------------------------------------------

resource "random_password" "db_password" {
  count   = var.db_password == "" ? 1 : 0
  length  = 32
  special = false
}

locals {
  db_password                  = var.db_password != "" ? var.db_password : random_password.db_password[0].result
  fleet_manager_image_tag      = var.fleet_manager_image_tag != "" ? var.fleet_manager_image_tag : var.superplane_image_tag
  fleet_manager_image_registry = var.fleet_manager_image_registry != "" ? var.fleet_manager_image_registry : var.image_registry
  fleet_manager_enabled        = trimspace(var.fleet_manager_config) != ""
  kubernetes_version           = trimspace(var.kubernetes_version) != "" ? var.kubernetes_version : null
  use_local_helm_chart         = trimspace(var.helm_chart_path) != ""
  helm_chart                   = local.use_local_helm_chart ? "${path.module}/${var.helm_chart_path}" : var.helm_chart_name
  helm_chart_repository        = local.use_local_helm_chart ? "" : var.helm_chart_repository
  db_host                      = var.create_postgresql_flexible_server ? azurerm_postgresql_flexible_server.superplane[0].fqdn : "postgres"
  db_ssl                       = var.create_postgresql_flexible_server ? "true" : "false"
  aks_kubeconfig = yamldecode(
    azurerm_kubernetes_cluster.superplane.kube_admin_config_raw != ""
    ? azurerm_kubernetes_cluster.superplane.kube_admin_config_raw
    : azurerm_kubernetes_cluster.superplane.kube_config_raw
  )
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

resource "random_string" "storage" {
  length  = 8
  upper   = false
  special = false
}
