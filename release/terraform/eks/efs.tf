# -----------------------------------------------------------------------------
# EFS File System for Runner Active Logs
# -----------------------------------------------------------------------------

resource "aws_efs_file_system" "runner_active_logs" {
  encrypted        = true
  performance_mode = "generalPurpose"
  throughput_mode  = "elastic"

  tags = {
    Name = "${var.cluster_name}-runner-active-logs"
  }
}

# The access point gives every application replica the same writable directory
# and enforces the UID/GID used by the SuperPlane pods.
resource "aws_efs_access_point" "runner_active_logs" {
  file_system_id = aws_efs_file_system.runner_active_logs.id

  posix_user {
    gid = 65534
    uid = 65534
  }

  root_directory {
    path = "/runner-active-logs"

    creation_info {
      owner_gid   = 65534
      owner_uid   = 65534
      permissions = "0770"
    }
  }

  tags = {
    Name = "${var.cluster_name}-runner-active-logs"
  }
}

# -----------------------------------------------------------------------------
# EFS Network Access
# -----------------------------------------------------------------------------

resource "aws_security_group" "efs" {
  name        = "${var.cluster_name}-efs-sg"
  description = "Allow EKS nodes to mount the runner active-log file system"
  vpc_id      = aws_vpc.superplane.id

  ingress {
    description     = "NFS from EKS nodes"
    from_port       = 2049
    to_port         = 2049
    protocol        = "tcp"
    security_groups = [aws_eks_cluster.superplane.vpc_config[0].cluster_security_group_id]
  }

  tags = {
    Name = "${var.cluster_name}-efs-sg"
  }
}

resource "aws_efs_mount_target" "runner_active_logs" {
  count = length(aws_subnet.private)

  file_system_id  = aws_efs_file_system.runner_active_logs.id
  subnet_id       = aws_subnet.private[count.index].id
  security_groups = [aws_security_group.efs.id]
}
