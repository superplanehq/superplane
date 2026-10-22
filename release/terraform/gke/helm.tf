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

  gcs_blob_storage_values = yamlencode({
    serviceAccount = {
      create = true
      name   = local.app_service_account_name
      annotations = {
        "iam.gke.io/gcp-service-account" = google_service_account.app.email
      }
    }
    blobStorage = {
      provider = "gcs"
      bucket   = google_storage_bucket.blobs.name
    }
  })

  runner_api_values = yamlencode({
    runner = {
      api = {
        enabled = true
      }
      workers = {
        enabled = true
      }
    }
  })

  fleet_manager_setup_values = yamlencode({
    # The published chart still defaults both components to v0.1.0.
    # Runner and Fleet Manager publish on separate release lines.
    componentReleases = {
      runnerVersion       = "v0.2.0"
      runnerVersions      = ["v0.2.0"]
      fleetManagerVersion = "v0.1.3"
    }
    fleetManagerSetup = {
      enabled            = true
      provider           = "gcp"
      managerId          = var.cluster_name
      namespace          = var.superplane_namespace
      serviceAccountName = local.fleet_manager_service_account_name
      infrastructure = {
        superplaneUrl        = "https://${var.domain_name}"
        projectId            = var.project_id
        zones                = local.runner_zones
        subnetwork           = google_compute_subnetwork.runners.id
        networkTags          = [local.runner_network_tag]
        imageAmd64           = var.runner_image_amd64
        imageArm64           = var.runner_image_arm64
        fleetManagerIdentity = google_service_account.fleet_manager.email
      }
    }
  })

}

resource "helm_release" "superplane" {
  name             = "superplane"
  repository       = local.superplane_chart_local ? null : "oci://ghcr.io/superplanehq"
  chart            = local.superplane_chart_local ? var.superplane_chart_path : "superplane-chart"
  version          = local.superplane_chart_local || var.superplane_chart_version == "" ? null : var.superplane_chart_version
  namespace        = var.superplane_namespace
  create_namespace = false
  timeout          = 1200 # Match EKS and AKS. First image pull and migrate can exceed 15 minutes.

  lifecycle {
    precondition {
      condition     = local.superplane_chart_local || var.superplane_chart_version != ""
      error_message = "Set superplane_chart_version or superplane_chart_path. An empty version installs the latest published chart, which may lack GKE ACME and migrate fixes."
    }

    precondition {
      condition     = !local.superplane_chart_local || var.superplane_image_tag != ""
      error_message = "With a local chart (superplane_chart_path), set superplane_image_tag. The local Chart.yaml appVersion (0.1.0) is a placeholder, not a published image."
    }
  }

  values = compact([
    local.gcs_blob_storage_values,
    local.runner_api_values,
    local.fleet_manager_setup_values,
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
    value = var.superplane_image_registry
  }

  set {
    name  = "image.name"
    value = "superplane"
  }

  dynamic "set" {
    for_each = var.superplane_image_tag == "" ? [] : [var.superplane_image_tag]
    content {
      name  = "image.tag"
      value = set.value
    }
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
    name  = "encryption.writeFormat"
    value = "versioned"
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
    google_project_iam_member.gke_nodes_artifact_registry,
    google_project_iam_member.fleet_manager_instance_admin,
    google_service_account_iam_member.fleet_manager_workload_identity,
  ]
}
