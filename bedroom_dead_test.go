package main

import (
	"image/color"
	"testing"
)

func TestWindowPartyDeadByTier(t *testing.T) {
	for tier := tierNormal; tier <= tierGold; tier++ {
		if got := windowPartyDead(false, tier, false); got != (tier >= tierBronze) {
			t.Errorf("tier %d: dead=%v", tier, got)
		}
	}
	if windowPartyDead(true, tierGold, true) {
		t.Error("the quiet ending scene keeps its sleeping neighbours")
	}
	if !windowPartyDead(true, tierNormal, false) {
		t.Error("everything won: dead")
	}
}

// every dead guest is still (no time input), draws blood, and stays inside the window (x 606..856, y 112..192).
func TestBodyDeadBloodAndBounds(t *testing.T) {
	type rect struct {
		x, y, w, h float64
		c          color.RGBA
	}
	for i := 0; i < 6; i++ {
		var rs []rect
		fr := func(x, y, w, h float64, c color.RGBA) { rs = append(rs, rect{x, y, w, h, c}) }
		fe := func(x, y, rx, ry float64, c color.RGBA) { rs = append(rs, rect{x - rx, y - ry, 2 * rx, 2 * ry, c}) }
		dx := 606.0 + 24 + float64(i)*42
		bodyDead(fr, fe, dx, 112+80-8, i, rgb(0x335577))
		reds := 0
		for _, r := range rs {
			if r.c.R > 0x60 && r.c.G < 0x20 && r.c.B < 0x20 {
				reds++
			}
			if r.x < 606 || r.x+r.w > 856 || r.y < 112 || r.y+r.h > 192 {
				t.Errorf("guest %d: %+v leaves the window", i, r)
			}
		}
		if reds < 5 {
			t.Errorf("guest %d: only %d red rects", i, reds)
		}
	}
}
