---
page_title: "hostkey_server Resource - hostkey"
subcategory: ""
description: |-
  Orders and manages a Hostkey server via InvAPI.
---

# hostkey_server (Resource)

Orders a **VPS or dedicated** server with `eq/order_instance`, waits for deploy, manages hostname, tags, power, reboot and reinstall. Destroy calls `whmcs/request_cancellation`.

There is no separate “dedic” resource: use any preset from `presets/list` (for example `vm.pico` or `v2-promo`) and a traffic plan compatible with that preset.

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

Dedicated presets use **different** traffic plan names than VMs. Typical dedic plans (confirm via `data.hostkey_traffic_plans`):

- `1Gbps 50TB - FREE`
- `1Gbps unmetered (10000 P)`

```hcl
resource "hostkey_server" "dedic" {
  preset_name       = "v2-promo"
  location_name     = "NL"
  os_name           = "Ubuntu 22.04"
  traffic_plan_name = "1Gbps 50TB - FREE"
  # traffic_plan_name = "1Gbps unmetered (10000 P)"
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

List plans for a location (and optionally for a preset id) with [`hostkey_traffic_plans`](../data-sources/traffic_plans.md). Names must match the catalog exactly (case-insensitive).

## Argument Reference

### Required

- `location_name` (String) Data-center code (`NL`, `US`, `FI`, `DE`, `RU`, …). Not the same as provider `region`.
- `root_pass` (String, Sensitive) Root password (8–30 chars: upper, lower, digit, and one of `% - _ +`; no `@`/`#`). Change triggers reinstall.

### Optional

- `preset_name` / `preset_id` — catalog preset (name preferred): VPS (`vm.*`) or dedicated (e.g. `v2-promo`). Change forces replace.
- `os_name` / `os_id` — OS. Change triggers reinstall.
- `soft_name` / `soft_id` — marketplace software. Change triggers reinstall.
- `traffic_plan_name` / `traffic_plan_id` — traffic plan for that preset (VM vs dedic names differ). Change forces replace.
- `hostname` — server hostname (rename via InvAPI).
- `ssh_key` — public key for deploy/reinstall. Change triggers reinstall.
- `post_install_script`, `own_os`, `root_size`, `os_template` — install options; changes trigger reinstall.
- `deploy_period` — `hourly`, `monthly`, `quarterly`, `semi-annually`, `annually`. Forces replace.
- `deploy_notify` — email when deploy finishes.
- `ipv4_amount`, `vlan`, `private_vlan`, `custom_domain`, `deploy_options`, `extra_order_params` — order-time options (mostly force replace).
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
- Do not use a VM traffic plan (e.g. `3 TB / 1 Gbps VM`) with a dedicated preset — resolve names from the catalog for that location/preset first.
