# -----------------------------------------------------------------------------
# PostgreSQL Flexible Server
# -----------------------------------------------------------------------------

resource "azurerm_private_dns_zone" "postgres" {
  name                = "privatelink.postgres.database.azure.com"
  resource_group_name = azurerm_resource_group.superplane.name
}

resource "azurerm_private_dns_zone_virtual_network_link" "postgres" {
  name                  = "${var.cluster_name}-postgres"
  resource_group_name   = azurerm_resource_group.superplane.name
  private_dns_zone_name = azurerm_private_dns_zone.postgres.name
  virtual_network_id    = azurerm_virtual_network.superplane.id
}

resource "azurerm_postgresql_flexible_server" "superplane" {
  count                         = var.create_postgresql_flexible_server ? 1 : 0
  name                          = var.db_server_name
  resource_group_name           = azurerm_resource_group.superplane.name
  location                      = azurerm_resource_group.superplane.location
  version                       = var.db_engine_version
  sku_name                      = var.db_sku_name
  storage_mb                    = var.db_storage_mb
  administrator_login           = var.db_username
  administrator_password        = local.db_password
  delegated_subnet_id           = azurerm_subnet.postgres.id
  private_dns_zone_id           = azurerm_private_dns_zone.postgres.id
  public_network_access_enabled = false
  backup_retention_days         = 7

  authentication {
    password_auth_enabled = true
  }

  lifecycle {
    prevent_destroy = true
  }

  depends_on = [
    azurerm_private_dns_zone_virtual_network_link.postgres
  ]
}

resource "azurerm_postgresql_flexible_server_database" "superplane" {
  count     = var.create_postgresql_flexible_server ? 1 : 0
  name      = var.db_name
  server_id = azurerm_postgresql_flexible_server.superplane[0].id
  charset   = "UTF8"
  collation = "en_US.utf8"
}

resource "azurerm_postgresql_flexible_server_configuration" "require_tls" {
  count     = var.create_postgresql_flexible_server ? 1 : 0
  name      = "require_secure_transport"
  server_id = azurerm_postgresql_flexible_server.superplane[0].id
  value     = "on"
}
