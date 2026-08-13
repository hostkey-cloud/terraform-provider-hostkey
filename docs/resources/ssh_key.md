---
page_title: "hostkey_ssh_key Resource - hostkey"
subcategory: ""
description: |-
  SSH public key in InvAPI account storage.
---

# hostkey_ssh_key (Resource)

Manages a public SSH key in InvAPI account storage (`ssh_keys`). Customer tokens typically cannot edit existing keys — changes to `name` / `key` / `default` force **replace**.

This is separate from `hostkey_server.ssh_key`, which injects a key during OS install. Use the same public key string in both if you want the key stored and installed.

## Example Usage

```hcl
resource "hostkey_ssh_key" "deploy" {
  name = "tf-deploy"
  key  = file("~/.ssh/id_ed25519.pub")
}
```

## Argument Reference

### Required

- `name` (String) Display name.
- `key` (String, Sensitive) Public key material (`ssh-ed25519`, `ssh-rsa`, …).

### Optional

- `default` (Boolean) Make this the account default key.

### Read-Only

- `id` (String) InvAPI key id.
- `created` (String) Creation timestamp when returned.

## Import

```shell
terraform import hostkey_ssh_key.deploy 123
```
