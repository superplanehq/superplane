# -----------------------------------------------------------------------------
# Kubernetes Namespace
# -----------------------------------------------------------------------------

resource "kubernetes_namespace" "superplane" {
  metadata {
    name = var.superplane_namespace
  }

  depends_on = [
    google_container_cluster.superplane
  ]
}

# -----------------------------------------------------------------------------
# Database Credentials Secret
# -----------------------------------------------------------------------------

resource "kubernetes_secret" "db_credentials" {
  metadata {
    name      = "superplane-db-credentials"
    namespace = kubernetes_namespace.superplane.metadata[0].name
  }

  data = {
    DB_HOST         = google_sql_database_instance.superplane.private_ip_address
    DB_PORT         = "5432"
    DB_NAME         = var.db_name
    DB_USERNAME     = "postgres"
    DB_PASSWORD     = local.db_password
    POSTGRES_DB_SSL = "false"
  }
}

# -----------------------------------------------------------------------------
# Session Secret
# -----------------------------------------------------------------------------

resource "kubernetes_secret" "session" {
  metadata {
    name      = "superplane-session"
    namespace = kubernetes_namespace.superplane.metadata[0].name
  }

  data = {
    SESSION_SECRET = random_password.session_secret.result
  }
}

# -----------------------------------------------------------------------------
# JWT Secret
# -----------------------------------------------------------------------------

resource "kubernetes_secret" "jwt" {
  metadata {
    name      = "superplane-jwt"
    namespace = kubernetes_namespace.superplane.metadata[0].name
  }

  data = {
    JWT_SECRET = random_password.jwt_secret.result
  }
}

# -----------------------------------------------------------------------------
# Encryption Key Secret
# -----------------------------------------------------------------------------

resource "kubernetes_secret" "encryption" {
  metadata {
    name      = "superplane-encryption"
    namespace = kubernetes_namespace.superplane.metadata[0].name
  }

  data = {
    ENCRYPTION_KEY = random_password.encryption_key.result
  }
}

# -----------------------------------------------------------------------------
# OIDC Key Secret
# -----------------------------------------------------------------------------

resource "kubernetes_secret" "oidc" {
  metadata {
    name      = "superplane-oidc"
    namespace = kubernetes_namespace.superplane.metadata[0].name
  }

  data = {
    "${time_static.oidc_key.unix}.pem" = tls_private_key.oidc.private_key_pem
  }
}

# -----------------------------------------------------------------------------
# FrontendConfig - Allow HTTP for ACME challenges
# -----------------------------------------------------------------------------

resource "kubectl_manifest" "frontend_config" {
  yaml_body = <<-YAML
    apiVersion: networking.gke.io/v1beta1
    kind: FrontendConfig
    metadata:
      name: superplane-frontend-config
      namespace: ${var.superplane_namespace}
    spec:
      redirectToHttps:
        enabled: false
  YAML

  depends_on = [
    kubernetes_namespace.superplane
  ]
}
