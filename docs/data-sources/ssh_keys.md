---
page_title: "hostkey_ssh_keys Data Source - hostkey"
subcategory: ""
description: |-
  Lists SSH keys in the Hostkey account.
---

# hostkey_ssh_keys (Data Source)

Lists keys from InvAPI SSH storage.

```hcl
data "hostkey_ssh_keys" "all" {}
```

## Argument Reference

### Optional

- `name` (String) Optional name filter.

### Read-Only

- `keys` — list of keys (`id`, `name`, `key`, …).
