# -----------------------------------------------------------------------------
# cert-manager Helm Release
# -----------------------------------------------------------------------------

resource "helm_release" "cert_manager" {
  name             = "cert-manager"
  repository       = "https://charts.jetstack.io"
  chart            = "cert-manager"
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

resource "helm_release" "superplane" {
  name             = "superplane"
  repository       = "oci://ghcr.io/superplanehq"
  chart            = "superplane-chart"
  namespace        = var.superplane_namespace
  create_namespace = false

  set {
    name  = "database.secretName"
    value = "superplane-db-credentials"
  }

  set {
    name  = "database.host"
    value = azurerm_postgresql_flexible_server.superplane.fqdn
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
    value = "true"
  }

  set {
    name  = "database.local.enabled"
    value = "false"
  }

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
    name  = "runnerAPI.enabled"
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
  }

  set {
    name  = "fleetManager.enabled"
    value = local.fleet_manager_enabled ? "true" : "false"
  }

  set {
    name  = "fleetManager.image.registry"
    value = "ghcr.io/superplanehq"
  }

  set {
    name  = "fleetManager.image.name"
    value = "superplane-fleet-manager"
  }

  set {
    name  = "fleetManager.image.tag"
    value = local.fleet_manager_image_tag
  }

  set {
    name  = "fleetManager.existingSecret"
    value = local.fleet_manager_enabled ? kubernetes_secret.fleet_manager[0].metadata[0].name : ""
  }

  set {
    name  = "fleetManager.serviceAccount.create"
    value = "true"
  }

  set {
    name  = "fleetManager.serviceAccount.name"
    value = "${var.cluster_name}-fleet-manager"
  }

  set {
    name  = "fleetManager.serviceAccount.annotations.azure\\.workload\\.identity/client-id"
    value = azurerm_user_assigned_identity.fleet_manager.client_id
  }

  set {
    name  = "fleetManager.podLabels.azure\\.workload\\.identity/use"
    value = "true"
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
    azurerm_federated_identity_credential.superplane,
    azurerm_federated_identity_credential.fleet_manager
  ]
}
