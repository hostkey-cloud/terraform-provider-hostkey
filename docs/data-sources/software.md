---
page_title: "hostkey_software Data Source - hostkey"
subcategory: ""
description: |-
  Lists Hostkey marketplace software.
---

# hostkey_software (Data Source)

Lists marketplace software (`software/list`) compatible with a preset or server.

```hcl
data "hostkey_software" "nl" {
  location    = "NL"
  instance_id = data.hostkey_presets.nl.presets[0].id
}
```

## Argument Reference

### Optional

- `location` (String) DC location.
- `instance_id` (Number) Preset id.
- `server_id` (Number) Existing server id.
- `bill_period` (String) Billing period filter (`monthly` / `hourly`).
- `name` (String) Substring filter on software name.

### Read-Only

- `software` — list of objects with `id`, `name`, `active`.
