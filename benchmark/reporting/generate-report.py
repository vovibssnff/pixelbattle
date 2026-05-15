#!/usr/bin/env python3
"""
Phase 1 vs Phase 2 (baseline vs candidate) comparison report.

Reads results/{baseline,candidate}/<run-id>/{benchmarks,k6,rum}/ trees and emits a
Markdown report covering:

  * Throughput delta (per storage:scenario)
  * P95 latency delta (per storage:scenario)
  * UX KPIs (e2e_pixel_latency_seconds p95, client_fps_avg)
  * k6 saturation_vus (min across all 3 k6 stages)
  * Phase 2 architecture-internal KPIs (when present): opstream_lag, replication_lag,
    gateway_fanout_recipients
  * Topology (free-form label embedded in BenchmarkResult.topology)

Prometheus scalars in ``prom/comparable_kpis.json`` are produced by Ansible
(``fetch_to_baseline.yml``): ``requests_per_second`` uses HTTP counter totals;
``error_rate`` is (5xx + ws_errors) over HTTP + WS message rates (no ``rate()``
on the misnamed ``requests_per_second`` series alone).
"""

import json
import math
import sys
from collections import defaultdict
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, List, Optional


# ---------------------------------------------------------------------------
# Loading helpers
# ---------------------------------------------------------------------------


def _load_latest_by_storage(results_dir: Path) -> Dict[str, List[Dict[str, Any]]]:
    """Walks <results_dir>/<run-id>/benchmarks/ or <results_dir>/benchmarks/.

    For a `results/<phase>/<run-id>/benchmarks/*.json` layout we pick the most recent
    run-id by lexical max (epoch-prefixed). For a flat `results/<phase>/benchmarks/*.json`
    we just load every file.
    """
    data: Dict[str, List[Dict[str, Any]]] = {}
    if not results_dir.exists():
        return data
    bench_dir = results_dir / "benchmarks"
    if not bench_dir.exists():
        # results/<phase>/<run-id>/benchmarks/*.json — pick latest run-id directory.
        candidates = [d for d in results_dir.iterdir() if d.is_dir() and (d / "benchmarks").exists()]
        if not candidates:
            return data
        candidates.sort(key=lambda d: d.name, reverse=True)
        bench_dir = candidates[0] / "benchmarks"

    files_by_storage: Dict[str, List[Path]] = defaultdict(list)
    for f in bench_dir.glob("*.json"):
        storage = f.stem.split("_")[0]
        files_by_storage[storage].append(f)

    for storage, files in files_by_storage.items():
        latest = max(files, key=lambda p: p.stem)
        payload = json.loads(latest.read_text())
        data[storage] = payload if isinstance(payload, list) else [payload]
    return data


def _latest_run_dir(phase_root: Path) -> Optional[Path]:
    if not phase_root.exists():
        return None
    if (phase_root / "benchmarks").exists():
        return phase_root
    candidates = [d for d in phase_root.iterdir() if d.is_dir() and any(d.iterdir())]
    if not candidates:
        return None
    candidates.sort(key=lambda d: d.name, reverse=True)
    return candidates[0]


def _scenario_map(results: Dict[str, List[Dict[str, Any]]], metric_key: str) -> Dict[str, float]:
    out: Dict[str, float] = {}
    for storage, rows in results.items():
        for row in rows:
            scenario = row.get("scenario", "unknown")
            value = float(row.get(metric_key, 0) or 0)
            out[f"{storage}:{scenario}"] = value
    return out


def _topology_label(results: Dict[str, List[Dict[str, Any]]]) -> str:
    for rows in results.values():
        for row in rows:
            label = (row.get("topology") or "").strip()
            if label:
                return label
    return ""


def _delta(base: Optional[float], cand: Optional[float]) -> str:
    if base is None or cand is None:
        return "n/a"
    if base == 0:
        return "inf" if cand and cand > 0 else "0.00%"
    return f"{((cand - base) / base) * 100:+.2f}%"


def _load_optional_metric(run_dir: Optional[Path], filename: str, key: str) -> Optional[float]:
    if run_dir is None:
        return None
    p = run_dir / "rum" / filename
    if not p.exists():
        return None
    try:
        obj = json.loads(p.read_text())
        val = obj.get(key)
        if val is None:
            return None
        fval = float(val)
        return fval if math.isfinite(fval) and fval > 0 else None
    except Exception:
        return None


def _load_k6_edge_metrics(run_dir: Optional[Path]) -> Dict[str, Optional[float]]:
    """Load edge-layer KPIs from the k6 nominal (medium), stress (heavy), and breakpoint summaries.

    Returns dict with keys like 'nominal_e2e_p95', 'stress_e2e_p95', etc.
    Only considers the file matching the run directory's epoch id.
    """
    result: Dict[str, Optional[float]] = {
        "nominal_e2e_p95": None,
        "nominal_ws_connect_p95": None,
        "nominal_checks_pass_rate": None,
        "stress_e2e_p95": None,
        "stress_ws_connect_p95": None,
        "stress_checks_pass_rate": None,
        "breakpoint_e2e_p95": None,
        "breakpoint_ws_connect_p95": None,
        "breakpoint_checks_pass_rate": None,
    }
    if run_dir is None:
        return result
    k6_dir = run_dir / "k6"
    if not k6_dir.exists():
        return result

    run_id = run_dir.name
    stage_map = [
        ("nominal", f"redis_medium_summary_{run_id}.json"),
        ("stress", f"redis_heavy_summary_{run_id}.json"),
        ("breakpoint", f"redis_breakpoint_summary_{run_id}.json"),
    ]

    for stage, filename in stage_map:
        p = k6_dir / filename
        if not p.exists():
            # Fall back to latest file of that type
            pattern = filename.replace(f"_{run_id}.json", "_*.json")
            matches = sorted(k6_dir.glob(pattern), reverse=True)
            if matches:
                p = matches[0]
            else:
                continue
        try:
            payload = json.loads(p.read_text())
        except Exception:
            continue

        metrics = payload.get("metrics", {})
        if not isinstance(metrics, dict):
            continue

        e2e = metrics.get("e2e_pixel_latency_seconds", {})
        if isinstance(e2e, dict):
            result[f"{stage}_e2e_p95"] = e2e.get("p(95)")

        ws = metrics.get("ws_connecting", {})
        if isinstance(ws, dict):
            result[f"{stage}_ws_connect_p95"] = ws.get("p(95)")

        checks = metrics.get("checks", {})
        if isinstance(checks, dict):
            val = checks.get("value")
            if val is not None:
                result[f"{stage}_checks_pass_rate"] = float(val)

    return result


def _load_saturation_vus(run_dir: Optional[Path]) -> Optional[int]:
    """Reads the LATEST breakpoint k6 summary and returns median saturation_vus.

    Only considers the file matching the run_dir name (epoch run-id) to avoid
    picking up stale results from older runs accumulated in the same k6/ tree.
    Falls back to max across breakpoint files if no run-id match is found.
    Uses the 'med' (median) field which represents the actual VU level at SLO break.
    """
    if run_dir is None:
        return None
    k6_dir = run_dir / "k6"
    if not k6_dir.exists():
        return None

    run_id = run_dir.name

    # Prefer the breakpoint summary matching this run's id
    target = k6_dir / f"redis_breakpoint_summary_{run_id}.json"
    candidates: List[Path] = []
    if target.exists():
        candidates = [target]
    else:
        candidates = sorted(k6_dir.glob("*breakpoint*summary*.json"), reverse=True)

    for jf in candidates:
        try:
            payload = json.loads(jf.read_text())
        except Exception:
            continue
        metrics = payload.get("metrics", {})
        sat = metrics.get("saturation_vus") if isinstance(metrics, dict) else None
        if not isinstance(sat, dict):
            continue
        # Use median as the representative saturation point; fall back to max, then avg
        v = sat.get("med") or sat.get("max") or sat.get("avg")
        if v is None:
            values = sat.get("values") or {}
            v = values.get("med") or values.get("max")
        if v is not None and float(v) > 0:
            return int(float(v))
    return None


def _fix_promql_json(text: str) -> str:
    """Fix malformed JSON where PromQL queries contain unescaped double quotes.

    The Ansible collection task writes query values with raw PromQL like:
      "query": "sum(rate(foo{label="bar"}[5m]))"
    which breaks JSON. We strip "query" fields to make the rest parseable.
    """
    import re
    # Remove entire "query": "..." lines — we don't need the raw PromQL for reporting
    fixed = re.sub(
        r'^\s*"query"\s*:.*$',
        '',
        text,
        flags=re.MULTILINE,
    )
    # Clean up trailing commas that become dangling after removal
    fixed = re.sub(r',(\s*[}\]])', r'\1', fixed)
    # Clean up leading commas
    fixed = re.sub(r'([\[{])\s*,', r'\1', fixed)
    return fixed


def _load_prom_kpi(run_dir: Optional[Path], metric_name: str) -> Optional[float]:
    """Extracts a single scalar from prom/comparable_kpis.json by query name.

    Returns None for NaN, empty results, or missing data.
    For multi-series results (e.g. pixels_placed_total per faculty), sums values.
    """

    if run_dir is None:
        return None
    p = run_dir / "prom" / "comparable_kpis.json"
    if not p.exists():
        return None
    try:
        raw = p.read_text()
        try:
            obj = json.loads(raw)
        except json.JSONDecodeError:
            obj = json.loads(_fix_promql_json(raw))
    except Exception:
        return None
    for q in obj.get("queries", []):
        if q.get("name") != metric_name:
            continue
        try:
            result = q.get("data", {}).get("data", {}).get("result", [])
            if not result:
                return None
            # Sum across series for rate metrics (e.g. per-faculty or per-instance)
            total = 0.0
            any_valid = False
            for series in result:
                val = series.get("value", [None, None])[1]
                if val is None:
                    continue
                fval = float(val)
                if not math.isfinite(fval):
                    continue
                total += fval
                any_valid = True
            return total if any_valid else None
        except Exception:
            return None
    return None


# ---------------------------------------------------------------------------
# Report rendering
# ---------------------------------------------------------------------------


def generate_report(root_dir: Path, output_file: Path) -> None:
    # Prefer two-tree comparison whenever both branches exist under root_dir.
    # This avoids accidentally taking single-phase mode when a legacy
    # root-level benchmarks/ directory is also present.
    baseline_branch = root_dir / "baseline"
    candidate_branch = root_dir / "candidate"
    if baseline_branch.is_dir() and candidate_branch.is_dir():
        baseline_dir = baseline_branch
        candidate_dir: Optional[Path] = candidate_branch
        baseline = _load_latest_by_storage(baseline_dir)
        candidate = _load_latest_by_storage(candidate_dir)
    elif (root_dir / "benchmarks").is_dir():
        baseline_dir = root_dir
        candidate_dir = None
        baseline = _load_latest_by_storage(baseline_dir)
        candidate: Dict[str, List[Dict[str, Any]]] = {}
    else:
        baseline_dir = baseline_branch
        candidate_dir = candidate_branch
        baseline = _load_latest_by_storage(baseline_dir)
        candidate = _load_latest_by_storage(candidate_dir)

    if not baseline and not candidate:
        raise RuntimeError("No baseline/candidate benchmark data found")

    base_run = _latest_run_dir(baseline_dir)
    cand_run = _latest_run_dir(candidate_dir) if candidate_dir else None

    base_tp = _scenario_map(baseline, "throughput_ops_per_sec")
    cand_tp = _scenario_map(candidate, "throughput_ops_per_sec")
    base_p95 = _scenario_map(baseline, "latency_p95_ms")
    cand_p95 = _scenario_map(candidate, "latency_p95_ms")

    keys = sorted(set(base_tp.keys()) | set(cand_tp.keys()))

    lines: List[str] = []
    lines.append("# Baseline vs Candidate Comparison Report")
    lines.append("")
    lines.append(f"Generated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    lines.append("")
    base_topology = _topology_label(baseline) or "monolith"
    cand_topology = _topology_label(candidate) or ("n/a" if not candidate else "kvm-swarm")
    lines.append(f"- **Baseline topology**: `{base_topology}`")
    lines.append(f"- **Candidate topology**: `{cand_topology}`")
    lines.append("")

    lines.append("## Throughput Delta (ops/sec)")
    lines.append("")
    lines.append("| Storage:Scenario | Baseline | Candidate | Δ |")
    lines.append("|---|---:|---:|---:|")
    for key in keys:
        b = base_tp.get(key)
        c = cand_tp.get(key)
        b_str = f"{b:.1f}" if b is not None else "n/a"
        c_str = f"{c:.1f}" if c is not None else "n/a"
        lines.append(f"| {key} | {b_str} | {c_str} | {_delta(b, c)} |")

    lines.append("")
    lines.append("## P95 Latency Delta (ms)")
    lines.append("")
    lines.append("| Storage:Scenario | Baseline | Candidate | Δ |")
    lines.append("|---|---:|---:|---:|")
    p95_keys = sorted(set(base_p95.keys()) | set(cand_p95.keys()))
    for key in p95_keys:
        b = base_p95.get(key)
        c = cand_p95.get(key)
        b_str = f"{b:.2f}" if b is not None else "n/a"
        c_str = f"{c:.2f}" if c is not None else "n/a"
        lines.append(f"| {key} | {b_str} | {c_str} | {_delta(b, c)} |")

    # --- k6 Edge-Layer Comparable KPIs (§2 metrics from synthetic load) ---
    base_k6 = _load_k6_edge_metrics(base_run)
    cand_k6 = _load_k6_edge_metrics(cand_run)

    lines.append("")
    lines.append("## k6 Edge-Layer KPIs (§2 Comparable Metrics)")
    lines.append("")
    lines.append("These are the thesis-comparable metrics measured at the client/edge boundary.")
    lines.append("Lower latency is better; higher check rate is better.")
    lines.append("")
    lines.append("| Stage / Metric | Baseline | Candidate | Δ |")
    lines.append("|---|---:|---:|---:|")
    for stage, label in [("nominal", "Nominal (200 VUs)"), ("stress", "Stress (500 VUs)"), ("breakpoint", "Breakpoint (→3000 VUs)")]:
        b_e2e = base_k6.get(f"{stage}_e2e_p95")
        c_e2e = cand_k6.get(f"{stage}_e2e_p95")
        b_ws = base_k6.get(f"{stage}_ws_connect_p95")
        c_ws = cand_k6.get(f"{stage}_ws_connect_p95")
        b_chk = base_k6.get(f"{stage}_checks_pass_rate")
        c_chk = cand_k6.get(f"{stage}_checks_pass_rate")

        def _fmt(v: Optional[float], precision: int = 1) -> str:
            return f"{v:.{precision}f}" if v is not None else "n/a"

        lines.append(f"| **{label}** | | | |")
        lines.append(f"|   e2e_pixel_latency P95 | {_fmt(b_e2e)} | {_fmt(c_e2e)} | {_delta(b_e2e, c_e2e)} |")
        lines.append(f"|   ws_connecting P95 (ms) | {_fmt(b_ws, 2)} | {_fmt(c_ws, 2)} | {_delta(b_ws, c_ws)} |")
        lines.append(f"|   checks pass rate | {_fmt(b_chk, 4)} | {_fmt(c_chk, 4)} | {_delta(b_chk, c_chk)} |")

    # --- UX KPIs ---
    base_e2e = _load_optional_metric(base_run, "summary.json", "e2e_latency_ms_p95")
    cand_e2e = _load_optional_metric(cand_run, "summary.json", "e2e_latency_ms_p95")
    base_fps = _load_optional_metric(base_run, "summary.json", "client_fps_avg")
    cand_fps = _load_optional_metric(cand_run, "summary.json", "client_fps_avg")

    lines.append("")
    lines.append("## UX Metrics (Playwright RUM)")
    lines.append("")
    lines.append("| Metric | Baseline | Candidate | Δ |")
    lines.append("|---|---:|---:|---:|")
    lines.append(f"| e2e_latency_ms_p95 | {base_e2e if base_e2e is not None else 'n/a'} | {cand_e2e if cand_e2e is not None else 'n/a'} | {_delta(base_e2e, cand_e2e)} |")
    lines.append(f"| client_fps_avg | {base_fps if base_fps is not None else 'n/a'} | {cand_fps if cand_fps is not None else 'n/a'} | {_delta(base_fps, cand_fps)} |")

    if cand_run is not None:
        rum_path = cand_run / "rum" / "summary.json"
        if rum_path.exists():
            try:
                rum = json.loads(rum_path.read_text())
                if float(rum.get("rum_beacon_total_success", 0) or 0) == 0:
                    lines.append("")
                    lines.append(
                        "> **Data quality**: Candidate `rum/summary.json` reports **zero** successful RUM beacons. "
                        "Typical causes: Playwright never logged in (see `playwright/run_log_*.txt`), stale JSON copied "
                        "from an older run (fixed in 2026-05-13: per-run `results/candidate/<id>/playwright/`), or an empty Prometheus window. "
                        "Re-run `benchmark/playbooks/run_playwright_candidate.yml` after `npx playwright install chromium` and stack checks."
                    )
            except Exception:
                pass

    # --- Saturation VUs (k6 breakpoint stage) ---
    base_sat = _load_saturation_vus(base_run)
    cand_sat = _load_saturation_vus(cand_run)
    lines.append("")
    lines.append("## k6 Saturation Point (`saturation_vus`)")
    lines.append("")
    lines.append("Minimum VU count at which the breakpoint stage first crossed the SLO breach line.")
    lines.append("Higher is better.")
    lines.append("")
    lines.append("| Stage | Baseline | Candidate | Δ |")
    lines.append("|---|---:|---:|---:|")
    base_sat_f = float(base_sat) if base_sat is not None else None
    cand_sat_f = float(cand_sat) if cand_sat is not None else None
    lines.append(f"| breakpoint min(saturation_vus) | {base_sat if base_sat is not None else 'n/a'} | {cand_sat if cand_sat is not None else 'n/a'} | {_delta(base_sat_f, cand_sat_f)} |")

    # --- Prometheus Comparable KPIs (snapshot-based) ---
    lines.append("")
    lines.append("## Prometheus Comparable KPIs (snapshot during run)")
    lines.append("")
    lines.append("| Metric | Baseline | Candidate | Δ |")
    lines.append("|---|---:|---:|---:|")
    for label, query, multiplier, unit in (
        ("pixel_write_visible P95", "pixel_write_visible_p95", 1000.0, "ms"),
        ("availability_sli (ok ratio)", "availability_sli", 1.0, ""),
        ("requests_per_second (total)", "requests_per_second", 1.0, "rps"),
        ("error_rate", "error_rate", 1.0, ""),
    ):
        b = _load_prom_kpi(base_run, query)
        c = _load_prom_kpi(cand_run, query)
        b_val = b * multiplier if b is not None else None
        c_val = c * multiplier if c is not None else None
        suffix = f" {unit}" if unit else ""
        b_str = f"{b_val:.2f}{suffix}" if b_val is not None else "n/a"
        c_str = f"{c_val:.2f}{suffix}" if c_val is not None else "n/a"
        lines.append(f"| {label} | {b_str} | {c_str} | {_delta(b_val, c_val)} |")

    output_file.write_text("\n".join(lines))


def main() -> None:
    root = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("results")
    output = root / f"comparison_report_{datetime.now().strftime('%Y%m%d_%H%M%S')}.md"
    generate_report(root, output)
    print(f"Report generated: {output}")


if __name__ == "__main__":
    main()
