package reporter

import (
	"fmt"
	"pb_backend/cmd/benchmark/benchmark"
	"strings"
)

func GenerateReport(results []benchmark.BenchmarkResult, storageType string) string {
	var sb strings.Builder

	sb.WriteString("=" + strings.Repeat("=", 80) + "\n")
	sb.WriteString(fmt.Sprintf("BENCHMARK RESULTS: %s\n", strings.ToUpper(storageType)))
	sb.WriteString("=" + strings.Repeat("=", 80) + "\n\n")

	for _, result := range results {
		sb.WriteString(fmt.Sprintf("Scenario: %s\n", result.Scenario))
		sb.WriteString(strings.Repeat("-", 80) + "\n")
		sb.WriteString(fmt.Sprintf("  Duration:        %.2f seconds\n", result.Duration))
		sb.WriteString(fmt.Sprintf("  Operations:      %d\n", result.Operations))
		sb.WriteString(fmt.Sprintf("  Throughput:      %.2f ops/sec\n", result.Throughput))
		sb.WriteString(fmt.Sprintf("  Latency P50:     %.2f ms\n", result.LatencyP50))
		sb.WriteString(fmt.Sprintf("  Latency P95:     %.2f ms\n", result.LatencyP95))
		sb.WriteString(fmt.Sprintf("  Latency P99:     %.2f ms\n", result.LatencyP99))
		sb.WriteString(fmt.Sprintf("  Latency Mean:    %.2f ms\n", result.LatencyMean))
		sb.WriteString(fmt.Sprintf("  Latency Min:     %.2f ms\n", result.LatencyMin))
		sb.WriteString(fmt.Sprintf("  Latency Max:     %.2f ms\n", result.LatencyMax))
		sb.WriteString(fmt.Sprintf("  Errors:          %d\n", result.ErrorCount))

		if result.TargetRate > 0 {
			sb.WriteString(fmt.Sprintf("  Target Rate:     %.0f ops/sec\n", result.TargetRate))
		}
		if result.HistoryDepth > 0 {
			sb.WriteString(fmt.Sprintf("  History Depth:   %d total rows\n", result.HistoryDepth))
		}
		if result.WarmupSkipped > 0 {
			sb.WriteString(fmt.Sprintf("  Warmup Skipped:  %d ops\n", result.WarmupSkipped))
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
				sb.WriteString(fmt.Sprintf("    First bucket (%.0fs): p50=%.2fms  p99=%.2fms  ops=%d\n",
					first.BucketStartSec, first.LatencyP50, first.LatencyP99, first.Operations))
				sb.WriteString(fmt.Sprintf("    Last  bucket (%.0fs): p50=%.2fms  p99=%.2fms  ops=%d\n",
					last.BucketStartSec, last.LatencyP50, last.LatencyP99, last.Operations))

				if first.LatencyP50 > 0 {
					change := ((last.LatencyP50 - first.LatencyP50) / first.LatencyP50) * 100
					direction := "stable"
					if change > 10 {
						direction = "degrading"
					} else if change < -10 {
						direction = "improving"
					}
					sb.WriteString(fmt.Sprintf("    Trend: %s (p50 %.1f%%)\n", direction, change))
				}
			}
		}

		sb.WriteString("\n")
	}

	sb.WriteString("=" + strings.Repeat("=", 80) + "\n")

	return sb.String()
}
