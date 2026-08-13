---
page_title: "hostkey_dns_record Resource - hostkey"
subcategory: ""
description: |-
  DNS record in a Hostkey pdns zone.
---

# hostkey_dns_record (Resource)

Manages a DNS record in a pdns zone (`pdns/add_dns`, `pdns/delete_dns`). The zone must already exist (for example via `hostkey_dns_domain`).

## Example Usage

```hcl
resource "hostkey_dns_record" "www" {
  zone    = hostkey_dns_domain.app.name
  name    = "www"
  type    = "A"
  content = hostkey_server.web.main_ipv4
  ttl     = 300
}
```

## Argument Reference

### Required

- `zone` (String) Zone FQDN.
- `name` (String) Record name relative to the zone (`www`, `@`, …).
- `type` (String) Record type (`A`, `AAAA`, `CNAME`, `MX`, `TXT`, `NS`, …).
- `content` (String) Record value.

### Optional

- `ttl` (Number) TTL in seconds (InvAPI default often `3600`).
- `priority` (Number) Priority for MX/SRV.

### Read-Only

- `id` (String) Synthetic id: `zone/name/type/content`.

## Import

```shell
terraform import hostkey_dns_record.www example.com/www/A/203.0.113.10
```

Import id format: `zone/name/type/content`.
