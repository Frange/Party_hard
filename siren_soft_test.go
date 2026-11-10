package main

import "testing"

// TestSirenLightsAreSoft: the police car and ambulance lights were too bright and blinked hard.
// Now: about half the old alpha (160), a wider radius than before (50), no white, a gentle cut of
// the darkness, a crossfade instead of an on/off blink, and at most 3 red-blue cycles per second.
func TestSirenLightsAreSoft(t *testing.T) {
	if sirenHz > 3 {
		t.Fatalf("siren cycle %.1f Hz, max 3", sirenHz)
	}
	prevRed := -1.0
	changes := 0
	for i := 0; i <= 600; i++ { // 10 s at 60 fps
		tm := float64(i) / 60
		ls := sirenLights(500, 300, tm)
		if len(ls) != 2 {
			t.Fatalf("want a red and a blue light, got %d", len(ls))
		}
		for _, l := range ls {
			if l.col.A > 90 || l.r < 55 || l.cut > 0.25 {
				t.Fatalf("t=%.2f: light too strong: %+v", tm, l)
			}
			if l.col.R > 150 && l.col.G > 150 && l.col.B > 150 {
				t.Fatalf("t=%.2f: white flash %+v", tm, l.col)
			}
		}
		red := float64(ls[0].col.A) - float64(ls[1].col.A)
		if prevRed != -1 && (red > 0) != (prevRed > 0) {
			changes++
		}
		if i > 0 {
			step := float64(ls[0].col.A) - float64(sirenLights(500, 300, float64(i-1)/60)[0].col.A)
			if step > 12 || step < -12 {
				t.Fatalf("t=%.2f: hard blink (alpha jumps %.0f in one frame)", tm, step)
			}
		}
		prevRed = red
	}
	if hz := float64(changes) / 2 / 10; hz > 3 || hz < 0.5 {
		t.Fatalf("red/blue swaps at %.1f Hz: want visible but at most 3 Hz", hz)
	}
}
