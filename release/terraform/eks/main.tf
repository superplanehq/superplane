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
