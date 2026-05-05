#!/usr/bin/env python3

import json
import sys
from collections import defaultdict
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple


def _load_latest_by_storage(results_dir: Path) -> Dict[str, List[Dict[str, Any]]]:
    data: Dict[str, List[Dict[str, Any]]] = {}
    bench_dir = results_dir / "benchmarks"
    if not bench_dir.exists():
        return data

    files_by_storage: Dict[str, List[Path]] = defaultdict(list)
    for f in bench_dir.glob("*.json"):
        storage = f.stem.split("_")[0]
        files_by_storage[storage].append(f)

    for storage, files in files_by_storage.items():
        latest = max(files, key=lambda p: p.stem)
        payload = json.loads(latest.read_text())
        data[storage] = payload if isinstance(payload, list) else [payload]
    return data


def _scenario_map(results: Dict[str, List[Dict[str, Any]]], metric_key: str) -> Dict[str, float]:
    out: Dict[str, float] = {}
    for storage, rows in results.items():
        for row in rows:
            scenario = row.get("scenario", "unknown")
            value = float(row.get(metric_key, 0) or 0)
            out[f"{storage}:{scenario}"] = value
    return out


def _delta(base: Optional[float], cand: Optional[float]) -> str:
    if base is None or cand is None:
        return "n/a"
    if base == 0:
        return "inf" if cand > 0 else "0.0%"
    return f"{((cand - base) / base) * 100:.2f}%"


def _load_optional_metric(results_dir: Path, filename: str, key: str) -> Optional[float]:
    p = results_dir / "rum" / filename
    if not p.exists():
        return None
    try:
        obj = json.loads(p.read_text())
        return float(obj.get(key, 0) or 0)
    except Exception:
        return None


def generate_report(root_dir: Path, output_file: Path) -> None:
    baseline_dir = root_dir / "baseline"
    candidate_dir = root_dir / "candidate"

    baseline = _load_latest_by_storage(baseline_dir)
    candidate = _load_latest_by_storage(candidate_dir)

    if not baseline and not candidate:
        raise RuntimeError("No baseline/candidate benchmark data found")

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
    lines.append("## Throughput Delta")
    lines.append("")
    lines.append("| Storage:Scenario | Baseline ops/sec | Candidate ops/sec | Delta |")
    lines.append("|---|---:|---:|---:|")
    for key in keys:
        b = base_tp.get(key)
        c = cand_tp.get(key)
        lines.append(f"| {key} | {b if b is not None else 'n/a'} | {c if c is not None else 'n/a'} | {_delta(b, c)} |")

    lines.append("")
    lines.append("## P95 Latency Delta")
    lines.append("")
    lines.append("| Storage:Scenario | Baseline p95 ms | Candidate p95 ms | Delta |")
    lines.append("|---|---:|---:|---:|")
    p95_keys = sorted(set(base_p95.keys()) | set(cand_p95.keys()))
    for key in p95_keys:
        b = base_p95.get(key)
        c = cand_p95.get(key)
        lines.append(f"| {key} | {b if b is not None else 'n/a'} | {c if c is not None else 'n/a'} | {_delta(b, c)} |")

    base_e2e = _load_optional_metric(baseline_dir, "summary.json", "e2e_latency_ms_p95")
    cand_e2e = _load_optional_metric(candidate_dir, "summary.json", "e2e_latency_ms_p95")
    base_fps = _load_optional_metric(baseline_dir, "summary.json", "client_fps_avg")
    cand_fps = _load_optional_metric(candidate_dir, "summary.json", "client_fps_avg")

    lines.append("")
    lines.append("## UX Metrics")
    lines.append("")
    lines.append("| Metric | Baseline | Candidate | Delta |")
    lines.append("|---|---:|---:|---:|")
    lines.append(f"| e2e_latency_ms_p95 | {base_e2e if base_e2e is not None else 'n/a'} | {cand_e2e if cand_e2e is not None else 'n/a'} | {_delta(base_e2e, cand_e2e)} |")
    lines.append(f"| client_fps_avg | {base_fps if base_fps is not None else 'n/a'} | {cand_fps if cand_fps is not None else 'n/a'} | {_delta(base_fps, cand_fps)} |")

    output_file.write_text("\n".join(lines))


def main() -> None:
    root = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("results")
    output = root / f"comparison_report_{datetime.now().strftime('%Y%m%d_%H%M%S')}.md"
    generate_report(root, output)
    print(f"Report generated: {output}")


if __name__ == "__main__":
    main()

