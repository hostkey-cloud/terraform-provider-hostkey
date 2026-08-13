---
page_title: "hostkey_dns_domain Resource - hostkey"
subcategory: ""
description: |-
  DNS zone in Hostkey pdns.
---

# hostkey_dns_domain (Resource)

Creates or deletes a DNS domain (`pdns/add_domain`, `pdns/delete_domain`). If the zone disappears outside Terraform, the next refresh removes it from state.

## Example Usage

```hcl
resource "hostkey_dns_domain" "app" {
  name = "example.com"
}
```

## Argument Reference

### Required

- `name` (String) Zone FQDN (e.g. `example.com`).

### Optional

- `server_id` (Number) Optional InvAPI server id to associate with the domain.

### Read-Only

- `id` (String) InvAPI domain id.

## Import

```shell
terraform import hostkey_dns_domain.app example.com
```
