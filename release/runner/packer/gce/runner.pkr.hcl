packer {
  required_version = ">= 1.16.1"

  required_plugins {
    googlecompute = {
      source  = "github.com/hashicorp/googlecompute"
      version = "~> 1.1"
    }
  }
}

variable "project_id" {
  type        = string
  description = "GCP project that builds and stores the image."

  validation {
    condition     = trimspace(var.project_id) != ""
    error_message = "Project ID must not be empty."
  }
}

variable "zone" {
  type        = string
  description = "Zone for the temporary build instance."
  default     = "us-central1-a"
}

variable "network" {
  type        = string
  description = "VPC network for the temporary build instance."
  default     = "default"
}

variable "subnetwork" {
  type        = string
  description = "Subnetwork for the temporary build instance. Empty uses the network default for the zone region."
  default     = ""
}

variable "use_iap" {
  type        = bool
  description = "Connect over Identity-Aware Proxy and give the build instance no external IP address."
  default     = false
}

variable "source_revision" {
  type        = string
  description = "SuperPlane revision recorded as an image label. Use lowercase letters, digits, dashes, or underscores."
  default     = "local"
}

variable "image_name_prefix" {
  type        = string
  description = "Prefix for the image names and image family."
  default     = "superplane-runner"
}

variable "architecture" {
  type        = string
  description = "Target image architecture: amd64 or arm64."

  validation {
    condition     = contains(["amd64", "arm64"], var.architecture)
    error_message = "Architecture must be amd64 or arm64."
  }
}

variable "machine_type" {
  type        = string
  description = "Machine type for the build instance. Use an Arm machine type, such as t2a-standard-4, for arm64."

  validation {
    condition     = trimspace(var.machine_type) != ""
    error_message = "Machine type must not be empty."
  }
}

variable "claude_code_version" {
  type        = string
  description = "Claude Code version installed in the image."
  default     = "2.1.282"
}

variable "opencode_version" {
  type        = string
  description = "OpenCode version installed in the image."
  default     = "1.18.31"
}

variable "codex_version" {
  type        = string
  description = "Codex version installed in the image."
  default     = "0.144.5"
}

variable "playwright_version" {
  type        = string
  description = "Playwright version installed in the image."
  default     = "1.63.0"
}

locals {
  timestamp = formatdate("YYYYMMDD-hhmmss", timestamp())
}

source "googlecompute" "ubuntu" {
  project_id              = var.project_id
  zone                    = var.zone
  machine_type            = var.machine_type
  network                 = var.network
  subnetwork              = var.subnetwork != "" ? var.subnetwork : null
  use_iap                 = var.use_iap
  use_internal_ip         = var.use_iap
  omit_external_ip        = var.use_iap
  source_image_family     = "ubuntu-2404-lts-${var.architecture}"
  source_image_project_id = ["ubuntu-os-cloud"]
  ssh_username            = "packer"

  disk_size = 30
  disk_type = "pd-balanced"

  enable_secure_boot          = true
  enable_vtpm                 = true
  enable_integrity_monitoring = true

  image_name         = "${var.image_name_prefix}-${var.architecture}-${local.timestamp}"
  image_family       = "${var.image_name_prefix}-${var.architecture}"
  image_architecture = var.architecture == "arm64" ? "ARM64" : "X86_64"
  image_description  = "SuperPlane runner image for linux/${var.architecture}, built ${local.timestamp}"
  image_labels = {
    architecture    = var.architecture
    managed_by      = "packer"
    os              = "ubuntu-2404"
    source_revision = var.source_revision
  }
}

build {
  name    = "superplane-runner"
  sources = ["source.googlecompute.ubuntu"]

  provisioner "shell" {
    script          = "${path.root}/scripts/prepare.sh"
    execute_command = "chmod +x {{ .Path }}; sudo -E sh -c '{{ .Vars }} {{ .Path }}'"
    pause_before    = "15s"
  }

  provisioner "shell" {
    script = "${path.root}/scripts/install.sh"
    environment_vars = [
      "CLAUDE_CODE_VERSION=${var.claude_code_version}",
      "OPENCODE_VERSION=${var.opencode_version}",
      "CODEX_VERSION=${var.codex_version}",
      "PLAYWRIGHT_VERSION=${var.playwright_version}",
    ]
    execute_command = "chmod +x {{ .Path }}; sudo -E sh -c '{{ .Vars }} {{ .Path }}'"
  }

  provisioner "shell" {
    script          = "${path.root}/../aws/scripts/install-media-tools.sh"
    execute_command = "chmod +x {{ .Path }}; sudo -E sh -c '{{ .Vars }} {{ .Path }}'"
  }

  provisioner "shell" {
    script          = "${path.root}/scripts/cleanup.sh"
    execute_command = "chmod +x {{ .Path }}; sudo -E sh -c '{{ .Vars }} {{ .Path }}'"
  }

  post-processor "manifest" {
    output     = "${path.root}/manifest.json"
    strip_path = true
  }
}
