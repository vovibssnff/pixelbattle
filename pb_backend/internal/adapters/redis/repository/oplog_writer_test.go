package repository

import "testing"

func TestWriterFromPixelPayload(t *testing.T) {
	if got := writerFromPixelPayload([]byte(`{"userid":"u1","faculty":"f","color":[1,2,3],"timestamp":1}`)); got != "u1" {
		t.Fatalf("string userid: %q", got)
	}
	if got := writerFromPixelPayload([]byte(`{"userid":42,"faculty":"f","color":[1,2,3],"timestamp":1}`)); got != "42" {
		t.Fatalf("numeric userid: %q", got)
	}
	if got := writerFromPixelPayload([]byte(`{`)); got != "" {
		t.Fatalf("invalid json: %q", got)
	}
}
