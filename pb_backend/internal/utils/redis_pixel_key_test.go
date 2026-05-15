package utils

import "testing"

func TestParseRedisPixelKeyLegacy(t *testing.T) {
	x, y, err := ParseRedisPixelKey("pixel:2:3")
	if err != nil || x != 3 || y != 2 {
		t.Fatalf("got x=%d y=%d err=%v", x, y, err)
	}
}

func TestParseRedisPixelKeyHashTag(t *testing.T) {
	x, y, err := ParseRedisPixelKey("pixel:{2:3}")
	if err != nil || x != 3 || y != 2 {
		t.Fatalf("got x=%d y=%d err=%v", x, y, err)
	}
}
