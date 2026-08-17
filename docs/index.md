---
page_title: "hostkey Provider"
description: |-
  Terraform provider for Hostkey InvAPI (servers, SSH keys, IPs, DNS).
---

# Hostkey Provider

Manage Hostkey infrastructure via [InvAPI](https://hostkey.com/documentation/apidocs/api_index/) ([RU index](https://hostkey.ru/documentation/apidocs/api_index/)).  
Account API keys: InvAPI → **Username → API keys** ([COM](https://hostkey.com/documentation/account/api_key_account/) · [RU](https://hostkey.ru/documentation/account/api_key_account/)).

Quick start and catalog notes: [GitHub README (RU)](https://github.com/hostkey-cloud/terraform-provider-hostkey/blob/main/README.md) · [README (EN)](https://github.com/hostkey-cloud/terraform-provider-hostkey/blob/main/README.en.md).

## Example Usage

```hcl
terraform {
  required_providers {
    hostkey = {
      source  = "hostkey-cloud/hostkey"
      version = "~> 0.1"
    }
  }
}

provider "hostkey" {
  region = "COM"
  # api_key from HOSTKEY_API_KEY / HOSTKEY_API_TOKEN, or set explicitly
}
```

## Schema

### Optional

- `api_key` (String, Sensitive) Account InvAPI API key. Env: `HOSTKEY_API_KEY` or `HOSTKEY_API_TOKEN`.
- `base_url` (String) InvAPI base URL. Overrides `region` when set. Env: `HOSTKEY_BASE_URL` or `HOSTKEY_API_URL`.
- `region` (String) `COM` (default) or `RU` when `base_url` is not set. Selects billing/API endpoint, not the data center (`location_name` on resources).
- `token_ttl` (Number) Session token TTL in seconds for `auth/login` (default `3600`).
- `http_timeout` (Number) HTTP client timeout in seconds (default `60`).
- `max_retries` (Number) Max attempts for retryable InvAPI HTTP failures (default `3`).

## Troubleshooting

| Error / symptom | What to do |
|-----------------|------------|
| `auth/login: No appropriate servers found` | Use an account API key (`Any`); check provider `region` (RU vs COM) matches your account portal |
| `Catalog verification failed` | Run `terraform plan` with a configured provider; confirm preset/OS/traffic ids via data sources |
| Ambiguous `traffic_plan_name` | List plans with [hostkey_traffic_plans](data-sources/traffic_plans.md) and `instance_id`; use `(10000 P)` / `- FREE` hints or `traffic_plan_id` |
| `pending:<invoice>` id | Deploy still running; re-run `apply` — the provider resumes the same order, it does not re-order |
| Reinstall after import | Declaring install fields on import does not reinstall until those fields differ from state; use [reinstall_trigger](resources/server.md) to force reinstall |
