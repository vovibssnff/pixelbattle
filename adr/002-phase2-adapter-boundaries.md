# ADR-002: Phase 2 — adapter boundaries for metrics and messaging

## Status

Accepted

## Date

2026-05-10

## Context

The research metrics plan splits **comparable** KPIs (edge + client, identical across monolith and distributed) from **architecture-internal** diagnostics (gRPC, op-log lag, CRDT paths, shard labels). Phase 2 adds Redis Streams, optional gRPC between gateways, and Redis Cluster.

Without clear rules, instrumentation leaks into domain packages and breaks the dependency graph (domain → metrics → adapters).

## Options Considered

### Option A: Allow domain and `internal/metrics` to import each other via facades

- Rejected: risks cycles and makes unit tests heavyweight.

### Option B: Pass a generic `MetricsRecorder` interface into every domain service

- Rejected for domain purity; acceptable only at application layer if needed later, not for CRDT ticks.

### Option C: Emit internal metrics only from **adapters** and **composition root** (`cmd/*`)

- Accepted: matches hexagonal ports/adapters — domain returns outcomes (`ResolutionPath`, errors); adapters translate to Prometheus and logs.

## Decision

1. **`internal/core/domain` and `internal/core/domain/crdt`** never import **`internal/metrics`**.
2. **Comparable-layer** signals remain where they are today (HTTP middleware, WS handler, RUM ingest) per plan §2.
3. **Architecture-internal** counters/histograms (`crdt_resolved_total`, future `opstream_lag_seconds`, gRPC interceptors, shard-scoped `database_operation_duration_seconds`) are recorded in:
   - Redis / Cluster **repository** implementations after domain decisions, or
   - gRPC server middleware registered in **`cmd/gateway`** (when introduced), not inside `domain/crdt`.
4. **Application services** (`internal/core/service`) depend on **domain ports** (`domain.CanvasRepository`, etc.), not concrete Redis types — composition root wires implementations.

## Consequences

- When op-log write paths land, the Redis adapter loads stored + incoming `LWWState`, calls `crdt.DecideLWW`, persists if `replace`, then `metrics.IncrementCRDTResolved(string(path))`.
- New binaries (gateway) reuse the same domain packages; only `cmd/*` and `adapters/*` gain new imports for gRPC and cluster clients.

## References

- ADR-001: `adr/001-crdt-hexagonal-layering.md`
- Plan: `~/.cursor/plans/pixelbattle_research_metrics_plan_04530117.plan.md` (§2 vs §3 metrics, §11.4 op-log)
