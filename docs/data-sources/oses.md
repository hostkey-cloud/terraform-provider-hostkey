---
page_title: "hostkey_oses Data Source - hostkey"
subcategory: ""
description: |-
  Lists Hostkey operating systems.
---

# hostkey_oses (Data Source)

Lists OS images (`os/list`).

```hcl
data "hostkey_oses" "nl" {
  location    = "NL"
  instance_id = data.hostkey_presets.nl.presets[0].id
}
```

## Argument Reference

### Optional

- `location` (String) DC location.
- `instance_id` (Number) Preset id — filter OS compatible with this preset.
- `server_id` (Number) Existing server id for compatible OS list.
- `bill_period` (String) Billing period filter when required by InvAPI.
- `name` (String) Optional name filter.

### Read-Only

- `oses` — list of objects with `id`, `name`, `active`.
