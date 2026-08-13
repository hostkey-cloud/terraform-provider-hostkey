# Security

## Reporting

Report vulnerabilities privately to the maintainers (GitHub Security Advisories preferred when the repository is public). Do not open public issues that include API keys, tokens, or customer server details.

## Credentials

- Prefer `HOSTKEY_API_KEY` / `HOSTKEY_API_TOKEN` in the environment.
- Never commit API keys, `terraform.tfvars`, or state files.
- Session tokens from `auth/login` are short-lived and kept only in process memory.

## Sensitive state

| Attribute | Resource | Notes |
|-----------|----------|--------|
| `root_pass` | `hostkey_server` | Required by InvAPI order/reinstall; marked Sensitive |
| `key` | `hostkey_ssh_key` | Public key material; marked Sensitive |

Protect remote state. Rotate passwords and keys if state is exposed.

## Operational safety

Orders and cancellations hit **production** billing. Acceptance tests (`TF_ACC=1`) create disposable paid resources — use a dedicated key when possible.
