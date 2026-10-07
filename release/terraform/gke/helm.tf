# -----------------------------------------------------------------------------
# cert-manager Helm Release
# -----------------------------------------------------------------------------

resource "helm_release" "cert_manager" {
  name             = "cert-manager"
  repository       = "https://charts.jetstack.io"
  chart            = "cert-manager"
  namespace        = "cert-manager"
  create_namespace = true
  timeout          = 600 # 10 minutes

  set {
    name  = "installCRDs"
    value = "true"
  }

  depends_on = [
    google_container_cluster.superplane
  ]
}

# -----------------------------------------------------------------------------
# ClusterIssuer for Let's Encrypt
# -----------------------------------------------------------------------------

resource "kubectl_manifest" "letsencrypt_issuer" {
  yaml_body = <<-YAML
    apiVersion: cert-manager.io/v1
    kind: ClusterIssuer
    metadata:
      name: letsencrypt-prod
    spec:
      acme:
        server: https://acme-v02.api.letsencrypt.org/directory
        email: ${var.letsencrypt_email}
        privateKeySecretRef:
          name: letsencrypt-prod
        solvers:
          - http01:
              ingress:
                name: superplane
                serviceType: ClusterIP
  YAML

  depends_on = [
    helm_release.cert_manager
  ]
}

# -----------------------------------------------------------------------------
# SuperPlane Helm Release
# -----------------------------------------------------------------------------

locals {
  superplane_chart_local = var.superplane_chart_path != ""

  gcs_blob_storage_values = local.gcs_blob_storage_enabled ? yamlencode({
    serviceAccount = {
      create = true
      name   = local.app_service_account_name
      annotations = {
        "iam.gke.io/gcp-service-account" = google_service_account.app[0].email
      }
    }
    blobStorage = {
      provider = "gcs"
      bucket   = google_storage_bucket.blobs[0].name
    }
  }) : ""

  runner_api_values = local.runners_enabled ? yamlencode({
    runnerAPI = {
      enabled = true
    }
  }) : ""

  fleet_manager_values = local.fleet_manager_enabled ? yamlencode({
    fleetManager = {
      enabled        = true
      existingSecret = kubernetes_secret.fleet_manager[0].metadata[0].name
      image = {
        registry = var.fleet_manager_image_registry
        tag      = var.fleet_manager_image_tag
      }
      serviceAccount = {
        create = true
        name   = local.fleet_manager_service_account_name
        annotations = {
          "iam.gke.io/gcp-service-account" = google_service_account.fleet_manager[0].email
        }
      }
      podAnnotations = {
        "checksum/config" = sha256(local.fleet_manager_config)
      }
    }
  }) : ""
}

resource "helm_release" "superplane" {
  name             = "superplane"
  repository       = local.superplane_chart_local ? null : "oci://ghcr.io/superplanehq"
  chart            = local.superplane_chart_local ? var.superplane_chart_path : "superplane-chart"
  version          = local.superplane_chart_local || var.superplane_chart_version == "" ? null : var.superplane_chart_version
  namespace        = var.superplane_namespace
  create_namespace = false

  values = compact([
    local.gcs_blob_storage_values,
    local.runner_api_values,
    local.fleet_manager_values,
  ])

  # Database configuration
  set {
    name  = "database.secretName"
    value = "superplane-db-credentials"
  }

  set {
    name  = "database.host"
    value = google_sql_database_instance.superplane.private_ip_address
  }

  set {
    name  = "database.port"
    value = "5432"
  }

  set {
    name  = "database.username"
    value = "postgres"
  }

  set_sensitive {
    name  = "database.password"
    value = local.db_password
  }

  set {
    name  = "database.ssl"
    value = "false"
  }

  set {
    name  = "database.local.enabled"
    value = "false"
  }

  # Image configuration
  set {
    name  = "image.registry"
    value = "ghcr.io/superplanehq"
  }

  set {
    name  = "image.name"
    value = "superplane"
  }

  set {
    name  = "image.tag"
    value = var.superplane_image_tag
  }

  set {
    name  = "image.pullPolicy"
    value = "IfNotPresent"
  }

  # Domain configuration
  set {
    name  = "domain.name"
    value = var.domain_name
  }

  # Ingress configuration
  set {
    name  = "ingress.enabled"
    value = "true"
  }

  set {
    name  = "ingress.className"
    value = "gce"
  }

  set {
    name  = "ingress.staticIpName"
    value = var.static_ip_name
  }

  # FrontendConfig annotation - allows HTTP for ACME challenges
  set {
    name  = "ingress.annotations.networking\\.gke\\.io/v1beta1\\.FrontendConfig"
    value = "superplane-frontend-config"
  }

  # SSL configuration
  set {
    name  = "ingress.ssl.enabled"
    value = "true"
  }

  set {
    name  = "ingress.ssl.type"
    value = "cert-manager"
  }

  set {
    name  = "ingress.ssl.certManager.issuerRef.name"
    value = "letsencrypt-prod"
  }

  set {
    name  = "ingress.ssl.certManager.issuerRef.kind"
    value = "ClusterIssuer"
  }

  set {
    name  = "ingress.ssl.certManager.secretName"
    value = "superplane-tls-secret"
  }

  # Authentication (disabled by default)
  set {
    name  = "authentication.github.enabled"
    value = "false"
  }

  set {
    name  = "authentication.google.enabled"
    value = "false"
  }

  # Telemetry (disabled by default)
  set {
    name  = "telemetry.opentelemetry.enabled"
    value = "false"
  }

  set {
    name  = "telemetry.opentelemetry.endpoint"
    value = ""
  }

  # Secrets
  set {
    name  = "session.secretName"
    value = "superplane-session"
  }

  set {
    name  = "jwt.secretName"
    value = "superplane-jwt"
  }

  set {
    name  = "encryption.secretName"
    value = "superplane-encryption"
  }

  set {
    name  = "oidc.secretName"
    value = "superplane-oidc"
  }

  set {
    name  = "podSecurityContext.runAsNonRoot"
    value = "true"
  }

  set {
    name  = "podSecurityContext.runAsUser"
    value = "65534"
  }

  set {
    name  = "podSecurityContext.fsGroup"
    value = "65534"
  }

  set {
    name  = "securityContext.allowPrivilegeEscalation"
    value = "false"
  }

  set {
    name  = "securityContext.readOnlyRootFilesystem"
    value = "true"
  }

  set {
    name  = "securityContext.runAsNonRoot"
    value = "true"
  }

  set {
    name  = "securityContext.capabilities.drop[0]"
    value = "ALL"
  }

  set {
    name  = "securityContext.seccompProfile.type"
    value = "RuntimeDefault"
  }

  depends_on = [
    kubernetes_namespace.superplane,
    kubernetes_secret.db_credentials,
    kubernetes_secret.session,
    kubernetes_secret.jwt,
    kubernetes_secret.encryption,
    kubernetes_secret.oidc,
    helm_release.cert_manager,
    kubectl_manifest.letsencrypt_issuer,
    kubectl_manifest.frontend_config,
    google_sql_database.superplane,
    google_storage_bucket_iam_member.app_blobs,
    google_service_account_iam_member.app_sign_blob,
    google_service_account_iam_member.app_workload_identity,
    google_project_service.iamcredentials,
    google_project_iam_member.fleet_manager_instance_admin,
    google_service_account_iam_member.fleet_manager_workload_identity,
  ]
}
