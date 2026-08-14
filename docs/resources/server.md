---
page_title: "hostkey_server Resource - hostkey"
subcategory: ""
description: |-
  Orders and manages a Hostkey server via InvAPI.
---

# hostkey_server (Resource)

Orders a server with [`eq/order_instance`](https://hostkey.com/documentation/apidocs/eq/#order_instance) ([RU](https://hostkey.ru/documentation/apidocs/eq/#order_instance)), waits for deploy, manages hostname, tags, power, reboot and reinstall. Destroy calls `whmcs/request_cancellation`.

One resource covers the whole catalog. There is no separate GPU/VDS type:

| Prefix | Typical `server_type` | Example `preset_name` |
|--------|----------------------|------------------------|
| `vm.*` | Virtual Private Server | `vm.pico` |
| `vds.*` | Virtual Dedicated Server | `vds.ryzen-8` |
| `bm.*` | Instant Dedicated Server | `bm.v2-promo` |
| `gpu.*` | Dedicated GPU Server | `gpu.v2-a5000` |
| `vgpu.*` | VDS with GPU | `vgpu.v2-a4000` |

Always pair the preset with a traffic plan from `traffic/get_plans` for that **preset id** (`instance_id`). GPU dedic often uses unmetered-style plans; vGPU often looks like dedic (`1Gbps / 50TB`). Confirm names and prices in the catalog.

Changing OS / software / `root_pass` / `ssh_key` (or `reinstall_trigger`) **reinstalls the same server id** — disk is wiped. Changes to preset, location, traffic plan or billing period force **replace** (new order).

## Example Usage

### VPS

```hcl
resource "hostkey_server" "web" {
  preset_name       = "vm.pico"
  location_name     = "NL"
  os_name           = "Ubuntu 22.04"
  traffic_plan_name = "3 TB / 1 Gbps VM"
  deploy_period     = "monthly"
  root_pass         = var.root_pass
  power_state       = "on"
  cancellation_type = 1

  tags = {
    env = "prod"
  }

  timeouts {
    create = "90m"
    update = "90m"
    delete = "30m"
  }
}
```

### Dedicated

Dedicated presets use the same resource. Catalog names differ from the control-panel labels:

| Panel / docs | InvAPI `preset_name` / plan |
|--------------|-----------------------------|
| v2-promo | `bm.v2-promo` |
| 1Gbps unmetered (10000 ₽) | `1Gbps unmetered (10000 P)` (or id from catalog) |
| 1 IPv4 - Free | default (`ipv4_amount` omitted or `1`) |
| IPv6 /64 block | `ipv6_block = true` — **only if the panel shows the checkbox** for this preset (NL/US; not all bm) |

Bare names like `1Gbps unmetered` can match **two** InvAPI rows (different prices). Prefer a price hint (`- FREE`, `(10000 P)`) or `traffic_plan_id`. Confirm with [`hostkey_traffic_plans`](../data-sources/traffic_plans.md) (`instance_id` = preset id).

Disk layout (panel «Конфигурация дисков») maps to `disk_mirror` and `no_lvm`. Omit `root_size` for «100% от загрузочного диска».

Set `disk_mirror` only when InvAPI `presets/list` shows **2+ disks** (`hdd` like `2x960` or description `/2x1TB`). One-disk presets (`hdd=1000`, `/1TB NVMe`) leave panel RAID type empty — **omit** `disk_mirror`; sending `hba` or RAID is not processed. `raid1`/`raid0` need two disks; `raid10` needs four.

```hcl
resource "hostkey_server" "dedic" {
  preset_name       = "bm.v2-promo"
  location_name     = "NL"
  os_name           = "Ubuntu 22.04"
  traffic_plan_name = "1Gbps unmetered (10000 P)"
  deploy_period     = "monthly"
  root_pass         = var.root_pass
  power_state       = "on"
  cancellation_type = 1

  # Catalog shows 1 disk — omit disk_mirror (panel RAID type is empty).
  no_lvm      = true  # classic partitions instead of LVM
  # root_size = 480   # GB; omit = full boot disk

  # Network (panel «Сетевые настройки»)
  ipv4_amount = 1       # default 1 IPv4
  # ipv6_block = true   # only when panel shows «IPv6 /64 block» for this preset (NL/US)

  timeouts {
    create = "90m"
    update = "90m"
    delete = "30m"
  }
}
```

List plans for a location (and optionally for a preset id) with [`hostkey_traffic_plans`](../data-sources/traffic_plans.md).

### Dedicated GPU (`gpu.*`)

Same resource. Pick an OS compatible with the GPU (CUDA images often appear as `Ubuntu 22.04 CUDA 12.8` / similar in `os/list`). Traffic plans for GPU presets are **not** VM plans — list them with `instance_id` = GPU preset id.

```hcl
resource "hostkey_server" "gpu" {
  preset_name       = "gpu.v2-a5000"
  location_name     = "NL"
  os_name           = "Ubuntu 22.04 CUDA 12.8"
  traffic_plan_name = "1Gbps unmetered (10000 P)"
  deploy_period     = "monthly"
  root_pass         = var.root_pass
  power_state       = "on"
  cancellation_type = 1

  timeouts {
    create = "90m"
    update = "90m"
    delete = "30m"
  }
}
```

If that traffic name does not resolve, copy the exact `name` from [`hostkey_traffic_plans`](../data-sources/traffic_plans.md) or use `traffic_plan_id`.

### VDS with GPU (`vgpu.*`)

```hcl
resource "hostkey_server" "vgpu" {
  preset_name       = "vgpu.v2-a4000"
  location_name     = "NL"
  os_name           = "Ubuntu 22.04"
  traffic_plan_name = "1Gbps 50TB - FREE"
  deploy_period     = "monthly"
  root_pass         = var.root_pass
  power_state       = "on"
  cancellation_type = 1

  timeouts {
    create = "90m"
    update = "90m"
    delete = "30m"
  }
}
```

## Argument Reference

### Required

- `location_name` (String) Data-center code (`NL`, `US`, `FI`, `DE`, `RU`, …). Not the same as provider `region`.
- `root_pass` (String, Sensitive) Root password (8–30 chars: upper, lower, digit, and one of `% - _ +`; no `@`/`#`). Change triggers reinstall.

### Optional

- `preset_name` / `preset_id` — catalog preset (name preferred): `vm.*`, `vds.*`, `bm.*`, `gpu.*`, `vgpu.*`. Change forces replace.
- `os_name` / `os_id` — OS. Change triggers reinstall.
- `soft_name` / `soft_id` — marketplace software. Change triggers reinstall.
- `traffic_plan_name` / `traffic_plan_id` — traffic plan for that preset (VM vs dedic names differ; dedic often needs `- FREE` / `(NNNN P)` hints or an id). Change forces replace.
- `hostname` — server hostname (rename via InvAPI).
- `ssh_key` — public key for deploy/reinstall. Change triggers reinstall.
- `post_install_script`, `own_os`, `root_size`, `disk_mirror`, `no_lvm`, `os_template` — install / disk options for bare metal (`bm.*`, `gpu.*`); changes trigger reinstall.
- `deploy_period` — `hourly`, `monthly`, `quarterly`, `semi-annually`, `annually`. Forces replace.
- `deploy_notify` — email when deploy finishes.
- `ipv4_amount`, `ipv6_block`, `vlan`, `private_vlan`, `custom_domain`, `deploy_options`, `extra_order_params` — order-time options (mostly force replace). `extra_order_params` is closed (any key is rejected). `ipv6_block`: dedicated catalog `virtual=0` and location **NL or US**. InvAPI does not expose a per-preset IPv6 checkbox — set it only when the panel shows it.
- `tags` (Map of String) — user tags only (Hostkey system tags are not synced back).
- `power_state` — `on` / `off`. Omit to leave power unmanaged.
- `power_off_hard` — use `eq/hard_off` when turning off.
- `reboot_trigger` — change the string to call `eq/reboot` once.
- `reinstall_trigger` — change the string to force reinstall with current OS/software.
- `cancellation_type` — `0` end of period, `1` immediate (when allowed). Used on destroy.
- `cancellation_reason` — reason for cancellation.
- `poll_interval_seconds` — deploy poll interval (default `15`).
- `timeouts` — `create` / `update` / `delete`.

### Read-Only

- `id` — InvAPI server id, or `pending:<invoice>` while deploy is in progress.
- `main_ipv4` — primary IPv4 after deploy.
- `status` — last known status.
- `invoice` — WHMCS invoice id after Paid.

## Import

```shell
terraform import hostkey_server.web 12345
```

Import by InvAPI server id. Align HCL with the live server so the next plan is empty.

## Notes

- `root_pass` is stored in Terraform state (sensitive).
- After a successful create, a second `apply` should be a no-op if config matches remote.
- Pending create only resumes the resource’s own `pending:<invoice>` — foreign Pending Instant servers are never adopted.
- Do not use a VM traffic plan (e.g. `3 TB / 1 Gbps VM`) with `bm.*` / `gpu.*` / `vgpu.*` — resolve names from the catalog for that location and preset id first.
- `disk_mirror` / `no_lvm` / `root_size` match InvAPI [`eq/order_instance`](https://hostkey.com/documentation/apidocs/eq/#order_instance) ([RU](https://hostkey.ru/documentation/apidocs/eq/#order_instance)) install fields. Disk count is taken from `presets/list` `hdd` + `description`. Available RAID levels depend on hardware (see [RAID docs](https://hostkey.com/documentation/technical/exist_server_using/raid_create/); [RU](https://hostkey.ru/documentation/technical/exist_server_using/raid_create/)).
- `ipv6_block` sends `ipv6=1` at order time for dedicated presets (`virtual=0`) in **NL or US**. InvAPI `presets/list` has no IPv6 flag — if the panel has no checkbox, omit `ipv6_block`.
