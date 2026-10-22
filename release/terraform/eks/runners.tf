# -----------------------------------------------------------------------------
# Standalone Fleet Manager and runner VM infrastructure
# -----------------------------------------------------------------------------

resource "aws_security_group" "runners" {
  count       = var.enable_runners ? 1 : 0
  name        = "${var.cluster_name}-runners"
  description = "Outbound-only security group for SuperPlane runner VMs"
  vpc_id      = aws_vpc.superplane.id

  egress {
    description = "Outbound access for releases, AWS APIs, and job dependencies"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.cluster_name}-runners"
  }

  lifecycle {
    precondition {
      condition     = trimspace(var.runner_image_amd64) != ""
      error_message = "Runners need runner_image_amd64. Build the AMI with release/runner/packer/aws/runner.pkr.hcl and set runner_image_amd64 = \"ami-...\". Set enable_runners = false to install without runners."
    }
  }
}

resource "aws_iam_role" "runner" {
  count = var.enable_runners ? 1 : 0
  name  = "${var.cluster_name}-runner"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = "sts:AssumeRole"
      Principal = {
        Service = "ec2.amazonaws.com"
      }
    }]
  })
}

resource "aws_iam_instance_profile" "runner" {
  count = var.enable_runners ? 1 : 0
  name  = "${var.cluster_name}-runner"
  role  = aws_iam_role.runner[0].name
}

resource "aws_iam_role" "fleet_manager" {
  count = var.enable_runners ? 1 : 0
  name  = "${var.cluster_name}-fleet-manager"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = "sts:AssumeRoleWithWebIdentity"
      Principal = {
        Federated = aws_iam_openid_connect_provider.eks.arn
      }
      Condition = {
        StringEquals = {
          "${replace(aws_eks_cluster.superplane.identity[0].oidc[0].issuer, "https://", "")}:aud" = "sts.amazonaws.com"
          "${replace(aws_eks_cluster.superplane.identity[0].oidc[0].issuer, "https://", "")}:sub" = "system:serviceaccount:${var.superplane_namespace}:superplane-fleet-manager"
        }
      }
    }]
  })
}

resource "aws_iam_role_policy" "fleet_manager" {
  count = var.enable_runners ? 1 : 0
  name  = "${var.cluster_name}-fleet-manager"
  role  = aws_iam_role.fleet_manager[0].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "ReadRunnerInstances"
        Effect   = "Allow"
        Action   = ["ec2:DescribeInstances"]
        Resource = "*"
      },
      {
        Sid    = "UseRunnerLaunchResources"
        Effect = "Allow"
        Action = ["ec2:RunInstances"]
        Resource = concat(
          [
            "arn:aws:ec2:${var.region}::image/*",
            "arn:aws:ec2:${var.region}:${data.aws_caller_identity.current.account_id}:key-pair/*",
            aws_security_group.runners[0].arn
          ],
          aws_subnet.private[*].arn
        )
      },
      {
        Sid      = "UseRunnerNetworkInterfaces"
        Effect   = "Allow"
        Action   = ["ec2:RunInstances"]
        Resource = "arn:aws:ec2:${var.region}:${data.aws_caller_identity.current.account_id}:network-interface/*"
        Condition = {
          ArnEquals = {
            "ec2:Subnet" = aws_subnet.private[*].arn
          }
        }
      },
      {
        Sid      = "CreateRunnerInstances"
        Effect   = "Allow"
        Action   = ["ec2:RunInstances"]
        Resource = "arn:aws:ec2:${var.region}:${data.aws_caller_identity.current.account_id}:instance/*"
        Condition = {
          StringEquals = {
            "aws:RequestTag/Name" = "superplane-runner"
          }
          ArnEquals = {
            "ec2:InstanceProfile" = aws_iam_instance_profile.runner[0].arn
          }
        }
      },
      {
        Sid      = "CreateRunnerVolumes"
        Effect   = "Allow"
        Action   = ["ec2:RunInstances"]
        Resource = "arn:aws:ec2:${var.region}:${data.aws_caller_identity.current.account_id}:volume/*"
        Condition = {
          StringEquals = {
            "aws:RequestTag/Name" = "superplane-runner"
          }
        }
      },
      {
        Sid    = "TagRunnerResourcesDuringLaunch"
        Effect = "Allow"
        Action = ["ec2:CreateTags"]
        Resource = [
          "arn:aws:ec2:${var.region}:${data.aws_caller_identity.current.account_id}:instance/*",
          "arn:aws:ec2:${var.region}:${data.aws_caller_identity.current.account_id}:volume/*"
        ]
        Condition = {
          StringEquals = {
            "ec2:CreateAction"    = "RunInstances"
            "aws:RequestTag/Name" = "superplane-runner"
          }
        }
      },
      {
        Sid      = "TerminateManagedRunnerInstances"
        Effect   = "Allow"
        Action   = ["ec2:TerminateInstances"]
        Resource = "arn:aws:ec2:${var.region}:${data.aws_caller_identity.current.account_id}:instance/*"
        Condition = {
          StringEquals = {
            "ec2:ResourceTag/Name" = "superplane-runner"
          }
          ArnEquals = {
            "ec2:InstanceProfile" = aws_iam_instance_profile.runner[0].arn
          }
        }
      },
      {
        Sid      = "PassRunnerRoleOnly"
        Effect   = "Allow"
        Action   = ["iam:PassRole"]
        Resource = aws_iam_role.runner[0].arn
        Condition = {
          StringEquals = {
            "iam:PassedToService" = "ec2.amazonaws.com"
          }
        }
      }
    ]
  })
}
