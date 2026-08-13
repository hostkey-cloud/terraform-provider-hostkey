# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Docs: dedicated examples (`v2-promo`) and dedic traffic plans (`1Gbps 50TB - FREE`, `1Gbps unmetered (10000 P)`) in README and `docs/resources/server.md`
- Documentation reorganized for public release (README RU/EN, Registry `docs/`, consolidated contributor docs)

## [0.1.0] - TBD

First public release (tag `v0.1.0`) after GitHub + Registry setup.

### Added

- Provider `hostkey` for Hostkey InvAPI (`RU` / `COM`)
- Resources: `hostkey_server`, `hostkey_server_ip`, `hostkey_ssh_key`, `hostkey_dns_domain`, `hostkey_dns_record`
- Data sources: presets, preset, oses, traffic_plans, software, ssh_keys, dns_domains
- Server: catalog name → id resolve, tags, hostname, power, reboot trigger, reinstall, cancellation
- Provider knobs: `http_timeout`, `max_retries`; env aliases `HOSTKEY_API_TOKEN`, `HOSTKEY_API_URL`
- GoReleaser + GitHub Actions (CI / Release)
