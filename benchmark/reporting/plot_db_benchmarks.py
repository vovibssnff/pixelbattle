#!/usr/bin/env python3
"""
Plot database-layer (Go binary) benchmark results from results/<dir>/benchmarks/*.json.

Phases match pb_backend/cmd/benchmark/benchmark/benchmark.go RunAllBenchmarks order.

Each phase 1–6 PNG: throughput + P95 latency. Latency uses a linear Y axis when the
data range allows; otherwise log with human-readable tick labels (no 10^n notation).
Phase 6 uses three rows: throughput, then write latency, then read latency (separate
scales so sub-ms writes are visible next to multi-second reads).

Uses the latest JSON file per storage backend (same rule as plot-graphs.py).
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from collections import defaultdict
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt
import matplotlib.ticker as mticker
import numpy as np

STORAGE_ORDER = ["redis", "postgres", "sqlite"]

# Use linear latency axis when max/min ratio stays below this (same order of magnitude band).
_LATENCY_LINEAR_RATIO_MAX = 80.0

# Strip "Phase N — " from titles for display (filenames still use db_phaseNN_).
_PHASE_NUM_PREFIX_RE = re.compile(r"^Phase\s+\d+\s*[—–-]\s*", re.IGNORECASE)


def strip_phase_prefix(text: str) -> str:
    return _PHASE_NUM_PREFIX_RE.sub("", text).strip()


def _format_ms_tick(y: float) -> str:
    """Plain ms text for axis ticks (no scientific / math superscripts)."""
    if not np.isfinite(y) or y <= 0:
        return ""
    if y >= 1000:
        return str(int(round(y)))
    if y >= 100:
        return str(int(round(y)))
    if y >= 10:
        return f"{y:.0f}"
    if y >= 1:
        s = f"{y:.1f}"
        return s.rstrip("0").rstrip(".")
    s = f"{y:.3f}"
    return s.rstrip("0").rstrip(".")


def _apply_smart_latency_axis(ax: plt.Axes, values_ms: List[float]) -> None:
    """
    Choose linear Y (from 0) when the spread is modest; otherwise log with explicit
    numeric labels at 1–2–5–10 style positions.
    """
    vals = [float(v) for v in values_ms if np.isfinite(v) and v > 0]
    ax.set_ylabel("P95 latency (ms)")
    if not vals:
        ax.set_ylim(0, 1)
        _plain_scalar_y_axis(ax)
        return

    lo, hi = min(vals), max(vals)
    ratio = hi / max(lo, 1e-15)

    if ratio <= _LATENCY_LINEAR_RATIO_MAX:
        ax.set_yscale("linear")
        top = hi * 1.12
        ax.set_ylim(0, max(top, 1e-6))
        ax.yaxis.set_major_locator(
            mticker.MaxNLocator(nbins=9, steps=[1, 2, 2.5, 5, 10], min_n_ticks=4)
        )
        _plain_scalar_y_axis(ax)
        return

    # Wide range: log scale with 1–2–5 ticks per decade; labels stay plain ms (not 10^n).
    ax.set_yscale("log")
    ax.yaxis.set_major_locator(mticker.LogLocator(base=10.0, subs=(1.0, 2.0, 5.0)))
    ax.yaxis.set_minor_formatter(mticker.NullFormatter())
    ax.yaxis.set_major_formatter(mticker.FuncFormatter(lambda y, _: _format_ms_tick(y)))
    ax.set_ylim(bottom=max(lo * 0.35, 1e-6), top=hi * 1.2)


def _plain_scalar_y_axis(ax: plt.Axes) -> None:
    """No scientific notation, no offset like +1e6 on linear axes."""
    fmt = mticker.ScalarFormatter(useMathText=False)
    fmt.set_scientific(False)
    fmt.set_useOffset(False)
    ax.yaxis.set_major_formatter(fmt)


def _plain_throughput_axis(ax: plt.Axes) -> None:
    _plain_scalar_y_axis(ax)
    ax.ticklabel_format(style="plain", axis="y")


@dataclass(frozen=True)
class BarPhase:
    """Single- or multi-scenario phase rendered as grouped bars."""

    num: int
    slug: str
    title: str
    scenarios: Tuple[str, ...]


# Order must match RunAllBenchmarks in benchmark.go (lines 132–154).
BAR_PHASES: Tuple[BarPhase, ...] = (
    BarPhase(1, "sequential_write", "Phase 1 — Sequential write (fills canvas)", ("sequential_write",)),
    BarPhase(2, "heatmap_load", "Phase 2 — HeatMap load (post-fill)", ("heatmap_load",)),
    BarPhase(3, "concurrent_write", "Phase 3 — Concurrent write", ("concurrent_write",)),
    BarPhase(4, "full_canvas_read", "Phase 4 — Full canvas read", ("full_canvas_read",)),
    BarPhase(5, "mixed_workload", "Phase 5 — Mixed workload", ("mixed_workload",)),
    BarPhase(
        6,
        "read_under_write",
        "Phase 6 — Read under write",
        ("read_under_write_writes", "read_under_write_reads"),
    ),
)


def phase_display_title(phase: BarPhase) -> str:
    """Phase name without leading 'Phase N —' (e.g. 'Mixed workload')."""
    return strip_phase_prefix(phase.title)


def load_latest_by_storage(results_dir: Path) -> Dict[str, List[Dict[str, Any]]]:
    """Load newest epoch JSON per storage type."""
    out: Dict[str, List[Dict[str, Any]]] = {}
    benchmarks = results_dir / "benchmarks"
    if not benchmarks.is_dir():
        return out

    files_by_type: Dict[str, List[Path]] = defaultdict(list)
    for f in benchmarks.glob("*.json"):
        stem = f.stem
        if "_benchmark_" not in stem:
            continue
        storage = stem.split("_benchmark_")[0]
        files_by_type[storage].append(f)

    for storage, files in files_by_type.items():
        latest = max(files, key=lambda p: p.stem)
        with open(latest, "r", encoding="utf-8") as fh:
            data = json.load(fh)
        out[storage] = data if isinstance(data, list) else [data]

    return out


def index_by_scenario(rows: List[Dict[str, Any]]) -> Dict[str, Dict[str, Any]]:
    return {r.get("scenario", ""): r for r in rows}


def _scenario_xtick_labels(scenarios: List[str]) -> List[str]:
    return [s.replace("_", " ").title() for s in scenarios]


def _bar_grouped(
    ax: plt.Axes,
    scenarios: List[str],
    results: Dict[str, List[Dict[str, Any]]],
    storages: List[str],
    value_key: str,
    title: Optional[str],
    ylabel: str,
    *,
    show_xlabels: bool = True,
    xlabel: Optional[str] = None,
) -> List[float]:
    """Draw grouped bars; returns all plotted Y values (for latency axis tuning)."""
    all_heights: List[float] = []
    if not scenarios or not storages:
        return all_heights
    n = len(scenarios)
    m = len(storages)
    width = min(0.8 / max(m, 1), 0.22)
    x = np.arange(n)
    for i, st in enumerate(storages):
        by_s = index_by_scenario(results.get(st, []))
        vals: List[float] = []
        for sc in scenarios:
            row = by_s.get(sc)
            if row is None:
                vals.append(0.0)
            else:
                v = row.get(value_key, 0) or 0
                vals.append(float(v))
        all_heights.extend(vals)
        ax.bar(x + (i - (m - 1) / 2) * width, vals, width, label=st)
    if title:
        ax.set_title(title, fontsize=10)
    ax.set_ylabel(ylabel)
    ax.set_xticks(x)
    if show_xlabels:
        rot = 18 if n <= 2 else 28
        ha = "center" if n <= 2 else "right"
        ax.set_xticklabels(_scenario_xtick_labels(scenarios), rotation=rot, ha=ha)
    else:
        ax.set_xticklabels([])
    if xlabel:
        ax.set_xlabel(xlabel, fontsize=9)
    ax.legend(loc="upper right", fontsize=8)
    ax.grid(True, alpha=0.3, axis="y")
    return all_heights


def _save_fig(fig: plt.Figure, path: Path, *, hspace: float = 0.45) -> None:
    fig.subplots_adjust(left=0.11, right=0.97, top=0.88, bottom=0.10, hspace=hspace)
    fig.savefig(path, dpi=200, bbox_inches="tight", pad_inches=0.4)


def plot_bar_phases(
    results: Dict[str, List[Dict[str, Any]]], out_dir: Path
) -> None:
    storages = [s for s in STORAGE_ORDER if s in results]
    if not storages:
        return
    out_dir.mkdir(parents=True, exist_ok=True)

    for phase in BAR_PHASES:
        scenarios = list(phase.scenarios)

        # Phase 6: writes vs reads throughput differ by ~4–5 orders of magnitude; use two
        # side-by-side throughput panels so read bars are not crushed on one Y scale.
        if phase.num == 6:
            fig = plt.figure(figsize=(10, 9.5))
            gs = fig.add_gridspec(3, 2, height_ratios=[1, 1, 1], hspace=0.5, wspace=0.32)
            fig.suptitle(phase_display_title(phase), fontsize=11, fontweight="medium", y=0.995)
            ax_wtp = fig.add_subplot(gs[0, 0])
            ax_rtp = fig.add_subplot(gs[0, 1])
            ax_wlat = fig.add_subplot(gs[1, :])
            ax_rlat = fig.add_subplot(gs[2, :])

            _bar_grouped(
                ax_wtp,
                ["read_under_write_writes"],
                results,
                storages,
                "throughput_ops_per_sec",
                "Writes throughput",
                "Throughput (ops/sec)",
                show_xlabels=False,
            )
            _plain_throughput_axis(ax_wtp)

            _bar_grouped(
                ax_rtp,
                ["read_under_write_reads"],
                results,
                storages,
                "throughput_ops_per_sec",
                "Reads throughput",
                "Throughput (ops/sec)",
                show_xlabels=False,
            )
            _plain_throughput_axis(ax_rtp)

            h1 = _bar_grouped(
                ax_wlat,
                ["read_under_write_writes"],
                results,
                storages,
                "latency_p95_ms",
                "P95 latency — writes",
                "P95 latency (ms)",
                show_xlabels=False,
            )
            _apply_smart_latency_axis(ax_wlat, h1)

            h2 = _bar_grouped(
                ax_rlat,
                ["read_under_write_reads"],
                results,
                storages,
                "latency_p95_ms",
                "P95 latency — reads",
                "P95 latency (ms)",
                show_xlabels=False,
            )
            _apply_smart_latency_axis(ax_rlat, h2)
            fig.subplots_adjust(left=0.1, right=0.97, top=0.90, bottom=0.06)
        else:
            fig, axes = plt.subplots(2, 1, figsize=(10, 7), height_ratios=[1, 1])
            fig.suptitle(phase_display_title(phase), fontsize=11, fontweight="medium", y=0.995)
            _bar_grouped(
                axes[0],
                scenarios,
                results,
                storages,
                "throughput_ops_per_sec",
                "Throughput",
                "Throughput (ops/sec)",
                show_xlabels=False,
            )
            _plain_throughput_axis(axes[0])

            lat_vals = _bar_grouped(
                axes[1],
                scenarios,
                results,
                storages,
                "latency_p95_ms",
                "P95 latency",
                "P95 latency (ms)",
                show_xlabels=True,
            )
            _apply_smart_latency_axis(axes[1], lat_vals)

        name = f"db_phase{phase.num:02d}_{phase.slug}.png"
        p = out_dir / name
        hspace = 0.45
        if phase.num != 6:
            _save_fig(fig, p, hspace=hspace)
        else:
            fig.savefig(p, dpi=200, bbox_inches="tight", pad_inches=0.4)
        plt.close(fig)
        print(f"Saved: {p}")


def collect_sustained(
    results: Dict[str, List[Dict[str, Any]]],
) -> Tuple[List[int], Dict[str, List[Tuple[int, float, float]]]]:
    rate_re = re.compile(r"^sustained_throughput_(\d+)$")
    by_storage: Dict[str, Dict[int, Dict[str, Any]]] = defaultdict(dict)
    all_rates: set[int] = set()

    for st, rows in results.items():
        for r in rows:
            m = rate_re.match(r.get("scenario", ""))
            if not m:
                continue
            rate = int(m.group(1))
            all_rates.add(rate)
            by_storage[st][rate] = r

    rates = sorted(all_rates)
    series: Dict[str, List[Tuple[int, float, float]]] = {}
    for st in STORAGE_ORDER:
        if st not in by_storage:
            continue
        pts = []
        for rate in rates:
            row = by_storage[st].get(rate)
            if row is None:
                continue
            tp = float(row.get("throughput_ops_per_sec", 0) or 0)
            p95 = float(row.get("latency_p95_ms", 0) or 0)
            pts.append((rate, tp, p95))
        if pts:
            series[st] = pts

    return rates, series


def plot_phase7_sustained(results: Dict[str, List[Dict[str, Any]]], out_dir: Path) -> None:
    rates, series = collect_sustained(results)
    if not series:
        print("No sustained_throughput_* rows; skipping Phase 7 plots.")
        return

    title_base = strip_phase_prefix("Phase 7 — Sustained throughput")

    fig, ax = plt.subplots(figsize=(8, 5))
    for st in STORAGE_ORDER:
        if st not in series:
            continue
        xs = [p[0] for p in series[st]]
        ys = [p[1] for p in series[st]]
        ax.plot(xs, ys, marker="o", label=st)
    if rates:
        ax.plot(rates, rates, "k--", alpha=0.4, label="ideal (target = achieved)")
    ax.set_xlabel("Target rate (ops/sec)")
    ax.set_ylabel("Achieved throughput (ops/sec)")
    ax.set_title(f"{title_base} — achieved vs target")
    ax.legend()
    ax.grid(True, alpha=0.3)
    _plain_throughput_axis(ax)
    p = out_dir / "db_phase07_sustained_throughput.png"
    _save_fig(fig, p)
    plt.close(fig)
    print(f"Saved: {p}")

    fig, ax = plt.subplots(figsize=(8, 5))
    all_p95: List[float] = []
    for st in STORAGE_ORDER:
        if st not in series:
            continue
        xs = [p[0] for p in series[st]]
        ys = [max(p[2], 1e-9) for p in series[st]]
        all_p95.extend(ys)
        ax.plot(xs, ys, marker="o", label=st)
    ax.set_xlabel("Target rate (ops/sec)")
    ax.set_ylabel("P95 latency (ms)")
    ax.set_title(f"{title_base} — P95 latency vs target rate")
    ax.legend()
    ax.grid(True, alpha=0.3, which="both")
    _apply_smart_latency_axis(ax, all_p95)
    p = out_dir / "db_phase07_sustained_latency_p95.png"
    _save_fig(fig, p)
    plt.close(fig)
    print(f"Saved: {p}")


def collect_history_rounds(
    results: Dict[str, List[Dict[str, Any]]],
) -> Optional[Tuple[List[int], Dict[str, Dict[str, List[Optional[Dict[str, Any]]]]]]]:
    pat = re.compile(r"^history_growth_(write|read|heatmap)_round_(\d+)$")
    structure: Dict[str, Dict[int, Dict[str, Dict[str, Any]]]] = defaultdict(
        lambda: defaultdict(dict)
    )
    rounds_set: set[int] = set()

    for st, rows in results.items():
        for r in rows:
            sc = r.get("scenario", "")
            m = pat.match(sc)
            if not m:
                continue
            kind, rn = m.group(1), int(m.group(2))
            rounds_set.add(rn)
            structure[st][rn][kind] = r

    if not rounds_set:
        return None
    rounds = sorted(rounds_set)
    kinds = ["write", "read", "heatmap"]
    out: Dict[str, Dict[str, List[Optional[Dict[str, Any]]]]] = {}
    for st in STORAGE_ORDER:
        if st not in structure:
            continue
        out[st] = {}
        for kind in kinds:
            out[st][kind] = []
            for rn in rounds:
                out[st][kind].append(structure[st].get(rn, {}).get(kind))

    return rounds, out


def plot_phase8_history(results: Dict[str, List[Dict[str, Any]]], out_dir: Path) -> None:
    parsed = collect_history_rounds(results)
    if not parsed:
        print("No history_growth_*_round_* rows; skipping Phase 8 plot.")
        return
    rounds, by_st = parsed
    storages = [s for s in STORAGE_ORDER if s in by_st]
    if not storages:
        return

    fig, axes = plt.subplots(1, 3, figsize=(14, 4.8))
    kinds = ["write", "read", "heatmap"]
    titles = [
        strip_phase_prefix("Phase 8 — History growth — write (throughput)"),
        strip_phase_prefix("Phase 8 — History growth — read (P95 ms)"),
        strip_phase_prefix("Phase 8 — History growth — heatmap (P95 ms)"),
    ]
    y_keys = ["throughput_ops_per_sec", "latency_p95_ms", "latency_p95_ms"]
    is_latency_flags = [False, True, True]

    x = np.arange(len(rounds))
    width = min(0.22, 0.8 / max(len(storages), 1))

    for ax, kind, title, yk, is_lat in zip(axes, kinds, titles, y_keys, is_latency_flags):
        heights: List[float] = []
        for i, st in enumerate(storages):
            ys = []
            for row in by_st[st][kind]:
                if row is None:
                    ys.append(0.0)
                else:
                    ys.append(float(row.get(yk, 0) or 0))
            heights.extend(ys)
            ax.bar(x + (i - (len(storages) - 1) / 2) * width, ys, width, label=st)
        ax.set_xticks(x)
        ax.set_xticklabels([f"round {r}" for r in rounds])
        ax.set_title(title)
        ax.grid(True, alpha=0.3, axis="y")
        if is_lat:
            _apply_smart_latency_axis(ax, heights)
        else:
            ax.set_ylabel("Throughput (ops/sec)")
            _plain_throughput_axis(ax)
        ax.legend(fontsize=8)

    p = out_dir / "db_phase08_history_growth.png"
    _save_fig(fig, p)
    plt.close(fig)
    print(f"Saved: {p}")


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Plot DB-layer benchmark JSON results (one figure per RunAllBenchmarks phase)."
    )
    parser.add_argument(
        "results_dir",
        nargs="?",
        default="results",
        help="Directory containing benchmarks/ (default: results)",
    )
    parser.add_argument(
        "-o",
        "--output-subdir",
        default="db_benchmarks",
        help="Subfolder under <results>/graphs/ for PNG output (default: db_benchmarks)",
    )
    args = parser.parse_args()
    results_path = Path(args.results_dir).resolve()
    graphs = results_path / "graphs" / args.output_subdir

    results = load_latest_by_storage(results_path)
    if not results:
        print(f"No benchmark JSON found under {results_path / 'benchmarks'}", file=sys.stderr)
        sys.exit(1)

    print(f"Loaded backends: {', '.join(sorted(results.keys()))}")
    plot_bar_phases(results, graphs)
    plot_phase7_sustained(results, graphs)
    plot_phase8_history(results, graphs)
    print(f"\nAll phase plots written to: {graphs}")


if __name__ == "__main__":
    main()
