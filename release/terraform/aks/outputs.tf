output "cluster_name" {
  description = "Name of the AKS cluster"
  value       = azurerm_kubernetes_cluster.superplane.name
}

output "cluster_fqdn" {
  description = "AKS cluster FQDN"
  value       = azurerm_kubernetes_cluster.superplane.fqdn
}

output "database_fqdn" {
  description = "PostgreSQL Flexible Server FQDN"
  value       = azurerm_postgresql_flexible_server.superplane.fqdn
}

output "storage_account_name" {
  description = "Blob storage account name"
  value       = azurerm_storage_account.superplane.name
}

output "storage_container_name" {
  description = "Blob container name"
  value       = azurerm_storage_container.blobs.name
}

output "app_identity_client_id" {
  description = "Workload identity client ID for SuperPlane pods"
  value       = azurerm_user_assigned_identity.superplane.client_id
}

output "fleet_manager_identity_client_id" {
  description = "Workload identity client ID for Fleet Manager"
  value       = azurerm_user_assigned_identity.fleet_manager.client_id
}

output "runner_identity_id" {
  description = "User-assigned identity resource ID for runner VMs"
  value       = azurerm_user_assigned_identity.runner.id
}

output "runner_subnet_id" {
  description = "Subnet ID for runner VMs"
  value       = azurerm_subnet.runners.id
}

output "runner_nsg_id" {
  description = "Network security group ID for runner VMs"
  value       = azurerm_network_security_group.runners.id
}

output "runner_gallery_id" {
  description = "Azure Compute Gallery ID for runner images"
  value       = azurerm_shared_image_gallery.runners.id
}

output "runner_image_amd64_id" {
  description = "Gallery image definition ID for linux/amd64 runners"
  value       = azurerm_shared_image.runner_amd64.id
}

output "runner_image_arm64_id" {
  description = "Gallery image definition ID for linux/arm64 runners"
  value       = azurerm_shared_image.runner_arm64.id
}

output "superplane_namespace" {
  description = "Kubernetes namespace where SuperPlane is deployed"
  value       = var.superplane_namespace
}

output "superplane_url" {
  description = "URL to access SuperPlane"
  value       = "https://${var.domain_name}"
}

output "kubectl_config_command" {
  description = "Command to configure kubectl to connect to the cluster"
  value       = "az aks get-credentials --resource-group ${azurerm_resource_group.superplane.name} --name ${var.cluster_name} --admin"
}

output "load_balancer_ip_command" {
  description = "Command to get the load balancer IP for DNS configuration"
  value       = "kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}'"
}

output "next_steps" {
  description = "Instructions to complete the setup"
  value       = <<-EOT

    ============================================================
    NEXT STEPS - Configure DNS
    ============================================================

    1. Configure kubectl:
       az aks get-credentials --resource-group ${azurerm_resource_group.superplane.name} --name ${var.cluster_name} --admin

    2. Get the Load Balancer IP:
       kubectl get svc -n ingress-nginx ingress-nginx-controller -o jsonpath='{.status.loadBalancer.ingress[0].ip}'

    3. Create an A record in your DNS provider:
       - Type: A
       - Name: ${var.domain_name}
       - Value: <IP from step 2>

    4. Wait for DNS propagation and certificate issuance (~5-10 min)

    5. Access SuperPlane at: https://${var.domain_name}

    ============================================================
  EOT
}
