# Contributing

Thanks for helping improve the Hostkey Terraform provider.

## Development

Requirements: Go **1.26+** (see `go.mod`), Terraform CLI **>= 1.0** for local plan/apply.

```bash
go test ./...
go install -ldflags "-X main.version=dev"
```

For local Terraform against your build, copy [examples/dev-terraform.rc](examples/dev-terraform.rc) to the Terraform CLI config (`~/.terraformrc` or `%APPDATA%\terraform.d\terraform.rc`) and point it at `$(go env GOPATH)/bin`.

Useful Make targets: `make test`, `make build`, `make install`, `make lint`, `make testacc`.

## Layout

| Path | Role |
|------|------|
| `internal/provider` | Framework resources and data sources |
| `internal/invapi` | InvAPI HTTP client and auth |
| `cmd/smoke` | Auth smoke check (read-only InvAPI) |
| `docs/` | Registry documentation |
| `examples/` | Sample configurations |

## Tests

- Unit: `go test ./...` (no API key).
- Smoke: `HOSTKEY_API_KEY=… go run ./cmd/smoke` (read-only InvAPI).
- Acceptance (billed, production InvAPI):

```bash
export TF_ACC=1
export HOSTKEY_API_KEY=…
go test ./internal/provider -v -timeout 120m -run TestAcc
```

DNS acceptance needs `HOSTKEY_ACC_DNS_DOMAIN`. Do not point tests at servers you must not destroy.

## Pull requests

- Keep diffs focused; match existing style.
- Update `docs/` and README when changing user-facing schema.
- Do not commit secrets, state, or personal account IDs.
- CI must pass (`gofmt`, `vet`, lint, unit tests).

## Scope

Features that map poorly to declarative Terraform (S3, backups, ISO as resources, one-shot console/IPMI) should be discussed before implementation.

## License

Contributions are under [MPL-2.0](LICENSE).
