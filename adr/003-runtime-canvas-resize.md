# ADR-003: Runtime canvas resize (expand-only)

## Status

Accepted

## Date

2026-05-10

## Context

The research metrics plan (§11.6) requires **runtime canvas resize**: operators grow the drawable grid without redeploy, new cells default to white, and all connected clients must update WebGL texture bounds. The monolith today fixes `CANVAS_HEIGHT` / `CANVAS_WIDTH` at boot only.

Constraints:

- **Expand-only** avoids orphaning in-flight pixels and matches the plan (“expand only”).
- **Redis** is the production canvas store; dimensions must be recoverable for snapshots, PNG init, and WS `validPixel` bounds.
- **Comparable KPI contract**: edge metrics stay on existing paths; we add `canvas_dimensions{axis}` as an internal/operational gauge (plan §11.6).

## Options Considered

### Option A: Config-only dimensions (restart to resize)

- Pros: no new state, simplest.
- Cons: violates Phase 2 acceptance (“resize at runtime”); bad for live events.

### Option B: Derive size only from max key coordinates (no meta key)

- Pros: no extra Redis key.
- Cons: ambiguous for sparse canvases; expensive scans; cluster-unfriendly as sole source of truth.

### Option C: Persist logical size in Redis + broadcast `RESIZE` over WebSocket

- Pros: O(1) read; explicit contract; snapshotter and HTTP init can call `CanvasDimensions`; WS hub updates `canvasWidth`/`canvasHeight` on the same goroutine that owns `clients` (no map races).
- Cons: SQLite/Postgres benchmark adapters use infer-from-data when meta returns (0,0).

## Decision

1. Store **width** and **height** in a small Redis HASH (`canvas:size` standalone, `{canvas}:size` when hash-tagged for cluster), set on `InitializeCanvas` and after successful `ExpandCanvas`.
2. **`CanvasService.ExpandCanvas`**: reject shrink; fill new coordinates with white pixels via existing `WritePixel`; then `SetCanvasDimensions`.
3. **`POST /api/admin/canvas/resize`** (existing route): body `{"width":W,"height":H}`; on success emit **`canvas_dimensions`**, then **`NotifyCanvasResize`** with JSON `{"event":"RESIZE","width":W,"height":H}` processed on the **WebSocket `Run` loop** (resize notice channel → per-client `control` channel → text frame).
4. **Frontend**: handle `RESIZE` in the JSON WS branch; **`GLWindow.expandTextureTo`** reallocates texture, copies overlapping region, fills extension white, updates uniforms/zoom.
5. **Instrumentation**: `canvas_dimensions{axis="width|height"}`; internal layer, not a comparable KPI.

## Consequences

- `RestHandlers` depends on **`WSResizeNotifier`** (implemented by `*websockets.WsServer`); `main` starts WS before REST so the hub is non-nil for admin.
- Clients that only parse binary pixels still receive **text** `RESIZE` frames; parsers must ignore non-pixel JSON (already required for control messages per §11.2).
- **Grafana:** Runtime Admin dashboard provisions a **Business Form** panel (`monitoring/grafana/dashboards/runtime-admin.json`) posting to `${api_origin}/api/admin/canvas/resize` with `X-Admin-Token` (plan §12.2).
- References: **`// See ADR-003`** at Redis size key helper and REST notifier interface.

## References

- ADR-002: `adr/002-phase2-adapter-boundaries.md`
- Plan: `~/.cursor/plans/pixelbattle_research_metrics_plan_04530117.plan.md` (§11.6)
