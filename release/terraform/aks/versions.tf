terraform {
  required_version = ">= 1.5.0"

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.25"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.12"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }
    time = {
      source  = "hashicorp/time"
      version = "~> 0.10"
    }
  }
}

provider "azurerm" {
  features {}
  subscription_id = var.subscription_id
}

provider "kubernetes" {
  host                   = azurerm_kubernetes_cluster.superplane.kube_admin_config[0].host
  client_certificate     = base64decode(azurerm_kubernetes_cluster.superplane.kube_admin_config[0].client_certificate)
  client_key             = base64decode(azurerm_kubernetes_cluster.superplane.kube_admin_config[0].client_key)
  cluster_ca_certificate = base64decode(azurerm_kubernetes_cluster.superplane.kube_admin_config[0].cluster_ca_certificate)
}

provider "helm" {
  kubernetes {
    host                   = azurerm_kubernetes_cluster.superplane.kube_admin_config[0].host
    client_certificate     = base64decode(azurerm_kubernetes_cluster.superplane.kube_admin_config[0].client_certificate)
    client_key             = base64decode(azurerm_kubernetes_cluster.superplane.kube_admin_config[0].client_key)
    cluster_ca_certificate = base64decode(azurerm_kubernetes_cluster.superplane.kube_admin_config[0].cluster_ca_certificate)
  }
}
