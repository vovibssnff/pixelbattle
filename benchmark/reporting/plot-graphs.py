#!/usr/bin/env python3

import json
import sys
from pathlib import Path
from collections import defaultdict
import matplotlib

matplotlib.use("Agg")  # Use non-interactive backend
import matplotlib.pyplot as plt
import numpy as np


def load_benchmark_results(results_dir: str):
    """Load benchmark results: latest epoch per storage type only."""
    results = {}
    results_path = Path(results_dir) / "benchmarks"

    if not results_path.exists():
        print(f"Warning: Benchmark results directory not found: {results_path}")
        return results

    files_by_type = defaultdict(list)
    for result_file in results_path.glob("*.json"):
        storage_type = result_file.stem.split("_")[0]
        files_by_type[storage_type].append(result_file)

    for storage_type, files in files_by_type.items():
        latest = max(files, key=lambda p: p.stem)
        with open(latest, "r") as f:
            data = json.load(f)
        if isinstance(data, list):
            results[storage_type] = data
        else:
            results[storage_type] = [data]

    return results


def _collect_scenario_metrics(results):
    """
    Build per-scenario metrics from latest results. Duplicate scenario names
    (same storage) last-write-wins — should not occur with latest-epoch load.
    """
    throughputs = {}
    latencies_p95 = {}

    for storage_type in ["redis", "postgres", "sqlite"]:
        if storage_type not in results:
            continue
        for result in results[storage_type]:
            scenario = result.get("scenario", "unknown")
            tp = float(result.get("throughput_ops_per_sec", 0) or 0)
            p95 = float(result.get("latency_p95_ms", 0) or 0)
            if scenario not in throughputs:
                throughputs[scenario] = {}
            if scenario not in latencies_p95:
                latencies_p95[scenario] = {}
            throughputs[scenario][storage_type] = tp
            latencies_p95[scenario][storage_type] = p95

    scenarios = sorted(throughputs.keys())
    return scenarios, throughputs, latencies_p95


def plot_comparison_graphs(results, output_dir: str):
    """Generate comparison graphs."""
    output_path = Path(output_dir)
    output_path.mkdir(parents=True, exist_ok=True)

    storage_types = [s for s in ["redis", "postgres", "sqlite"] if s in results]
    if not storage_types:
        print("No storage results to plot.")
        return

    scenarios, throughputs, latencies_p95 = _collect_scenario_metrics(results)
    if not scenarios:
        print("No scenarios to plot.")
        return

    width = 0.25
    x = np.arange(len(scenarios))

    # Throughput comparison (linear scale)
    fig, ax = plt.subplots(figsize=(14, 7))
    for i, storage_type in enumerate(storage_types):
        values = [throughputs[s].get(storage_type, 0) for s in scenarios]
        ax.bar(x + i * width, values, width, label=storage_type)

    ax.set_xlabel("Scenario")
    ax.set_ylabel("Throughput (ops/sec)")
    ax.set_title("Write Throughput Comparison")
    ax.set_xticks(x + width)
    ax.set_xticklabels([s.replace("_", " ").title() for s in scenarios], rotation=45, ha="right")
    ax.legend()
    ax.grid(True, alpha=0.3)
    plt.tight_layout()
    plt.savefig(output_path / "throughput_comparison.png", dpi=300, bbox_inches="tight")
    print(f"Saved: {output_path / 'throughput_comparison.png'}")
    plt.close()

    # P95 latency — log scale to avoid single outlier dominating the chart
    fig, ax = plt.subplots(figsize=(14, 7))
    for i, storage_type in enumerate(storage_types):
        raw = [latencies_p95[s].get(storage_type, 0) for s in scenarios]
        values = [max(v, 1e-6) for v in raw]
        ax.bar(x + i * width, values, width, label=storage_type)

    ax.set_xlabel("Scenario")
    ax.set_ylabel("P95 Latency (ms, log scale)")
    ax.set_title("P95 Latency Comparison")
    ax.set_yscale("log")
    ax.set_xticks(x + width)
    ax.set_xticklabels([s.replace("_", " ").title() for s in scenarios], rotation=45, ha="right")
    ax.legend()
    ax.grid(True, alpha=0.3, which="both")
    plt.tight_layout()
    plt.savefig(output_path / "latency_comparison.png", dpi=300, bbox_inches="tight")
    print(f"Saved: {output_path / 'latency_comparison.png'}")
    plt.close()


def main():
    if len(sys.argv) < 2:
        results_dir = "results"
    else:
        results_dir = sys.argv[1]

    results = load_benchmark_results(results_dir)

    if not results:
        print("No benchmark results found. Please run benchmarks first.")
        sys.exit(1)

    output_dir = Path(results_dir) / "graphs"
    plot_comparison_graphs(results, str(output_dir))

    print(f"\nGraphs saved to: {output_dir}")


if __name__ == "__main__":
    main()
