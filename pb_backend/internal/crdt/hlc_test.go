package crdt

import (
	"bytes"
	"testing"
)

func TestClockTickAdvancesWithWall(t *testing.T) {
	var c Clock
	x := c.Tick(100)
	if x.Millis != 100 || x.Logical != 0 {
		t.Fatalf("Tick: got %+v", x)
	}
	y := c.Tick(100)
	if y.Millis != 100 || y.Logical != 1 {
		t.Fatalf("same wall: got %+v", y)
	}
	z := c.Tick(200)
	if z.Millis != 200 || z.Logical != 0 {
		t.Fatalf("wall jump: got %+v", z)
	}
}

func TestClockWitnessPastRemote(t *testing.T) {
	var c Clock
	c.Tick(50)
	c.Witness(Clock{Millis: 100, Logical: 3}, 90)
	if c.Millis != 100 || c.Logical != 4 {
		t.Fatalf("after witness: got %+v", c)
	}
}

func TestCompareLexicographic(t *testing.T) {
	if Compare(Clock{10, 0}, Clock{20, 0}) != -1 {
		t.Fatal("expected -1")
	}
	if Compare(Clock{20, 1}, Clock{20, 0}) != 1 {
		t.Fatal("expected 1")
	}
	if Compare(Clock{5, 5}, Clock{5, 5}) != 0 {
		t.Fatal("expected 0")
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	want := Clock{Millis: 1_700_000_000_123, Logical: 42}
	buf := Encode(nil, want)
	got, err := Decode(buf)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestEncodeAppends(t *testing.T) {
	prefix := []byte("hdr:")
	buf := Encode(prefix, Clock{1, 2})
	if !bytes.HasPrefix(buf, prefix) {
		t.Fatalf("prefix lost: %q", buf)
	}
	got, err := Decode(buf)
	if err != nil || got.Millis != 1 || got.Logical != 2 {
		t.Fatalf("decode: %+v err=%v", got, err)
	}
}

func TestCompareStreamIDs(t *testing.T) {
	less, err := CompareStreamIDs("1000-0", "1000-1")
	if err != nil || less != -1 {
		t.Fatalf("less: %d %v", less, err)
	}
	eq, err := CompareStreamIDs("1000-5", "1000-5")
	if err != nil || eq != 0 {
		t.Fatalf("eq: %d %v", eq, err)
	}
	gt, err := CompareStreamIDs("1001-0", "1000-99")
	if err != nil || gt != 1 {
		t.Fatalf("gt: %d %v", gt, err)
	}
	if _, err := CompareStreamIDs("bad", "1000-0"); err == nil {
		t.Fatal("expected error")
	}
}
