package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type placeAndWatch struct {
	WS2Render     []float64 `json:"wsToRender"`
	Click2Render  []float64 `json:"clickToRender"`
}

var (
	clickToRender = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "playwright_click_to_render_ms",
			Help:    "Playwright measured click-to-render latency in milliseconds",
			Buckets: []float64{5, 10, 20, 40, 80, 120, 250, 500, 1000, 2000},
		},
	)
)

func main() {
	resultsDir := flag.String("results-dir", "results/playwright", "Directory with playwright JSON artifacts")
	port := flag.Int("port", 9092, "Metrics listen port")
	flag.Parse()

	prometheus.MustRegister(clickToRender)

	files, _ := filepath.Glob(filepath.Join(*resultsDir, "place-and-watch-*.json"))
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var payload placeAndWatch
		if err := json.Unmarshal(b, &payload); err != nil {
			continue
		}
		for _, v := range payload.Click2Render {
			clickToRender.Observe(v)
		}
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	addr := fmt.Sprintf(":%d", *port)
	_ = http.ListenAndServe(addr, mux)
}

