## Summary

- Describe what changed and why.

## Security checklist

- [ ] No secrets committed (`.env`, keys, tokens)
- [ ] CI security jobs pass (`gitleaks`, `gosec`, `govulncheck`, `semgrep`, `trivy`)
- [ ] Dockerfile lint and app lint jobs pass
- [ ] If config changed, `docker compose config` is valid

## Verification

- [ ] Local checks executed (or justify why not)
- [ ] Relevant logs/screenshots attached if useful
