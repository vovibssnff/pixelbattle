#!/usr/bin/env python3

import json
import os
import sys
from pathlib import Path
from datetime import datetime
from typing import Dict, List, Any

def load_benchmark_results(results_dir: str) -> Dict[str, List[Dict[str, Any]]]:
    """Load all benchmark result files."""
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

def generate_markdown_report(results: Dict[str, List[Dict[str, Any]]], output_file: str):
    """Generate a markdown comparison report."""
    report = []
    report.append("# Database Benchmark Comparison Report")
    report.append(f"\nGenerated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n")
    report.append("=" * 80)
    report.append("\n")
    
    # Executive Summary
    report.append("## Executive Summary\n")
    report.append("This report compares the performance of Redis, PostgreSQL, and SQLite")
    report.append("for the PixelBattle application under various workload scenarios.\n")
    
    # Results by Storage Type
    for storage_type in ["redis", "postgres", "sqlite"]:
        if storage_type not in results:
            continue
        
        report.append(f"\n## {storage_type.upper()} Results\n")
        report.append("| Scenario | Throughput (ops/sec) | P50 Latency (ms) | P95 Latency (ms) | P99 Latency (ms) | Errors |")
        report.append("|----------|---------------------|------------------|------------------|------------------|--------|")
        
        for result in results[storage_type]:
            scenario = result.get("scenario", "unknown")
            throughput = result.get("throughput", 0)
            p50 = result.get("latency_p50_ms", 0)
            p95 = result.get("latency_p95_ms", 0)
            p99 = result.get("latency_p99_ms", 0)
            errors = result.get("error_count", 0)
            
            report.append(f"| {scenario} | {throughput:.2f} | {p50:.2f} | {p95:.2f} | {p99:.2f} | {errors} |")
    
    # Comparison Section
    report.append("\n## Performance Comparison\n")
    report.append("### Write Throughput\n")
    
    for scenario in ["sequential_write", "concurrent_write", "mixed_workload"]:
        report.append(f"\n#### {scenario.replace('_', ' ').title()}\n")
        report.append("| Storage | Throughput (ops/sec) | P95 Latency (ms) |")
        report.append("|---------|---------------------|------------------|")
        
        for storage_type in ["redis", "postgres", "sqlite"]:
            if storage_type not in results:
                continue
            
            for result in results[storage_type]:
                if result.get("scenario") == scenario:
                    throughput = result.get("throughput", 0)
                    p95 = result.get("latency_p95_ms", 0)
                    report.append(f"| {storage_type} | {throughput:.2f} | {p95:.2f} |")
                    break
    
    # Recommendations
    report.append("\n## Recommendations\n")
    report.append("Based on the benchmark results:\n")
    report.append("- **Redis**: Best for high-throughput write workloads with low latency requirements\n")
    report.append("- **PostgreSQL**: Provides ACID guarantees but may have lower write throughput\n")
    report.append("- **SQLite**: Good for single-instance deployments but limited concurrency\n")
    
    # Write report
    report_text = "\n".join(report)
    with open(output_file, 'w') as f:
        f.write(report_text)
    
    print(f"Report generated: {output_file}")

def main():
    if len(sys.argv) < 2:
        results_dir = "results"
    else:
        results_dir = sys.argv[1]
    
    results = load_benchmark_results(results_dir)
    
    if not results:
        print("No benchmark results found. Please run benchmarks first.")
        sys.exit(1)
    
    output_file = Path(results_dir) / f"comparison_report_{datetime.now().strftime('%Y%m%d_%H%M%S')}.md"
    generate_markdown_report(results, str(output_file))
    
    print(f"\nReport saved to: {output_file}")

if __name__ == "__main__":
    main()
