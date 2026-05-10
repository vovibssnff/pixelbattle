# ADR-001: CRDT / LWW logic in the domain hexagon

## Status

Accepted

## Date

2026-05-10

## Context

Phase 2 of the research metrics plan introduces Redis Streams as the canonical last-write-wins (LWW) timestamp and a Hybrid Logical Clock (HLC) as the ordering fallback when stream IDs are missing or unreliable (see plan §5, §11.4). The backend already follows a rough ports-and-adapters layout (`internal/core/domain` for ports and models, `internal/core/service` for use cases, `internal/adapters/*` for Redis/HTTP/WebSocket).

We need a single place for **pure ordering rules** so that:

- The same comparator can be unit-tested without Redis or Prometheus.
- Infrastructure code (Redis Lua/scripts, cluster client) only **persists** outcomes; it does not redefine correctness.
- Prometheus labels such as `crdt_resolved_total{path="stream_id|hlc_fallback"}` are recorded at the **adapter** boundary, not inside domain logic.

## Options Considered

### Option A: Implement LWW only inside the Redis adapter (Lua or Go repository)

- Pros: Fast to ship; all Redis details in one file.
- Cons: Business rules are untestable in isolation; HTTP/gRPC entrypoints risk duplicating rules; violates dependency rule (domain should not depend on Redis, but **ordering policy** should not live only in Redis either).

### Option B: Put CRDT helpers next to Prometheus (`internal/metrics`)

- Pros: Co-located with `crdt_resolved_total`.
- Cons: Metrics package must stay a leaf dependency; mixing correctness with instrumentation creates import cycles and hides domain rules.

### Option C: Domain subpackage `internal/core/domain/crdt` (pure Go, no infra imports)

- Pros: Aligns with hexagonal/clean architecture: **domain** holds algorithms and value types; **adapters** call into domain and translate results to storage + metrics.
- Cons: One extra import path for Redis code when merging pixels (acceptable).

## Decision

We place **HLC, stream-ID comparison, and LWW decision** in **`pb_backend/internal/core/domain/crdt`**, a package with **no** imports from adapters, `metrics`, or `service`.

- **Domain (`domain/crdt`)** exposes data structures and functions such as `DecideLWW`, returning a typed `ResolutionPath` (`stream_id` vs `hlc_fallback`) for use cases and adapters.
- **Application (`internal/core/service`)** continues to orchestrate canvas writes; it may depend on `domain` and `domain/crdt` when serialization or timestamps need HLC ticks (future op-log path).
- **Driving adapters (Redis repository, future gateway op-log consumer)** perform I/O, then call `domain/crdt` to decide overwrite, and **only then** call `internal/metrics.IncrementCRDTResolved(string(path))` so instrumentation stays outward.

**Canvas persistence port:** `CanvasService` depends on **`domain.CanvasRepository`** (the interface in `interfaces.go`), not on the Redis concrete type, so alternate storage and tests can substitute the port without import leakage from `adapters/redis`.

## Consequences

- New Redis Cluster / Streams code must **delegate** LWW comparisons to `domain/crdt` rather than re-encode ordering in Lua when possible; Lua may still atomically read fields and pass them to a shared decision (or duplicate the minimal compare if required for single round-trip — if so, add a comment linking this ADR and keep Go tests as source of truth).
- Any change to conflict policy requires updating **domain tests** first, then adapters.
- Grafana and thesis metrics remain valid: `path` label values match domain constants.

## References

- Plan: `~/.cursor/plans/pixelbattle_research_metrics_plan_04530117.plan.md` (§5, §11.4, comparable vs internal metrics).
- Code: `pb_backend/internal/core/domain/crdt/` — `// See adr/001-crdt-hexagonal-layering.md`.
