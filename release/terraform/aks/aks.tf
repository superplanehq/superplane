# -----------------------------------------------------------------------------
# AKS
# -----------------------------------------------------------------------------

resource "azurerm_kubernetes_cluster" "superplane" {
  name                = var.cluster_name
  location            = azurerm_resource_group.superplane.location
  resource_group_name = azurerm_resource_group.superplane.name
  dns_prefix          = var.cluster_name
  kubernetes_version  = local.kubernetes_version
  sku_tier            = "Standard"

  oidc_issuer_enabled       = true
  workload_identity_enabled = true
  local_account_disabled    = false

  default_node_pool {
    name                         = "system"
    vm_size                      = var.node_vm_size
    node_count                   = var.node_count
    vnet_subnet_id               = azurerm_subnet.aks.id
    os_disk_size_gb              = 64
    only_critical_addons_enabled = false

    upgrade_settings {
      max_surge = "10%"
    }
  }

  identity {
    type = "SystemAssigned"
  }

  network_profile {
    network_plugin      = "azure"
    network_plugin_mode = "overlay"
    network_policy      = "azure"
    outbound_type       = "userAssignedNATGateway"
    load_balancer_sku   = "standard"
    service_cidr        = var.aks_service_cidr
    dns_service_ip      = var.aks_dns_service_ip
  }

  depends_on = [
    azurerm_subnet_nat_gateway_association.aks,
    azurerm_nat_gateway_public_ip_association.superplane
  ]
}

resource "azurerm_role_assignment" "aks_network" {
  scope                = azurerm_virtual_network.superplane.id
  role_definition_name = "Network Contributor"
  principal_id         = azurerm_kubernetes_cluster.superplane.identity[0].principal_id
}

resource "azurerm_role_assignment" "aks_acr" {
  count                = trimspace(var.container_registry_id) != "" ? 1 : 0
  scope                = var.container_registry_id
  role_definition_name = "AcrPull"
  principal_id         = azurerm_kubernetes_cluster.superplane.kubelet_identity[0].object_id
}
