# -----------------------------------------------------------------------------
# Kubernetes Namespace
# -----------------------------------------------------------------------------

resource "kubernetes_namespace" "superplane" {
  metadata {
    name = var.superplane_namespace
  }

  depends_on = [
    # Keep storage drivers running until namespace PVC deletion finishes.
    aws_eks_addon.ebs_csi,
    aws_eks_addon.efs_csi
  ]
}

# -----------------------------------------------------------------------------
# Default StorageClass (gp3)
# -----------------------------------------------------------------------------

resource "kubernetes_storage_class" "gp3_default" {
  metadata {
    name = "gp3"
    annotations = {
      "storageclass.kubernetes.io/is-default-class" = "true"
    }
  }

  storage_provisioner = "ebs.csi.aws.com"
  reclaim_policy      = "Delete"
  volume_binding_mode = "WaitForFirstConsumer"

  parameters = {
    type   = "gp3"
    fsType = "ext4"
  }

  depends_on = [
    aws_eks_addon.ebs_csi
  ]
}

# Bind the chart's runner API and workers to the same EFS access point.
resource "kubernetes_persistent_volume_v1" "runner_active_logs" {
  metadata {
    name = "${var.cluster_name}-runner-active-logs"
  }

  spec {
    capacity = {
      storage = var.runner_active_log_volume_capacity
    }
    access_modes                     = ["ReadWriteMany"]
    persistent_volume_reclaim_policy = "Retain"
    storage_class_name               = ""
    mount_options                    = ["tls"]

    persistent_volume_source {
      csi {
        driver        = "efs.csi.aws.com"
        volume_handle = "${aws_efs_file_system.runner_active_logs.id}::${aws_efs_access_point.runner_active_logs.id}"
      }
    }
  }

  depends_on = [aws_eks_addon.efs_csi, aws_efs_mount_target.runner_active_logs]
}

resource "kubernetes_persistent_volume_claim_v1" "runner_active_logs" {
  metadata {
    name      = "superplane-runner-active-logs"
    namespace = kubernetes_namespace.superplane.metadata[0].name
  }

  spec {
    access_modes       = ["ReadWriteMany"]
    storage_class_name = ""
    volume_name        = kubernetes_persistent_volume_v1.runner_active_logs.metadata[0].name

    resources {
      requests = {
        storage = var.runner_active_log_volume_capacity
      }
    }
  }
}

# -----------------------------------------------------------------------------
# Database Credentials Secret
# -----------------------------------------------------------------------------

resource "kubernetes_secret" "db_credentials" {
  metadata {
    name      = "superplane-db-credentials"
    namespace = kubernetes_namespace.superplane.metadata[0].name
  }

  data = {
    DB_HOST         = aws_db_instance.superplane.address
    DB_PORT         = "5432"
    DB_NAME         = var.db_name
    DB_USERNAME     = var.db_username
    DB_PASSWORD     = local.db_password
    POSTGRES_DB_SSL = "true"
  }
}

# -----------------------------------------------------------------------------
# Session Secret
# -----------------------------------------------------------------------------

resource "kubernetes_secret" "session" {
  metadata {
    name      = "superplane-session"
    namespace = kubernetes_namespace.superplane.metadata[0].name
  }

  data = {
    SESSION_SECRET = random_password.session_secret.result
  }
}

# -----------------------------------------------------------------------------
# JWT Secret
# -----------------------------------------------------------------------------

resource "kubernetes_secret" "jwt" {
  metadata {
    name      = "superplane-jwt"
    namespace = kubernetes_namespace.superplane.metadata[0].name
  }

  data = {
    JWT_SECRET = random_password.jwt_secret.result
  }
}

# -----------------------------------------------------------------------------
# Encryption Key Secret
# -----------------------------------------------------------------------------

resource "kubernetes_secret" "encryption" {
  metadata {
    name      = "superplane-encryption"
    namespace = kubernetes_namespace.superplane.metadata[0].name
  }

  data = {
    ENCRYPTION_KEY = random_password.encryption_key.result
  }
}

# -----------------------------------------------------------------------------
# OIDC Key Secret
# -----------------------------------------------------------------------------

resource "kubernetes_secret" "oidc" {
  metadata {
    name      = "superplane-oidc"
    namespace = kubernetes_namespace.superplane.metadata[0].name
  }

  data = {
    "${time_static.oidc_key.unix}.pem" = tls_private_key.oidc.private_key_pem
  }
}
