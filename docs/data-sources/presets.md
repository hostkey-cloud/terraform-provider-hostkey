---
page_title: "hostkey_presets Data Source - hostkey"
subcategory: ""
description: |-
  Lists Hostkey server presets.
---

# hostkey_presets (Data Source)

Lists presets from InvAPI (`presets/list`), optionally filtered by location and name.

Names are the InvAPI `name` field, not the panel short label. Prefixes:

- `vm.*` — VPS
- `vds.*` — virtual dedicated
- `bm.*` — instant dedicated (`bm.v2-promo`, not `v2-promo`)
- `gpu.*` — dedicated GPU (`gpu.v2-a5000`, `gpu.v3-4090t`, …)
- `vgpu.*` — VDS with GPU

Each preset includes `id`, `name`, `description`, `locations`. Use `description` and HDD fields from [hostkey_preset](preset.md) / InvAPI `presets/list` to infer disk count before setting `disk_mirror` on [hostkey_server](../resources/server.md) (RAID needs 2+ disks in catalog).

```hcl
data "hostkey_presets" "nl" {
  location = "NL"
  name     = "vm.pico"
}

data "hostkey_presets" "gpu" {
  location = "NL"
  name     = "gpu.v2-a5000"
}
```

## Argument Reference

### Optional

- `location` (String) DC location filter (e.g. `NL`).
- `name` (String) Exact preset name filter.

### Read-Only

- `presets` — list of objects with `id`, `name`, `description`, `locations`. Disk layout hints (`hdd`, description) affect whether `disk_mirror` is valid on order — see [hostkey_server](../resources/server.md).
