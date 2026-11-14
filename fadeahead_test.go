package main

import "testing"

// Tall things (tree tops, roofs) fade MORE than before and start BEFORE the person is hidden: the closer he gets to the
// zone behind the sprite, the stronger; far away nothing. The fade itself is soft (a share per frame, never a jump).
func TestSeeThroughStartsBeforeAndGoesFurther(t *testing.T) {
	if fadeAmp <= 0.55 || fadeAmp > 0.8 {
		t.Errorf("fadeAmp %.2f: should be a bit more see-through than the old 0.55 (and never vanish)", fadeAmp)
	}
	if fadeAheadMargin < 20 {
		t.Errorf("fadeAheadMargin %.0f: too small to notice the anticipation", fadeAheadMargin)
	}
	// a tree top: sprite 60x80 at (100, 100), its sort line (trunk foot) at y = 180
	x0, y0, w, h, sortY := 100.0, 100.0, 60.0, 80.0, 180.0
	in := Vec{130, 150}
	if k := behindKRect(x0, y0, w, h, sortY, in); k != 1 {
		t.Errorf("hidden behind: %.2f, want 1", k)
	}
	// walking up towards it from the front (below the sort line) and from a side: it already starts, and grows
	for _, path := range [][]Vec{
		{{130, 179 + 40}, {130, 179 + 25}, {130, 179 + 12}, {130, 179 + 3}},
		{{102 - 40, 150}, {102 - 25, 150}, {102 - 12, 150}, {102 - 3, 150}},
		{{158 + 40, 150}, {158 + 25, 150}, {158 + 12, 150}, {158 + 3, 150}},
	} {
		prev := -1.0
		for i, p := range path {
			k := behindKRect(x0, y0, w, h, sortY, p)
			if i == 0 && k != 0 {
				t.Errorf("%v: 40 px away it already fades (%.2f)", p, k)
			}
			if i > 0 && (k <= prev || k <= 0) {
				t.Errorf("%v: %.2f, not more than %.2f a step before", p, k, prev)
			}
			if i == len(path)-1 && k < 0.8 {
				t.Errorf("%v: 3 px from the zone only %.2f", p, k)
			}
			prev = k
		}
	}
	// the fade is soft: from solid to the full see-through takes several frames, each step a small one
	fade, steps := 0.0, 0
	for fade < 0.95 {
		next := fade + (1-fade)*fadeRate
		if next-fade > 0.15 {
			t.Fatalf("a jump of %.2f in one frame", next-fade)
		}
		fade = next
		steps++
	}
	if steps < 12 {
		t.Errorf("the fade takes only %d frames", steps)
	}
}
