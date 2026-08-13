---
page_title: "hostkey_traffic_plans Data Source - hostkey"
subcategory: ""
description: |-
  Lists Hostkey traffic plans.
---

# hostkey_traffic_plans (Data Source)

Lists traffic plans (`traffic_plans/list`). InvAPI often requires `location`. The provider requests this list **without** a customer session token (sending a token frequently breaks the call).

```hcl
data "hostkey_traffic_plans" "nl" {
  location = "NL"
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
