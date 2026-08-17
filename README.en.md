# Hostkey | Terraform Provider

[![Terraform Registry](https://img.shields.io/badge/registry-hostkey--cloud%2Fhostkey-623CE4)](https://registry.terraform.io/providers/hostkey-cloud/hostkey/latest)

Terraform provider for [Hostkey](https://hostkey.com/): VPS, dedicated, GPU, and DNS via [InvAPI](https://hostkey.com/documentation/apidocs/api_index/).

Русский: [README.md](README.md) · Registry: [`hostkey-cloud/hostkey`](https://registry.terraform.io/providers/hostkey-cloud/hostkey/latest)

## Documentation

Full attribute reference is under [`docs/`](docs/) ([Terraform Registry](https://registry.terraform.io/providers/hostkey-cloud/hostkey/latest/docs)). Examples: [`examples/`](examples/).

### Resources

| Resource | Purpose |
|----------|---------|
| [`hostkey_server`](docs/resources/server.md) | Order and manage a server (VPS, dedic, GPU, vGPU) |
| [`hostkey_server_ip`](docs/resources/server_ip.md) | Extra IPv4 on a server |
| [`hostkey_ssh_key`](docs/resources/ssh_key.md) | SSH public key in InvAPI account storage |
| [`hostkey_dns_domain`](docs/resources/dns_domain.md) | DNS zone (PowerDNS) |
| [`hostkey_dns_record`](docs/resources/dns_record.md) | Record in a DNS zone |

### Data sources

| Data source | Purpose |
|-------------|---------|
| [`hostkey_presets`](docs/data-sources/presets.md) | Preset list (`presets/list`) |
| [`hostkey_preset`](docs/data-sources/preset.md) | Single preset by id or name |
| [`hostkey_oses`](docs/data-sources/oses.md) | OS images for a preset or server |
| [`hostkey_traffic_plans`](docs/data-sources/traffic_plans.md) | Traffic plans for a location / preset |
| [`hostkey_software`](docs/data-sources/software.md) | Marketplace software for a preset |
| [`hostkey_ssh_keys`](docs/data-sources/ssh_keys.md) | Account SSH keys |
| [`hostkey_dns_domains`](docs/data-sources/dns_domains.md) | Account DNS zones |

One `hostkey_server` resource covers the whole catalog: `vm.*`, `vds.*`, `bm.*`, `gpu.*`, `vgpu.*`. Dedicated / GPU — [docs/resources/server.md](docs/resources/server.md).

## Requirements

* [Terraform](https://developer.hashicorp.com/terraform/tutorials/aws-get-started/install-cli) **>= 1.0**
* Account-wide InvAPI API key (`Any`): [documentation](https://hostkey.com/documentation/account/api_key_account/)
* Server orders are **billed**; deploy may take up to ~90 minutes

## Quick start

### 1. Configuration

Create a project directory and `main.tf`:

```hcl
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
  # api_key from HOSTKEY_API_KEY (below) or explicitly: api_key = var.hostkey_api_key
}

variable "hostkey_region" {
  type        = string
  description = "InvAPI endpoint: COM (.com) or RU (.ru). Not the same as location_name (DC)."
  default     = "COM"
}

variable "root_pass" {
  type        = string
  sensitive   = true
  description = "Root password (8–30 chars; see docs/resources/server.md)."
}

# Confirm catalog names before ordering (read-only, free):
data "hostkey_presets" "pico" {
  location = "NL"
  name     = "vm.pico"
}

data "hostkey_traffic_plans" "vm" {
  location    = "NL"
  instance_id = data.hostkey_presets.pico.presets[0].id
}

resource "hostkey_server" "web" {
  preset_name       = "vm.pico"
  location_name     = "NL"
  os_name           = "Ubuntu 22.04"
  traffic_plan_name = "3 TB / 1 Gbps VM"
  deploy_period     = "monthly"
  root_pass         = var.root_pass
  power_state       = "on"

  # On destroy: 0 = end of billing period, 1 = immediate (when allowed)
  cancellation_type   = 1
  cancellation_reason = "terraform"

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
```

Copy [`examples/basic/terraform.tfvars.example`](examples/basic/terraform.tfvars.example) → `terraform.tfvars` (gitignored, **do not commit**):

```hcl
root_pass = "StrongPass1%"
```

### 2. API key

InvAPI → **Username → API keys**: [hostkey.com](https://hostkey.com/documentation/account/api_key_account/) · [invapi.hostkey.com](https://invapi.hostkey.com).

**Option 1 — environment variable (recommended):**

```bash
export HOSTKEY_API_KEY="your-key"
```

```powershell
$env:HOSTKEY_API_KEY = "your-key"
```

**Option 2 — provider block** (add a variable and `terraform.tfvars`):

```hcl
variable "hostkey_api_key" {
  type      = string
  sensitive = true
}

provider "hostkey" {
  region  = var.hostkey_region
  api_key = var.hostkey_api_key
}
```

Env aliases: `HOSTKEY_API_TOKEN`. URL override: `HOSTKEY_BASE_URL` / `HOSTKEY_API_URL`.

### 3. Init, validate, plan, apply

```bash
terraform init
terraform validate   # Success! The configuration is valid.
terraform plan
terraform apply
```

Terraform prints the plan and asks for confirmation — type **`yes`** and press Enter.

Orders are **billed**. Deploy is asynchronous (often tens of minutes; default create timeout 90m).

### 4. Destroy

```bash
terraform destroy
```

Confirm with **`yes`**. Calls `whmcs/request_cancellation` using `cancellation_type` / `cancellation_reason` on the resource.

## Hostkey specifics (InvAPI)

* Provider **`region`** selects the API endpoint (`invapi.hostkey.com` / `.ru`); schema default is **`COM`**. Resource **`location_name`** is the data center (`NL`, `US`, `RU`, …).
* **`preset_name` / `os_name` / `traffic_plan_name`** must match **InvAPI** exactly, not short panel labels (`bm.v2-promo`, not `v2-promo`).
* Before ordering: `data.hostkey_presets` + `data.hostkey_traffic_plans` with **`instance_id`** = preset id.
* Dedicated catalogs often have **two rows with the same `name` and different `price`** — use a panel hint (`- FREE`, `(10000 P)`) or `traffic_plan_id`.
* Set **`disk_mirror`** only when `presets/list` shows **2+ disks** for the preset; omit on single-disk presets (including `bm.v2-promo`).
* **`extra_order_params`** is closed: any key fails plan validation (order fields are typed attributes).
* BM / GPU / vGPU, RAID, IPv6, reinstall — [docs/resources/server.md](docs/resources/server.md).

Local build without the Registry: `go install` + [dev_overrides](examples/dev-terraform.rc) — [CONTRIBUTING.md](CONTRIBUTING.md).

## Import

```bash
terraform import hostkey_server.web 12345
```

Import by numeric InvAPI id — see [Registry: hostkey_server → Import](https://registry.terraform.io/providers/hostkey-cloud/hostkey/latest/docs/resources/server#import).

## Troubleshooting

See [Registry: Troubleshooting](https://registry.terraform.io/providers/hostkey-cloud/hostkey/latest/docs#troubleshooting).

## Development

* [CONTRIBUTING.md](CONTRIBUTING.md)
* [SECURITY.md](SECURITY.md)
* [CHANGELOG.md](CHANGELOG.md)
* License [MPL-2.0](LICENSE)
