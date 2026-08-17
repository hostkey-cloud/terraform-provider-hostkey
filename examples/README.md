# Examples

| Path | Purpose |
|------|---------|
| [provider](provider/) | Minimal provider block |
| [basic](basic/) | Server + presets data source (billed apply) |
| [data-sources/catalog](data-sources/catalog/) | Presets / OS / traffic plans (read-only) |
| [resources/hostkey_server](resources/hostkey_server/) | Full server example |
| [resources/hostkey_ssh_key](resources/hostkey_ssh_key/) | SSH key storage |
| [resources/hostkey_dns_domain](resources/hostkey_dns_domain/) | DNS zone |
| [dev-terraform.rc](dev-terraform.rc) | Local `dev_overrides` template |

Copy `*.tfvars.example` → `terraform.tfvars` (gitignored). Prefer `HOSTKEY_API_KEY` in the environment.

**Registry:** [`hostkey-cloud/hostkey`](https://registry.terraform.io/providers/hostkey-cloud/hostkey/latest) — use `terraform init` as in README.

**Local provider build:** `go install` + `dev_overrides` from [dev-terraform.rc](dev-terraform.rc) — [CONTRIBUTING.md](../CONTRIBUTING.md).
