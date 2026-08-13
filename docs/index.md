---
page_title: "hostkey Provider"
description: |-
  Terraform provider for Hostkey InvAPI (servers, SSH keys, IPs, DNS).
---

# Hostkey Provider

Manage Hostkey infrastructure via [InvAPI](https://hostkey.com/documentation/apidocs/api_index/).

Русский гайд: [README.md](../README.md) · English: [README.en.md](../README.en.md).

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
  region = "RU"
  # api_key from HOSTKEY_API_KEY / HOSTKEY_API_TOKEN, or set explicitly
}
```

## Schema

### Optional

- `api_key` (String, Sensitive) Account InvAPI API key (Configuration → API keys). Env: `HOSTKEY_API_KEY` or `HOSTKEY_API_TOKEN`.
- `base_url` (String) InvAPI base URL. Overrides `region` when set. Env: `HOSTKEY_BASE_URL` or `HOSTKEY_API_URL`.
- `region` (String) `COM` (default) or `RU` when `base_url` is not set. Selects billing/API endpoint, not the data center.
- `token_ttl` (Number) Session token TTL in seconds for `auth/login` (default `3600`).
- `http_timeout` (Number) HTTP client timeout in seconds (default `60`).
- `max_retries` (Number) Max attempts for retryable InvAPI HTTP failures (default `3`).
