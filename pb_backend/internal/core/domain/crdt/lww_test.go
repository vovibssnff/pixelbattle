package crdt

import "testing"

func TestDecideLWW_StreamIDBeatsOlder(t *testing.T) {
	stored := LWWState{StreamID: "1000-0", HLC: Clock{1, 0}}
	incoming := LWWState{StreamID: "1000-1", HLC: Clock{0, 0}}
	rep, path := DecideLWW(stored, incoming)
	if !rep || path != ResolvedByStreamID {
		t.Fatalf("got replace=%v path=%s", rep, path)
	}
}

func TestDecideLWW_StreamIDRejectsOlder(t *testing.T) {
	stored := LWWState{StreamID: "2000-0", HLC: Clock{0, 0}}
	incoming := LWWState{StreamID: "1000-99", HLC: Clock{99, 99}}
	rep, path := DecideLWW(stored, incoming)
	if rep || path != ResolvedByStreamID {
		t.Fatalf("got replace=%v path=%s", rep, path)
	}
}

func TestDecideLWW_EmptyStoredValidIncoming(t *testing.T) {
	stored := LWWState{StreamID: "", HLC: Clock{}}
	incoming := LWWState{StreamID: "100-0", HLC: Clock{5, 1}}
	rep, path := DecideLWW(stored, incoming)
	if !rep || path != ResolvedByStreamID {
		t.Fatalf("got replace=%v path=%s", rep, path)
	}
}

func TestDecideLWW_HLCFallbackWhenIncomingLacksStream(t *testing.T) {
	stored := LWWState{StreamID: "1000-0", HLC: Clock{10, 0}}
	incoming := LWWState{StreamID: "", HLC: Clock{20, 0}}
	rep, path := DecideLWW(stored, incoming)
	if !rep || path != ResolvedByHLCFallback {
		t.Fatalf("got replace=%v path=%s", rep, path)
	}
}

func TestDecideLWW_HLCFallbackBothMissingStream(t *testing.T) {
	stored := LWWState{HLC: Clock{10, 5}}
	incoming := LWWState{HLC: Clock{10, 6}}
	rep, path := DecideLWW(stored, incoming)
	if !rep || path != ResolvedByHLCFallback {
		t.Fatalf("got replace=%v path=%s", rep, path)
	}
}
