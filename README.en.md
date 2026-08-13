# Hostkey | Terraform Provider

Manage [Hostkey](https://hostkey.com/) infrastructure (VPS and dedicated) through [InvAPI](https://hostkey.com/documentation/apidocs/api_index/) using HCL and Terraform plans.

Русская версия: [README.md](README.md).

More about Terraform: [developer.hashicorp.com/terraform](https://developer.hashicorp.com/terraform/docs).

## Documentation

Resource and data source reference lives under [`docs/`](docs/) (Terraform Registry pages).

### Resources

* [hostkey_server](docs/resources/server.md) — order and manage a server
* [hostkey_server_ip](docs/resources/server_ip.md) — additional IPv4
* [hostkey_ssh_key](docs/resources/ssh_key.md) — SSH key in account storage
* [hostkey_dns_domain](docs/resources/dns_domain.md) — DNS zone
* [hostkey_dns_record](docs/resources/dns_record.md) — DNS record

### Data sources

* [hostkey_presets](docs/data-sources/presets.md) — preset list
* [hostkey_preset](docs/data-sources/preset.md) — single preset by id
* [hostkey_oses](docs/data-sources/oses.md) — operating systems
* [hostkey_traffic_plans](docs/data-sources/traffic_plans.md) — traffic plans
* [hostkey_software](docs/data-sources/software.md) — marketplace software
* [hostkey_ssh_keys](docs/data-sources/ssh_keys.md) — account SSH keys
* [hostkey_dns_domains](docs/data-sources/dns_domains.md) — DNS domains

Examples: [`examples/`](examples/).

## Quick start

### 1. Install Terraform

Follow the [official install guide](https://developer.hashicorp.com/terraform/tutorials/aws-get-started/install-cli). Requires **Terraform >= 1.0**.

### 2. Create a configuration

For example, directory `hostkey-terraform` with `main.tf`:

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
  region = "RU" # or COM — billing/API endpoint (.ru / .com), not the DC
}

resource "hostkey_server" "web" {
  preset_name       = "vm.pico"
  location_name     = "NL"
  os_name           = "Ubuntu 22.04"
  traffic_plan_name = "3 TB / 1 Gbps VM"
  deploy_period     = "monthly"
  root_pass         = var.root_pass
}

# Dedicated — different preset and traffic plan (not VM):
# resource "hostkey_server" "dedic" {
#   preset_name       = "v2-promo"
#   location_name     = "NL"
#   os_name           = "Ubuntu 22.04"
#   traffic_plan_name = "1Gbps 50TB - FREE"
#   # traffic_plan_name = "1Gbps unmetered (10000 P)"
#   deploy_period     = "monthly"
#   root_pass         = var.root_pass
# }
```

VPS and dedicated use **different** traffic plan names. Confirm via `data.hostkey_traffic_plans` / `data.hostkey_presets`. See [docs/resources/server.md](docs/resources/server.md).

For local development without the Registry: `go install` and [dev_overrides](examples/dev-terraform.rc) — see [CONTRIBUTING.md](CONTRIBUTING.md).

### 3. API key

Create a key in InvAPI: **Configuration → API keys**  
([Hostkey docs](https://hostkey.com/documentation/controlpanel/apikey/)).

```bash
export HOSTKEY_API_KEY="your-key"
```

```powershell
$env:HOSTKEY_API_KEY = "your-key"
```

Or in configuration:

```hcl
provider "hostkey" {
  region  = "RU"
  api_key = var.hostkey_api_key
}
```

Aliases: `HOSTKEY_API_TOKEN` for the key; `HOSTKEY_BASE_URL` / `HOSTKEY_API_URL` for the InvAPI base URL.

**Note:** `region` selects the InvAPI endpoint (`.ru` / `.com`). The data center is `location_name` on the resource (for example `NL`, `RU`).

### 4. Plan / Apply / Destroy

```bash
terraform init
terraform plan
terraform apply
terraform destroy
```

Ordering a server is **billed**. Deploy can take from tens of minutes up to about 90 minutes.  
`destroy` requests service cancellation (`whmcs/request_cancellation`). Optional `cancellation_type` / `cancellation_reason` on `hostkey_server` control cancel behaviour.

## Authentication

| Method | Notes |
|--------|--------|
| `HOSTKEY_API_KEY` or `HOSTKEY_API_TOKEN` | Preferred |
| `provider.api_key` | Explicit in HCL (use a variable; do not commit secrets) |

The provider exchanges the account key for a short-lived session token (`auth/login`) and attaches it to InvAPI calls.

## Development and release

* Contributing: [CONTRIBUTING.md](CONTRIBUTING.md)
* Cutting a release: [RELEASE.md](RELEASE.md)
* Security: [SECURITY.md](SECURITY.md)
* Changelog: [CHANGELOG.md](CHANGELOG.md)
* License: [MPL-2.0](LICENSE)

## License

MPL-2.0
