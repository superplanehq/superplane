# The GCE ingress controller syncs once while Helm is still creating
# BackendConfig objects. That sync fails, and the controller does not retry.
# The load balancer then has no address, so Let's Encrypt cannot complete.
# Annotate the Ingress after Helm, then wait until HTTPS answers.
resource "terraform_data" "gce_ingress_ready" {
  # depends_on does not rerun this provisioner. Replace it when the
  # release identity changes, including a move to another namespace.
  triggers_replace = [
    helm_release.superplane.name,
    helm_release.superplane.namespace,
    var.cluster_name,
    var.project_id,
    var.zone,
    var.domain_name,
    var.superplane_chart_version,
    var.superplane_image_tag,
  ]

  depends_on = [
    helm_release.superplane
  ]

  provisioner "local-exec" {
    interpreter = ["bash", "-c"]
    command     = <<-EOT
      set -euo pipefail
      gcloud container clusters get-credentials ${var.cluster_name} \
        --zone=${var.zone} \
        --project=${var.project_id}
      # The first sync creates the load balancer and skips the ACME path.
      # The second sync adds that path so Let's Encrypt can finish.
      for _ in 1 2; do
        kubectl --namespace ${var.superplane_namespace} annotate ingress superplane \
          superplane.io/gce-ingress-sync="$(date -u +%s)" --overwrite
        sleep 45
      done
      deadline=$(( $(date -u +%s) + 1200 ))
      until curl --fail --silent --show-error --max-time 20 \
        "https://${var.domain_name}/health" >/dev/null
      do
        if [ "$(date -u +%s)" -ge "$deadline" ]; then
          echo "https://${var.domain_name}/health did not respond within 20 minutes" >&2
          exit 1
        fi
        sleep 15
      done
    EOT
  }
}
