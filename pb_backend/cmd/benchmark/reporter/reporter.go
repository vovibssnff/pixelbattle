package reporter

import (
	"fmt"
	"pb_backend/cmd/benchmark/benchmark"
	"strings"
)

func GenerateReport(results []benchmark.BenchmarkResult, storageType string) string {
	var sb strings.Builder

	sb.WriteString("=" + strings.Repeat("=", 80) + "\n")
	fmt.Fprintf(&sb, "BENCHMARK RESULTS: %s\n", strings.ToUpper(storageType))
	sb.WriteString("=" + strings.Repeat("=", 80) + "\n\n")

	for _, result := range results {
		fmt.Fprintf(&sb, "Scenario: %s\n", result.Scenario)
		sb.WriteString(strings.Repeat("-", 80) + "\n")
		fmt.Fprintf(&sb, "  Duration:        %.2f seconds\n", result.Duration)
		fmt.Fprintf(&sb, "  Operations:      %d\n", result.Operations)
		fmt.Fprintf(&sb, "  Throughput:      %.2f ops/sec\n", result.Throughput)
		fmt.Fprintf(&sb, "  Latency P50:     %.2f ms\n", result.LatencyP50)
		fmt.Fprintf(&sb, "  Latency P95:     %.2f ms\n", result.LatencyP95)
		fmt.Fprintf(&sb, "  Latency P99:     %.2f ms\n", result.LatencyP99)
		fmt.Fprintf(&sb, "  Latency Mean:    %.2f ms\n", result.LatencyMean)
		fmt.Fprintf(&sb, "  Latency Min:     %.2f ms\n", result.LatencyMin)
		fmt.Fprintf(&sb, "  Latency Max:     %.2f ms\n", result.LatencyMax)
		fmt.Fprintf(&sb, "  Errors:          %d\n", result.ErrorCount)

		if result.TargetRate > 0 {
			fmt.Fprintf(&sb, "  Target Rate:     %.0f ops/sec\n", result.TargetRate)
		}
		if result.HistoryDepth > 0 {
			fmt.Fprintf(&sb, "  History Depth:   %d total rows\n", result.HistoryDepth)
		}
		if result.WarmupSkipped > 0 {
			fmt.Fprintf(&sb, "  Warmup Skipped:  %d ops\n", result.WarmupSkipped)
		}
		if result.Operations == 0 && result.WarmupSkipped > 0 {
			sb.WriteString("  *** All operations fell within warmup window; no measured latency data ***\n")
		}

		if len(result.TimeBuckets) > 1 {
			sb.WriteString("  Time Trend:\n")
			first := result.TimeBuckets[0]
			last := result.TimeBuckets[len(result.TimeBuckets)-1]
			// Find the last bucket that actually has operations
			for i := len(result.TimeBuckets) - 1; i >= 0; i-- {
				if result.TimeBuckets[i].Operations > 0 {
					last = result.TimeBuckets[i]
					break
				}
			}
			if first.Operations > 0 && last.Operations > 0 {
				fmt.Fprintf(&sb, "    First bucket (%.0fs): p50=%.2fms  p99=%.2fms  ops=%d\n",
					first.BucketStartSec, first.LatencyP50, first.LatencyP99, first.Operations)
				fmt.Fprintf(&sb, "    Last  bucket (%.0fs): p50=%.2fms  p99=%.2fms  ops=%d\n",
					last.BucketStartSec, last.LatencyP50, last.LatencyP99, last.Operations)

				if first.LatencyP50 > 0 {
					change := ((last.LatencyP50 - first.LatencyP50) / first.LatencyP50) * 100
					direction := "stable"
					if change > 10 {
						direction = "degrading"
					} else if change < -10 {
						direction = "improving"
					}
					fmt.Fprintf(&sb, "    Trend: %s (p50 %.1f%%)\n", direction, change)
				}
			}
		}

		sb.WriteString("\n")
	}

	sb.WriteString("=" + strings.Repeat("=", 80) + "\n")

	return sb.String()
}
