# Shared blob storage for the API, workers, and runner pods.
resource "aws_s3_bucket" "blobs" {
  bucket_prefix = "${var.cluster_name}-blobs-"
}

resource "aws_s3_bucket_public_access_block" "blobs" {
  bucket = aws_s3_bucket.blobs.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "blobs" {
  bucket = aws_s3_bucket.blobs.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_iam_role" "blob_storage" {
  name = "${var.cluster_name}-blob-storage"

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
          "${replace(aws_eks_cluster.superplane.identity[0].oidc[0].issuer, "https://", "")}:aud" = "sts.amazonaws.com"
          "${replace(aws_eks_cluster.superplane.identity[0].oidc[0].issuer, "https://", "")}:sub" = "system:serviceaccount:${var.superplane_namespace}:superplane"
        }
      }
    }]
  })
}

resource "aws_iam_role_policy" "blob_storage" {
  name = "${var.cluster_name}-blob-storage"
  role = aws_iam_role.blob_storage.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:ListBucket"]
        Resource = aws_s3_bucket.blobs.arn
      },
      {
        Effect   = "Allow"
        Action   = ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"]
        Resource = "${aws_s3_bucket.blobs.arn}/*"
      }
    ]
  })
}
