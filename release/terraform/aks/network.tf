# -----------------------------------------------------------------------------
# Resource groups
# -----------------------------------------------------------------------------

resource "azurerm_resource_group" "superplane" {
  name     = var.resource_group_name
  location = var.location
}

resource "azurerm_resource_group" "runners" {
  name     = "${var.resource_group_name}-runners"
  location = var.location
}

# -----------------------------------------------------------------------------
# Virtual network
# -----------------------------------------------------------------------------

resource "azurerm_virtual_network" "superplane" {
  name                = "${var.cluster_name}-vnet"
  location            = azurerm_resource_group.superplane.location
  resource_group_name = azurerm_resource_group.superplane.name
  address_space       = [var.vnet_cidr]
}

resource "azurerm_subnet" "aks" {
  name                 = "${var.cluster_name}-aks"
  resource_group_name  = azurerm_resource_group.superplane.name
  virtual_network_name = azurerm_virtual_network.superplane.name
  address_prefixes     = [var.aks_subnet_cidr]
  service_endpoints    = ["Microsoft.Storage"]
}

resource "azurerm_subnet" "postgres" {
  name                 = "${var.cluster_name}-postgres"
  resource_group_name  = azurerm_resource_group.superplane.name
  virtual_network_name = azurerm_virtual_network.superplane.name
  address_prefixes     = [var.postgres_subnet_cidr]

  delegation {
    name = "postgres"
    service_delegation {
      name    = "Microsoft.DBforPostgreSQL/flexibleServers"
      actions = ["Microsoft.Network/virtualNetworks/subnets/join/action"]
    }
  }
}

resource "azurerm_subnet" "runners" {
  name                 = "${var.cluster_name}-runners"
  resource_group_name  = azurerm_resource_group.superplane.name
  virtual_network_name = azurerm_virtual_network.superplane.name
  address_prefixes     = [var.runner_subnet_cidr]
}

# -----------------------------------------------------------------------------
# NAT Gateway
# -----------------------------------------------------------------------------

resource "azurerm_public_ip" "nat" {
  name                = "${var.cluster_name}-nat"
  location            = azurerm_resource_group.superplane.location
  resource_group_name = azurerm_resource_group.superplane.name
  allocation_method   = "Static"
  sku                 = "Standard"
}

resource "azurerm_nat_gateway" "superplane" {
  name                    = "${var.cluster_name}-nat"
  location                = azurerm_resource_group.superplane.location
  resource_group_name     = azurerm_resource_group.superplane.name
  sku_name                = "Standard"
  idle_timeout_in_minutes = 10
}

resource "azurerm_nat_gateway_public_ip_association" "superplane" {
  nat_gateway_id       = azurerm_nat_gateway.superplane.id
  public_ip_address_id = azurerm_public_ip.nat.id
}

resource "azurerm_subnet_nat_gateway_association" "aks" {
  subnet_id      = azurerm_subnet.aks.id
  nat_gateway_id = azurerm_nat_gateway.superplane.id
}

resource "azurerm_subnet_nat_gateway_association" "runners" {
  subnet_id      = azurerm_subnet.runners.id
  nat_gateway_id = azurerm_nat_gateway.superplane.id
}

# -----------------------------------------------------------------------------
# Network security groups
# -----------------------------------------------------------------------------

resource "azurerm_network_security_group" "aks" {
  name                = "${var.cluster_name}-aks"
  location            = azurerm_resource_group.superplane.location
  resource_group_name = azurerm_resource_group.superplane.name
}

resource "azurerm_network_security_group" "runners" {
  name                = "${var.cluster_name}-runners"
  location            = azurerm_resource_group.superplane.location
  resource_group_name = azurerm_resource_group.superplane.name

  security_rule {
    name                       = "deny-internet-inbound"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Deny"
    protocol                   = "*"
    source_port_range          = "*"
    destination_port_range     = "*"
    source_address_prefix      = "Internet"
    destination_address_prefix = "*"
  }
}

resource "azurerm_subnet_network_security_group_association" "aks" {
  subnet_id                 = azurerm_subnet.aks.id
  network_security_group_id = azurerm_network_security_group.aks.id
}

resource "azurerm_subnet_network_security_group_association" "runners" {
  subnet_id                 = azurerm_subnet.runners.id
  network_security_group_id = azurerm_network_security_group.runners.id
}
