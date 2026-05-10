package rest

import (
	"encoding/json"
	"net/http"
	"os"
	"pb_backend/internal/core/service"
	"strconv"
	"time"
)

type rumCorrection struct {
	Reason string `json:"reason"`
}

type rumBeaconPayload struct {
	Session          string          `json:"session"`
	Version          string          `json:"version"`
	FPSAvg           float64         `json:"fps_avg"`
	FPSP10           float64         `json:"fps_p10"`
	FrameTimeP95Ms   float64         `json:"frame_time_ms_p95"`
	LCPMs            float64         `json:"lcp_ms"`
	INPMs            float64         `json:"inp_ms"`
	TTFBMs           float64         `json:"ttfb_ms"`
	WSRenderLatP95Ms float64         `json:"ws_render_lat_ms_p95"`
	Errors           []string        `json:"errors"`
	Corrections      []rumCorrection `json:"optimistic_corrections"`
}

func (h *RestHandlers) HandleRUMBeacon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		service.IncrementRUMBeacon("method_not_allowed")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()
	var payload rumBeaconPayload
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&payload); err != nil {
		service.IncrementRUMBeacon("decode_error")
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if payload.FPSAvg > 0 {
		service.ObserveClientFPS(payload.FPSAvg)
	}
	if payload.FrameTimeP95Ms > 0 {
		service.ObserveClientFrameTimeP95Ms(payload.FrameTimeP95Ms)
	}
	if payload.LCPMs > 0 {
		service.ObserveClientWebVitalMs("lcp", payload.LCPMs)
	}
	if payload.INPMs > 0 {
		service.ObserveClientWebVitalMs("inp", payload.INPMs)
	}
	if payload.TTFBMs > 0 {
		service.ObserveClientWebVitalMs("ttfb", payload.TTFBMs)
	}
	if payload.WSRenderLatP95Ms > 0 {
		service.ObserveClientWSRenderLatencyMs(payload.WSRenderLatP95Ms)
		service.ObserveE2EPixelLatency("rum_ws_to_render", floatMillisToDuration(payload.WSRenderLatP95Ms))
	}

	for range payload.Errors {
		service.IncrementWSError("client")
	}

	for _, c := range payload.Corrections {
		service.IncrementOptimisticCorrection(c.Reason)
	}

	service.IncrementRUMBeacon("success")
	w.WriteHeader(http.StatusAccepted)
}

func floatMillisToDuration(v float64) time.Duration {
	return time.Duration(v * float64(time.Millisecond))
}

func (h *RestHandlers) HandleClientConfig(w http.ResponseWriter, _ *http.Request) {
	sampleRate := 0.05
	if raw := os.Getenv("RUM_SAMPLE_RATE"); raw != "" {
		if parsed, err := strconv.ParseFloat(raw, 64); err == nil && parsed >= 0 && parsed <= 1 {
			sampleRate = parsed
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]float64{
		"rum_sample_rate": sampleRate,
	})
}
