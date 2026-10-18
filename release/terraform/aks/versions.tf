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
  storage_use_azuread = true
  features {}
  subscription_id = var.subscription_id
}

provider "kubernetes" {
  host                   = local.aks_kubeconfig.clusters[0].cluster.server
  client_certificate     = base64decode(local.aks_kubeconfig.users[0].user["client-certificate-data"])
  client_key             = base64decode(local.aks_kubeconfig.users[0].user["client-key-data"])
  cluster_ca_certificate = base64decode(local.aks_kubeconfig.clusters[0].cluster["certificate-authority-data"])
}

provider "helm" {
  kubernetes {
    host                   = local.aks_kubeconfig.clusters[0].cluster.server
    client_certificate     = base64decode(local.aks_kubeconfig.users[0].user["client-certificate-data"])
    client_key             = base64decode(local.aks_kubeconfig.users[0].user["client-key-data"])
    cluster_ca_certificate = base64decode(local.aks_kubeconfig.clusters[0].cluster["certificate-authority-data"])
  }
}
