# -----------------------------------------------------------------------------
# AWS Load Balancer Controller IAM Role
# -----------------------------------------------------------------------------

data "aws_caller_identity" "current" {}

resource "aws_iam_role" "aws_load_balancer_controller" {
  name = "${var.cluster_name}-aws-load-balancer-controller"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRoleWithWebIdentity"
      Effect = "Allow"
      Principal = {
        Federated = aws_iam_openid_connect_provider.eks.arn
      }
      Condition = {
        StringEquals = {
          "${replace(aws_eks_cluster.superplane.identity[0].oidc[0].issuer, "https://", "")}:sub" = "system:serviceaccount:kube-system:aws-load-balancer-controller"
          "${replace(aws_eks_cluster.superplane.identity[0].oidc[0].issuer, "https://", "")}:aud" = "sts.amazonaws.com"
        }
      }
    }]
  })
}

# Custom policy with resource-based conditions scoped to cluster-tagged resources
resource "aws_iam_role_policy" "aws_load_balancer_controller" {
  name = "${var.cluster_name}-lb-controller-policy"
  role = aws_iam_role.aws_load_balancer_controller.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ec2:DescribeAvailabilityZones",
          "ec2:DescribeSecurityGroups",
          "ec2:DescribeSubnets",
          "ec2:DescribeVpcs",
          "ec2:DescribeTags",
          "ec2:DescribeInstances",
          "ec2:DescribeNetworkInterfaces",
          "ec2:DescribeAccountAttributes",
          "ec2:DescribeAddresses",
          "ec2:DescribeInternetGateways",
          "ec2:DescribeCoipPools",
          "ec2:GetCoipPoolUsage",
          "ec2:DescribeVpcPeeringConnections"
        ]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "ec2:CreateSecurityGroup"
        ]
        Resource = "arn:aws:ec2:${var.region}:${data.aws_caller_identity.current.account_id}:vpc/${aws_vpc.superplane.id}"
      },
      {
        Effect = "Allow"
        Action = [
          "ec2:CreateSecurityGroup"
        ]
        Resource = "arn:aws:ec2:${var.region}:${data.aws_caller_identity.current.account_id}:security-group/*"
        Condition = {
          StringEquals = {
            "aws:RequestTag/kubernetes.io/cluster/${var.cluster_name}" = "owned"
          }
        }
      },
      {
        Effect = "Allow"
        Action = [
          "ec2:AuthorizeSecurityGroupIngress",
          "ec2:RevokeSecurityGroupIngress",
          "ec2:DeleteSecurityGroup"
        ]
        Resource = "*"
        Condition = {
          StringEquals = {
            "aws:ResourceTag/kubernetes.io/cluster/${var.cluster_name}" = "owned"
          }
        }
      },
      {
        Effect = "Allow"
        Action = [
          "ec2:CreateTags"
        ]
        Resource = [
          "arn:aws:ec2:${var.region}:${data.aws_caller_identity.current.account_id}:security-group/*",
          "arn:aws:ec2:${var.region}:${data.aws_caller_identity.current.account_id}:subnet/*"
        ]
        Condition = {
          StringEquals = {
            "aws:RequestTag/kubernetes.io/cluster/${var.cluster_name}" = "owned"
          }
        }
      },
      {
        Effect = "Allow"
        Action = [
          "ec2:DeleteTags"
        ]
        Resource = [
          "arn:aws:ec2:${var.region}:${data.aws_caller_identity.current.account_id}:security-group/*",
          "arn:aws:ec2:${var.region}:${data.aws_caller_identity.current.account_id}:subnet/*"
        ]
        Condition = {
          StringEquals = {
            "aws:ResourceTag/kubernetes.io/cluster/${var.cluster_name}" = ["owned", "shared"]
          }
        }
      },
      {
        Effect = "Allow"
        Action = [
          "elasticloadbalancing:DescribeLoadBalancers",
          "elasticloadbalancing:DescribeLoadBalancerAttributes",
          "elasticloadbalancing:DescribeListeners",
          "elasticloadbalancing:DescribeListenerAttributes",
          "elasticloadbalancing:DescribeListenerCertificates",
          "elasticloadbalancing:DescribeSSLPolicies",
          "elasticloadbalancing:DescribeRules",
          "elasticloadbalancing:DescribeTargetGroups",
          "elasticloadbalancing:DescribeTargetGroupAttributes",
          "elasticloadbalancing:DescribeTargetHealth",
          "elasticloadbalancing:DescribeTags"
        ]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "elasticloadbalancing:CreateLoadBalancer",
          "elasticloadbalancing:CreateTargetGroup"
        ]
        Resource = "*"
        Condition = {
          StringEquals = {
            "aws:RequestTag/kubernetes.io/cluster/${var.cluster_name}" = "owned"
          }
        }
      },
      {
        Effect = "Allow"
        Action = [
          "elasticloadbalancing:DeleteLoadBalancer",
          "elasticloadbalancing:DeleteTargetGroup",
          "elasticloadbalancing:ModifyLoadBalancerAttributes",
          "elasticloadbalancing:ModifyTargetGroup",
          "elasticloadbalancing:ModifyTargetGroupAttributes",
          "elasticloadbalancing:SetIpAddressType",
          "elasticloadbalancing:SetSecurityGroups",
          "elasticloadbalancing:SetSubnets"
        ]
        Resource = "*"
        Condition = {
          StringEquals = {
            "aws:ResourceTag/kubernetes.io/cluster/${var.cluster_name}" = "owned"
          }
        }
      },
      {
        Effect = "Allow"
        Action = [
          "elasticloadbalancing:CreateListener",
          "elasticloadbalancing:DeleteListener",
          "elasticloadbalancing:CreateRule",
          "elasticloadbalancing:DeleteRule",
          "elasticloadbalancing:ModifyListener",
          "elasticloadbalancing:ModifyRule",
          "elasticloadbalancing:AddListenerCertificates",
          "elasticloadbalancing:RemoveListenerCertificates"
        ]
        Resource = "*"
        Condition = {
          StringEquals = {
            "aws:ResourceTag/kubernetes.io/cluster/${var.cluster_name}" = "owned"
          }
        }
      },
      {
        Effect = "Allow"
        Action = [
          "elasticloadbalancing:RegisterTargets",
          "elasticloadbalancing:DeregisterTargets"
        ]
        Resource = "*"
        Condition = {
          StringEquals = {
            "aws:ResourceTag/kubernetes.io/cluster/${var.cluster_name}" = "owned"
          }
        }
      },
      {
        Effect = "Allow"
        Action = [
          "elasticloadbalancing:AddTags"
        ]
        Resource = [
          "arn:aws:elasticloadbalancing:${var.region}:${data.aws_caller_identity.current.account_id}:loadbalancer/*",
          "arn:aws:elasticloadbalancing:${var.region}:${data.aws_caller_identity.current.account_id}:targetgroup/*",
          "arn:aws:elasticloadbalancing:${var.region}:${data.aws_caller_identity.current.account_id}:listener/net/*/*",
          "arn:aws:elasticloadbalancing:${var.region}:${data.aws_caller_identity.current.account_id}:listener/app/*/*"
        ]
        Condition = {
          StringEquals = {
            "aws:RequestTag/kubernetes.io/cluster/${var.cluster_name}" = "owned"
          }
        }
      },
      {
        Effect = "Allow"
        Action = [
          "elasticloadbalancing:RemoveTags"
        ]
        Resource = [
          "arn:aws:elasticloadbalancing:${var.region}:${data.aws_caller_identity.current.account_id}:loadbalancer/*",
          "arn:aws:elasticloadbalancing:${var.region}:${data.aws_caller_identity.current.account_id}:targetgroup/*",
          "arn:aws:elasticloadbalancing:${var.region}:${data.aws_caller_identity.current.account_id}:listener/net/*/*",
          "arn:aws:elasticloadbalancing:${var.region}:${data.aws_caller_identity.current.account_id}:listener/app/*/*"
        ]
        Condition = {
          StringEquals = {
            "aws:ResourceTag/kubernetes.io/cluster/${var.cluster_name}" = "owned"
          }
        }
      },
      {
        Effect = "Allow"
        Action = [
          "iam:CreateServiceLinkedRole"
        ]
        Resource = "*"
        Condition = {
          StringEquals = {
            "iam:AWSServiceName" = "elasticloadbalancing.amazonaws.com"
          }
        }
      },
      {
        Effect = "Allow"
        Action = [
          "acm:DescribeCertificate",
          "acm:ListCertificates",
          "cognito-idp:DescribeUserPoolClient"
        ]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "wafv2:GetWebACL",
          "wafv2:GetWebACLForResource",
          "wafv2:AssociateWebACL",
          "wafv2:DisassociateWebACL"
        ]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "shield:GetSubscriptionState",
          "shield:DescribeProtection",
          "shield:CreateProtection",
          "shield:DeleteProtection"
        ]
        Resource = "*"
      }
    ]
  })
}

# -----------------------------------------------------------------------------
# AWS Load Balancer Controller Helm Release
# -----------------------------------------------------------------------------

resource "helm_release" "aws_load_balancer_controller" {
  name       = "aws-load-balancer-controller"
  repository = "https://aws.github.io/eks-charts"
  chart      = "aws-load-balancer-controller"
  namespace  = "kube-system"
  timeout    = 600
  wait       = true

  set {
    name  = "clusterName"
    value = var.cluster_name
  }

  set {
    name  = "region"
    value = var.region
  }

  set {
    name  = "vpcId"
    value = aws_vpc.superplane.id
  }

  set {
    name  = "serviceAccount.create"
    value = "true"
  }

  set {
    name  = "serviceAccount.name"
    value = "aws-load-balancer-controller"
  }

  set {
    name  = "serviceAccount.annotations.eks\\.amazonaws\\.com/role-arn"
    value = aws_iam_role.aws_load_balancer_controller.arn
  }

  # Configure controller to tag resources with cluster tag for security
  set {
    name  = "defaultTags.kubernetes\\.io/cluster/${var.cluster_name}"
    value = "owned"
  }

  depends_on = [
    aws_eks_node_group.superplane,
    aws_iam_role_policy.aws_load_balancer_controller
  ]
}

# Wait for the AWS Load Balancer Controller webhook to be fully ready
resource "time_sleep" "wait_for_alb_controller" {
  depends_on = [helm_release.aws_load_balancer_controller]

  create_duration = "60s"
}

# -----------------------------------------------------------------------------
# cert-manager Helm Release
# -----------------------------------------------------------------------------

resource "helm_release" "cert_manager" {
  name             = "cert-manager"
  repository       = "https://charts.jetstack.io"
  chart            = "cert-manager"
  namespace        = "cert-manager"
  create_namespace = true
  timeout          = 600
  wait             = true

  set {
    name  = "installCRDs"
    value = "true"
  }

  depends_on = [
    time_sleep.wait_for_alb_controller
  ]
}

# -----------------------------------------------------------------------------
# Traefik Ingress Controller with NLB
# -----------------------------------------------------------------------------

resource "helm_release" "traefik" {
  name             = "traefik"
  repository       = "https://traefik.github.io/charts"
  chart            = "traefik"
  version          = "41.7.0"
  namespace        = "traefik"
  create_namespace = true
  timeout          = 1200 # 20 minutes - NLB deletion can take longer than creation

  values = [yamlencode({
    ingressClass = {
      isDefaultClass = false
    }
    providers = {
      kubernetesIngress = {
        ingressClass = "traefik"
      }
    }
    service = {
      type = "LoadBalancer"
      annotations = {
        "service.beta.kubernetes.io/aws-load-balancer-type"            = "external"
        "service.beta.kubernetes.io/aws-load-balancer-nlb-target-type" = "ip"
        "service.beta.kubernetes.io/aws-load-balancer-scheme"          = "internet-facing"
      }
    }
  })]

  depends_on = [
    time_sleep.wait_for_alb_controller
  ]
}

# -----------------------------------------------------------------------------
# SuperPlane Helm Release
# -----------------------------------------------------------------------------

resource "helm_release" "superplane" {
  name             = "superplane"
  repository       = "oci://ghcr.io/superplanehq"
  chart            = "superplane-chart"
  version          = var.superplane_chart_version
  namespace        = var.superplane_namespace
  create_namespace = false
  timeout          = 1200

  # Database configuration
  set {
    name  = "database.secretName"
    value = "superplane-db-credentials"
  }

  set {
    name  = "database.host"
    value = aws_db_instance.superplane.address
  }

  set {
    name  = "database.port"
    value = "5432"
  }

  set {
    name  = "database.username"
    value = var.db_username
  }

  set_sensitive {
    name  = "database.password"
    value = local.db_password
  }

  set {
    name  = "database.ssl"
    value = "true"
  }

  set {
    name  = "database.local.enabled"
    value = "false"
  }

  # Image configuration
  set {
    name  = "image.registry"
    value = "ghcr.io/superplanehq"
  }

  set {
    name  = "image.name"
    value = "superplane"
  }

  set {
    name  = "image.tag"
    value = var.superplane_image_tag
  }

  set {
    name  = "image.pullPolicy"
    value = "IfNotPresent"
  }

  # Share blobs across the API, workers, and runner pods.
  set {
    name  = "blobStorage.provider"
    value = "s3"
  }

  set {
    name  = "blobStorage.bucket"
    value = aws_s3_bucket.blobs.id
  }

  set {
    name  = "blobStorage.region"
    value = var.region
  }

  set {
    name  = "serviceAccount.create"
    value = "true"
  }

  set {
    name  = "serviceAccount.annotations.eks\\.amazonaws\\.com/role-arn"
    value = aws_iam_role.blob_storage.arn
  }

  # Integrated Runner API and workers share active logs through EFS.
  set {
    name  = "runner.api.enabled"
    value = "true"
  }

  set {
    name  = "runner.activeLogs.existingClaim"
    value = kubernetes_persistent_volume_claim_v1.runner_active_logs.metadata[0].name
  }

  # Domain configuration
  set {
    name  = "domain.name"
    value = var.domain_name
  }

  # Ingress configuration for Traefik
  set {
    name  = "ingress.enabled"
    value = "true"
  }

  set {
    name  = "ingress.className"
    value = "traefik"
  }

  # SSL configuration with cert-manager
  set {
    name  = "ingress.ssl.enabled"
    value = "true"
  }

  set {
    name  = "ingress.ssl.type"
    value = "cert-manager"
  }

  set {
    name  = "ingress.ssl.certManager.issuerRef.name"
    value = "letsencrypt-prod"
  }

  set {
    name  = "ingress.ssl.certManager.issuerRef.kind"
    value = "ClusterIssuer"
  }

  set {
    name  = "ingress.ssl.certManager.secretName"
    value = "superplane-tls-secret"
  }

  set {
    name  = "ingress.ssl.certManager.createClusterIssuer"
    value = "true"
  }

  set {
    name  = "ingress.ssl.certManager.acme.email"
    value = var.letsencrypt_email
  }

  # Authentication (disabled by default)
  set {
    name  = "authentication.github.enabled"
    value = "false"
  }

  set {
    name  = "authentication.google.enabled"
    value = "false"
  }

  # Telemetry (disabled by default)
  set {
    name  = "telemetry.opentelemetry.enabled"
    value = "false"
  }

  set {
    name  = "telemetry.opentelemetry.endpoint"
    value = ""
  }

  # Secrets
  set {
    name  = "session.secretName"
    value = "superplane-session"
  }

  set {
    name  = "jwt.secretName"
    value = "superplane-jwt"
  }

  set {
    name  = "encryption.secretName"
    value = "superplane-encryption"
  }

  set {
    name  = "oidc.secretName"
    value = "superplane-oidc"
  }

  set {
    name  = "podSecurityContext.runAsNonRoot"
    value = "true"
  }

  set {
    name  = "podSecurityContext.runAsUser"
    value = "65534"
  }

  set {
    name  = "podSecurityContext.fsGroup"
    value = "65534"
  }

  set {
    name  = "securityContext.allowPrivilegeEscalation"
    value = "false"
  }

  set {
    name  = "securityContext.readOnlyRootFilesystem"
    value = "true"
  }

  set {
    name  = "securityContext.runAsNonRoot"
    value = "true"
  }

  set {
    name  = "securityContext.capabilities.drop[0]"
    value = "ALL"
  }

  set {
    name  = "securityContext.seccompProfile.type"
    value = "RuntimeDefault"
  }

  depends_on = [
    kubernetes_namespace.superplane,
    kubernetes_secret.db_credentials,
    kubernetes_secret.session,
    kubernetes_secret.jwt,
    kubernetes_secret.encryption,
    kubernetes_secret.oidc,
    helm_release.cert_manager,
    helm_release.traefik,
    aws_iam_role_policy.blob_storage,
    aws_s3_bucket_public_access_block.blobs,
    aws_s3_bucket_server_side_encryption_configuration.blobs,
    aws_db_instance.superplane
  ]
}
