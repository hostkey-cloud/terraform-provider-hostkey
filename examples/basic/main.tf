terraform {
  required_providers {
    hostkey = {
      source  = "hostkey-cloud/hostkey"
      version = "~> 0.1"
    }
  }
  required_version = ">= 1.0"
}

provider "hostkey" {
  region = var.hostkey_region
}

variable "hostkey_region" {
  type    = string
  default = "RU"
}

variable "root_pass" {
  type      = string
  sensitive = true
}

data "hostkey_presets" "all" {
  location = "NL"
}

resource "hostkey_server" "web" {
  preset_name       = "vm.pico"
  location_name     = "NL"
  os_name           = "Ubuntu 22.04"
  traffic_plan_name = "3 TB / 1 Gbps VM"
  deploy_period     = "monthly"
  root_pass         = var.root_pass
  power_state       = "on"
  cancellation_type = 1

  tags = {
    env = "example"
  }

  timeouts {
    create = "90m"
    delete = "30m"
  }
}

output "server_id" {
  value = hostkey_server.web.id
}

output "main_ipv4" {
  value = hostkey_server.web.main_ipv4
}

output "preset_count" {
  value = length(data.hostkey_presets.all.presets)
}
