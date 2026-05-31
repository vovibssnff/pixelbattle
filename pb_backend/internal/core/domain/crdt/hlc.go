// Package crdt holds pure last-write-wins ordering: Hybrid Logical Clocks and Redis stream IDs.
// See adr/001-crdt-hexagonal-layering.md at repo root (ADR-001).
package crdt

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrInvalidStreamID indicates a malformed Redis stream ID (expected "milliseconds-sequence").
var ErrInvalidStreamID = errors.New("invalid redis stream id")

// Clock is a Hybrid Logical Clock tuple used when Redis stream IDs are unavailable
// (partition, replay, or cross-stream merge). Wall time is Unix milliseconds;
// Logical ticks disambiguate concurrent events in the same millisecond.
type Clock struct {
	Millis  uint64
	Logical uint32
}

// Tick advances the clock for a local event, using wallMs as the physical component.
func (c *Clock) Tick(wallMs uint64) Clock {
	if wallMs > c.Millis {
		c.Millis = wallMs
		c.Logical = 0
	} else {
		c.Logical++
	}
	return *c
}

// Witness merges a remote clock and the local physical reading, then advances strictly past remote.
func (c *Clock) Witness(remote Clock, wallMs uint64) Clock {
	if wallMs > c.Millis {
		c.Millis = wallMs
		c.Logical = 0
	}
	switch {
	case remote.Millis > c.Millis:
		c.Millis = remote.Millis
		c.Logical = remote.Logical + 1
	case remote.Millis == c.Millis && remote.Logical >= c.Logical:
		c.Logical = remote.Logical + 1
	default:
		c.Logical++
	}
	return *c
}

// Compare orders two HLC tuples lexicographically (millis, then logical).
func Compare(a, b Clock) int {
	switch {
	case a.Millis < b.Millis:
		return -1
	case a.Millis > b.Millis:
		return 1
	case a.Logical < b.Logical:
		return -1
	case a.Logical > b.Logical:
		return 1
	default:
		return 0
	}
}

const encodedClockLen = 12

// Encode appends a fixed-width big-endian encoding of c to buf and returns the extended slice.
func Encode(buf []byte, c Clock) []byte {
	n := len(buf)
	out := make([]byte, n+encodedClockLen)
	copy(out, buf)
	binary.BigEndian.PutUint64(out[n:n+8], c.Millis)
	binary.BigEndian.PutUint32(out[n+8:n+12], c.Logical)
	return out
}

// Decode parses Encode output at the end of data (last 12 bytes).
func Decode(data []byte) (Clock, error) {
	if len(data) < encodedClockLen {
		return Clock{}, fmt.Errorf("crdt: need %d bytes, got %d", encodedClockLen, len(data))
	}
	i := len(data) - encodedClockLen
	return Clock{
		Millis:  binary.BigEndian.Uint64(data[i : i+8]),
		Logical: binary.BigEndian.Uint32(data[i+8 : i+12]),
	}, nil
}

// CompareStreamIDs compares Redis stream IDs (milliseconds-sequence). Returns -1 if a < b, 0 if equal, +1 if a > b.
func CompareStreamIDs(a, b string) (int, error) {
	ams, aseq, err := parseStreamID(a)
	if err != nil {
		return 0, err
	}
	bms, bseq, err := parseStreamID(b)
	if err != nil {
		return 0, err
	}
	switch {
	case ams < bms:
		return -1, nil
	case ams > bms:
		return 1, nil
	case aseq < bseq:
		return -1, nil
	case aseq > bseq:
		return 1, nil
	default:
		return 0, nil
	}
}

// ValidStreamID reports whether s parses as a Redis stream ID.
func ValidStreamID(s string) bool {
	_, _, err := parseStreamID(s)
	return err == nil
}

func parseStreamID(s string) (ms uint64, seq uint64, err error) {
	s = strings.TrimSpace(s)
	dash := strings.IndexByte(s, '-')
	if dash <= 0 || dash >= len(s)-1 {
		return 0, 0, fmt.Errorf("%w: %q", ErrInvalidStreamID, s)
	}
	ms, err = strconv.ParseUint(s[:dash], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: %w", ErrInvalidStreamID, err)
	}
	seq, err = strconv.ParseUint(s[dash+1:], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("%w: %w", ErrInvalidStreamID, err)
	}
	return ms, seq, nil
}
