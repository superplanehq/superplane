# -----------------------------------------------------------------------------
# cert-manager Helm Release
# -----------------------------------------------------------------------------

resource "helm_release" "cert_manager" {
  name             = "cert-manager"
  repository       = "https://charts.jetstack.io"
  chart            = "cert-manager"
  version          = var.cert_manager_chart_version
  namespace        = "cert-manager"
  create_namespace = true
  timeout          = 600
  wait             = true

  set {
    name  = "installCRDs"
    value = "true"
  }

  depends_on = [
    azurerm_kubernetes_cluster.superplane
  ]
}

# -----------------------------------------------------------------------------
# NGINX Ingress Controller
# -----------------------------------------------------------------------------

resource "helm_release" "nginx_ingress" {
  name             = "ingress-nginx"
  repository       = "https://kubernetes.github.io/ingress-nginx"
  chart            = "ingress-nginx"
  version          = var.ingress_nginx_chart_version
  namespace        = "ingress-nginx"
  create_namespace = true
  timeout          = 1200

  set {
    name  = "controller.service.type"
    value = "LoadBalancer"
  }

  set {
    name  = "controller.service.annotations.service\\.beta\\.kubernetes\\.io/azure-load-balancer-health-probe-request-path"
    value = "/healthz"
  }

  depends_on = [
    azurerm_kubernetes_cluster.superplane,
    azurerm_role_assignment.aks_network
  ]
}

# -----------------------------------------------------------------------------
# SuperPlane Helm Release
# -----------------------------------------------------------------------------

locals {
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
      provider           = "azure"
      managerId          = var.cluster_name
      namespace          = var.superplane_namespace
      serviceAccountName = "superplane-fleet-manager"
      infrastructure = {
        superplaneUrl        = "https://${var.domain_name}"
        subscriptionId       = var.subscription_id
        resourceGroup        = azurerm_resource_group.runners.name
        location             = azurerm_resource_group.runners.location
        zones                = var.runner_zones
        subnetId             = azurerm_subnet.runners.id
        securityGroupIds     = azurerm_network_security_group.runners.id
        imageAmd64           = var.runner_image_amd64
        imageArm64           = var.runner_image_arm64
        runnerIdentity       = azurerm_user_assigned_identity.runner.id
        fleetManagerIdentity = azurerm_user_assigned_identity.fleet_manager.client_id
      }
    }
  })
}

resource "helm_release" "superplane" {
  name             = "superplane"
  repository       = local.helm_chart_repository
  chart            = local.helm_chart
  namespace        = var.superplane_namespace
  create_namespace = false
  timeout          = 1200
  wait             = true
  version          = local.use_local_helm_chart ? null : var.superplane_chart_version

  lifecycle {
    precondition {
      condition     = local.use_local_helm_chart || var.superplane_chart_version != ""
      error_message = "Set superplane_chart_version, or set helm_chart_path for a local chart."
    }

    precondition {
      condition     = !local.use_local_helm_chart || var.superplane_image_tag != ""
      error_message = "With a local chart (helm_chart_path), set superplane_image_tag. The local Chart.yaml appVersion (0.1.0) is a placeholder, not a published image."
    }
  }

  values = [local.fleet_manager_setup_values]

  set {
    name  = "componentReleases.fleetManagerVersion"
    value = var.fleet_manager_version
  }

  set {
    name  = "database.secretName"
    value = "superplane-db-credentials"
  }

  set {
    name  = "database.host"
    value = local.db_host
  }

  set {
    name  = "database.port"
    value = "5432"
  }

  set {
    name  = "database.username"
    value = var.db_username
  }

  set_sensitive {
    name  = "database.password"
    value = local.db_password
  }

  set {
    name  = "database.ssl"
    value = local.db_ssl
  }

  set {
    name  = "database.local.enabled"
    value = var.create_postgresql_flexible_server ? "false" : "true"
  }

  set {
    name  = "image.registry"
    value = var.image_registry
  }

  set {
    name  = "image.name"
    value = var.image_name
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

  set {
    name  = "domain.name"
    value = var.domain_name
  }

  set {
    name  = "ingress.enabled"
    value = "true"
  }

  set {
    name  = "ingress.className"
    value = "nginx"
  }

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

  set {
    name  = "ingress.ssl.certManager.createClusterIssuer"
    value = "true"
  }

  set {
    name  = "ingress.ssl.certManager.acme.email"
    value = var.letsencrypt_email
  }

  set {
    name  = "authentication.github.enabled"
    value = "false"
  }

  set {
    name  = "authentication.google.enabled"
    value = "false"
  }

  set {
    name  = "telemetry.opentelemetry.enabled"
    value = "false"
  }

  set {
    name  = "telemetry.opentelemetry.endpoint"
    value = ""
  }

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
    name  = "blobStorage.provider"
    value = "azure"
  }

  set {
    name  = "blobStorage.account"
    value = azurerm_storage_account.superplane.name
  }

  set {
    name  = "blobStorage.bucket"
    value = azurerm_storage_container.blobs.name
  }

  set {
    name  = "runner.api.enabled"
    value = "true"
  }

  set {
    name  = "serviceAccount.create"
    value = "true"
  }

  set {
    name  = "serviceAccount.name"
    value = var.cluster_name
  }

  set {
    name  = "serviceAccount.annotations.azure\\.workload\\.identity/client-id"
    value = azurerm_user_assigned_identity.superplane.client_id
  }

  set {
    name  = "podLabels.azure\\.workload\\.identity/use"
    value = "true"
    type  = "string"
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
    helm_release.nginx_ingress,
    azurerm_postgresql_flexible_server_database.superplane,
    azurerm_role_assignment.blob_data,
    azurerm_role_assignment.aks_acr,
    azurerm_federated_identity_credential.superplane,
    azurerm_federated_identity_credential.fleet_manager
  ]
}
