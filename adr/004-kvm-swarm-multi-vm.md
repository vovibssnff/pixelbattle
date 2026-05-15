# ADR-004: KVM micro-VMs + Docker Swarm for multi-VM distributed deployment

## Status

Accepted

## Date

2026-05-10

## Context

Phase 2 of the research metrics plan (`~/.cursor/plans/pixelbattle_research_metrics_plan_04530117.plan.md`, §5, §7.2, §13.2) requires the upgraded distributed service to run with Redis Cluster masters and backend gateway instances on **separate VMs** so the candidate column of the comparison report contains true cross-instance numbers (gRPC fan-out, per-shard `database_operation_duration_seconds`, `redis_replication_lag_seconds`, `opstream_lag_seconds`).

The existing single-VM `redis-cluster` profile in [docker-compose.yml](../docker-compose.yml) puts all three masters and the single backend on one 16 GB VM. That layout cannot expose:

- cross-VM gRPC RPCs between gateways (everything is in-process / loopback);
- per-shard Redis network behaviour (slot redirects, gossip, replication lag);
- per-VM resource pressure (one ceiling, not five).

Constraints:

- **Hardware budget is fixed**: one physical host with 12 CPU / 24 GB RAM (Phase 1 used the same host as a single VM).
- **Comparable-KPI contract** (plan §2): every comparable signal must measure identically in baseline and candidate runs. New per-host overhead must not bias the comparable metrics, so per-VM caps + scrape paths must mirror Phase 1 service caps.
- **6 VMs are required** by user direction: 3× Redis + 2× Backend + 1× Manager/Monitoring.

## Options Considered

### Option A — Single-node Docker Swarm on the existing host (no separate VMs)

- Pros: zero VM overhead; cluster gossip works on local bridge; all Phase 1 caps fit unchanged; trivial operations.
- Cons: containers share one kernel and one host network namespace; Redis Cluster gossip latency is sub-millisecond regardless of placement, so cross-shard latency is not realistic; gRPC server / client both run on the same kernel, hiding the failure modes the thesis is supposed to expose; Phase 1 vs Phase 2 comparison would be one-VM vs same-one-VM, which weakens the architectural claim.

### Option B — LXC containers as "VMs" on the same physical host

- Pros: ~100 MB overhead per LXC container (~600 MB total); per-namespace network; quick to provision via `lxc launch`.
- Cons: shared kernel — can't observe kernel-level isolation; not a recognisably "VM-shaped" topology in the thesis; chrony / time-skew measurements degrade because all guests inherit the host clock with sub-microsecond drift.

### Option C — KVM micro-VMs + Docker Swarm

- Pros: each VM has its own kernel, its own clock (chrony per VM produces real `node_timex_offset_seconds` skew), its own libvirt-bridge IP; Docker Swarm's overlay network handles service discovery; `deploy.placement.constraints` pin services to node labels; Redis Cluster `--cluster-announce-ip` set to each VM's bridge IP gives realistic slot-redirect behaviour; gRPC fan-out genuinely crosses two kernel boundaries.
- Cons: ~2 GB combined VM overhead leaves 22 GB for services (so Phase 1 per-service caps must shrink: Redis 4 GB → 1.5 GB, backend 8 GB → 4 GB); nested virtualization has to be available on the physical host.

### Option D — Kubernetes (k3s on the same physical host or split across LXC/KVM)

- Pros: richer scheduling, declarative manifests, mature observability story.
- Cons: disproportionate operational complexity for a 6-node academic benchmark fleet; the Phase 2 plan's `deploy.placement.constraints` model maps 1:1 onto a Swarm stack file with far less control-plane overhead.

## Decision

We pick **Option C — KVM micro-VMs + Docker Swarm**.

- 6 KVM guests are provisioned on the host's `default` libvirt network (`192.168.122.0/24`):
  - `vm-manager` 2 vCPU / 6 GB RAM at `192.168.122.10` — Swarm manager; runs Caddy, MongoDB, Prometheus, Grafana, blackbox-exporter.
  - `vm-backend-1` 3 vCPU / 5 GB RAM at `192.168.122.11` — backend gateway replica.
  - `vm-backend-2` 3 vCPU / 5 GB RAM at `192.168.122.12` — backend gateway replica.
  - `vm-redis-1/2/3` 1 vCPU / 2 GB RAM at `192.168.122.13/14/15` — Redis Cluster masters.
  - Total: 11 / 12 vCPU and 22 GB / 24 GB RAM (≈ 2 GB host headroom).
- VMs are created by [deploy/playbooks/kvm-provision.yml](../deploy/playbooks/kvm-provision.yml) using `virt-install` + cloud-init on Ubuntu 24.04 cloud images. Each guest installs Docker CE and chrony from cloud-init `runcmd`.
- The Swarm is formed by [deploy/playbooks/swarm-init.yml](../deploy/playbooks/swarm-init.yml): `docker swarm init --advertise-addr 192.168.122.10` on the manager, then `docker swarm join` on the 5 workers; nodes are labelled `role=manager|backend|redis`.
- Services are deployed via [deploy/stack.yml](../deploy/stack.yml) using `docker stack deploy`. `deploy.placement.constraints` pin each service to the right label.
- Redis Cluster is bootstrapped by [deploy/playbooks/swarm-redis-cluster.yml](../deploy/playbooks/swarm-redis-cluster.yml) — `redis-cli --cluster create` is called with the three VM bridge IPs (not the overlay IPs) so that a returning slot-redirect resolves on the libvirt bridge. Each redis-cN service sets `--cluster-announce-ip $NODE_IP` from the per-task environment.

### Container caps (reduced from Phase 1)

| Service | Phase 1 cap | Phase 2 cap |
|---|---|---|
| Redis (per master) | 4 GB | 1.5 GB |
| Backend (per replica) | 8 GB | 4 GB |
| MongoDB | 1 GB | 1 GB |
| Prometheus | 1 GB | 1 GB |
| Grafana | 512 MB | 512 MB |
| Caddy | 512 MB | 512 MB |
| blackbox-exporter | 128 MB | 128 MB |
| node-exporter | n/a | 64 MB (per node, global service) |

Reduced caps are documented in the thesis measurement section as a constraint of the single-host setup. Actual RSS at the §7.0 nominal/stress load levels is well under the new caps in Phase 1 traces (`redis` ≈ 200 MB, `backend` ≈ 600 MB), so the reduction is non-binding.

## Consequences

- **Ansible inventory grows**: [deploy/inventory.yml](../deploy/inventory.yml) gets `swarm_manager`, `swarm_backend_workers`, `swarm_redis_workers` host groups in addition to the legacy `pixelbattle_servers`. The Phase 1 single-VM target stays for backward compatibility with `run_monolith_suite.yml`.
- **Caddy load balancer**: Caddy on `vm-manager` round-robins between gateway-1 and gateway-2 via Swarm's built-in routing mesh. WebSocket sticky-session is not required because the gateways gossip pixels via Redis Streams (XREADGROUP) + gRPC, so any client landing on either gateway sees the same canvas state.
- **Latency disclosure for thesis**: cross-VM latency on a local KVM bridge is ~0.1–0.5 ms vs ~1–3 ms on real LAN. The architecture proof (Cluster gossip works, gRPC fan-out works, partition tolerance works) is valid; absolute latency numbers in `e2e_pixel_latency_seconds` and `redis_replication_lag_seconds` are documented as **lower bounds** vs real distributed deployments.
- **Phase 1 fallback path**: if nested virtualization is not available on a future host (e.g. running benchmarks in a cloud VM that does not enable nested), we fall back to **Option A** (single-node Docker Swarm, same stack file). The placement constraints can all be satisfied by labelling a single node with all three roles. The fallback decision is recorded in the run-id artefact tree.
- **Network bonus**: with KVM bridge IPs as Swarm advertise / Redis announce addresses, future hardware moves are easier — replacing a physical host with three real machines just means re-running the cloud-init step against three IPs that are no longer 192.168.122.x.

## References

- ADR-001: `adr/001-crdt-hexagonal-layering.md` (CRDT/HLC stays in the domain layer; not affected).
- ADR-002: `adr/002-phase2-adapter-boundaries.md` (gRPC interceptors remain in the adapter / composition root; not affected).
- ADR-003: `adr/003-runtime-canvas-resize.md` (Redis size HASH; the same key now lives in the cluster slot owned by the same hash-tag).
- Plan: `~/.cursor/plans/pixelbattle_research_metrics_plan_04530117.plan.md` (§5 reference architecture, §7.2 distributed suite, §13.2 merge gate).
- Workspace plan: `~/.cursor/plans/phase_2_swarm_completion_62dffe4b.plan.md`.
