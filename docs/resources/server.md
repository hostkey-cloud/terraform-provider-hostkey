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

Always pair the preset with a traffic plan for that **preset id** (`instance_id` in [hostkey_traffic_plans](../data-sources/traffic_plans.md)). GPU dedic often uses unmetered-style plans; vGPU often uses dedic-style names (`1Gbps 50TB - FREE`). Confirm names and prices in the catalog.

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

Disk layout (panel «Disk configuration» / «Конфигурация дисков») maps to `disk_mirror` and `no_lvm`. Omit `root_size` for «100% от загрузочного диска».

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

List plans with [`hostkey_traffic_plans`](../data-sources/traffic_plans.md).

### Dedicated GPU (`gpu.*`) and vGPU (`vgpu.*`)

Same resource and attributes as VPS/dedicated — change **`preset_name`** and pick OS/traffic from the catalog for that preset (`instance_id`). Examples: `gpu.v2-a5000` + CUDA OS images; `vgpu.v2-a4000` + plans like `1Gbps 50TB - FREE`. Do **not** reuse VM traffic plan names.

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
- `ipv4_amount`, `ipv6_block`, `vlan`, `private_vlan`, `custom_domain`, `deploy_options`, `extra_order_params` — order-time options (mostly force replace). `extra_order_params` is **closed** (any key fails plan validation; use typed attributes instead). `deploy_options` and `os_template` are forwarded as-is — invalid values fail at order/reinstall time. `ipv6_block`: dedicated catalog `virtual=0` and location **NL or US**. InvAPI does not expose a per-preset IPv6 checkbox — set it only when the panel shows it.
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

Import by InvAPI server id (numeric only).

After import, Terraform state contains mainly **`id`**, **`main_ipv4`**, **`status`**, and power-related fields from `eq/show`. Catalog fields (`preset_name`, `os_name`, `traffic_plan_name`, computed ids) are **not** filled from the panel automatically — set them in HCL to document intent.

**First apply after import:** declaring `os_name`, `traffic_plan_name`, `root_pass`, etc. when those attributes were **empty in state** does **not** trigger reinstall. Reinstall runs when an install-time field **changes from a value already stored in state**, or when you set a non-empty **`reinstall_trigger`**.

To avoid surprise drift, either:

- keep HCL aligned with the live server and accept that some attributes stay unknown in state until a reinstall, or
- set **`reinstall_trigger`** once to align the OS/software with HCL (disk wipe).

## Notes

- `root_pass` is stored in Terraform state (sensitive).
- Pending create only resumes this resource's own `pending:<invoice>` — foreign Pending orders are never adopted.
- RAID / disk layout: [Hostkey RAID docs](https://hostkey.com/documentation/technical/exist_server_using/raid_create/) ([RU](https://hostkey.ru/documentation/technical/exist_server_using/raid_create/)).
