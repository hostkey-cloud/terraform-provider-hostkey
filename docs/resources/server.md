---
page_title: "hostkey_server Resource - hostkey"
subcategory: ""
description: |-
  Orders and manages a Hostkey server via InvAPI.
---

# hostkey_server (Resource)

Orders a server with `eq/order_instance`, waits for deploy, manages hostname, tags, power, reboot and reinstall. Destroy calls `whmcs/request_cancellation`.

Changing OS / software / `root_pass` / `ssh_key` (or `reinstall_trigger`) **reinstalls the same server id** — disk is wiped. Changes to preset, location, traffic plan or billing period force **replace** (new order).

## Example Usage

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

## Argument Reference

### Required

- `location_name` (String) Data-center code (`NL`, `US`, `FI`, `DE`, `RU`, …). Not the same as provider `region`.
- `root_pass` (String, Sensitive) Root password (8–30 chars: upper, lower, digit, and one of `% - _ +`; no `@`/`#`). Change triggers reinstall.

### Optional

- `preset_name` / `preset_id` — catalog preset (name preferred). Change forces replace.
- `os_name` / `os_id` — OS. Change triggers reinstall.
- `soft_name` / `soft_id` — marketplace software. Change triggers reinstall.
- `traffic_plan_name` / `traffic_plan_id` — traffic plan. Change forces replace.
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
