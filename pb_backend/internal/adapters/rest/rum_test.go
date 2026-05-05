package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// goldenRUMPayload is stable JSON matching the frontend wire format (field order for readability).
const goldenRUMPayload = `{"session":"","version":"1","fps_avg":59.5,"fps_p10":55,"frame_time_ms_p95":16.7,"lcp_ms":1200,"inp_ms":80,"ttfb_ms":95,"ws_render_lat_ms_p95":25,"errors":[]}`

func TestRUMBeaconGoldenJSONAccepted(t *testing.T) {
	if err := json.Unmarshal([]byte(goldenRUMPayload), new(rumBeaconPayload)); err != nil {
		t.Fatalf("fixture must unmarshal: %v", err)
	}

	h := &RestHandlers{}
	srv := httptest.NewServer(http.HandlerFunc(h.HandleRUMBeacon))
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodPost, srv.URL, bytes.NewReader([]byte(goldenRUMPayload)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status %d want %d", resp.StatusCode, http.StatusAccepted)
	}
}
