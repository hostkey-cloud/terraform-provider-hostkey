---
page_title: "hostkey_dns_domains Data Source - hostkey"
subcategory: ""
description: |-
  Lists DNS domains in the Hostkey account.
---

# hostkey_dns_domains (Data Source)

Lists pdns domains available to the account.

```hcl
data "hostkey_dns_domains" "all" {}
```

## Argument Reference

### Read-Only

- `domains` — list of domains (`id`, `name`, …).
