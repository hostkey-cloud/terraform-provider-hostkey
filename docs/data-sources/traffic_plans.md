---
page_title: "hostkey_traffic_plans Data Source - hostkey"
subcategory: ""
description: |-
  Lists Hostkey traffic plans.
---

# hostkey_traffic_plans (Data Source)

Lists traffic plans (`traffic_plans/list`). InvAPI often requires `location`. The provider requests this list **without** a customer session token (sending a token frequently breaks the call).

VPS and dedicated presets use **different** plan names. Examples (exact strings change — always check the list):

| Kind | Example `name` |
|------|----------------|
| VPS | `3 TB / 1 Gbps VM` |
| Dedicated | `1Gbps 50TB - FREE` |
| Dedicated | `1Gbps unmetered (10000 P)` |

Pass `instance_id` (preset id) to see plans compatible with that preset.

```hcl
data "hostkey_traffic_plans" "nl" {
  location = "NL"
}

# Plans compatible with a dedicated preset (pass preset id from presets/list):
data "hostkey_traffic_plans" "for_dedic" {
  location    = "NL"
  instance_id = 12345 # e.g. id of v2-promo
  name        = "1Gbps"
}
```

## Argument Reference

### Optional

- `location` (String) DC location (recommended).
- `instance_id` (Number) Preset id for compatible plans.
- `server_id` (Number) Existing server id.
- `name` (String) Substring filter on plan name.

### Read-Only

- `traffic_plans` — list of matching plans (`id`, `name`, …).
