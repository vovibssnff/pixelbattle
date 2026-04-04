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
		sb.WriteString("\n")
	}

	sb.WriteString("=" + strings.Repeat("=", 80) + "\n")

	return sb.String()
}
