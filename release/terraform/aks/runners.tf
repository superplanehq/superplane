# -----------------------------------------------------------------------------
# Runner identities and gallery
# -----------------------------------------------------------------------------

resource "azurerm_user_assigned_identity" "fleet_manager" {
  name                = "${var.cluster_name}-fleet-manager"
  location            = azurerm_resource_group.superplane.location
  resource_group_name = azurerm_resource_group.superplane.name
}

resource "azurerm_user_assigned_identity" "runner" {
  name                = "${var.cluster_name}-runner"
  location            = azurerm_resource_group.runners.location
  resource_group_name = azurerm_resource_group.runners.name
}

resource "azurerm_role_assignment" "fleet_manager_vm" {
  scope                = azurerm_resource_group.runners.id
  role_definition_name = "Virtual Machine Contributor"
  principal_id         = azurerm_user_assigned_identity.fleet_manager.principal_id
}

resource "azurerm_role_assignment" "fleet_manager_network" {
  scope                = azurerm_resource_group.runners.id
  role_definition_name = "Network Contributor"
  principal_id         = azurerm_user_assigned_identity.fleet_manager.principal_id
}

resource "azurerm_role_assignment" "fleet_manager_gallery" {
  scope                = azurerm_shared_image_gallery.runners.id
  role_definition_name = "Reader"
  principal_id         = azurerm_user_assigned_identity.fleet_manager.principal_id
}

resource "azurerm_role_assignment" "fleet_manager_runner_identity" {
  scope                = azurerm_user_assigned_identity.runner.id
  role_definition_name = "Managed Identity Operator"
  principal_id         = azurerm_user_assigned_identity.fleet_manager.principal_id
}

resource "azurerm_federated_identity_credential" "fleet_manager" {
  name                = "${var.cluster_name}-fleet-manager"
  resource_group_name = azurerm_resource_group.superplane.name
  parent_id           = azurerm_user_assigned_identity.fleet_manager.id
  audience            = ["api://AzureADTokenExchange"]
  issuer              = azurerm_kubernetes_cluster.superplane.oidc_issuer_url
  subject             = "system:serviceaccount:${var.superplane_namespace}:${var.cluster_name}-fleet-manager"
}

resource "azurerm_shared_image_gallery" "runners" {
  name                = replace("${var.cluster_name}runners", "-", "")
  resource_group_name = azurerm_resource_group.runners.name
  location            = azurerm_resource_group.runners.location
  description         = "SuperPlane runner images"
}

resource "azurerm_shared_image" "runner_amd64" {
  name                     = "superplane-runner-amd64"
  gallery_name             = azurerm_shared_image_gallery.runners.name
  resource_group_name      = azurerm_resource_group.runners.name
  location                 = azurerm_resource_group.runners.location
  os_type                  = "Linux"
  hyper_v_generation       = "V2"
  architecture             = "x64"
  trusted_launch_supported = true

  identifier {
    publisher = "superplane"
    offer     = "runner"
    sku       = "ubuntu-2404-amd64"
  }
}

resource "azurerm_shared_image" "runner_arm64" {
  name                     = "superplane-runner-arm64"
  gallery_name             = azurerm_shared_image_gallery.runners.name
  resource_group_name      = azurerm_resource_group.runners.name
  location                 = azurerm_resource_group.runners.location
  os_type                  = "Linux"
  hyper_v_generation       = "V2"
  architecture             = "Arm64"
  trusted_launch_supported = true

  identifier {
    publisher = "superplane"
    offer     = "runner"
    sku       = "ubuntu-2404-arm64"
  }
}
