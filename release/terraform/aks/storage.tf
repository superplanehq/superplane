# -----------------------------------------------------------------------------
# Blob storage
# -----------------------------------------------------------------------------

resource "azurerm_storage_account" "superplane" {
  name                            = "sp${random_string.storage.result}"
  resource_group_name             = azurerm_resource_group.superplane.name
  location                        = azurerm_resource_group.superplane.location
  account_tier                    = "Standard"
  account_replication_type        = "GRS"
  min_tls_version                 = "TLS1_2"
  https_traffic_only_enabled      = true
  allow_nested_items_to_be_public = false
  # Terraform waits on the blob data plane. That wait needs account keys or
  # Azure AD. Keys stay enabled for the provider. SuperPlane pods still use
  # workload identity, not account keys.
  shared_access_key_enabled = true
  local_user_enabled        = false
}

resource "azurerm_storage_container" "blobs" {
  name                  = "superplane"
  storage_account_id    = azurerm_storage_account.superplane.id
  container_access_type = "private"
}

resource "azurerm_user_assigned_identity" "superplane" {
  name                = "${var.cluster_name}-app"
  location            = azurerm_resource_group.superplane.location
  resource_group_name = azurerm_resource_group.superplane.name
}

resource "azurerm_role_assignment" "blob_data" {
  scope                = azurerm_storage_account.superplane.id
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = azurerm_user_assigned_identity.superplane.principal_id
}

resource "azurerm_role_assignment" "blob_delegator" {
  scope                = azurerm_storage_account.superplane.id
  role_definition_name = "Storage Blob Delegator"
  principal_id         = azurerm_user_assigned_identity.superplane.principal_id
}

resource "azurerm_federated_identity_credential" "superplane" {
  name                      = "${var.cluster_name}-app"
  user_assigned_identity_id = azurerm_user_assigned_identity.superplane.id
  audience                  = ["api://AzureADTokenExchange"]
  issuer                    = azurerm_kubernetes_cluster.superplane.oidc_issuer_url
  subject                   = "system:serviceaccount:${var.superplane_namespace}:${var.cluster_name}"
}
