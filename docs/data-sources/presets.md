---
page_title: "hostkey_presets Data Source - hostkey"
subcategory: ""
description: |-
  Lists Hostkey server presets.
---

# hostkey_presets (Data Source)

Lists presets from InvAPI (`presets/list`), optionally filtered by location and name.

```hcl
data "hostkey_presets" "nl" {
  location = "NL"
  name     = "vm.pico"
}
```

## Argument Reference

### Optional

- `location` (String) DC location filter (e.g. `NL`).
- `name` (String) Exact preset name filter.

### Read-Only

- `presets` — list of objects with `id`, `name`, `description`, `locations`.
