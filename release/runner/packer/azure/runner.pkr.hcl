packer {
  required_version = ">= 1.16.1"

  required_plugins {
    azure = {
      source  = "github.com/hashicorp/azure"
      version = "= 2.3.3"
    }
  }
}

variable "subscription_id" {
  type        = string
  description = "Azure subscription that owns the Compute Gallery."
}

variable "location" {
  type        = string
  description = "Azure region in which to build the image."
  default     = "eastus"
}

variable "gallery_resource_group" {
  type        = string
  description = "Resource group that contains the Compute Gallery."
}

variable "gallery_name" {
  type        = string
  description = "Azure Compute Gallery name."
}

variable "image_name" {
  type        = string
  description = "Gallery image definition name. Must match the architecture."
}

variable "image_version" {
  type        = string
  description = "Gallery image version to publish. Use a unique version for each build."
}

variable "architecture" {
  type        = string
  description = "Target image architecture: amd64 or arm64."

  validation {
    condition     = contains(["amd64", "arm64"], var.architecture)
    error_message = "Architecture must be amd64 or arm64."
  }
}

variable "vm_size" {
  type        = string
  description = "VM size used to build the image."

  validation {
    condition     = trimspace(var.vm_size) != ""
    error_message = "VM size must not be empty."
  }
}

variable "claude_code_version" {
  type        = string
  description = "Claude Code version installed in the images."
  default     = "2.1.282"
}

variable "opencode_version" {
  type        = string
  description = "OpenCode version installed in the images."
  default     = "1.18.31"
}

variable "codex_version" {
  type        = string
  description = "Codex version installed in the images."
  default     = "0.144.5"
}

variable "playwright_version" {
  type        = string
  description = "Playwright version installed in the images."
  default     = "1.63.0"
}

locals {
  image_sku = var.architecture == "arm64" ? "24_04-lts-arm64" : "24_04-lts-gen2"
}

source "azure-arm" "ubuntu" {
  use_azure_cli_auth = true
  subscription_id    = var.subscription_id
  location           = var.location
  vm_size            = var.vm_size

  os_type         = "Linux"
  image_publisher = "Canonical"
  image_offer     = "0001-com-ubuntu-server-noble"
  image_sku       = local.image_sku
  image_version   = "latest"

  secure_boot_enabled = true
  vtpm_enabled        = true
  security_type       = "TrustedLaunch"

  azure_tags = {
    Architecture = var.architecture
    ManagedBy    = "packer"
    Name         = "superplane-runner-${var.architecture}"
    OS           = "ubuntu-24.04"
  }

  shared_image_gallery_destination {
    subscription         = var.subscription_id
    resource_group       = var.gallery_resource_group
    gallery_name         = var.gallery_name
    image_name           = var.image_name
    image_version        = var.image_version
    storage_account_type = "Standard_LRS"

    target_region {
      name = var.location
    }
  }
}

build {
  name    = "superplane-runner"
  sources = ["source.azure-arm.ubuntu"]

  provisioner "shell" {
    script = "${path.root}/../aws/scripts/install.sh"
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
    script          = "${path.root}/../aws/scripts/install-media-tools.sh"
    execute_command = "chmod +x {{ .Path }}; sudo -E sh -c '{{ .Vars }} {{ .Path }}'"
  }

  provisioner "shell" {
    execute_command = "chmod +x {{ .Path }}; {{ .Vars }} sudo -E sh '{{ .Path }}'"
    inline = [
      "/usr/sbin/waagent -force -deprovision+user && export HISTSIZE=0 && sync",
    ]
    skip_clean = true
  }
}
