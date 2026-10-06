# -----------------------------------------------------------------------------
# Random password for database (if not provided)
# -----------------------------------------------------------------------------

resource "random_password" "db_password" {
  count   = var.db_password == "" ? 1 : 0
  length  = 32
  special = false
}

locals {
  db_password             = var.db_password != "" ? var.db_password : random_password.db_password[0].result
  fleet_manager_image_tag = var.fleet_manager_image_tag != "" ? var.fleet_manager_image_tag : var.superplane_image_tag
  fleet_manager_enabled   = trimspace(var.fleet_manager_config) != ""
  kubernetes_version      = trimspace(var.kubernetes_version) != "" ? var.kubernetes_version : null
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
