terraform {
  required_providers {
    hostkey = {
      source = "registry.terraform.io/hkadm/hostkey"
    }
  }
}

provider "hostkey" {
  region = "RU"
}

variable "ssh_public_key" {
  type      = string
  sensitive = true
}

resource "hostkey_ssh_key" "example" {
  name    = "tf-example"
  key     = var.ssh_public_key
  default = false
}

output "ssh_key_id" {
  value = hostkey_ssh_key.example.id
}
