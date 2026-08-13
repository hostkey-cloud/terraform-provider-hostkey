# Release

How to publish `hostkey-cloud/hostkey` to the [Terraform Registry](https://registry.terraform.io/).

## Prerequisites

1. Public GitHub repository: `github.com/hostkey-cloud/terraform-provider-hostkey`
2. Registry namespace ownership for **`hostkey-cloud`**
3. GPG key for checksum signing
4. GitHub Actions secrets: `GPG_PRIVATE_KEY`, `PASSPHRASE`

## Before tagging

- [ ] No API keys, `*.tfvars`, or Terraform state in the tree
- [ ] [CHANGELOG.md](CHANGELOG.md) updated for the version
- [ ] `go test ./...` and CI green
- [ ] Optional paid smoke on a disposable server only (create → refresh → plan no-op → destroy)

## Cut a release

```bash
git tag v0.1.0
git push origin v0.1.0
```

[`.github/workflows/release.yml`](.github/workflows/release.yml) runs GoReleaser and attaches signed artifacts to the GitHub Release.

## Registry

1. [Publish a provider](https://registry.terraform.io/publish/provider)
2. Select the GitHub repository
3. Wait for the tag to sync

Users then:

```hcl
terraform {
  required_providers {
    hostkey = {
      source  = "hostkey-cloud/hostkey"
      version = "~> 0.1"
    }
  }
}
```

Guide: [Publishing providers](https://developer.hashicorp.com/terraform/registry/providers/publishing).

## Local snapshot (no publish)

```bash
make release
```
