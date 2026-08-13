terraform {
  required_providers {
    hostkey = {
      source  = "hkadm/hostkey"
      version = "~> 0.1"
    }
  }
  required_version = ">= 1.0"
}

provider "hostkey" {
  # Prefer env: HOSTKEY_API_KEY (or HOSTKEY_API_TOKEN)
  # region selects invapi.hostkey.com vs .ru when base_url is unset
  region = "RU"

  # Optional knobs:
  # http_timeout = 60
  # max_retries  = 3
  # token_ttl    = 3600
  # base_url     = "https://invapi.hostkey.ru/"
}
