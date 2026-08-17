---
page_title: "hostkey_preset Data Source - hostkey"
subcategory: ""
description: |-
  Single Hostkey preset by id.
---

# hostkey_preset (Data Source)

Reads one preset (`presets/show`). Preset ids differ by account and catalog — resolve from [`hostkey_presets`](presets.md) first.

```hcl
data "hostkey_presets" "pico" {
  location = "NL"
  name     = "vm.pico"
}

data "hostkey_preset" "pico" {
  id = data.hostkey_presets.pico.presets[0].id
}
```

## Argument Reference

### Required

- `id` (Number) Preset id.

### Read-Only

- `name` (String) Preset name (e.g. `vm.pico`).
- `description` (String) Hardware description.
- `locations` (String) Comma-separated available locations.
