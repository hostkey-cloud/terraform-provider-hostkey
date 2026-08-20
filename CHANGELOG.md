# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.7] - 2026-08-20

### Fixed

- `hostkey_server`: `main_ipv4` and `status` use `UseStateForUnknown`, so a no-op plan no longer perpetually shows `1 to change` with only those fields as `(known after apply)`.
- `hostkey_server`: when both `os_name`/`soft_name`/`traffic_plan_name` and an explicit conflicting `*_id` are set in HCL, plan fails with a clear catalog conflict message instead of `Provider produced invalid plan`.
- `hostkey_server`: pending deploy timeout on Update now emits a **warning** (state kept as `pending:<invoice>`), matching Create — avoids failed apply / replace pressure that could drop a paid order from state on destroy.
- `hostkey_server`: after create, if InvAPI did not apply `hostname`, provider attempts `eq/rename_server` and warns on residual drift; Read records live hostname from `eq/show` when available.
- `hostkey_server`: Read removes the resource from state when InvAPI reports the server as not found (cancelled/deleted outside Terraform), instead of failing every plan.
- `hostkey_server`: plan warns when `location_name` is unknown so catalog checks are deferred to apply (instead of a silent skip).
- `hostkey_server`: import-safe RequiresReplace — null→known after import no longer forces replace (AUD-001).
- `hostkey_server`: Create generates a unique default `hostname` when unset and serializes snapshot+`order_instance` (AUD-005) to reduce pending mis-correlation under parallelism.
- InvAPI: `APIError` messages redact secrets in Message/Result at construct and in `Error()`.

### Changed

- `root_size` is documented and validated as a **percentage** of total disk (1–100), matching InvAPI — not GB. The earlier GB-vs-`hdd` capacity check was incorrect and was removed.
- Pending wait polls log InvAPI status hints via `tflog` (`TF_LOG=INFO`) — Terraform CLI still hardcodes the `Still creating...` line and cannot show custom status there.
- Create with `ssh_key` emits a warning to verify key login (InvAPI does not expose authorized_keys confirmation).

## [0.1.6] - 2026-08-19

### Fixed

- `hostkey_server` Create: when `eq/list` / `eq/update_servers` exposes exactly one new server id after the pre-order snapshot, apply now links it immediately and does not wait for `eq/show` hostname fields to appear. This fixes a case where the server was already created/active but Terraform kept printing `Still creating...`.

### Added

- Example: `examples/resources/hostkey_server_pending_resume/` for reproducing and verifying pending-create resume/link behavior during local testing.

## [0.1.5] - 2026-08-19

### Fixed

- `hostkey_server` Create: pending deploy no longer waits forever when the server is already active but callback data is incomplete or missing. Apply can now finish via a safe `eq/list` fallback with hostname disambiguation, without blindly adopting unrelated servers.

## [0.1.4] - 2026-08-19

### Fixed

- `hostkey_server`: interrupted create (`pending:<invoice>`) no longer plans as no-op. Next apply waits for **this** invoice/callback (`deploy_keys[invoice]`), not the first new `eq/list` id. Transient `eq/update_servers` errors are retried until timeout. Empty/failed pre-order `eq/list` refuses `order_instance`.
- InvAPI client: `eq/order_instance` is never HTTP-retried (avoids duplicate paid orders after timeout/5xx).
- `hostkey_dns_record` destroy: `pdns/delete_dns` includes record **content** (and MX priority when set) so destroy removes one row, not every record of that type on the name.
- `hostkey_ssh_key` Read: drop state only when the key is missing; keep state on InvAPI/network errors (avoids recreating the account default key after a timeout).
- InvAPI HTTP client: follow redirects only on the same origin (307/308 no longer replay `token`/`root_pass` to a third host). `base_url` must be HTTPS except localhost. Login JSON `invapi` is applied only for Hostkey hosts and never downgrades TLS.
- Diagnostics: login errors no longer dump response bodies; `token`/`key`/`password`/`root_pass` are redacted in truncated HTTP bodies.
- `hostkey_server` reinstall: send only install fields (`os_id`, software, `root_pass`, SSH, RAID/LVM, scripts) — not location, traffic plan, extra IPv4, VLAN, or IPv6.
- `hostkey_server` plan: changing only `os_name` / `soft_name` / `traffic_plan_name` now syncs the matching `*_id` (computed ids from state no longer block catalog verify).
- `hostkey_server` reinstall: if `WaitForCallback` fails, the next apply resumes waiting and does not start a second reinstall (prevents double disk wipes).
- `hostkey_server`: added bounded length validators for `os_template`, `deploy_options`, `post_install_script` to avoid unbounded client-side payloads.
- Release workflow: pin GitHub Actions to commit SHAs; `persist-credentials: false` on checkout.
- CI: acc helpers in `provider_test.go` are behind `//go:build acceptance` (same as `acc_test.go`) so `unused` lint does not fail default `golangci-lint`

### Added

- `hostkey_server` plan: warning when install-time fields trigger reinstall (disk wipe) even though Terraform shows update in-place.
- `hostkey_server` plan: add an `os_name`-scoped attribute warning for disk wipe on OS change.
- `hostkey_server` plan: warn when `ipv4_amount > 1` because extra IPv4 addresses may be billed.
- `hostkey_ssh_key` plan: warn when `default = true` because future server deploys may use the account default key automatically.
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
