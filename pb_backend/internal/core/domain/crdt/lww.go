package crdt

// ResolutionPath identifies which ordering rule resolved an LWW conflict (thesis / plan §11.4).
// Adapters map these to Prometheus label values on crdt_resolved_total{path=...}.
type ResolutionPath string

const (
	// ResolvedByStreamID means both sides carried valid Redis stream IDs and those IDs were compared.
	ResolvedByStreamID ResolutionPath = "stream_id"
	// ResolvedByHLCFallback means at least one side lacked a valid stream ID, or both lacked one, so HLC tuples were compared.
	ResolvedByHLCFallback ResolutionPath = "hlc_fallback"
)

// LWWState is the minimum stored/incoming state needed to decide last-writer-wins for one coordinate.
type LWWState struct {
	StreamID string
	HLC      Clock
}

// DecideLWW returns whether incoming should replace stored.
// If both stream IDs were marked valid but CompareStreamIDs fails, comparison falls back to HLC (path hlc_fallback).
func DecideLWW(stored, incoming LWWState) (replace bool, path ResolutionPath) {
	stOK := ValidStreamID(stored.StreamID)
	inOK := ValidStreamID(incoming.StreamID)

	switch {
	case !stOK && inOK:
		// First durable write (or legacy row without stream id): accept canonical stream id.
		return true, ResolvedByStreamID
	case stOK && inOK:
		cmp, err := CompareStreamIDs(incoming.StreamID, stored.StreamID)
		if err != nil {
			c := Compare(incoming.HLC, stored.HLC)
			return c > 0, ResolvedByHLCFallback
		}
		return cmp > 0, ResolvedByStreamID
	default:
		// Missing stream id on incoming, or neither valid: plan §5 HLC fallback.
		c := Compare(incoming.HLC, stored.HLC)
		return c > 0, ResolvedByHLCFallback
	}
}
