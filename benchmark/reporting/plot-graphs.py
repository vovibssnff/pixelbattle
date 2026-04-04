#!/usr/bin/env python3

import json
import sys
from pathlib import Path
import matplotlib
matplotlib.use('Agg')  # Use non-interactive backend
import matplotlib.pyplot as plt
import numpy as np

def load_benchmark_results(results_dir: str):
    """Load benchmark results."""
    results = {}
    results_path = Path(results_dir) / "benchmarks"
    
    if not results_path.exists():
        print(f"Warning: Benchmark results directory not found: {results_path}")
        return results
    
    for result_file in results_path.glob("*.json"):
        storage_type = result_file.stem.split("_")[0]
        if storage_type not in results:
            results[storage_type] = []
        
        with open(result_file, 'r') as f:
            data = json.load(f)
            if isinstance(data, list):
                results[storage_type].extend(data)
            else:
                results[storage_type].append(data)
    
    return results

def plot_comparison_graphs(results, output_dir: str):
    """Generate comparison graphs."""
    output_path = Path(output_dir)
    output_path.mkdir(parents=True, exist_ok=True)
    
    # Extract data for comparison
    storage_types = []
    scenarios = []
    throughputs = {}
    latencies_p95 = {}
    
    for storage_type in ["redis", "postgres", "sqlite"]:
        if storage_type not in results:
            continue
        
        storage_types.append(storage_type)
        for result in results[storage_type]:
            scenario = result.get("scenario", "unknown")
            if scenario not in scenarios:
                scenarios.append(scenario)
            
            if scenario not in throughputs:
                throughputs[scenario] = {}
            if scenario not in latencies_p95:
                latencies_p95[scenario] = {}
            
            throughputs[scenario][storage_type] = result.get("throughput", 0)
            latencies_p95[scenario][storage_type] = result.get("latency_p95_ms", 0)
    
    # Plot throughput comparison
    if throughputs:
        fig, ax = plt.subplots(figsize=(12, 6))
        x = np.arange(len(scenarios))
        width = 0.25
        
        for i, storage_type in enumerate(storage_types):
            values = [throughputs[s].get(storage_type, 0) for s in scenarios]
            ax.bar(x + i * width, values, width, label=storage_type)
        
        ax.set_xlabel('Scenario')
        ax.set_ylabel('Throughput (ops/sec)')
        ax.set_title('Write Throughput Comparison')
        ax.set_xticks(x + width)
        ax.set_xticklabels([s.replace('_', ' ').title() for s in scenarios])
        ax.legend()
        ax.grid(True, alpha=0.3)
        
        plt.tight_layout()
        plt.savefig(output_path / "throughput_comparison.png", dpi=300)
        print(f"Saved: {output_path / 'throughput_comparison.png'}")
        plt.close()
    
    # Plot latency comparison
    if latencies_p95:
        fig, ax = plt.subplots(figsize=(12, 6))
        x = np.arange(len(scenarios))
        width = 0.25
        
        for i, storage_type in enumerate(storage_types):
            values = [latencies_p95[s].get(storage_type, 0) for s in scenarios]
            ax.bar(x + i * width, values, width, label=storage_type)
        
        ax.set_xlabel('Scenario')
        ax.set_ylabel('P95 Latency (ms)')
        ax.set_title('P95 Latency Comparison')
        ax.set_xticks(x + width)
        ax.set_xticklabels([s.replace('_', ' ').title() for s in scenarios])
        ax.legend()
        ax.grid(True, alpha=0.3)
        
        plt.tight_layout()
        plt.savefig(output_path / "latency_comparison.png", dpi=300)
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
