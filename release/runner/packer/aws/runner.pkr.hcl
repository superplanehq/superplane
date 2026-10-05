packer {
  required_version = ">= 1.16.1"

  required_plugins {
    amazon = {
      source  = "github.com/hashicorp/amazon"
      version = "= 1.8.2"
    }
  }
}

variable "aws_region" {
  type        = string
  description = "AWS region in which to build the AMIs."
  default     = "us-east-1"
}

variable "ami_name_prefix" {
  type        = string
  description = "Prefix for the AMI names."
  default     = "superplane-runner"
}

variable "architecture" {
  type        = string
  description = "Target AMI architecture: amd64 or arm64."

  validation {
    condition     = contains(["amd64", "arm64"], var.architecture)
    error_message = "Architecture must be amd64 or arm64."
  }
}

variable "instance_type" {
  type        = string
  description = "EC2 instance type used to build the AMI."

  validation {
    condition     = trimspace(var.instance_type) != ""
    error_message = "Instance type must not be empty."
  }
}

variable "claude_code_version" {
  type        = string
  description = "Claude Code version installed in the AMIs."
  default     = "2.1.282"
}

variable "opencode_version" {
  type        = string
  description = "OpenCode version installed in the AMIs."
  default     = "1.18.31"
}

variable "codex_version" {
  type        = string
  description = "Codex version installed in the AMIs."
  default     = "0.144.5"
}

variable "playwright_version" {
  type        = string
  description = "Playwright version installed in the AMIs."
  default     = "1.63.0"
}

locals {
  timestamp = formatdate("YYYYMMDD-hhmmss", timestamp())
}

source "amazon-ebs" "ubuntu" {
  region        = var.aws_region
  instance_type = var.instance_type

  source_ami_filter {
    filters = {
      name                = var.architecture == "arm64" ? "ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-arm64-server-*" : "ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-*"
      root-device-type    = "ebs"
      virtualization-type = "hvm"
    }
    most_recent = true
    owners      = ["099720109477"]
  }

  ssh_username = "ubuntu"

  launch_block_device_mappings {
    device_name           = "/dev/sda1"
    delete_on_termination = true
    volume_size           = 30
    volume_type           = "gp3"
  }

  ami_name        = "${var.ami_name_prefix}-${var.architecture}-${local.timestamp}"
  ami_description = "SuperPlane runner AMI for linux/${var.architecture}, built ${local.timestamp}"

  tags = {
    Architecture = var.architecture
    ManagedBy    = "packer"
    Name         = "${var.ami_name_prefix}-${var.architecture}"
    OS           = "ubuntu-24.04"
  }
}

build {
  name    = "superplane-runner"
  sources = ["source.amazon-ebs.ubuntu"]

  provisioner "shell" {
    script = "${path.root}/scripts/install.sh"
    environment_vars = [
      "CLAUDE_CODE_VERSION=${var.claude_code_version}",
      "OPENCODE_VERSION=${var.opencode_version}",
      "CODEX_VERSION=${var.codex_version}",
      "PLAYWRIGHT_VERSION=${var.playwright_version}",
    ]
    execute_command = "chmod +x {{ .Path }}; sudo -E sh -c '{{ .Vars }} {{ .Path }}'"
    pause_before    = "15s"
  }

  provisioner "shell" {
    script          = "${path.root}/scripts/install-media-tools.sh"
    execute_command = "chmod +x {{ .Path }}; sudo -E sh -c '{{ .Vars }} {{ .Path }}'"
  }

  post-processor "manifest" {
    output     = "${path.root}/manifest.json"
    strip_path = true
  }
}
