---
name: pixelbattle.ops
---

# pixelbattle.ops

Ansible collection for PixelBattle deployment, production operations, and benchmark automation.

## Layout

- `playbooks/` - canonical playbook entry points for deployment, hardening, validation, and benchmarks.
- `roles/deploy/` - single-node Compose deployment role.
- `roles/benchmark/` - benchmark orchestration role.
- `inventory/` - sample inventories for deploy and benchmark flows.
- `group_vars/all/` - non-secret defaults plus `vault.yml.example`.
- `tests/` - CI inventory and syntax fixtures.

## Running

From the repository root:

```bash
ansible-galaxy collection install -r requirements.yml
ansible-playbook -i ansible_collections/pixelbattle/ops/inventory/deploy.yml \
  ansible_collections/pixelbattle/ops/playbooks/deploy-stack.yml
```

Secrets belong in an encrypted `vault.yml` outside version control. The collection includes only
`vault.yml.example`.
