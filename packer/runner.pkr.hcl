packer {
  required_version = ">= 1.10.0"
  required_plugins {
    amazon = {
      version = ">= 1.3.0"
      source  = "github.com/hashicorp/amazon"
    }
  }
}

# ---------------------------------------------------------------------------
# Variables
# ---------------------------------------------------------------------------

variable "aws_region" {
  description = "AWS region to build and publish the AMI in."
  default     = "us-east-1"
}

variable "runner_s3_uri_amd64" {
  description = "S3 URI of the amd64 runner binary, e.g. s3://superplane-runner/latest/runner-linux-amd64"
}

variable "runner_s3_uri_arm64" {
  description = "S3 URI of the arm64 runner binary, e.g. s3://superplane-runner/latest/runner-linux-arm64"
}

variable "build_instance_type_amd64" {
  description = "Instance type used for the amd64 build (does not need to match prod)."
  default     = "t3.medium"
}

variable "build_instance_type_arm64" {
  description = "Instance type used for the arm64 build."
  default     = "t4g.medium"
}

variable "ami_name_prefix" {
  description = "Prefix for the resulting AMI names."
  default     = "superplane-runner"
}

# ---------------------------------------------------------------------------
# Locals
# ---------------------------------------------------------------------------

locals {
  timestamp = formatdate("YYYYMMDD-hhmmss", timestamp())
}

# ---------------------------------------------------------------------------
# Sources
# ---------------------------------------------------------------------------

source "amazon-ebs" "ubuntu_amd64" {
  region        = var.aws_region
  instance_type = var.build_instance_type_amd64

  # Ubuntu 24.04 LTS (Noble) amd64 — always pick the latest official Canonical image.
  source_ami_filter {
    filters = {
      name                = "ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-*"
      root-device-type    = "ebs"
      virtualization-type = "hvm"
    }
    owners      = ["099720109477"] # Canonical
    most_recent = true
  }

  ssh_username = "ubuntu"

  ami_name        = "${var.ami_name_prefix}-amd64-${local.timestamp}"
  ami_description = "SuperPlane runner AMI — amd64, Ubuntu 24.04, built ${local.timestamp}"

  tags = {
    Name           = "${var.ami_name_prefix}-amd64"
    BuildTimestamp = local.timestamp
    Arch           = "amd64"
    OS             = "ubuntu-24.04"
    ManagedBy      = "packer"
  }
}

source "amazon-ebs" "ubuntu_arm64" {
  region        = var.aws_region
  instance_type = var.build_instance_type_arm64

  source_ami_filter {
    filters = {
      name                = "ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-arm64-server-*"
      root-device-type    = "ebs"
      virtualization-type = "hvm"
    }
    owners      = ["099720109477"] # Canonical
    most_recent = true
  }

  ssh_username = "ubuntu"

  ami_name        = "${var.ami_name_prefix}-arm64-${local.timestamp}"
  ami_description = "SuperPlane runner AMI — arm64, Ubuntu 24.04, built ${local.timestamp}"

  tags = {
    Name           = "${var.ami_name_prefix}-arm64"
    BuildTimestamp = local.timestamp
    Arch           = "arm64"
    OS             = "ubuntu-24.04"
    ManagedBy      = "packer"
  }
}

# ---------------------------------------------------------------------------
# Builds
# ---------------------------------------------------------------------------

build {
  name    = "superplane-runner-amd64"
  sources = ["source.amazon-ebs.ubuntu_amd64"]

  provisioner "shell" {
    script           = "packer/scripts/install.sh"
    environment_vars = ["RUNNER_S3_URI=${var.runner_s3_uri_amd64}"]
    pause_before     = "15s"
  }

  post-processor "manifest" {
    output     = "packer/manifest.json"
    strip_path = true
  }
}

build {
  name    = "superplane-runner-arm64"
  sources = ["source.amazon-ebs.ubuntu_arm64"]

  provisioner "shell" {
    script           = "packer/scripts/install.sh"
    environment_vars = ["RUNNER_S3_URI=${var.runner_s3_uri_arm64}"]
    pause_before     = "15s"
  }

  post-processor "manifest" {
    output     = "packer/manifest.json"
    strip_path = true
  }
}
