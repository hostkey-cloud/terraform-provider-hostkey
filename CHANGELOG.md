# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed

- CI: acc helpers in `provider_test.go` are behind `//go:build acceptance` (same as `acc_test.go`) so `unused` lint does not fail default `golangci-lint`

### Added

- README RU/EN and Registry troubleshooting: install via Yandex Cloud public Terraform provider mirror when `registry.terraform.io` is blocked (`terraform-mirror.yandexcloud.net`; no Yandex Cloud account)

## [0.1.3] - 2026-08-17

### Fixed

- `hostkey_server` Create: reject server ids from order callback / `WaitForNewServerID` that were already in `eq/list` before `order_instance` (`acceptNewServerID` + known-id check on deploy_keys path)
- Catalog verify: OS/traffic/software must be **active** in InvAPI lists (no fallback to inactive rows); cross-check `*_name` vs `*_id` when both are set
- `ModifyPlan`: error when the provider is not configured (`api_key` / env missing) instead of skipping catalog validation
- Import / first apply: declaring install fields (`os_name`, `traffic_plan_name`, `root_pass`, …) when state had them empty no longer triggers unintended reinstall
- Remove dead `OrderInstanceRequest.Extra` from InvAPI client (order fields are typed; `extra_order_params` stays closed in schema)

### Changed

- README RU/EN: Timeweb-style quick start; dedicated/GPU details in `docs/resources/server.md`; import/troubleshooting link to Registry docs
- `docs/index.md`: provider schema and troubleshooting (no duplicate resource index — Registry sidebar)
- `docs/resources/server.md`: import notes, dedicated example, compact GPU/vGPU section (removed duplicate HCL blocks)
- `docs/data-sources/presets.md`, `traffic_plans.md`: disk-count hints; traffic example without hardcoded preset id
- `examples/README.md`: Registry is published; local dev path clarified
- Plan warnings for `os_template` and `deploy_options`; order response no longer logged with raw InvAPI body
- Acceptance tests behind `//go:build acceptance` — `go test ./...` skips paid deploy even when `TF_ACC=1` is set (`make testacc` / `-tags=acceptance`)
- Sanitize public ids in tests and docs; `.gitignore` for `SECURITY_AUDIT*.md` and `acc-*.log`; `CONTRIBUTING.md`

## [0.1.2] - 2026-08-14

### Fixed

- CI: `gofmt` on `validators_common.go`; unused assignment in `catalog_resolve_test.go`

## [0.1.1] - 2026-08-14

### Added

- `hostkey_server`: bare-metal disk options `disk_mirror` (hba/raid0/raid1/raid10), `no_lvm`, and network `ipv6_block` (NL/US) with validation and bm.* docs example
- Schema and plan validation across provider/resources/data sources: InvAPI URLs, location codes, IPv4/IPv6, DNS zones/records, SSH keys, root password, server IDs, import IDs, cross-field server/DNS checks
- Plan: `disk_mirror` is checked against InvAPI `presets/list` disk count (`hdd`/`description`); 1 disk → omit the field; RAID10 needs 4+ disks. `extra_order_params` is closed (any key rejected, not forwarded)
- Catalog hardening: plan/apply re-check preset/OS/traffic/software against InvAPI lists; exact catalog names only; duplicate same-price traffic names require `traffic_plan_id` or a price hint

### Fixed

- Resolve dedicated traffic plans when InvAPI returns duplicate names: accept panel-style hints (`- FREE`, `(10000 P)`); ambiguous same-price rows require `traffic_plan_id`
- Documentation links: account API keys (`account/api_key_account`), RU README on `hostkey.ru`, EN/Registry on `hostkey.com`

### Changed

- Docs: dedicated uses `bm.v2-promo`; traffic plan examples aligned with InvAPI (`1Gbps 50TB - FREE`, `1Gbps unmetered (10000 P)`); GPU (`gpu.*`) and vGPU (`vgpu.*`) examples on `hostkey_server`
- Documentation reorganized for public release (README RU/EN, Registry `docs/`, consolidated contributor docs)

## [0.1.0] - TBD

First public release (tag `v0.1.0`) after GitHub + Registry setup.

### Added

- Provider `hostkey` for Hostkey InvAPI (`RU` / `COM`)
- Resources: `hostkey_server`, `hostkey_server_ip`, `hostkey_ssh_key`, `hostkey_dns_domain`, `hostkey_dns_record`
- Data sources: presets, preset, oses, traffic_plans, software, ssh_keys, dns_domains
- Server: catalog name → id resolve, tags, hostname, power, reboot, reinstall, cancellation
- Provider knobs: `http_timeout`, `max_retries`; env aliases `HOSTKEY_API_TOKEN`, `HOSTKEY_API_URL`
- GoReleaser + GitHub Actions (CI / Release)
