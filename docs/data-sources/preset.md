---
page_title: "hostkey_preset Data Source - hostkey"
subcategory: ""
description: |-
  Single Hostkey preset by id.
---

# hostkey_preset (Data Source)

Reads one preset (`presets/show`).

```hcl
data "hostkey_preset" "pico" {
  id = 108
}
```

## Argument Reference

### Required

- `id` (Number) Preset id.

### Read-Only

- `name` (String) Preset name (e.g. `vm.pico`).
- `description` (String) Hardware description.
- `locations` (String) Comma-separated available locations.
