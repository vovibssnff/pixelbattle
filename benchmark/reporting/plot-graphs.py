#!/usr/bin/env python3

import json
import sys
from collections import defaultdict
from pathlib import Path
from typing import Dict, List, Optional

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt
import numpy as np


def _load_latest_by_storage(results_dir: Path) -> Dict[str, List[dict]]:
    out: Dict[str, List[dict]] = {}
    bench_dir = results_dir / "benchmarks"
    if not bench_dir.exists():
        return out

    files_by_storage: Dict[str, List[Path]] = defaultdict(list)
    for f in bench_dir.glob("*.json"):
        files_by_storage[f.stem.split("_")[0]].append(f)

    for storage, files in files_by_storage.items():
        latest = max(files, key=lambda p: p.stem)
        payload = json.loads(latest.read_text())
        out[storage] = payload if isinstance(payload, list) else [payload]
    return out


def _scenario_metric(data: Dict[str, List[dict]], key: str) -> Dict[str, float]:
    out: Dict[str, float] = {}
    for storage, rows in data.items():
        for row in rows:
            scenario = row.get("scenario", "unknown")
            out[f"{storage}:{scenario}"] = float(row.get(key, 0) or 0)
    return out


def _load_summary_metric(results_dir: Path, key: str) -> Optional[float]:
    p = results_dir / "rum" / "summary.json"
    if not p.exists():
        return None
    try:
        obj = json.loads(p.read_text())
        return float(obj.get(key, 0) or 0)
    except Exception:
        return None


def _plot_baseline_candidate(metric_name: str, baseline: Dict[str, float], candidate: Dict[str, float], out_file: Path):
    labels = sorted(set(baseline.keys()) | set(candidate.keys()))
    if not labels:
        return
    x = np.arange(len(labels))
    width = 0.4
    bvals = [baseline.get(label, 0) for label in labels]
    cvals = [candidate.get(label, 0) for label in labels]

    fig, ax = plt.subplots(figsize=(16, 8))
    ax.bar(x - width / 2, bvals, width=width, label="baseline")
    ax.bar(x + width / 2, cvals, width=width, label="candidate")
    ax.set_title(metric_name)
    ax.set_xticks(x)
    ax.set_xticklabels(labels, rotation=45, ha="right")
    ax.grid(True, alpha=0.3)
    ax.legend()
    plt.tight_layout()
    plt.savefig(out_file, dpi=200, bbox_inches="tight")
    plt.close(fig)


def _plot_single_metric(metric_name: str, baseline: Optional[float], candidate: Optional[float], out_file: Path):
    fig, ax = plt.subplots(figsize=(6, 5))
    labels = ["baseline", "candidate"]
    values = [baseline or 0, candidate or 0]
    ax.bar(labels, values, color=["#4c78a8", "#f58518"])
    ax.set_title(metric_name)
    ax.grid(True, alpha=0.3)
    plt.tight_layout()
    plt.savefig(out_file, dpi=200, bbox_inches="tight")
    plt.close(fig)


def main() -> None:
    root = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("results")
    out_dir = root / "graphs"
    out_dir.mkdir(parents=True, exist_ok=True)

    baseline = _load_latest_by_storage(root / "baseline")
    candidate = _load_latest_by_storage(root / "candidate")

    _plot_baseline_candidate(
        "Throughput Comparison",
        _scenario_metric(baseline, "throughput_ops_per_sec"),
        _scenario_metric(candidate, "throughput_ops_per_sec"),
        out_dir / "throughput_comparison.png",
    )
    _plot_baseline_candidate(
        "Latency P95 Comparison",
        _scenario_metric(baseline, "latency_p95_ms"),
        _scenario_metric(candidate, "latency_p95_ms"),
        out_dir / "latency_comparison.png",
    )

    _plot_single_metric(
        "E2E Latency (ms p95)",
        _load_summary_metric(root / "baseline", "e2e_latency_ms_p95"),
        _load_summary_metric(root / "candidate", "e2e_latency_ms_p95"),
        out_dir / "e2e_latency_comparison.png",
    )
    _plot_single_metric(
        "Client FPS Avg",
        _load_summary_metric(root / "baseline", "client_fps_avg"),
        _load_summary_metric(root / "candidate", "client_fps_avg"),
        out_dir / "client_fps_comparison.png",
    )
    print(f"Saved graphs in: {out_dir}")


if __name__ == "__main__":
    main()

