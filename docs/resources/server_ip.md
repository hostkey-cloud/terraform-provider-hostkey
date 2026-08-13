---
page_title: "hostkey_server_ip Resource - hostkey"
subcategory: ""
description: |-
  Additional IPv4 address on a Hostkey server.
---

# hostkey_server_ip (Resource)

Adds or removes an extra IPv4 on a server (`net/add_ipv4`, `net/remove_ipv4`). May be billed depending on the account and location.

## Example Usage

```hcl
resource "hostkey_server_ip" "extra" {
  server_id = tonumber(hostkey_server.web.id)
}
```

## Argument Reference

### Required

- `server_id` (Number) InvAPI server id.

### Optional

- `ip` (String) IPv4. Optional on create (InvAPI assigns). Required after create / for import.
- `port` (String) Network port name (default `eth0`).

### Read-Only

- `id` (String) `<server_id>/<ip>`.
- `vlan` (Number) VLAN when returned by InvAPI.

## Import

```shell
terraform import hostkey_server_ip.extra 5860/1.2.3.4
```
