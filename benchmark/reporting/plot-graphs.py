#!/usr/bin/env python3
"""
Generate thesis-quality comparison charts: baseline (monolith) vs candidate (distributed).

Reads k6 summary JSONs and Go benchmark JSONs from results/{baseline,candidate}/.
Produces 5 charts in results/graphs/:
  1. e2e_latency_by_stage.png       — grouped bars, 3 stages × 2 configs (log Y)
  2. ws_latency_by_stage.png        — grouped bars, 3 stages × 2 configs
  3. saturation_vus_comparison.png   — two-bar chart
  4. storage_throughput_comparison.png — key scenarios only
  5. storage_latency_comparison.png   — key scenarios only
"""

import json
import sys
from pathlib import Path
from typing import Dict, Optional, Tuple

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt
import numpy as np

plt.rcParams["font.family"] = "DejaVu Sans"

BASELINE_COLOR = "#4c78a8"
CANDIDATE_COLOR = "#f58518"


def _latest_run_dir(phase_root: Path) -> Optional[Path]:
    """Find the latest run directory that contains actual k6 summary data."""
    if not phase_root.exists():
        return None
    dirs = [d for d in phase_root.iterdir() if d.is_dir()]
    if not dirs:
        return None
    dirs.sort(key=lambda d: d.name, reverse=True)
    for d in dirs:
        k6_dir = d / "k6"
        if k6_dir.exists() and list(k6_dir.glob("*_summary_*.json")):
            return d
    return dirs[0]


def _find_k6_summary(run_dir: Path, prefix: str) -> Optional[Path]:
    """Find the latest k6 summary file matching prefix in the run's k6/ folder."""
    k6_dir = run_dir / "k6"
    if not k6_dir.exists():
        return None
    matches = sorted(k6_dir.glob(f"{prefix}_summary_*.json"), key=lambda p: p.stem, reverse=True)
    if not matches:
        return None
    return matches[0]


def _load_k6_metric(summary_path: Path, metric_key: str, stat: str = "p(95)") -> Optional[float]:
    try:
        data = json.loads(summary_path.read_text())
        m = data.get("metrics", {}).get(metric_key)
        if m is None:
            return None
        return float(m.get(stat, 0))
    except Exception:
        return None


def _load_stage_metrics(phase_root: Path) -> Dict[str, Dict[str, Optional[float]]]:
    """Load per-stage k6 metrics for a phase (baseline or candidate)."""
    run_dir = _latest_run_dir(phase_root)
    if run_dir is None:
        return {}

    result: Dict[str, Dict[str, Optional[float]]] = {}
    stage_map = {
        "Номинальная\n(200 ВП)": "redis_medium",
        "Стрессовая\n(500 ВП)": "redis_heavy",
        "Предельная\n(→3000 ВП)": "redis_breakpoint",
    }

    for stage_label, prefix in stage_map.items():
        summary = _find_k6_summary(run_dir, prefix)
        if summary is None:
            result[stage_label] = {"e2e": None, "ws": None}
            continue
        result[stage_label] = {
            "e2e": _load_k6_metric(summary, "e2e_pixel_latency_seconds"),
            "ws": _load_k6_metric(summary, "ws_connecting"),
        }
    return result


def _load_saturation(phase_root: Path) -> Optional[float]:
    run_dir = _latest_run_dir(phase_root)
    if run_dir is None:
        return None
    summary = _find_k6_summary(run_dir, "redis_breakpoint")
    if summary is None:
        return None
    return _load_k6_metric(summary, "saturation_vus", stat="med")


def _load_benchmark_scenarios(phase_root: Path) -> Dict[str, Dict[str, float]]:
    """Load Go DB benchmark results for key scenarios."""
    run_dir = _latest_run_dir(phase_root)
    if run_dir is None:
        return {}
    bench_dir = run_dir / "benchmarks"
    if not bench_dir.exists():
        return {}

    latest_file = None
    for f in sorted(bench_dir.glob("redis_benchmark_*.json"), key=lambda p: p.stem, reverse=True):
        latest_file = f
        break
    if latest_file is None:
        return {}

    try:
        payload = json.loads(latest_file.read_text())
    except Exception:
        return {}

    if not isinstance(payload, list):
        payload = [payload]

    out: Dict[str, Dict[str, float]] = {}
    for row in payload:
        scenario = row.get("scenario", "")
        out[scenario] = {
            "throughput": float(row.get("throughput_ops_per_sec", 0) or 0),
            "latency_p95": float(row.get("latency_p95_ms", 0) or 0),
        }
    return out


def _plot_grouped_bars(
    title: str,
    labels: list,
    baseline_vals: list,
    candidate_vals: list,
    ylabel: str,
    out_file: Path,
    log_scale: bool = False,
):
    x = np.arange(len(labels))
    width = 0.35

    fig, ax = plt.subplots(figsize=(10, 6))
    bars1 = ax.bar(x - width / 2, baseline_vals, width, label="Монолит", color=BASELINE_COLOR)
    bars2 = ax.bar(x + width / 2, candidate_vals, width, label="Распределённая", color=CANDIDATE_COLOR)

    ax.set_ylabel(ylabel, fontsize=12)
    ax.set_title(title, fontsize=13, fontweight="bold")
    ax.set_xticks(x)
    ax.set_xticklabels(labels, fontsize=11)
    ax.legend(fontsize=11)
    ax.grid(True, alpha=0.3, axis="y")

    if log_scale:
        ax.set_yscale("log")
        ax.set_ylim(bottom=1)

    for bar_group in [bars1, bars2]:
        for bar in bar_group:
            height = bar.get_height()
            if height > 0:
                label_text = f"{int(height)}" if height >= 10 else f"{height:.1f}"
                ax.annotate(
                    label_text,
                    xy=(bar.get_x() + bar.get_width() / 2, height),
                    xytext=(0, 4),
                    textcoords="offset points",
                    ha="center",
                    va="bottom",
                    fontsize=9,
                )

    plt.tight_layout()
    plt.savefig(out_file, dpi=200, bbox_inches="tight")
    plt.close(fig)


def _plot_two_bars(title: str, baseline_val: float, candidate_val: float, ylabel: str, out_file: Path):
    fig, ax = plt.subplots(figsize=(7, 5))
    labels = ["Монолит", "Распределённая"]
    values = [baseline_val, candidate_val]
    bars = ax.bar(labels, values, color=[BASELINE_COLOR, CANDIDATE_COLOR], width=0.5)

    ax.set_ylabel(ylabel, fontsize=12)
    ax.set_title(title, fontsize=13, fontweight="bold")
    ax.grid(True, alpha=0.3, axis="y")

    for bar in bars:
        height = bar.get_height()
        ax.annotate(
            f"{int(height)}",
            xy=(bar.get_x() + bar.get_width() / 2, height),
            xytext=(0, 4),
            textcoords="offset points",
            ha="center",
            va="bottom",
            fontsize=11,
            fontweight="bold",
        )

    delta = ((candidate_val - baseline_val) / baseline_val) * 100 if baseline_val else 0
    ax.text(
        0.5,
        0.92,
        f"Δ = +{delta:.0f}% (×{candidate_val / baseline_val:.1f})" if delta > 0 else f"Δ = {delta:.0f}%",
        transform=ax.transAxes,
        ha="center",
        fontsize=12,
        style="italic",
    )

    plt.tight_layout()
    plt.savefig(out_file, dpi=200, bbox_inches="tight")
    plt.close(fig)


KEY_STORAGE_SCENARIOS = [
    ("concurrent_write", "Парал.\nзапись"),
    ("sustained_throughput_500", "Целевая\n500 оп/с"),
    ("sustained_throughput_1000", "Целевая\n1000 оп/с"),
    ("sustained_throughput_2000", "Целевая\n2000 оп/с"),
]


def main() -> None:
    root = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("results")
    out_dir = root / "graphs"
    out_dir.mkdir(parents=True, exist_ok=True)

    baseline_root = root / "baseline"
    candidate_root = root / "candidate"

    # --- Chart 1: e2e latency by stage ---
    base_metrics = _load_stage_metrics(baseline_root)
    cand_metrics = _load_stage_metrics(candidate_root)

    stages = ["Номинальная\n(200 ВП)", "Стрессовая\n(500 ВП)", "Предельная\n(→3000 ВП)"]
    base_e2e = [base_metrics.get(s, {}).get("e2e") or 0 for s in stages]
    cand_e2e = [cand_metrics.get(s, {}).get("e2e") or 0 for s in stages]

    if any(v > 0 for v in base_e2e + cand_e2e):
        _plot_grouped_bars(
            "Сквозная задержка размещения пикселя P95",
            stages,
            base_e2e,
            cand_e2e,
            "e2e_pixel_latency P95",
            out_dir / "e2e_latency_by_stage.png",
            log_scale=True,
        )
        print(f"  [OK] e2e_latency_by_stage.png (values: base={base_e2e}, cand={cand_e2e})")
    else:
        print("  [SKIP] e2e_latency_by_stage.png — no k6 data")

    # --- Chart 2: WS connecting latency by stage ---
    base_ws = [base_metrics.get(s, {}).get("ws") or 0 for s in stages]
    cand_ws = [cand_metrics.get(s, {}).get("ws") or 0 for s in stages]

    if any(v > 0 for v in base_ws + cand_ws):
        _plot_grouped_bars(
            "Задержка установления WebSocket-соединения P95",
            stages,
            base_ws,
            cand_ws,
            "ws_connecting P95, мс",
            out_dir / "ws_latency_by_stage.png",
            log_scale=True,
        )
        print(f"  [OK] ws_latency_by_stage.png (values: base={[f'{v:.1f}' for v in base_ws]}, cand={[f'{v:.1f}' for v in cand_ws]})")
    else:
        print("  [SKIP] ws_latency_by_stage.png — no k6 data")

    # --- Chart 3: Saturation VUs ---
    base_sat = _load_saturation(baseline_root)
    cand_sat = _load_saturation(candidate_root)

    if base_sat and cand_sat:
        _plot_two_bars(
            "Точка насыщения (виртуальные пользователи)",
            base_sat,
            cand_sat,
            "Виртуальные пользователи (ВП)",
            out_dir / "saturation_vus_comparison.png",
        )
        print(f"  [OK] saturation_vus_comparison.png ({base_sat} vs {cand_sat})")
    else:
        print(f"  [SKIP] saturation_vus_comparison.png — base={base_sat}, cand={cand_sat}")

    # --- Charts 4 & 5: Storage throughput and latency ---
    base_bench = _load_benchmark_scenarios(baseline_root)
    cand_bench = _load_benchmark_scenarios(candidate_root)

    scenario_labels = []
    base_tp = []
    cand_tp = []
    base_lat = []
    cand_lat = []

    for scenario_key, label in KEY_STORAGE_SCENARIOS:
        if scenario_key in base_bench or scenario_key in cand_bench:
            scenario_labels.append(label)
            base_tp.append(base_bench.get(scenario_key, {}).get("throughput", 0))
            cand_tp.append(cand_bench.get(scenario_key, {}).get("throughput", 0))
            base_lat.append(base_bench.get(scenario_key, {}).get("latency_p95", 0))
            cand_lat.append(cand_bench.get(scenario_key, {}).get("latency_p95", 0))

    if scenario_labels:
        _plot_grouped_bars(
            "Пропускная способность хранилища (ключевые сценарии)",
            scenario_labels,
            base_tp,
            cand_tp,
            "Операций/с",
            out_dir / "storage_throughput_comparison.png",
            log_scale=True,
        )
        print(f"  [OK] storage_throughput_comparison.png ({len(scenario_labels)} scenarios)")

        _plot_grouped_bars(
            "Задержка P95 хранилища (ключевые сценарии)",
            scenario_labels,
            base_lat,
            cand_lat,
            "Задержка P95, мс",
            out_dir / "storage_latency_comparison.png",
            log_scale=False,
        )
        print(f"  [OK] storage_latency_comparison.png ({len(scenario_labels)} scenarios)")
    else:
        print("  [SKIP] storage charts — no benchmark data found")

    print(f"\nAll graphs saved to: {out_dir}")


if __name__ == "__main__":
    main()
