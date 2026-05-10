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
"""

import json
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
        return float(obj.get(key, 0) or 0)
    except Exception:
        return None


def _load_saturation_vus(run_dir: Optional[Path]) -> Optional[int]:
    """Reads k6 summary JSONs and returns min(saturation_vus) across all stages."""
    if run_dir is None:
        return None
    k6_dir = run_dir / "k6"
    if not k6_dir.exists():
        return None
    best: Optional[int] = None
    for jf in k6_dir.glob("**/*.json"):
        try:
            payload = json.loads(jf.read_text())
        except Exception:
            continue
        # k6 summary: metrics["saturation_vus"]["values"]["min"]
        metrics = payload.get("metrics", {})
        sat = metrics.get("saturation_vus") if isinstance(metrics, dict) else None
        if not isinstance(sat, dict):
            continue
        values = sat.get("values") or {}
        v = values.get("min")
        if v is None:
            continue
        v_int = int(v)
        if v_int > 0 and (best is None or v_int < best):
            best = v_int
    return best


def _load_prom_kpi(run_dir: Optional[Path], metric_name: str) -> Optional[float]:
    """Extracts a single scalar from prom/comparable_kpis.json by query name."""
    if run_dir is None:
        return None
    p = run_dir / "prom" / "comparable_kpis.json"
    if not p.exists():
        return None
    try:
        obj = json.loads(p.read_text())
    except Exception:
        return None
    for q in obj.get("queries", []):
        if q.get("name") != metric_name:
            continue
        try:
            result = q.get("data", {}).get("data", {}).get("result", [])
            if not result:
                return None
            val = result[0].get("value", [None, None])[1]
            return float(val) if val is not None else None
        except Exception:
            return None
    return None


# ---------------------------------------------------------------------------
# Report rendering
# ---------------------------------------------------------------------------


def generate_report(root_dir: Path, output_file: Path) -> None:
    # Phase 1 fetch layout: results/baseline/<run-id>/benchmarks/*.json
    # Legacy layout: results/{baseline,candidate}/benchmarks/*.json under root_dir
    if (root_dir / "benchmarks").is_dir():
        baseline_dir = root_dir
        candidate_dir: Optional[Path] = None
        baseline = _load_latest_by_storage(baseline_dir)
        candidate: Dict[str, List[Dict[str, Any]]] = {}
    else:
        baseline_dir = root_dir / "baseline"
        candidate_dir = root_dir / "candidate"
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

    # --- Phase 2 architecture-internal KPIs (only meaningful when candidate has them) ---
    lines.append("")
    lines.append("## Phase 2 Architecture-Internal KPIs (from Prometheus snapshot)")
    lines.append("")
    lines.append("| Metric | Baseline | Candidate |")
    lines.append("|---|---:|---:|")
    for label, query in (
        ("e2e_pixel_latency_seconds p95", "e2e_pixel_latency_seconds"),
        ("pixel_write_visible_seconds p95", "pixel_write_visible_p95"),
        ("availability_sli (ok ratio)", "availability_sli"),
        ("error_rate", "error_rate"),
    ):
        b = _load_prom_kpi(base_run, query)
        c = _load_prom_kpi(cand_run, query)
        b_str = f"{b:.4f}" if b is not None else "n/a"
        c_str = f"{c:.4f}" if c is not None else "n/a"
        lines.append(f"| {label} | {b_str} | {c_str} |")

    output_file.write_text("\n".join(lines))


def main() -> None:
    root = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("results")
    output = root / f"comparison_report_{datetime.now().strftime('%Y%m%d_%H%M%S')}.md"
    generate_report(root, output)
    print(f"Report generated: {output}")


if __name__ == "__main__":
    main()
